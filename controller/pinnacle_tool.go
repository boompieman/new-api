package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type pinnacleToolConfigRequest struct {
	Enabled  bool   `json:"enabled"`
	APIToken string `json:"api_token"`
}

func GetPinnacleToolConfig(c *gin.Context) {
	settings := operation_setting.GetPinnacleToolSetting()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"enabled":                     settings.Enabled,
			"token_configured":            operation_setting.IsPinnacleToolTokenConfigured(),
			"price_per_thousand_requests": operation_setting.PinnacleToolPricePerThousandUSD,
			"currency":                    "USD",
		},
	})
}

func UpdatePinnacleToolConfig(c *gin.Context) {
	var request pinnacleToolConfigRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid Pinnacle tool configuration"})
		return
	}

	apiToken := strings.TrimSpace(request.APIToken)
	if len(apiToken) > 4096 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Apify API token is too long"})
		return
	}
	if request.Enabled && apiToken == "" && !operation_setting.IsPinnacleToolTokenConfigured() {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "An Apify API token is required before enabling the Pinnacle tool"})
		return
	}

	updates := map[string]string{
		"pinnacle_tool.enabled": strconv.FormatBool(request.Enabled),
	}
	if apiToken != "" {
		updates["pinnacle_tool.api_token"] = apiToken
	}
	if err := model.UpdateOptionsBulk(updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Pinnacle API tool configuration saved",
		"data": gin.H{
			"enabled":          request.Enabled,
			"token_configured": operation_setting.IsPinnacleToolTokenConfigured(),
		},
	})
}

func TestPinnacleToolConnection(c *gin.Context) {
	if !operation_setting.IsPinnacleToolTokenConfigured() {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Apify API token is not configured"})
		return
	}

	client := service.NewPinnacleToolClient(operation_setting.GetPinnacleToolAPIToken(), service.GetHttpClient())
	response, err := client.Do(c.Request.Context(), service.PinnacleEndpointTimezones, nil)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": err.Error()})
		return
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": "Pinnacle upstream rejected the configured credential"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Pinnacle API connection verified"})
}

func RelayPinnacleTool(c *gin.Context, endpoint service.PinnacleEndpoint) {
	requestID := c.GetString(common.RequestIdKey)
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	if !operation_setting.IsPinnacleToolEnabled() {
		writePinnacleToolError(c, http.StatusServiceUnavailable, "pinnacle_tool_unavailable", "Pinnacle API tool is not configured", requestID)
		return
	}
	if err := service.ValidatePinnacleToolQuery(endpoint, c.Request.URL.Query()); err != nil {
		writePinnacleToolError(c, http.StatusBadRequest, "invalid_pinnacle_query", err.Error(), requestID)
		return
	}

	common.SetContextKey(c, constant.ContextKeyRequestStartTime, time.Now())
	common.SetContextKey(c, constant.ContextKeyOriginalModel, operation_setting.PinnacleToolModelName)
	relayInfo, err := relaycommon.GenRelayInfo(c, types.RelayFormatTask, nil, nil)
	if err != nil {
		writePinnacleToolError(c, http.StatusInternalServerError, "pinnacle_tool_internal_error", err.Error(), requestID)
		return
	}
	relayInfo.OriginModelName = operation_setting.PinnacleToolModelName
	relayInfo.Action = string(endpoint)

	quotaDecimal := decimal.NewFromFloat(operation_setting.PinnacleToolPricePerThousandUSD).
		Div(decimal.NewFromInt(1000)).
		Mul(decimal.NewFromFloat(common.QuotaPerUnit))
	quota, clamp := common.QuotaFromDecimalChecked(quotaDecimal)
	relayInfo.QuotaClamp = clamp
	if apiErr := service.PreConsumeBilling(c, quota, relayInfo); apiErr != nil {
		apiErr.SetMessage(common.MessageWithRequestId(apiErr.Error(), requestID))
		c.JSON(apiErr.StatusCode, gin.H{"error": apiErr.ToOpenAIError()})
		return
	}

	settled := false
	defer func() {
		if !settled && relayInfo.Billing != nil {
			relayInfo.Billing.Refund(c)
		}
	}()

	client := service.NewPinnacleToolClient(operation_setting.GetPinnacleToolAPIToken(), service.GetHttpClient())
	response, err := client.Do(c.Request.Context(), endpoint, c.Request.URL.Query())
	if err != nil {
		writePinnacleToolError(c, http.StatusBadGateway, "pinnacle_upstream_error", err.Error(), requestID)
		return
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		contentType := response.ContentType
		if contentType == "" {
			contentType = "application/json"
		}
		c.Data(response.StatusCode, contentType, response.Body)
		return
	}

	if err := service.SettleBilling(c, relayInfo, quota); err != nil {
		writePinnacleToolError(c, http.StatusInternalServerError, "pinnacle_billing_error", err.Error(), requestID)
		return
	}
	settled = true

	useTime := int(time.Since(relayInfo.StartTime).Seconds())
	model.UpdateUserUsedQuotaAndRequestCount(relayInfo.UserId, quota)
	model.RecordConsumeLog(c, relayInfo.UserId, model.RecordConsumeLogParams{
		ModelName:      operation_setting.PinnacleToolModelName,
		TokenName:      c.GetString("token_name"),
		Quota:          quota,
		Content:        fmt.Sprintf("Pinnacle API tool call: %s", endpoint),
		TokenId:        relayInfo.TokenId,
		UseTimeSeconds: useTime,
		Group:          relayInfo.UsingGroup,
		Other: map[string]interface{}{
			"api_tool":                    "pinnacle",
			"operation":                   string(endpoint),
			"price_per_thousand_requests": operation_setting.PinnacleToolPricePerThousandUSD,
			"request_path":                c.Request.URL.Path,
		},
	})

	contentType := response.ContentType
	if contentType == "" {
		contentType = "application/json"
	}
	c.Data(response.StatusCode, contentType, response.Body)
}

func writePinnacleToolError(c *gin.Context, status int, code types.ErrorCode, message, requestID string) {
	errorMessage := common.MessageWithRequestId(message, requestID)
	c.JSON(status, gin.H{
		"error": types.OpenAIError{
			Message: errorMessage,
			Type:    "api_tool_error",
			Code:    code,
		},
	})
}
