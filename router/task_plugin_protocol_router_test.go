package router

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostProtocolRegistryDrivesProtocolRoutesOnce(t *testing.T) {
	engine := gin.New()
	SetTaskPluginProtocolRouter(engine)

	expected := []string{
		"POST /v1/responses",
		"GET /v1/responses/:response_id",
		"POST /v1/videos",
		"GET /v1/videos/:task_id",
		"GET /v1/videos/:task_id/content",
		"HEAD /v1/videos/:task_id/content",
	}
	actual := make([]string, 0, len(engine.Routes()))
	for _, route := range engine.Routes() {
		actual = append(actual, fmt.Sprintf("%s %s", route.Method, route.Path))
	}
	sort.Strings(expected)
	sort.Strings(actual)
	assert.Equal(t, expected, actual)
}

func TestResponsesRouteRecordsAuthenticatedRequest(t *testing.T) {
	setupRelayRouterTestDB(t)
	require.NoError(t, i18n.Init())
	t.Setenv("REQUEST_CONTENT_LOG_ENABLED", "true")
	require.NoError(t, model.DB.AutoMigrate(&model.RequestContent{}, &model.RequestContentAccess{}))
	user := model.User{Username: "content-route", Status: common.UserStatusEnabled, Group: "default", Quota: 100}
	require.NoError(t, model.DB.Create(&user).Error)
	token := model.Token{UserId: user.Id, Key: "contentroutekey", Status: common.TokenStatusEnabled, ExpiredTime: -1, UnlimitedQuota: true}
	require.NoError(t, model.DB.Create(&token).Error)

	engine := gin.New()
	engine.Use(middleware.RequestId())
	SetRelayRouter(engine)
	SetTaskPluginProtocolRouter(engine)
	for _, authenticated := range []bool{false, true} {
		request := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"input":"record this question","stream":true}`))
		request.Header.Set("Content-Type", "application/json")
		if authenticated {
			request.Header.Set("Authorization", "Bearer "+token.Key)
		}
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, request)
		requestID := recorder.Header().Get(common.RequestIdKey)
		require.NotEmpty(t, requestID)
		entry, err := model.GetRequestContent(context.Background(), requestID, user.Id, true)
		if !authenticated {
			assert.Equal(t, http.StatusUnauthorized, recorder.Code)
			assert.Nil(t, entry)
			assert.Error(t, err)
			continue
		}
		// Even a routing rejection must retain the authenticated caller's input.
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
		require.NoError(t, err)
		assert.Equal(t, user.Id, entry.UserId)
		assert.Equal(t, token.Id, entry.TokenId)
		assert.Equal(t, "record this question", entry.Preview)
		assert.JSONEq(t, `[{"role":"user","text":"record this question"}]`, entry.Input)
		assert.False(t, entry.Complete)
	}
}
