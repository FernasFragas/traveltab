package httpserver

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"weatherservice/internal/application"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type fakePlaceSuggester struct {
	queries []string
	places  []application.PlaceSuggestion
	err     error
}

func (f *fakePlaceSuggester) Suggest(_ context.Context, query string) ([]application.PlaceSuggestion, error) {
	f.queries = append(f.queries, query)
	return f.places, f.err
}

func TestSuggestPlaces(t *testing.T) {
	server, _, _ := createTestServer(gomock.NewController(t))
	source := &fakePlaceSuggester{places: []application.PlaceSuggestion{{Name: "Lisbon", Region: "Lisbon", Country: "Portugal", CountryCode: "PT"}}}
	server.SetPlaceSuggester(source)
	for _, path := range []string{"/suggest?city_name=l", "/suggest?city_name=%20%20"} {
		status, body := doRequest(t, server, httptest.NewRequest("GET", path, nil))
		assert.Equal(t, 200, status)
		assert.Empty(t, body)
	}
	status, body := doRequest(t, server, httptest.NewRequest("GET", "/suggest?city_name=%20Lis%2C%20Portugal", nil))
	require.Equal(t, 200, status)
	assert.Contains(t, body, "Lisbon, Portugal")
	assert.Contains(t, body, `data-country-code="PT"`)
	assert.Contains(t, body, `role="option"`)
	assert.Equal(t, []string{"Lis"}, source.queries)
	status, _ = doRequest(t, server, httptest.NewRequest("GET", "/suggest?city_name=lis", nil))
	assert.Equal(t, 200, status)
	assert.Len(t, source.queries, 1, "case-insensitive queries share the cache")
}

func TestSuggestPlacesUpstreamError(t *testing.T) {
	server, _, _ := createTestServer(gomock.NewController(t))
	server.SetPlaceSuggester(&fakePlaceSuggester{err: errors.New("offline")})
	status, body := doRequest(t, server, httptest.NewRequest("GET", "/suggest?city_name=lis", nil))
	assert.Equal(t, 200, status)
	assert.Empty(t, strings.TrimSpace(body))
}

func TestSelectedCountryCodeUsedForSearch(t *testing.T) {
	server, weather, _ := createTestServer(gomock.NewController(t))
	weather.EXPECT().GenerateReport(gomock.Any(), "Paris, US").Return(nil, errors.New("stop after checking query"))
	status, _ := doRequest(t, server, httptest.NewRequest("GET", "/process-form/?city_name=Paris%2C+United+States&country_code=us", nil))
	assert.Equal(t, 500, status)
}

type selectedWeatherReporter struct {
	selected application.PlaceSelection
}

func (r *selectedWeatherReporter) GenerateReport(context.Context, string) (*application.GeneralWeatherInfo, error) {
	return nil, errors.New("name lookup should not run")
}

func (r *selectedWeatherReporter) GenerateReportForPlace(_ context.Context, place application.PlaceSelection) (*application.GeneralWeatherInfo, error) {
	r.selected = place
	return nil, errors.New("stop after checking coordinates")
}

func TestSelectedCoordinatesUsedForNamesakeCity(t *testing.T) {
	server, _, _ := createTestServer(gomock.NewController(t))
	reporter := &selectedWeatherReporter{}
	server.weatherReporters = reporter
	status, _ := doRequest(t, server, httptest.NewRequest("GET", "/process-form/?city_name=Paris%2C+United+States&country_code=US&place_lat=36.302&place_lon=-88.32671", nil))
	assert.Equal(t, 500, status)
	assert.Equal(t, application.PlaceSelection{Name: "Paris", CountryCode: "US", Lat: 36.302, Lon: -88.32671}, reporter.selected)
}
