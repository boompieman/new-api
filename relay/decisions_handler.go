package relay

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

func DecisionsHelper(c *gin.Context, info *relaycommon.RelayInfo) (newAPIError *types.NewAPIError) {
	info.InitChannelMeta(c)

	switch info.ChannelType {
	case constant.ChannelTypeOpenRouter,
		constant.ChannelTypeSub2API,
		constant.ChannelTypeNewAPI,
		constant.ChannelTypeAdvancedCustom:
	default:
		return types.NewError(
			errors.New("channel does not support the decisions API"),
			types.ErrorCodeInvalidRequest,
		)
	}

	request, ok := info.Request.(*dto.DecisionsRequest)
	if !ok {
		return types.NewErrorWithStatusCode(
			fmt.Errorf("invalid request type, expected *dto.DecisionsRequest, got %T", info.Request),
			types.ErrorCodeInvalidRequest,
			http.StatusBadRequest,
			types.ErrOptionWithSkipRetry(),
		)
	}

	err := helper.ModelMappedHelper(c, info, request)
	if err != nil {
		return types.NewError(err, types.ErrorCodeChannelModelMappedError, types.ErrOptionWithSkipRetry())
	}

	jsonData, err := buildDecisionsRequestBody(request.RawBody, info.OriginModelName, info.UpstreamModelName)
	if err != nil {
		return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
	}

	if len(info.ParamOverride) > 0 {
		jsonData, err = relaycommon.ApplyParamOverrideWithRelayInfo(jsonData, info)
		if err != nil {
			return newAPIErrorFromParamOverride(err)
		}
	}

	logger.LogDebug(c, "requestBody: %s", jsonData)
	body, closer, err := relaycommon.NewOutboundJSONBody(jsonData)
	if err != nil {
		return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
	}
	defer closer.Close()

	adaptor := GetAdaptor(info.ApiType)
	if adaptor == nil {
		return types.NewError(fmt.Errorf("invalid api type: %d", info.ApiType), types.ErrorCodeInvalidApiType, types.ErrOptionWithSkipRetry())
	}
	adaptor.Init(info)

	resp, err := adaptor.DoRequest(c, info, body)
	if err != nil {
		return types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
	}

	statusCodeMappingStr := c.GetString("status_code_mapping")
	httpResp, ok := resp.(*http.Response)
	if !ok || httpResp == nil {
		return types.NewOpenAIError(errors.New("invalid http response"), types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		newAPIError = service.RelayErrorHandler(c.Request.Context(), httpResp, false)
		service.ResetStatusCode(newAPIError, statusCodeMappingStr)
		return newAPIError
	}

	responseBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return types.NewError(err, types.ErrorCodeBadResponseBody, types.ErrOptionWithSkipRetry())
	}

	usage := usageFromDecisionsResponse(responseBody)
	if usage == nil {
		promptTokens := info.GetEstimatePromptTokens()
		usage = &dto.Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: 0,
			TotalTokens:      promptTokens,
			InputTokens:      promptTokens,
		}
	}

	if contentType := httpResp.Header.Get("Content-Type"); contentType != "" {
		c.Writer.Header().Set("Content-Type", contentType)
	}
	c.Writer.WriteHeader(httpResp.StatusCode)
	if _, err := c.Writer.Write(responseBody); err != nil {
		service.PostTextConsumeQuota(c, info, usage, nil)
		return types.NewError(err, types.ErrorCodeDoRequestFailed, types.ErrOptionWithSkipRetry())
	}

	service.PostTextConsumeQuota(c, info, usage, nil)
	return nil
}

func buildDecisionsRequestBody(rawBody []byte, originModel, upstreamModel string) ([]byte, error) {
	return buildAlphaSearchRequestBody(rawBody, originModel, upstreamModel)
}

type decisionsUsageEnvelope struct {
	Usage *decisionsUsage `json:"usage"`
}

type decisionsUsage struct {
	InputTokens      int `json:"input_tokens"`
	OutputTokens     int `json:"output_tokens"`
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func usageFromDecisionsResponse(body []byte) *dto.Usage {
	var parsed decisionsUsageEnvelope
	if err := common.Unmarshal(body, &parsed); err != nil || parsed.Usage == nil {
		return nil
	}
	u := parsed.Usage
	prompt := u.PromptTokens
	if prompt == 0 {
		prompt = u.InputTokens
	}
	completion := u.CompletionTokens
	if completion == 0 {
		completion = u.OutputTokens
	}
	if prompt == 0 && completion == 0 && u.TotalTokens == 0 {
		return nil
	}
	total := u.TotalTokens
	if total == 0 {
		total = prompt + completion
	}
	return &dto.Usage{
		PromptTokens:     prompt,
		CompletionTokens: completion,
		TotalTokens:      total,
		InputTokens:      prompt,
		OutputTokens:     completion,
	}
}
