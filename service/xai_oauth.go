package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

const (
	xaiOAuthIssuer           = "https://auth.x.ai"
	xaiOAuthClientID         = "b1a00492-073a-47ea-816f-4c329264a828"
	xaiOAuthTokenURL         = "https://auth.x.ai/oauth2/token"
	xaiOAuthScopeKey         = xaiOAuthIssuer + "::" + xaiOAuthClientID
	xaiOAuthRefreshThreshold = 2 * time.Minute
	xaiOAuthRefreshTimeout   = 20 * time.Second
)

type XaiOAuthCredential struct {
	Type         string `json:"type"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    string `json:"expires_at"`
	Issuer       string `json:"issuer"`
	ClientID     string `json:"client_id"`
	LastRefresh  string `json:"last_refresh,omitempty"`
}

type xaiOAuthInput struct {
	Type         string `json:"type"`
	AccessToken  string `json:"access_token"`
	Key          string `json:"key"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    string `json:"expires_at"`
	Issuer       string `json:"issuer"`
	OIDCIssuer   string `json:"oidc_issuer"`
	ClientID     string `json:"client_id"`
	OIDCClientID string `json:"oidc_client_id"`
	LastRefresh  string `json:"last_refresh"`
}

func ParseXaiOAuthCredential(raw string) (*XaiOAuthCredential, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || !strings.HasPrefix(trimmed, "{") {
		return nil, errors.New("xAI OAuth credential must be a JSON object")
	}

	var input xaiOAuthInput
	if err := common.Unmarshal([]byte(trimmed), &input); err != nil {
		return nil, errors.New("xAI OAuth credential must be valid JSON")
	}
	if strings.TrimSpace(input.AccessToken) == "" && strings.TrimSpace(input.Key) == "" {
		var authStore map[string]xaiOAuthInput
		if err := common.Unmarshal([]byte(trimmed), &authStore); err != nil {
			return nil, errors.New("xAI OAuth credential must be valid JSON")
		}
		var ok bool
		input, ok = authStore[xaiOAuthScopeKey]
		if !ok {
			return nil, errors.New("xAI OAuth credential does not contain the official xAI login")
		}
	}

	accessToken := strings.TrimSpace(input.AccessToken)
	if accessToken == "" {
		accessToken = strings.TrimSpace(input.Key)
	}
	issuer := strings.TrimSpace(input.Issuer)
	if issuer == "" {
		issuer = strings.TrimSpace(input.OIDCIssuer)
	}
	clientID := strings.TrimSpace(input.ClientID)
	if clientID == "" {
		clientID = strings.TrimSpace(input.OIDCClientID)
	}
	if issuer == "" {
		issuer = xaiOAuthIssuer
	}
	if clientID == "" {
		clientID = xaiOAuthClientID
	}

	credential := &XaiOAuthCredential{
		Type:         "xai_oauth",
		AccessToken:  accessToken,
		RefreshToken: strings.TrimSpace(input.RefreshToken),
		ExpiresAt:    strings.TrimSpace(input.ExpiresAt),
		Issuer:       issuer,
		ClientID:     clientID,
		LastRefresh:  strings.TrimSpace(input.LastRefresh),
	}
	if credential.AccessToken == "" || credential.RefreshToken == "" || credential.ExpiresAt == "" {
		return nil, errors.New("xAI OAuth credential requires access_token, refresh_token, and expires_at")
	}
	if credential.Issuer != xaiOAuthIssuer || credential.ClientID != xaiOAuthClientID {
		return nil, errors.New("xAI OAuth credential issuer or client_id is not the official xAI client")
	}
	if _, err := time.Parse(time.RFC3339, credential.ExpiresAt); err != nil {
		return nil, errors.New("xAI OAuth credential expires_at must be RFC3339")
	}
	return credential, nil
}

