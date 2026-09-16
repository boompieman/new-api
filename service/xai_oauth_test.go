package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestXaiOAuthCredentialLifecycle(t *testing.T) {
	expiresAt := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	officialAuthJSON := `{
  "https://auth.x.ai::b1a00492-073a-47ea-816f-4c329264a828": {
    "key": "access-old",
    "refresh_token": "refresh-old",
    "expires_at": "` + expiresAt + `",
    "oidc_issuer": "https://auth.x.ai",
    "oidc_client_id": "b1a00492-073a-47ea-816f-4c329264a828",
    "email": "not-persisted@example.invalid"
  }
}`

	normalized, err := NormalizeXaiOAuthCredential(officialAuthJSON)
	require.NoError(t, err)
	assert.NotContains(t, normalized, "not-persisted@example.invalid")
	assert.NotContains(t, normalized, `"key"`)

	credential, err := ParseXaiOAuthCredential(normalized)
	require.NoError(t, err)
	assert.Equal(t, "access-old", credential.AccessToken)
	assert.Equal(t, "refresh-old", credential.RefreshToken)
	assert.Equal(t, xaiOAuthIssuer, credential.Issuer)
	assert.Equal(t, xaiOAuthClientID, credential.ClientID)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "refresh_token", r.Form.Get("grant_type"))
		assert.Equal(t, xaiOAuthClientID, r.Form.Get("client_id"))
		assert.Equal(t, "refresh-old", r.Form.Get("refresh_token"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"access-new","refresh_token":"refresh-new","expires_in":900}`))
	}))
	defer server.Close()

	refreshed, err := refreshXaiOAuthToken(t.Context(), server.Client(), server.URL, credential)
	require.NoError(t, err)
	assert.Equal(t, "access-new", refreshed.AccessToken)
	assert.Equal(t, "refresh-new", refreshed.RefreshToken)
	assert.WithinDuration(t, time.Now().Add(15*time.Minute), mustParseRFC3339(t, refreshed.ExpiresAt), 2*time.Second)

	baseURL := "https://api.x.ai"
	channel := &model.Channel{Type: constant.ChannelTypeXai, Key: normalized, BaseURL: &baseURL}
	accessToken, err := ResolveXaiChannelAccessToken(context.Background(), channel)
	require.NoError(t, err)
	assert.Equal(t, "access-old", accessToken)
}

func TestXaiOAuthCredentialRejectsUntrustedDestinations(t *testing.T) {
	credential := XaiOAuthCredential{
		Type:         "xai_oauth",
		AccessToken:  "access",
		RefreshToken: "refresh",
		ExpiresAt:    time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
		Issuer:       "https://evil.example",
		ClientID:     xaiOAuthClientID,
	}
	raw, err := common.Marshal(credential)
	require.NoError(t, err)
	_, err = ParseXaiOAuthCredential(string(raw))
	assert.EqualError(t, err, "xAI OAuth credential issuer or client_id is not the official xAI client")

	for _, baseURL := range []string{
		"http://api.x.ai",
		"https://api.x.ai.evil.example",
		"https://api.x.ai:8443",
		"https://api.x.ai?token=leak",
	} {
		t.Run(strings.ReplaceAll(baseURL, "/", "_"), func(t *testing.T) {
			assert.Error(t, ValidateXaiOAuthBaseURL(baseURL))
		})
	}
	assert.NoError(t, ValidateXaiOAuthBaseURL("https://api.x.ai/v1"))
}

func TestXaiOAuthRefreshDatabaseMatrix(t *testing.T) {
	for _, dialect := range []struct {
		name common.DatabaseType
		env  string
	}{
		{common.DatabaseTypeSQLite, ""},
		{common.DatabaseTypeMySQL, "TEST_MYSQL_DSN"},
		{common.DatabaseTypePostgreSQL, "TEST_POSTGRES_DSN"},
	} {
		t.Run(string(dialect.name), func(t *testing.T) {
			var driver gorm.Dialector = sqlite.Open("file:xai_oauth?mode=memory&cache=shared")
			if dialect.env != "" {
				dsn := strings.TrimSpace(os.Getenv(dialect.env))
				if dsn == "" {
					t.Skip(dialect.env + " is not configured")
				}
				if dialect.name == common.DatabaseTypeMySQL {
					driver = mysql.Open(dsn)
				} else {
					driver = postgres.New(postgres.Config{DSN: dsn, PreferSimpleProtocol: true})
				}
			}

			db, err := gorm.Open(driver, &gorm.Config{})
			require.NoError(t, err)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			sqlDB.SetMaxOpenConns(4)
			t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

			previousDB := model.DB
			previousType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
			previousMemoryCache := common.MemoryCacheEnabled
			model.DB = db
			common.SetDatabaseTypes(dialect.name, previousLogType)
			common.MemoryCacheEnabled = false
			t.Cleanup(func() {
				model.DB = previousDB
				common.SetDatabaseTypes(previousType, previousLogType)
				common.MemoryCacheEnabled = previousMemoryCache
			})
			require.NoError(t, db.AutoMigrate(&model.Channel{}))

			credential := XaiOAuthCredential{
				Type:         "xai_oauth",
				AccessToken:  "access-old",
				RefreshToken: "refresh-old",
				ExpiresAt:    time.Now().Add(-time.Minute).UTC().Format(time.RFC3339),
				Issuer:       xaiOAuthIssuer,
				ClientID:     xaiOAuthClientID,
			}
			encoded, err := common.Marshal(credential)
			require.NoError(t, err)
			baseURL := "https://api.x.ai"
			channel := model.Channel{
				Type: constant.ChannelTypeXai, Key: string(encoded), Name: "xai-oauth-matrix",
				BaseURL: &baseURL, Status: common.ChannelStatusEnabled,
			}
			require.NoError(t, db.Create(&channel).Error)

			var refreshes atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				refreshes.Add(1)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"access_token":"access-new","refresh_token":"refresh-new","expires_in":900}`))
			}))
			defer server.Close()

			results := make(chan string, 2)
			errors := make(chan error, 2)
			var group sync.WaitGroup
			for range 2 {
				group.Go(func() {
					token, refreshErr := refreshXaiChannelCredential(t.Context(), channel.Id, server.URL)
					results <- token
					errors <- refreshErr
				})
			}
			group.Wait()
			close(results)
			close(errors)
			for refreshErr := range errors {
				require.NoError(t, refreshErr)
			}
			for token := range results {
				assert.Equal(t, "access-new", token)
			}
			assert.Equal(t, int32(1), refreshes.Load())

			stored, err := model.GetChannelById(channel.Id, true)
			require.NoError(t, err)
			storedCredential, err := ParseXaiOAuthCredential(stored.Key)
			require.NoError(t, err)
			assert.Equal(t, "refresh-new", storedCredential.RefreshToken)
		})
	}
}

func mustParseRFC3339(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	require.NoError(t, err)
	return parsed
}
