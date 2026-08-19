package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestCreateManualBankTransferTopUpWaitsForAdminCompletionBeforeCreditingQuota(t *testing.T) {
	oldDB := model.DB
	oldLogDB := model.LOG_DB
	oldMainDatabaseType := common.MainDatabaseType()
	oldLogDatabaseType := common.LogDatabaseType()
	oldQuotaPerUnit := common.QuotaPerUnit
	oldRedisEnabled := common.RedisEnabled
	oldPrice := operation_setting.Price
	oldMinTopUp := operation_setting.MinTopUp
	oldDisplayType := operation_setting.GetGeneralSetting().QuotaDisplayType
	oldPaymentSetting := *operation_setting.GetPaymentSetting()
	oldTransferConfig := *operation_setting.GetManualBankTransferConfig()

	db, err := gorm.Open(sqlite.Open("file:manual-bank-transfer?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.TopUp{}))
	model.DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	common.SetLogDatabaseType(common.DatabaseTypeSQLite)
	t.Setenv("LOG_SQL_DSN", "")
	require.NoError(t, model.InitLogDB())
	common.QuotaPerUnit = 500000
	common.RedisEnabled = false
	operation_setting.Price = 35
	operation_setting.MinTopUp = 1
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD
	operation_setting.GetPaymentSetting().ComplianceConfirmed = true
	operation_setting.GetPaymentSetting().ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion
	operation_setting.GetPaymentSetting().AmountDiscount = map[int]float64{}
	*operation_setting.GetManualBankTransferConfig() = operation_setting.ManualBankTransferConfig{
		Enabled:       true,
		BankName:      "Example Bank",
		BankCode:      "123",
		BranchName:    "Main Branch",
		AccountName:   "Example Company",
		AccountNumber: "0123456789",
		Instructions:  "Send the order number after transferring.",
	}
	t.Cleanup(func() {
		model.DB = oldDB
		model.LOG_DB = oldLogDB
		common.SetMainDatabaseType(oldMainDatabaseType)
		common.SetLogDatabaseType(oldLogDatabaseType)
		common.QuotaPerUnit = oldQuotaPerUnit
		common.RedisEnabled = oldRedisEnabled
		operation_setting.Price = oldPrice
		operation_setting.MinTopUp = oldMinTopUp
		operation_setting.GetGeneralSetting().QuotaDisplayType = oldDisplayType
		*operation_setting.GetPaymentSetting() = oldPaymentSetting
		*operation_setting.GetManualBankTransferConfig() = oldTransferConfig
		sqlDB, dbErr := db.DB()
		if dbErr == nil {
			require.NoError(t, sqlDB.Close())
		}
	})

	user := model.User{
		Id:       42,
		Username: "manual_transfer_user",
		Group:    "default",
		Quota:    1000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(&user).Error)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", user.Id)
	ctx.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/user/topup/manual-bank-transfer",
		strings.NewReader(`{"amount":10}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	CreateManualBankTransferTopUp(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			OrderID       string  `json:"order_id"`
			PaymentAmount float64 `json:"payment_amount"`
			BankName      string  `json:"bank_name"`
			AccountNumber string  `json:"account_number"`
			Status        string  `json:"status"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.NotEmpty(t, response.Data.OrderID)
	assert.Equal(t, float64(350), response.Data.PaymentAmount)
	assert.Equal(t, "Example Bank", response.Data.BankName)
	assert.Equal(t, "0123456789", response.Data.AccountNumber)
	assert.Equal(t, common.TopUpStatusPending, response.Data.Status)

	var topUp model.TopUp
	require.NoError(t, db.Where("trade_no = ?", response.Data.OrderID).First(&topUp).Error)
	assert.Equal(t, user.Id, topUp.UserId)
	assert.Equal(t, int64(10), topUp.Amount)
	assert.Equal(t, float64(350), topUp.Money)
	assert.Equal(t, manualBankTransferPaymentType, topUp.PaymentProvider)
	assert.Equal(t, common.TopUpStatusPending, topUp.Status)

	var reloaded model.User
	require.NoError(t, db.First(&reloaded, user.Id).Error)
	assert.Equal(t, 1000, reloaded.Quota)

	require.NoError(t, model.ManualCompleteTopUp(response.Data.OrderID, "127.0.0.1"))
	require.NoError(t, db.First(&reloaded, user.Id).Error)
	assert.Equal(t, 5001000, reloaded.Quota)

	// Completing the same order again is idempotent and must not credit twice.
	require.NoError(t, model.ManualCompleteTopUp(response.Data.OrderID, "127.0.0.1"))
	require.NoError(t, db.First(&reloaded, user.Id).Error)
	assert.Equal(t, 5001000, reloaded.Quota)
}
