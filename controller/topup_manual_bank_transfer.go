package controller

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

const manualBankTransferPaymentType = "manual_bank_transfer"

func isManualBankTransferTopUpEnabled() bool {
	config := operation_setting.GetManualBankTransferConfig()
	return operation_setting.IsPaymentComplianceConfirmed() &&
		config.Enabled &&
		strings.TrimSpace(config.BankName) != "" &&
		strings.TrimSpace(config.AccountName) != "" &&
		strings.TrimSpace(config.AccountNumber) != ""
}

func appendManualBankTransferPayMethod(payMethods []map[string]string) []map[string]string {
	if !isManualBankTransferTopUpEnabled() {
		return payMethods
	}
	for _, method := range payMethods {
		if method["type"] == manualBankTransferPaymentType {
			return payMethods
		}
	}
	return append(payMethods, map[string]string{
		"name":      "Manual Bank Transfer",
		"type":      manualBankTransferPaymentType,
		"icon":      "LuLandmark",
		"color":     "#0F766E",
		"min_topup": strconv.FormatInt(getMinTopup(), 10),
	})
}

// CreateManualBankTransferTopUp adapts manual transfers to the existing TopUp
// lifecycle. AdminCompleteTopUp remains the only path that credits the order.
func CreateManualBankTransferTopUp(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}
	if !isManualBankTransferTopUpEnabled() {
		common.ApiErrorMsg(c, "手动汇款未启用或银行资料不完整")
		return
	}

	var request AmountRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if request.Amount < getMinTopup() {
		common.ApiErrorMsg(c, fmt.Sprintf("充值数量不能小于 %d", getMinTopup()))
		return
	}

	userID := c.GetInt("id")
	if rejectInvalidTopUpQuota(c, userID, request.Amount) {
		return
	}
	group, err := model.GetUserGroup(userID, true)
	if err != nil {
		common.ApiErrorMsg(c, "获取用户分组失败")
		return
	}
	payMoney := getPayMoney(request.Amount, group)
	if payMoney < 0.01 || math.IsNaN(payMoney) || math.IsInf(payMoney, 0) {
		common.ApiErrorMsg(c, "汇款金额无效")
		return
	}

	normalizedAmount := request.Amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		normalizedAmount = decimal.NewFromInt(request.Amount).
			Div(decimal.NewFromFloat(common.QuotaPerUnit)).
			IntPart()
	}
	if normalizedAmount <= 0 {
		common.ApiErrorMsg(c, "无效的充值额度")
		return
	}

	tradeNo := fmt.Sprintf("MBT-%d-%d-%s", userID, time.Now().UnixMilli(), common.GetRandomString(6))
	topUp := &model.TopUp{
		UserId:          userID,
		Amount:          normalizedAmount,
		Money:           payMoney,
		TradeNo:         tradeNo,
		PaymentMethod:   manualBankTransferPaymentType,
		PaymentProvider: manualBankTransferPaymentType,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	if err := topUp.Insert(); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("手动汇款订单创建失败 user_id=%d amount=%d error=%q", userID, request.Amount, err.Error()))
		common.ApiErrorMsg(c, "创建订单失败")
		return
	}

	config := operation_setting.GetManualBankTransferConfig()
	logger.LogInfo(c.Request.Context(), fmt.Sprintf("手动汇款订单创建成功 user_id=%d trade_no=%s amount=%d money=%.2f", userID, tradeNo, request.Amount, payMoney))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "success",
		"data": gin.H{
			"order_id":       tradeNo,
			"payment_amount": payMoney,
			"bank_name":      strings.TrimSpace(config.BankName),
			"bank_code":      strings.TrimSpace(config.BankCode),
			"branch_name":    strings.TrimSpace(config.BranchName),
			"account_name":   strings.TrimSpace(config.AccountName),
			"account_number": strings.TrimSpace(config.AccountNumber),
			"instructions":   strings.TrimSpace(config.Instructions),
			"status":         common.TopUpStatusPending,
		},
	})
}
