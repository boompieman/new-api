package controller

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type OenPayRequest struct {
	Amount int64 `json:"amount"`
}

type OenWebhookPayload struct {
	MerchantID     string `json:"merchantId"`
	Success        bool   `json:"success"`
	ID             string `json:"id"`
	Purpose        string `json:"purpose"`
	Status         string `json:"status"`
	TransactionHID string `json:"transactionHid"`
	Action         string `json:"action"`
	Amount         int64  `json:"amount"`
	Currency       string `json:"currency"`
	OrderID        string `json:"orderId"`
	PaymentMethod  string `json:"paymentMethod"`
	Message        string `json:"message"`
}

func getOenMinTopUp() int64 {
	minimum := int64(setting.OenMinTopUp)
	if operation_setting.GetQuotaDisplayType() != operation_setting.QuotaDisplayTypeTokens {
		return minimum
	}

	displayMinimum, clamp := common.QuotaFromDecimalChecked(
		decimal.NewFromInt(minimum).Mul(decimal.NewFromFloat(common.QuotaPerUnit)),
	)
	if clamp != nil {
		return int64(common.MaxQuota)
	}
	return int64(displayMinimum)
}

func normalizeOenTopUpAmount(amount int64) (int64, error) {
	if amount <= 0 {
		return 0, errors.New("充值数量必须大于 0")
	}

	normalized := amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		normalized = decimal.NewFromInt(amount).Div(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart()
	}
	if normalized <= 0 {
		return 0, errors.New("无效的充值额度")
	}

	quota, clamp := common.QuotaFromDecimalChecked(
		decimal.NewFromInt(normalized).Mul(decimal.NewFromFloat(common.QuotaPerUnit)),
	)
	if clamp != nil || quota <= 0 {
		return 0, errors.New("无效的充值额度")
	}
	return normalized, nil
}

func getOenPayMoney(requestAmount int64, normalizedAmount int64, group string) (int64, error) {
	unitPrice := setting.OenUnitPriceTWD
	if unitPrice <= 0 || math.IsNaN(unitPrice) || math.IsInf(unitPrice, 0) {
		return 0, errors.New("OEN 支付单价配置无效")
	}

	groupRatio := common.GetTopupGroupRatio(group)
	if groupRatio <= 0 || math.IsNaN(groupRatio) || math.IsInf(groupRatio, 0) {
		return 0, errors.New("用户分组充值倍率无效")
	}

	discount := 1.0
	if configured, ok := operation_setting.GetPaymentSetting().AmountDiscount[int(requestAmount)]; ok && configured > 0 {
		discount = configured
	}
	if math.IsNaN(discount) || math.IsInf(discount, 0) {
		return 0, errors.New("充值折扣配置无效")
	}

	paymentAmount := decimal.NewFromInt(normalizedAmount).
		Mul(decimal.NewFromFloat(unitPrice)).
		Mul(decimal.NewFromFloat(groupRatio)).
		Mul(decimal.NewFromFloat(discount)).
		Round(0)
	if paymentAmount.LessThanOrEqual(decimal.Zero) || paymentAmount.GreaterThan(decimal.NewFromInt(math.MaxInt32)) {
		return 0, errors.New("OEN 支付金额超出允许范围")
	}
	return paymentAmount.IntPart(), nil
}

func RequestOenAmount(c *gin.Context) {
	if !isOenTopUpEnabled() {
		common.ApiErrorMsg(c, "OEN 支付未启用或配置不完整")
		return
	}

	var request OenPayRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if request.Amount < getOenMinTopUp() {
		common.ApiErrorMsg(c, fmt.Sprintf("充值数量不能小于 %d", getOenMinTopUp()))
		return
	}
	if rejectInvalidTopUpQuota(c, c.GetInt("id"), request.Amount) {
		return
	}

	normalizedAmount, err := normalizeOenTopUpAmount(request.Amount)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	group, err := model.GetUserGroup(c.GetInt("id"), true)
	if err != nil {
		common.ApiErrorMsg(c, "获取用户分组失败")
		return
	}
	paymentAmount, err := getOenPayMoney(request.Amount, normalizedAmount, group)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, strconv.FormatInt(paymentAmount, 10))
}

