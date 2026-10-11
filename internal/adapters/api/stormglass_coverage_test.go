package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStormGlass_ReturnsLatestWaveAndStation(t *testing.T) {
	api := NewStormGlassAPI("test-key")
	api.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, "test-key", req.Header.Get("Authorization"))
		switch req.URL.Path {
		case "/v2/weather/point":
			assert.Equal(t, "32.76", req.URL.Query().Get("lat"))
			assert.Equal(t, "-16.96", req.URL.Query().Get("lng"))
			assert.Equal(t, "waveHeight", req.URL.Query().Get("params"))
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"hours":[{"waveHeight":{"noaa":1.2}},{"waveHeight":{"noaa":2.4}}]}`))}, nil
		case "/v2/tide/stations":
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"data":[{"name":"Funchal","lat":32.65,"lng":-16.91}]}`))}, nil
		default:
			t.Fatalf("unexpected StormGlass request: %s", req.URL)
			return nil, nil
		}
	})}
	report, err := api.FetchReportData(context.Background(), "32.76,-16.96")
	require.NoError(t, err)
	assert.Equal(t, 2.4, report.Data.Height)
	general, err := api.FetchGeneralInfo(context.Background(), "Madeira")
	require.NoError(t, err)
	assert.Equal(t, "Funchal", general.Data.Country)
	assert.Equal(t, 32.65, general.Data.Lat)
}

func TestStormGlass_EmptyWaveListAndProviderErrors(t *testing.T) {
	api := &StormGlassAPI{client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"hours":[]}`))}, nil
	})}}
	report, err := api.FetchReportData(context.Background(), "32.76,-16.96")
	require.NoError(t, err)
	assert.Zero(t, report.Data.Height)
	_, err = api.FetchReportData(context.Background(), "invalid,-16.96")
	assert.Error(t, err)
	_, err = api.FetchReportData(context.Background(), "32.76,invalid")
	assert.Error(t, err)
	api.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("provider unavailable")
	})}
	_, err = api.FetchReportData(context.Background(), "32.76,-16.96")
	assert.ErrorContains(t, err, "provider unavailable")
	_, err = api.FetchStationsData(context.Background())
	assert.ErrorContains(t, err, "provider unavailable")
	_, err = api.FetchGeneralInfo(context.Background(), "Madeira")
	assert.ErrorContains(t, err, "provider unavailable")
}
