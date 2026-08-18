package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/stretchr/testify/require"
)

func TestValidateOenTransaction(t *testing.T) {
	topUp := &model.TopUp{
		TradeNo: "ORDER-1",
		Money:   1200,
	}
	payload := &OenWebhookPayload{
		ID:      "tx-1",
		OrderID: "ORDER-1",
	}
	valid := &service.OenTransaction{
		TransactionID: "tx-1",
		OrderID:       "ORDER-1",
		Action:        "onetime",
		Amount:        1200,
	}
	require.NoError(t, validateOenTransaction(topUp, payload, valid))

	testCases := []struct {
		name        string
		transaction service.OenTransaction
	}{
		{
			name: "transaction id mismatch",
			transaction: service.OenTransaction{
				TransactionID: "tx-other",
				OrderID:       "ORDER-1",
				Action:        "onetime",
				Amount:        1200,
			},
		},
		{
			name: "order mismatch",
			transaction: service.OenTransaction{
				TransactionID: "tx-1",
				OrderID:       "ORDER-2",
				Action:        "onetime",
				Amount:        1200,
			},
		},
		{
			name: "subscription action",
			transaction: service.OenTransaction{
				TransactionID: "tx-1",
				OrderID:       "ORDER-1",
				Action:        "subscription",
				Amount:        1200,
			},
		},
		{
			name: "amount mismatch",
			transaction: service.OenTransaction{
				TransactionID: "tx-1",
				OrderID:       "ORDER-1",
				Action:        "onetime",
				Amount:        1199,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Error(t, validateOenTransaction(topUp, payload, &testCase.transaction))
		})
	}
}
