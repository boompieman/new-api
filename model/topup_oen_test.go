package model

import (
	"math"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRechargeOenCreditsQuotaOnce(t *testing.T) {
	truncateTables(t)
	insertUserForPaymentGuardTest(t, 201, 100)
	insertTopUpForPaymentGuardTest(t, "oen-success", 201, PaymentProviderOen)

	require.NoError(t, RechargeOen("oen-success", "127.0.0.1"))
	require.NoError(t, RechargeOen("oen-success", "127.0.0.1"))

	expectedQuota := 100 + common.QuotaFromDecimal(decimal.NewFromInt(2).Mul(decimal.NewFromFloat(common.QuotaPerUnit)))
	assert.Equal(t, expectedQuota, getUserQuotaForPaymentGuardTest(t, 201))
	assert.Equal(t, common.TopUpStatusSuccess, getTopUpStatusForPaymentGuardTest(t, "oen-success"))
}

func TestRechargeOenRejectsProviderMismatch(t *testing.T) {
	truncateTables(t)
	insertUserForPaymentGuardTest(t, 202, 0)
	insertTopUpForPaymentGuardTest(t, "oen-mismatch", 202, PaymentProviderStripe)

	err := RechargeOen("oen-mismatch", "127.0.0.1")
	require.ErrorIs(t, err, ErrPaymentMethodMismatch)
	assert.Equal(t, 0, getUserQuotaForPaymentGuardTest(t, 202))
	assert.Equal(t, common.TopUpStatusPending, getTopUpStatusForPaymentGuardTest(t, "oen-mismatch"))
}

func TestRechargeOenRejectsQuotaOverflow(t *testing.T) {
	truncateTables(t)
	insertUserForPaymentGuardTest(t, 203, 0)
	topUp := &TopUp{
		UserId:          203,
		Amount:          math.MaxInt64,
		Money:           1000,
		TradeNo:         "oen-overflow",
		PaymentMethod:   PaymentMethodOen,
		PaymentProvider: PaymentProviderOen,
		Status:          common.TopUpStatusPending,
		CreateTime:      time.Now().Unix(),
	}
	require.NoError(t, topUp.Insert())

	err := RechargeOen("oen-overflow", "127.0.0.1")
	require.Error(t, err)
	assert.Equal(t, 0, getUserQuotaForPaymentGuardTest(t, 203))
	assert.Equal(t, common.TopUpStatusPending, getTopUpStatusForPaymentGuardTest(t, "oen-overflow"))
}