func NormalizeXaiOAuthCredential(raw string) (string, error) {
	credential, err := ParseXaiOAuthCredential(raw)
	if err != nil {
		return "", err
	}
	encoded, err := common.Marshal(credential)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func ValidateXaiOAuthBaseURL(baseURL string) error {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() != "api.x.ai" {
		return errors.New("xAI OAuth credentials may only be sent to https://api.x.ai")
	}
	if port := parsed.Port(); port != "" && port != "443" {
		return errors.New("xAI OAuth credentials may only be sent to https://api.x.ai")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("xAI OAuth base URL cannot contain user info, query, or fragment")
	}
	return nil
}

func ResolveXaiChannelAccessToken(ctx context.Context, channel *model.Channel) (string, error) {
	if channel == nil || channel.Type != constant.ChannelTypeXai {
		return "", errors.New("channel type is not xAI")
	}
	rawKey := strings.TrimSpace(channel.Key)
	if !strings.HasPrefix(rawKey, "{") {
		return rawKey, nil
	}
	if channel.ChannelInfo.IsMultiKey {
		return "", errors.New("xAI OAuth does not support multi-key channels")
	}
	if err := ValidateXaiOAuthBaseURL(xaiChannelBaseURL(channel)); err != nil {
		return "", err
	}
	credential, err := ParseXaiOAuthCredential(rawKey)
	if err != nil {
		return "", err
	}
	expiresAt, _ := time.Parse(time.RFC3339, credential.ExpiresAt)
	if time.Until(expiresAt) > xaiOAuthRefreshThreshold {
		return credential.AccessToken, nil
	}
	if channel.Id <= 0 {
		return "", errors.New("xAI OAuth credential is expiring; save the channel before retrying")
	}
	return refreshXaiChannelCredential(ctx, channel.Id, xaiOAuthTokenURL)
}

func refreshXaiChannelCredential(ctx context.Context, channelID int, tokenURL string) (string, error) {
	lock := model.GetChannelPollingLock(channelID)
	lock.Lock()
	defer lock.Unlock()

	refreshCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), xaiOAuthRefreshTimeout)
	defer cancel()

	var refreshedChannel *model.Channel
	var accessToken string
	err := model.DB.WithContext(refreshCtx).Transaction(func(tx *gorm.DB) error {
		channel, err := model.GetChannelByIdForUpdate(tx, channelID)
		if err != nil {
			return err
		}
		if channel.Type != constant.ChannelTypeXai {
			return errors.New("channel type is not xAI")
		}
		if err := ValidateXaiOAuthBaseURL(xaiChannelBaseURL(channel)); err != nil {
			return err
		}
		credential, err := ParseXaiOAuthCredential(strings.TrimSpace(channel.Key))
		if err != nil {
			return err
		}
		expiresAt, _ := time.Parse(time.RFC3339, credential.ExpiresAt)
		if time.Until(expiresAt) > xaiOAuthRefreshThreshold {
			accessToken = credential.AccessToken
			return nil
		}

		client, err := GetHttpClientWithProxy(strings.TrimSpace(channel.GetSetting().Proxy))
		if err != nil {
			return err
		}
		if client == nil {
			client = &http.Client{Timeout: xaiOAuthRefreshTimeout}
		} else {
			clientCopy := *client
			clientCopy.Timeout = xaiOAuthRefreshTimeout
			client = &clientCopy
		}
		updated, err := refreshXaiOAuthToken(refreshCtx, client, tokenURL, credential)
		if err != nil {
			return err
		}
		encoded, err := common.Marshal(updated)
		if err != nil {
			return err
		}
		if err := tx.Model(&model.Channel{}).Where("id = ?", channel.Id).Update("key", string(encoded)).Error; err != nil {
			return err
		}
		channel.Key = string(encoded)
		refreshedChannel = channel
		accessToken = updated.AccessToken
		return nil
	})
	if err != nil {
		return "", err
	}
	if refreshedChannel != nil {
		model.CacheUpdateChannel(refreshedChannel)
	}
	return accessToken, nil
}

func refreshXaiOAuthToken(ctx context.Context, client *http.Client, tokenURL string, credential *XaiOAuthCredential) (*XaiOAuthCredential, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", xaiOAuthClientID)
	form.Set("refresh_token", credential.RefreshToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("xAI OAuth refresh failed: status=%d", resp.StatusCode)
	}

	var payload struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := common.DecodeJson(resp.Body, &payload); err != nil {
		return nil, err
	}
	if strings.TrimSpace(payload.AccessToken) == "" || payload.ExpiresIn <= 0 || payload.ExpiresIn > int64((30*24*time.Hour)/time.Second) {
		return nil, errors.New("xAI OAuth refresh response missing or invalid fields")
	}
	refreshToken := strings.TrimSpace(payload.RefreshToken)
	if refreshToken == "" {
		refreshToken = credential.RefreshToken
	}
	return &XaiOAuthCredential{
		Type:         "xai_oauth",
		AccessToken:  strings.TrimSpace(payload.AccessToken),
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(payload.ExpiresIn) * time.Second).UTC().Format(time.RFC3339),
		Issuer:       xaiOAuthIssuer,
		ClientID:     xaiOAuthClientID,
		LastRefresh:  time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func xaiChannelBaseURL(channel *model.Channel) string {
	baseURL := strings.TrimSpace(channel.GetBaseURL())
	if baseURL == "" {
		baseURL = constant.GetChannelBaseURL(constant.ChannelTypeXai)
	}
	return baseURL
}
