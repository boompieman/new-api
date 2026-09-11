package codex

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNativeImageGeneration(t *testing.T) {
	a := &Adaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeCodex, ChannelBaseUrl: "https://chatgpt.com"},
		RelayMode:   relayconstant.RelayModeImagesGenerations,
	}
	url, err := a.GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://chatgpt.com/backend-api/codex/images/generations", url)
	request := dto.ImageRequest{Model: "gpt-image-2", Prompt: "a blue square", N: lo.ToPtr(uint(1)), Quality: "low", Size: "1024x1024", Background: json.RawMessage(`"transparent"`), ResponseFormat: "b64_json"}
	converted, err := a.ConvertImageRequest(nil, info, request)
	require.NoError(t, err)
	body, err := common.Marshal(converted)
	require.NoError(t, err)
	assert.JSONEq(t, `{"model":"gpt-image-2","prompt":"a blue square","n":1,"quality":"low","size":"1024x1024","background":"transparent"}`, string(body))

	for _, n := range []uint{0, dto.MaxImageN + 1} {
		invalid := request
		invalid.N = &n
		_, err = a.ConvertImageRequest(nil, info, invalid)
		require.Error(t, err)
	}
	streaming := request
	streaming.Stream = lo.ToPtr(true)
	_, err = a.ConvertImageRequest(nil, info, streaming)
	require.Error(t, err)
	referenced := request
	referenced.Images = json.RawMessage(`[{"image_url":"data:image/png;base64,aW1hZ2U="}]`)
	_, err = a.ConvertImageRequest(nil, info, referenced)
	require.Error(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
	response := `{"data":[{"b64_json":"aW1hZ2U="}],"usage":{"input_tokens":1000,"input_tokens_details":{"text_tokens":1000,"image_tokens":0,"cached_tokens":200},"output_tokens":500,"total_tokens":1500}}`
	result, apiErr := a.DoResponse(c, &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(response))}, info)
	require.Nil(t, apiErr)
	usage := result.(*dto.Usage)
	assert.Equal(t, 1000, usage.PromptTokens)
	assert.Equal(t, 500, usage.CompletionTokens)
	assert.Equal(t, 200, usage.PromptTokensDetails.CachedTokens)
	assert.JSONEq(t, response, w.Body.String())
	expression, ok := billing_setting.GetBillingExpr("gpt-image-2")
	require.True(t, ok)
	params := service.BuildTieredTokenParams(usage, false, billingexpr.UsedVars(expression))
	charge, err := billingexpr.ComputeTieredQuotaWithRequest(&billingexpr.BillingSnapshot{ExprString: expression, GroupRatio: 1, QuotaPerUnit: 500000}, params, billingexpr.RequestInput{})
	require.NoError(t, err)
	assert.Equal(t, 9625, charge.ActualQuotaAfterGroup)

	info.RelayMode = relayconstant.RelayModeImagesEdits
	_, err = a.ConvertImageRequest(nil, info, request)
	require.Error(t, err)
}

func TestGetRequestURLAlphaSearch(t *testing.T) {
	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:    constant.ChannelTypeCodex,
			ChannelBaseUrl: "https://chatgpt.com",
		},
		RelayMode: relayconstant.RelayModeAlphaSearch,
	}

	url, err := adaptor.GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://chatgpt.com/backend-api/codex/alpha/search", url)
}

// The Codex backend rejects these fields, so the adaptor clears them rather
// than forwarding what the client sent.
func TestConvertOpenAIResponsesRequestDropsPenalties(t *testing.T) {
	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeCodex},
		RelayMode:   relayconstant.RelayModeResponses,
	}

	converted, err := adaptor.ConvertOpenAIResponsesRequest(nil, info, dto.OpenAIResponsesRequest{
		Model:            "gpt-5-codex",
		Input:            json.RawMessage(`"hello"`),
		MaxOutputTokens:  lo.ToPtr(uint(128)),
		Temperature:      lo.ToPtr(1.0),
		FrequencyPenalty: json.RawMessage(`1.5`),
		PresencePenalty:  json.RawMessage(`1.5`),
	})
	require.NoError(t, err)

	request, ok := converted.(dto.OpenAIResponsesRequest)
	require.True(t, ok)
	assert.Nil(t, request.MaxOutputTokens)
	assert.Nil(t, request.Temperature)
	assert.Nil(t, request.FrequencyPenalty)
	assert.Nil(t, request.PresencePenalty)
}