func RequestOenPay(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}
	if !isOenTopUpEnabled() {
		common.ApiErrorMsg(c, "OEN 支付未启用或配置不完整")
		return
	}

	var request OenPayRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if request.Amount < getOenMinTopUp() {
		common.ApiErrorMsg(c, fmt.Sprintf("充值数量不能小于 %d", getOenMinTopUp()))
		return
	}

	userID := c.GetInt("id")
	if rejectInvalidTopUpQuota(c, userID, request.Amount) {
		return
	}
	normalizedAmount, err := normalizeOenTopUpAmount(request.Amount)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	group, err := model.GetUserGroup(userID, true)
	if err != nil {
		common.ApiErrorMsg(c, "获取用户分组失败")
		return
	}
	paymentAmount, err := getOenPayMoney(request.Amount, normalizedAmount, group)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	client, err := service.NewOenPaymentClient(setting.OenApiToken, setting.OenMerchantID, setting.OenTestMode)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("OEN client 初始化失败 user_id=%d error=%q", userID, err.Error()))
		common.ApiErrorMsg(c, "OEN 支付配置错误")
		return
	}

	tradeNo := fmt.Sprintf("OEN-%d-%d-%s", userID, time.Now().UnixMilli(), common.GetRandomString(6))
	topUp := &model.TopUp{
		UserId:          userID,
		Amount:          normalizedAmount,
		Money:           float64(paymentAmount),
		TradeNo:         tradeNo,
		PaymentMethod:   model.PaymentMethodOen,
		PaymentProvider: model.PaymentProviderOen,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	if err := topUp.Insert(); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("OEN 创建充值订单失败 user_id=%d trade_no=%s error=%q", userID, tradeNo, err.Error()))
		common.ApiErrorMsg(c, "创建订单失败")
		return
	}

	checkout, err := client.CreateCheckout(c.Request.Context(), service.OenCheckoutRequest{
		Amount:     paymentAmount,
		Currency:   "TWD",
		OrderID:    tradeNo,
		SuccessURL: paymentReturnPath("/wallet?show_history=true"),
		FailureURL: paymentReturnPath("/wallet"),
		Use3D:      setting.OenUse3D,
		CustomID:   tradeNo,
		UserID:     strconv.Itoa(userID),
		Note:       fmt.Sprintf("Top up %d", request.Amount),
	})
	if err != nil {
		topUp.Status = common.TopUpStatusFailed
		_ = topUp.Update()
		logger.LogError(c.Request.Context(), fmt.Sprintf("OEN 建立 checkout 失败 user_id=%d trade_no=%s error=%q", userID, tradeNo, err.Error()))
		common.ApiErrorMsg(c, "建立支付页面失败")
		return
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("OEN 充值订单创建成功 user_id=%d trade_no=%s transaction_id=%s amount=%d money=%d", userID, tradeNo, checkout.ID, request.Amount, paymentAmount))
	common.ApiSuccess(c, gin.H{
		"pay_link": client.CheckoutURL(checkout.ID),
		"order_id": tradeNo,
	})
}

func validateOenTransaction(topUp *model.TopUp, payload *OenWebhookPayload, transaction *service.OenTransaction) error {
	if transaction.TransactionID != payload.ID {
		return errors.New("OEN transaction ID mismatch")
	}
	if transaction.OrderID != payload.OrderID || transaction.OrderID != topUp.TradeNo {
		return errors.New("OEN order ID mismatch")
	}
	if transaction.Action != "onetime" {
		return errors.New("OEN transaction action mismatch")
	}
	expectedAmount := decimal.NewFromFloat(topUp.Money).Round(0).IntPart()
	if transaction.Amount != expectedAmount {
		return errors.New("OEN transaction amount mismatch")
	}
	return nil
}

