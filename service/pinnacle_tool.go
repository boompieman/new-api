package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type PinnacleEndpoint string

const (
	PinnacleEndpointTimezones     PinnacleEndpoint = "timezones"
	PinnacleEndpointSportList     PinnacleEndpoint = "sport_list"
	PinnacleEndpointSportLeagues  PinnacleEndpoint = "sport_leagues"
	PinnacleEndpointSportMatches  PinnacleEndpoint = "sport_matches"
	PinnacleEndpointLiveMatches   PinnacleEndpoint = "sport_matches_live"
	PinnacleEndpointLeagueMatches PinnacleEndpoint = "league_matches"
	PinnacleEndpointMatchDetails  PinnacleEndpoint = "match_details"
	PinnacleEndpointMatchOdds     PinnacleEndpoint = "match_odds"

	pinnacleDefaultBaseURL = "https://superapis--pinnacle.apify.actor"
	pinnacleResponseLimit  = 16 << 20
)

var pinnacleTimezonePattern = regexp.MustCompile(`^[A-Za-z0-9_+\-/]{1,64}$`)

type pinnacleEndpointSpec struct {
	path     string
	required map[string]struct{}
	optional map[string]struct{}
}

var pinnacleEndpointSpecs = map[PinnacleEndpoint]pinnacleEndpointSpec{
	PinnacleEndpointTimezones: {path: "/v1/timezones"},
	PinnacleEndpointSportList: {path: "/v1/sport/list"},
	PinnacleEndpointSportLeagues: {
		path:     "/v1/sport/leagues",
		required: map[string]struct{}{"sport_id": {}},
	},
	PinnacleEndpointSportMatches: {
		path:     "/v1/sport/matches",
		required: map[string]struct{}{"sport_id": {}},
		optional: map[string]struct{}{"league_id": {}, "date": {}, "timezone": {}},
	},
	PinnacleEndpointLiveMatches: {
		path:     "/v1/sport/matches/live",
		required: map[string]struct{}{"sport_id": {}},
		optional: map[string]struct{}{"league_id": {}, "timezone": {}},
	},
	PinnacleEndpointLeagueMatches: {
		path:     "/v1/league/matches",
		required: map[string]struct{}{"league_id": {}, "sport_id": {}},
		optional: map[string]struct{}{"date": {}, "timezone": {}},
	},
	PinnacleEndpointMatchDetails: {
		path:     "/v1/match/details",
		required: map[string]struct{}{"match_id": {}},
		optional: map[string]struct{}{"timezone": {}},
	},
	PinnacleEndpointMatchOdds: {
		path:     "/v1/match/odds",
		required: map[string]struct{}{"match_id": {}},
		optional: map[string]struct{}{"timezone": {}},
	},
}

type PinnacleToolClient struct {
	baseURL    string
	apiToken   string
	httpClient *http.Client
}

type PinnacleToolResponse struct {
	StatusCode  int
	ContentType string
	Body        []byte
}

func NewPinnacleToolClient(apiToken string, httpClient *http.Client) *PinnacleToolClient {
	return newPinnacleToolClient(pinnacleDefaultBaseURL, apiToken, httpClient)
}

func newPinnacleToolClient(baseURL, apiToken string, httpClient *http.Client) *PinnacleToolClient {
	return &PinnacleToolClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiToken:   strings.TrimSpace(apiToken),
		httpClient: httpClient,
	}
}

func ValidatePinnacleToolQuery(endpoint PinnacleEndpoint, input url.Values) error {
	spec, ok := pinnacleEndpointSpecs[endpoint]
	if !ok {
		return errors.New("unsupported Pinnacle endpoint")
	}
	_, err := validatePinnacleQuery(spec, input)
	return err
}

func (client *PinnacleToolClient) Do(ctx context.Context, endpoint PinnacleEndpoint, input url.Values) (*PinnacleToolResponse, error) {
	if client == nil || client.httpClient == nil {
		return nil, errors.New("Pinnacle HTTP client is not configured")
	}
	if client.apiToken == "" {
		return nil, errors.New("Pinnacle API token is not configured")
	}

	spec, ok := pinnacleEndpointSpecs[endpoint]
	if !ok {
		return nil, errors.New("unsupported Pinnacle endpoint")
	}
	query, err := validatePinnacleQuery(spec, input)
	if err != nil {
		return nil, err
	}
	query.Set("token", client.apiToken)

	requestURL, err := url.Parse(client.baseURL + spec.path)
	if err != nil {
		return nil, fmt.Errorf("build Pinnacle request URL: %w", err)
	}
	requestURL.RawQuery = query.Encode()

	requestContext, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(requestContext, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build Pinnacle request: %w", err)
	}
	request.Header.Set("Accept", "application/json")

	upstreamClient := *client.httpClient
	upstreamClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	response, err := upstreamClient.Do(request)
	if err != nil {
		// net/http errors can contain the complete URL. Do not surface them,
		// because the Apify credential is transported in its query string.
		return nil, errors.New("Pinnacle upstream request failed")
	}
	defer response.Body.Close()

	limitedBody := io.LimitReader(response.Body, pinnacleResponseLimit+1)
	body, err := io.ReadAll(limitedBody)
	if err != nil {
		return nil, fmt.Errorf("read Pinnacle response: %w", err)
	}
	if len(body) > pinnacleResponseLimit {
		return nil, errors.New("Pinnacle response exceeds 16 MiB limit")
	}

	return &PinnacleToolResponse{
		StatusCode:  response.StatusCode,
		ContentType: response.Header.Get("Content-Type"),
		Body:        body,
	}, nil
}

func validatePinnacleQuery(spec pinnacleEndpointSpec, input url.Values) (url.Values, error) {
	result := make(url.Values, len(input))
	for key, values := range input {
		if _, required := spec.required[key]; !required {
			if _, optional := spec.optional[key]; !optional {
				return nil, fmt.Errorf("unsupported query parameter %q", key)
			}
		}
		if len(values) != 1 || strings.TrimSpace(values[0]) == "" {
			return nil, fmt.Errorf("query parameter %q must have exactly one non-empty value", key)
		}
		value := strings.TrimSpace(values[0])
		switch key {
		case "sport_id", "league_id", "match_id":
			id, err := strconv.ParseInt(value, 10, 64)
			if err != nil || id <= 0 {
				return nil, fmt.Errorf("query parameter %q must be a positive integer", key)
			}
		case "date":
			if _, err := time.Parse("2006-01-02", value); err != nil {
				return nil, errors.New("query parameter \"date\" must use YYYY-MM-DD")
			}
		case "timezone":
			if !pinnacleTimezonePattern.MatchString(value) {
				return nil, errors.New("query parameter \"timezone\" must be a valid IANA timezone identifier")
			}
		}
		result.Set(key, value)
	}

	for key := range spec.required {
		if result.Get(key) == "" {
			return nil, fmt.Errorf("missing required query parameter %q", key)
		}
	}
	return result, nil
}
