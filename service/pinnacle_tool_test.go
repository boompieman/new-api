package service

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestPinnacleToolClientForwardsValidatedQueryAndServerToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1/sport/matches", r.URL.Path)
		assert.Equal(t, "11", r.URL.Query().Get("sport_id"))
		assert.Equal(t, "22", r.URL.Query().Get("league_id"))
		assert.Equal(t, "Asia/Taipei", r.URL.Query().Get("timezone"))
		assert.Equal(t, "test-apify-token", r.URL.Query().Get("token"))
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"matches":[{"id":33}]}`))
	}))
	t.Cleanup(server.Close)

	client := newPinnacleToolClient(server.URL, "test-apify-token", server.Client())
	response, err := client.Do(t.Context(), PinnacleEndpointSportMatches, url.Values{
		"sport_id":  {"11"},
		"league_id": {"22"},
		"timezone":  {"Asia/Taipei"},
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", response.ContentType)
	assert.JSONEq(t, `{"matches":[{"id":33}]}`, string(response.Body))
}

func TestPinnacleToolClientRejectsInvalidQueryBeforeCallingUpstream(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	client := newPinnacleToolClient(server.URL, "test-apify-token", server.Client())

	tests := []struct {
		name     string
		endpoint PinnacleEndpoint
		query    url.Values
		message  string
	}{
		{
			name:     "missing required sport",
			endpoint: PinnacleEndpointSportLeagues,
			query:    url.Values{},
			message:  `missing required query parameter "sport_id"`,
		},
		{
			name:     "non-positive match id",
			endpoint: PinnacleEndpointMatchOdds,
			query:    url.Values{"match_id": {"0"}},
			message:  `query parameter "match_id" must be a positive integer`,
		},
		{
			name:     "malformed date",
			endpoint: PinnacleEndpointSportMatches,
			query:    url.Values{"sport_id": {"1"}, "date": {"2026-02-30"}},
			message:  `query parameter "date" must use YYYY-MM-DD`,
		},
		{
			name:     "unknown parameter",
			endpoint: PinnacleEndpointTimezones,
			query:    url.Values{"token": {"client-token"}},
			message:  `unsupported query parameter "token"`,
		},
		{
			name:     "duplicate parameter",
			endpoint: PinnacleEndpointMatchDetails,
			query:    url.Values{"match_id": {"1", "2"}},
			message:  `query parameter "match_id" must have exactly one non-empty value`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := client.Do(t.Context(), test.endpoint, test.query)
			require.EqualError(t, err, test.message)
		})
	}
	assert.Zero(t, callCount)
}

func TestPinnacleToolClientRequiresConfiguration(t *testing.T) {
	client := NewPinnacleToolClient("", http.DefaultClient)

	_, err := client.Do(t.Context(), PinnacleEndpointTimezones, nil)

	require.EqualError(t, err, "Pinnacle API token is not configured")
}

func TestPinnacleToolClientDoesNotFollowRedirects(t *testing.T) {
	redirectTargetCalls := 0
	redirectTarget := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectTargetCalls++
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(redirectTarget.Close)

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, redirectTarget.URL+"?token="+r.URL.Query().Get("token"), http.StatusFound)
	}))
	t.Cleanup(origin.Close)

	client := newPinnacleToolClient(origin.URL, "test-apify-token", origin.Client())
	response, err := client.Do(t.Context(), PinnacleEndpointTimezones, nil)

	require.NoError(t, err)
	assert.Equal(t, http.StatusFound, response.StatusCode)
	assert.Zero(t, redirectTargetCalls)
}

func TestPinnacleToolClientSanitizesTransportErrors(t *testing.T) {
	httpClient := &http.Client{
		Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
			return nil, errors.New("request failed: " + request.URL.String())
		}),
	}
	client := newPinnacleToolClient("https://example.com", "test-apify-token", httpClient)

	_, err := client.Do(t.Context(), PinnacleEndpointTimezones, nil)

	require.EqualError(t, err, "Pinnacle upstream request failed")
	assert.NotContains(t, err.Error(), "test-apify-token")
}
