package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenMeteoGeocodingSuggest(t *testing.T) {
	fixture, err := os.ReadFile("testdata/openmeteo/geocoding_lis.json")
	require.NoError(t, err)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/search", r.URL.Path)
		assert.Equal(t, "lis", r.URL.Query().Get("name"))
		assert.Equal(t, "6", r.URL.Query().Get("count"))
		assert.Equal(t, "en", r.URL.Query().Get("language"))
		assert.Equal(t, "json", r.URL.Query().Get("format"))
		assert.Equal(t, UserAgent, r.Header.Get("User-Agent"))
		_, _ = w.Write(fixture)
	}))
	defer upstream.Close()
	api := NewOpenMeteoGeocodingAPI(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		clone := r.Clone(r.Context())
		clone.URL.Scheme = "http"
		clone.URL.Host = strings.TrimPrefix(upstream.URL, "http://")
		return http.DefaultTransport.RoundTrip(clone)
	})})

	places, err := api.Suggest(context.Background(), "lis")
	require.NoError(t, err)
	require.Len(t, places, 2)
	assert.Equal(t, "Lisbon", places[0].Name)
	assert.Equal(t, "Lisbon", places[0].Region)
	assert.Equal(t, "Portugal", places[0].Country)
	assert.Equal(t, "PT", places[0].CountryCode)
	assert.Equal(t, 38.71667, places[0].Lat)
	assert.Equal(t, -9.13333, places[0].Lon)
	assert.Equal(t, "Maine", places[1].Region)
}

func TestOpenMeteoGeocodingFailure(t *testing.T) {
	api := NewOpenMeteoGeocodingAPI(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 502, Body: io.NopCloser(strings.NewReader("offline"))}, nil
	})})
	_, err := api.Suggest(context.Background(), "lis")
	assert.ErrorContains(t, err, "502")
}