func OenWebhook(c *gin.Context) {
	if !isOenWebhookEnabled() {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("OEN webhook 被拒绝 reason=webhook_disabled client_ip=%s", c.ClientIP()))
		c.JSON(http.StatusServiceUnavailable, gin.H{"received": false})
		return
	}

	var payload OenWebhookPayload
	if err := common.DecodeJson(c.Request.Body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"received": false})
		return
	}
	payload.MerchantID = strings.TrimSpace(payload.MerchantID)
	payload.ID = strings.TrimSpace(payload.ID)
	payload.OrderID = strings.TrimSpace(payload.OrderID)
	if payload.MerchantID != strings.TrimSpace(setting.OenMerchantID) ||
		payload.ID == "" || payload.OrderID == "" || payload.Purpose != "charge" ||
		payload.Action != "onetime" || payload.Currency != "TWD" {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("OEN webhook 基本资料验证失败 merchant_id=%q transaction_id=%q order_id=%q purpose=%q action=%q currency=%q client_ip=%s", payload.MerchantID, payload.ID, payload.OrderID, payload.Purpose, payload.Action, payload.Currency, c.ClientIP()))
		c.JSON(http.StatusBadRequest, gin.H{"received": false})
		return
	}

	topUp := model.GetTopUpByTradeNo(payload.OrderID)
	if topUp == nil {
		c.JSON(http.StatusNotFound, gin.H{"received": false})
		return
	}
	if topUp.PaymentProvider != model.PaymentProviderOen {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("OEN webhook 订单支付网关不匹配 trade_no=%s provider=%s client_ip=%s", payload.OrderID, topUp.PaymentProvider, c.ClientIP()))
		c.JSON(http.StatusBadRequest, gin.H{"received": false})
		return
	}
	if topUp.Status == common.TopUpStatusSuccess {
		c.JSON(http.StatusOK, gin.H{"received": true})
		return
	}

	client, err := service.NewOenPaymentClient(setting.OenApiToken, setting.OenMerchantID, setting.OenTestMode)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"received": false})
		return
	}
	transaction, err := client.GetTransaction(c.Request.Context(), payload.ID)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("OEN webhook 查询交易失败 transaction_id=%s trade_no=%s error=%q", payload.ID, payload.OrderID, err.Error()))
		c.JSON(http.StatusBadGateway, gin.H{"received": false})
		return
	}
	if err := validateOenTransaction(topUp, &payload, transaction); err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("OEN webhook 远端交易验证失败 transaction_id=%s trade_no=%s error=%q client_ip=%s", payload.ID, payload.OrderID, err.Error(), c.ClientIP()))
		c.JSON(http.StatusBadRequest, gin.H{"received": false})
		return
	}

	switch transaction.Status {
	case "charged", "claimed":
		if err := model.RechargeOen(payload.OrderID, c.ClientIP()); err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("OEN webhook 充值失败 transaction_id=%s trade_no=%s error=%q", payload.ID, payload.OrderID, err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"received": false})
			return
		}
	case "failed":
		if err := model.UpdatePendingTopUpStatus(payload.OrderID, model.PaymentProviderOen, common.TopUpStatusFailed); err != nil &&
			!errors.Is(err, model.ErrTopUpStatusInvalid) {
			c.JSON(http.StatusInternalServerError, gin.H{"received": false})
			return
		}
	case "initiated", "charging":
		// The checkout is still in progress. Keep the local order pending.
	default:
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("OEN webhook 收到未知交易状态 transaction_id=%s trade_no=%s status=%q", payload.ID, payload.OrderID, transaction.Status))
		c.JSON(http.StatusBadRequest, gin.H{"received": false})
		return
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("OEN webhook 处理完成 transaction_id=%s transaction_hid=%s trade_no=%s remote_status=%s payment_method=%s", payload.ID, payload.TransactionHID, payload.OrderID, transaction.Status, payload.PaymentMethod))
	c.JSON(http.StatusOK, gin.H{"received": true})
}
