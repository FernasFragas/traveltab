package api

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// forecastStub builds a client whose transport always answers with body and
// status, and returns the requests it saw. An empty body serves the Kyoto
// fixture.
func forecastStub(t *testing.T, body string, status int) (*OpenMeteoForecastAPI, *[]*http.Request) {
	t.Helper()

	if body == "" {
		b, err := os.ReadFile("testdata/openmeteo/forecast_kyoto.json")
		require.NoError(t, err)
		body = string(b)
	}

	var requests []*http.Request
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests = append(requests, req)
		return &http.Response{
			StatusCode: status,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{},
		}, nil
	})}

	return NewOpenMeteoForecastAPI(client), &requests
}

func TestForecast_ReturnsSixteenDays(t *testing.T) {
	api, _ := forecastStub(t, "", http.StatusOK)

	forecast, err := api.DailyForecast(context.Background(), 35.0116, 135.7681)

	require.NoError(t, err)
	require.NotNil(t, forecast)
	assert.Len(t, forecast.Days, 16)
}

func TestForecast_ReturnsTheCityTimeZone(t *testing.T) {
	api, _ := forecastStub(t, "", http.StatusOK)

	forecast, err := api.DailyForecast(context.Background(), 35.0116, 135.7681)

	require.NoError(t, err)
	require.NotNil(t, forecast)
	assert.Equal(t, "Asia/Tokyo", forecast.Timezone)
}

func TestForecast_DatesAreMidnightInTheCityTimeZone(t *testing.T) {
	api, _ := forecastStub(t, "", http.StatusOK)

	forecast, err := api.DailyForecast(context.Background(), 35.0116, 135.7681)

	require.NoError(t, err)
	require.NotNil(t, forecast)
	require.Len(t, forecast.Days, 16)
	assert.Equal(t, "2026-09-19T00:00:00+09:00", forecast.Days[0].Date.Format("2006-01-02T15:04:05Z07:00"))
}

func TestForecast_KeepsTheRainAmount(t *testing.T) {
	api, _ := forecastStub(t, "", http.StatusOK)

	forecast, err := api.DailyForecast(context.Background(), 35.0116, 135.7681)

	require.NoError(t, err)
	require.NotNil(t, forecast)
	require.Len(t, forecast.Days, 16)
	require.NotNil(t, forecast.Days[1].RainMM)
	assert.Equal(t, 24.4, *forecast.Days[1].RainMM)
}

func TestForecast_NullRainBecomesNil(t *testing.T) {
	api, _ := forecastStub(t, "", http.StatusOK)

	forecast, err := api.DailyForecast(context.Background(), 35.0116, 135.7681)

	require.NoError(t, err)
	require.NotNil(t, forecast)
	require.Len(t, forecast.Days, 16)
	assert.Nil(t, forecast.Days[2].RainMM)
}

func TestForecast_ErrorsOnNon200(t *testing.T) {
	api, _ := forecastStub(t, "upstream is unwell", http.StatusBadGateway)

	_, err := api.DailyForecast(context.Background(), 35.0116, 135.7681)

	assert.ErrorContains(t, err, "502")
}

func TestForecast_ErrorsOnAnUnknownTimeZone(t *testing.T) {
	api, _ := forecastStub(t, `{"timezone":"Unknown/City","daily":{"time":[],"precipitation_sum":[]}}`, http.StatusOK)

	_, err := api.DailyForecast(context.Background(), 35.0116, 135.7681)

	assert.ErrorContains(t, err, "time zone")
}

func TestForecast_AsksForDailyRainInTheLocalTimeZone(t *testing.T) {
	api, requests := forecastStub(t, "", http.StatusOK)

	_, err := api.DailyForecast(context.Background(), 35.0116, 135.7681)

	require.NoError(t, err)
	require.Len(t, *requests, 1)
	sent := (*requests)[0]
	query := sent.URL.Query()
	assert.Equal(t, "precipitation_sum", query.Get("daily"))
	assert.Equal(t, "auto", query.Get("timezone"))
	assert.Equal(t, "16", query.Get("forecast_days"))
	assert.Equal(t, "35.0116", query.Get("latitude"))
	assert.Equal(t, "135.7681", query.Get("longitude"))
	assert.Equal(t, UserAgent, sent.Header.Get("User-Agent"))
}

func TestForecastLive_ReturnsKyoto(t *testing.T) {
	if os.Getenv("LIVE_API_TESTS") != "1" {
		t.Skip("set LIVE_API_TESTS=1 to run the live Open-Meteo check")
	}

	forecast, err := NewOpenMeteoForecastAPI(nil).DailyForecast(context.Background(), 35.0116, 135.7681)

	require.NoError(t, err)
	require.NotNil(t, forecast)
	assert.Equal(t, "Asia/Tokyo", forecast.Timezone)
	assert.Len(t, forecast.Days, 16)
}
