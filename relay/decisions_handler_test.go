package relay

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDecisionsRequestBodyRewritesMappedModel(t *testing.T) {
	raw := []byte(`{"model":"jev","state":"hi","questions":{"ok":{"type":"noul"}},"extra":1}`)
	out, err := buildDecisionsRequestBody(raw, "jev", "typesafe/jev-1.13")
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, common.Unmarshal(out, &body))
	assert.Equal(t, "typesafe/jev-1.13", body["model"])
	assert.Equal(t, "hi", body["state"])
	assert.Equal(t, float64(1), body["extra"])
}

func TestBuildDecisionsRequestBodyKeepsRawBytesWithoutMapping(t *testing.T) {
	raw := []byte(`{"model":"typesafe/jev-1.13","questions":{"ok":{"type":"noul"}}}`)
	out, err := buildDecisionsRequestBody(raw, "typesafe/jev-1.13", "typesafe/jev-1.13")
	require.NoError(t, err)
	assert.Equal(t, raw, out)
}

func TestUsageFromDecisionsResponse(t *testing.T) {
	t.Run("typeSafe input_tokens", func(t *testing.T) {
		usage := usageFromDecisionsResponse([]byte(`{"model":"jev-1.13.0","answers":{},"usage":{"input_tokens":422,"output_tokens":41}}`))
		require.NotNil(t, usage)
		assert.Equal(t, 422, usage.PromptTokens)
		assert.Equal(t, 41, usage.CompletionTokens)
		assert.Equal(t, 463, usage.TotalTokens)
	})
	t.Run("openAI prompt_tokens", func(t *testing.T) {
		usage := usageFromDecisionsResponse([]byte(`{"usage":{"prompt_tokens":10,"completion_tokens":2,"total_tokens":12}}`))
		require.NotNil(t, usage)
		assert.Equal(t, 10, usage.PromptTokens)
		assert.Equal(t, 2, usage.CompletionTokens)
		assert.Equal(t, 12, usage.TotalTokens)
	})
	t.Run("missing usage", func(t *testing.T) {
		assert.Nil(t, usageFromDecisionsResponse([]byte(`{"answers":{}}`)))
	})
}

func TestGetAndValidateDecisionsRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("valid", func(t *testing.T) {
		body := `{"model":"typesafe/jev-1.13","state":"urgent payouts","questions":{"is_urgent":{"type":"noul","instructions":"urgency"}}}`
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/systemone", bytes.NewBufferString(body))
		c.Request.Header.Set("Content-Type", "application/json")

		req, err := helper.GetAndValidateDecisionsRequest(c)
		require.NoError(t, err)
		assert.Equal(t, "typesafe/jev-1.13", req.Model)
		require.NotEmpty(t, req.RawBody)
	})

	t.Run("missing questions", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/systemone", bytes.NewBufferString(`{"model":"typesafe/jev-1.13","state":"hi"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		_, err := helper.GetAndValidateDecisionsRequest(c)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "questions")
	})
}

func TestDecisionsHelperForwardsOpenRouterDecisions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var gotPath string
	var gotAuth string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"jev-1.13.0","answers":{"is_urgent":{"type":"noul","noul":0.91}},"usage":{"input_tokens":12,"output_tokens":3}}`))
	}))
	t.Cleanup(upstream.Close)

	raw := []byte(`{"model":"typesafe/jev-1.13","state":"payouts failed","questions":{"is_urgent":{"type":"noul","instructions":"urgency"}}}`)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/systemone", bytes.NewReader(raw))
	c.Request.Header.Set("Content-Type", "application/json")
	common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeOpenRouter)
	common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, upstream.URL)
	common.SetContextKey(c, constant.ContextKeyChannelKey, "sk-or-test")

	info := &relaycommon.RelayInfo{
		Request:         &dto.DecisionsRequest{Model: "typesafe/jev-1.13", RawBody: raw},
		OriginModelName: "typesafe/jev-1.13",
		RelayMode:       relayconstant.RelayModeDecisions,
		RelayFormat:     types.RelayFormatDecisions,
		RequestURLPath:  "/v1/systemone",
	}

	func() {
		defer func() { _ = recover() }()
		_ = DecisionsHelper(c, info)
	}()
	assert.Equal(t, "/alpha/decisions", gotPath)
	assert.Equal(t, "Bearer sk-or-test", gotAuth)
	assert.Equal(t, "typesafe/jev-1.13", gotBody["model"])
	assert.Contains(t, recorder.Body.String(), `"noul":0.91`)
}

func TestDecisionsHelperRejectsUnsupportedChannel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/systemone", bytes.NewBufferString(`{}`))

	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeOpenAI},
		Request:     &dto.DecisionsRequest{Model: "typesafe/jev-1.13"},
	}
	err := DecisionsHelper(c, info)
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "does not support the decisions API")
}
