package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOenPaymentClientCreateCheckout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, "/checkout", request.URL.Path)
		assert.Equal(t, "Bearer test-token", request.Header.Get("Authorization"))

		var payload OenCheckoutRequest
		require.NoError(t, common.DecodeJson(request.Body, &payload))
		assert.Equal(t, "merchant", payload.MerchantID)
		assert.Equal(t, int64(1200), payload.Amount)
		assert.Equal(t, "ORDER-1", payload.OrderID)

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"code":"S0000","message":"","data":{"id":"tx-1","transactionHid":"P123"}}`))
	}))
	defer server.Close()

	client := &OenPaymentClient{
		apiBaseURL:     server.URL,
		checkoutDomain: "merchant.testing.oen.tw",
		token:          "test-token",
		merchantID:     "merchant",
		httpClient:     server.Client(),
	}
	checkout, err := client.CreateCheckout(context.Background(), OenCheckoutRequest{
		Amount:     1200,
		Currency:   "TWD",
		OrderID:    "ORDER-1",
		SuccessURL: "https://example.com/success",
		FailureURL: "https://example.com/failure",
		Use3D:      true,
	})
	require.NoError(t, err)
	assert.Equal(t, "tx-1", checkout.ID)
	assert.Equal(t, "P123", checkout.TransactionHID)
	assert.Equal(t, "https://merchant.testing.oen.tw/checkout/tx-1", client.CheckoutURL(checkout.ID))
}

func TestOenPaymentClientGetTransaction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, http.MethodGet, request.Method)
		assert.Equal(t, "/transactions/tx-1", request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"code":"S0000","message":"","data":{"id":"P123","transactionId":"tx-1","action":"onetime","amount":1200,"status":"charged","orderId":"ORDER-1"}}`))
	}))
	defer server.Close()

	client := &OenPaymentClient{
		apiBaseURL: server.URL,
		token:      "test-token",
		merchantID: "merchant",
		httpClient: server.Client(),
	}
	transaction, err := client.GetTransaction(context.Background(), "tx-1")
	require.NoError(t, err)
	assert.Equal(t, "tx-1", transaction.TransactionID)
	assert.Equal(t, "ORDER-1", transaction.OrderID)
	assert.Equal(t, int64(1200), transaction.Amount)
	assert.Equal(t, "charged", transaction.Status)
}

func TestNewOenPaymentClientRejectsInvalidConfiguration(t *testing.T) {
	_, err := NewOenPaymentClient("", "merchant", true)
	require.Error(t, err)

	_, err = NewOenPaymentClient("token", "merchant.example.com", true)
	require.Error(t, err)
}
