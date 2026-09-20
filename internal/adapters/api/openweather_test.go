package api

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type stubResponse struct {
	status int
	body   string
}

// newStubWeatherAPI answers the geocoding and weather endpoints with the given responses
// instead of calling OpenWeather, and records every request it receives.
func newStubWeatherAPI(geocoding, weather stubResponse) (*WeatherAPI, *[]*http.Request) {
	var requests []*http.Request

	api := NewWeatherAPI("test-key")
	api.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests = append(requests, req)

		stub := weather
		if strings.Contains(req.URL.Path, "/geo/") {
			stub = geocoding
		}

		return &http.Response{
			StatusCode: stub.status,
			Body:       io.NopCloser(strings.NewReader(stub.body)),
			Header:     http.Header{},
		}, nil
	})}

	return api, &requests
}

var (
	lisbonLocation = stubResponse{http.StatusOK, `[{"name":"Lisbon","lat":38.72,"lon":-9.14,"country":"PT"}]`}
	invalidAPIKey  = stubResponse{http.StatusUnauthorized, `{"cod":401, "message": "Invalid API key. Please see https://openweathermap.org/faq#error401 for more info."}`}
)

func Test_GetWeather(t *testing.T) {
	api, requests := newStubWeatherAPI(
		lisbonLocation,
		stubResponse{http.StatusOK, `{"coord":{"lon":-9.14,"lat":38.72},"weather":[{"main":"Clear","description":"clear sky"}],"main":{"temp":24.5,"feels_like":24.1,"humidity":60},"wind":{"speed":3.6},"sys":{"country":"PT"}}`},
	)

	report, err := api.FetchReportData(context.Background(), "Lisbon, PT")
	require.NoError(t, err)

	require.Len(t, *requests, 2)
	assert.Equal(t, "Lisbon,PT", (*requests)[0].URL.Query().Get("q"))
	assert.Equal(t, "test-key", (*requests)[0].URL.Query().Get("APPID"))
	assert.Equal(t, "38.720000", (*requests)[1].URL.Query().Get("lat"))
	assert.Equal(t, "-9.140000", (*requests)[1].URL.Query().Get("lon"))

	assert.Equal(t, "Lisbon", report.Data.City)
	assert.Equal(t, "PT", report.Data.Country)
	assert.Equal(t, 24.5, report.Data.Temperature)
	assert.Equal(t, 24.1, report.Data.FeelsLike)
	assert.Equal(t, 60.0, report.Data.Humidity)
	assert.Equal(t, 3.6, report.Data.Wind)
	assert.Equal(t, "clear sky", report.Data.Condition)
}

func Test_GetWeather_Errors(t *testing.T) {
	tests := []struct {
		name         string
		location     string
		geocoding    stubResponse
		weather      stubResponse
		wantErr      string
		wantRequests int
	}{
		{
			name:         "missing country",
			location:     "Lisbon",
			wantErr:      "city and country are required",
			wantRequests: 0,
		},
		{
			name:         "invalid API key",
			location:     "Lisbon, PT",
			geocoding:    invalidAPIKey,
			wantErr:      "status 401: Invalid API key",
			wantRequests: 1,
		},
		{
			name:         "unknown city",
			location:     "Atlantis, PT",
			geocoding:    stubResponse{http.StatusOK, `[]`},
			wantErr:      "no location found for Atlantis, PT",
			wantRequests: 1,
		},
		{
			name:         "weather request rejected",
			location:     "Lisbon, PT",
			geocoding:    lisbonLocation,
			weather:      invalidAPIKey,
			wantErr:      "status 401: Invalid API key",
			wantRequests: 2,
		},
		{
			name:         "non-JSON error page",
			location:     "Lisbon, PT",
			geocoding:    stubResponse{http.StatusBadGateway, `<html>Bad Gateway</html>`},
			wantErr:      "status 502: Bad Gateway",
			wantRequests: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api, requests := newStubWeatherAPI(tt.geocoding, tt.weather)

			report, err := api.FetchReportData(context.Background(), tt.location)

			assert.Nil(t, report)
			assert.ErrorContains(t, err, tt.wantErr)
			assert.Len(t, *requests, tt.wantRequests)
		})
	}
}
