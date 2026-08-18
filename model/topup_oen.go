package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const (
	PaymentMethodOen   = "oen"
	PaymentProviderOen = "oen"
)

func RechargeOen(tradeNo string, callerIP string) error {
	if tradeNo == "" {
		return errors.New("未提供支付单号")
	}

	var quotaToAdd int
	topUp := &TopUp{}
	refCol := "`trade_no`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		refCol = `"trade_no"`
	}

	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).Where(refCol+" = ?", tradeNo).First(topUp).Error; err != nil {
			return ErrTopUpNotFound
		}
		if topUp.PaymentProvider != PaymentProviderOen {
			return ErrPaymentMethodMismatch
		}
		if topUp.Status == common.TopUpStatusSuccess {
			return nil
		}
		if topUp.Status != common.TopUpStatusPending {
			return ErrTopUpStatusInvalid
		}

		quota, err := common.QuotaFromDecimalStrict(
			decimal.NewFromInt(topUp.Amount).Mul(decimal.NewFromFloat(common.QuotaPerUnit)),
		)
		if err != nil || quota <= 0 {
			return ErrInvalidTopUpQuota
		}
		quotaToAdd = quota

		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		if err := tx.Save(topUp).Error; err != nil {
			return err
		}
		return creditTopUpQuota(tx, topUp.UserId, quotaToAdd, nil)
	})
	if err != nil {
		common.SysError("OEN topup failed: " + err.Error())
		return err
	}

	if quotaToAdd > 0 {
		syncCreditUserQuotaCache(topUp.UserId, quotaToAdd, "OEN topup")
		RecordTopupLog(
			topUp.UserId,
			fmt.Sprintf("OEN充值成功，充值额度: %v，支付金额: %.0f TWD", logger.FormatQuota(quotaToAdd), topUp.Money),
			callerIP,
			topUp.PaymentMethod,
			PaymentMethodOen,
		)
	}
	return nil
}
