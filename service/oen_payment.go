package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const (
	oenProductionAPIBaseURL = "https://payment-api.oen.tw"
	oenTestingAPIBaseURL    = "https://payment-api.testing.oen.tw"
	oenResponseBodyLimit    = 1 << 20
)

var oenMerchantIDPattern = regexp.MustCompile(`^[A-Za-z0-9-]+$`)

type OenPaymentClient struct {
	apiBaseURL     string
	checkoutDomain string
	token          string
	merchantID     string
	httpClient     *http.Client
}

type OenCheckoutRequest struct {
	MerchantID string `json:"merchantId"`
	Amount     int64  `json:"amount"`
	Currency   string `json:"currency"`
	OrderID    string `json:"orderId"`
	SuccessURL string `json:"successUrl"`
	FailureURL string `json:"failureUrl"`
	Use3D      bool   `json:"use3d"`
	CustomID   string `json:"customId,omitempty"`
	UserID     string `json:"userId,omitempty"`
	Note       string `json:"note,omitempty"`
}

type OenCheckoutData struct {
	ID             string `json:"id"`
	TransactionHID string `json:"transactionHid"`
}

type OenTransaction struct {
	ID             string `json:"id"`
	TransactionID  string `json:"transactionId"`
	Action         string `json:"action"`
	Amount         int64  `json:"amount"`
	Status         string `json:"status"`
	OrderID        string `json:"orderId"`
	PaymentMethod  string `json:"paymentMethod"`
	TransactionHID string `json:"transactionHid"`
}

type oenAPIResponse[T any] struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

func NewOenPaymentClient(token string, merchantID string, testMode bool) (*OenPaymentClient, error) {
	token = strings.TrimSpace(token)
	merchantID = strings.TrimSpace(merchantID)
	if token == "" {
		return nil, fmt.Errorf("OEN API token is empty")
	}
	if !oenMerchantIDPattern.MatchString(merchantID) {
		return nil, fmt.Errorf("invalid OEN merchant ID")
	}

	apiBaseURL := oenProductionAPIBaseURL
	checkoutDomain := merchantID + ".oen.tw"
	if testMode {
		apiBaseURL = oenTestingAPIBaseURL
		checkoutDomain = merchantID + ".testing.oen.tw"
	}

	return &OenPaymentClient{
		apiBaseURL:     apiBaseURL,
		checkoutDomain: checkoutDomain,
		token:          token,
		merchantID:     merchantID,
		httpClient:     GetHttpClient(),
	}, nil
}

func (client *OenPaymentClient) CreateCheckout(ctx context.Context, request OenCheckoutRequest) (*OenCheckoutData, error) {
	request.MerchantID = client.merchantID
	var response oenAPIResponse[OenCheckoutData]
	if err := client.do(ctx, http.MethodPost, "/checkout", request, &response); err != nil {
		return nil, err
	}
	if response.Code != "S0000" {
		return nil, fmt.Errorf("OEN checkout failed: code=%s message=%s", response.Code, response.Message)
	}
	if strings.TrimSpace(response.Data.ID) == "" {
		return nil, fmt.Errorf("OEN checkout response missing transaction ID")
	}
	return &response.Data, nil
}

func (client *OenPaymentClient) GetTransaction(ctx context.Context, transactionID string) (*OenTransaction, error) {
	transactionID = strings.TrimSpace(transactionID)
	if transactionID == "" {
		return nil, fmt.Errorf("OEN transaction ID is empty")
	}

	var response oenAPIResponse[OenTransaction]
	path := "/transactions/" + url.PathEscape(transactionID)
	if err := client.do(ctx, http.MethodGet, path, nil, &response); err != nil {
		return nil, err
	}
	if response.Code != "S0000" {
		return nil, fmt.Errorf("OEN transaction query failed: code=%s message=%s", response.Code, response.Message)
	}
	return &response.Data, nil
}

func (client *OenPaymentClient) CheckoutURL(transactionID string) string {
	return "https://" + client.checkoutDomain + "/checkout/" + url.PathEscape(transactionID)
}

func (client *OenPaymentClient) do(ctx context.Context, method string, path string, requestBody any, responseBody any) error {
	var body io.Reader
	if requestBody != nil {
		payload, err := common.Marshal(requestBody)
		if err != nil {
			return err
		}
		body = bytes.NewReader(payload)
	}

	request, err := http.NewRequestWithContext(ctx, method, client.apiBaseURL+path, body)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+client.token)

	response, err := client.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	limitedBody := io.LimitReader(response.Body, oenResponseBodyLimit)
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		bodyBytes, _ := io.ReadAll(limitedBody)
		return fmt.Errorf("OEN API returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}
	if err := common.DecodeJson(limitedBody, responseBody); err != nil {
		return fmt.Errorf("decode OEN API response: %w", err)
	}
	return nil
}
