package httpserver

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"weatherservice/internal/application"
	"weatherservice/internal/writermap"
	"weatherservice/internal/writermap/writermaptest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type fakePlaceSuggester struct {
	queries []string
	places  []application.PlaceSuggestion
	results map[string][]application.PlaceSuggestion
	err     error
}

func (f *fakePlaceSuggester) Suggest(_ context.Context, query string) ([]application.PlaceSuggestion, error) {
	f.queries = append(f.queries, query)
	if f.results != nil {
		return f.results[query], f.err
	}
	return f.places, f.err
}

type blockingBaseSource struct {
	*writermaptest.FakeBaseSource
	started chan writermap.Area
	release chan struct{}
}

func (f *blockingBaseSource) Places(ctx context.Context, area writermap.Area) (writermap.BasePlaces, error) {
	result, err := f.FakeBaseSource.Places(ctx, area)
	f.started <- area
	<-f.release
	return result, err
}

// As a visitor typing a city, I want the top result cached before selection, while suggestions remain responsive.
func TestSuggestPlaces_WarmsTopSuggestionWithoutDelayingResponse(t *testing.T) {
	server, _, _ := createTestServer(gomock.NewController(t))
	server.SetPlaceSuggester(&fakePlaceSuggester{places: []application.PlaceSuggestion{
		{Name: "Funchal", Country: "Portugal", CountryCode: "PT", Lat: 32.6669, Lon: -16.9241},
		{Name: "Porto", Country: "Portugal", CountryCode: "PT", Lat: 41.1579, Lon: -8.6291},
	}})
	base := &blockingBaseSource{FakeBaseSource: &writermaptest.FakeBaseSource{}, started: make(chan writermap.Area, 1), release: make(chan struct{})}
	server.SetBasePlaceSource(base, true)

	request := httptest.NewRequest("GET", "/suggest?city_name=Fun", nil)
	request.Header.Set("Fly-Client-IP", "192.0.2.1")
	status, body := doRequest(t, server, request)
	require.Equal(t, 200, status)
	assert.Contains(t, body, "Funchal")
	select {
	case area := <-base.started:
		assert.Equal(t, writermap.Area{Name: "Funchal", Country: "Portugal", Lat: 32.6669, Lon: -16.9241}, area)
	case <-time.After(time.Second):
		t.Fatal("base warmup did not start")
	}
	assert.Equal(t, 1, base.PlacesCalls())
	close(base.release)
}

// As a visitor typing repeatedly, I want one base query per destination and no more than one new warmup every two seconds.
func TestSuggestPlaces_DeduplicatesAndThrottlesWarmups(t *testing.T) {
	server, _, _ := createTestServer(gomock.NewController(t))
	server.SetPlaceSuggester(&fakePlaceSuggester{results: map[string][]application.PlaceSuggestion{
		"Fun":  {{Name: "Funchal", Country: "Portugal", Lat: 32.6669, Lon: -16.9241}},
		"Port": {{Name: "Porto", Country: "Portugal", Lat: 41.1579, Lon: -8.6291}},
		"Lis":  {{Name: "Lisbon", Country: "Portugal", Lat: 38.7223, Lon: -9.1393}},
	}})
	base := &blockingBaseSource{FakeBaseSource: &writermaptest.FakeBaseSource{}, started: make(chan writermap.Area, 3), release: make(chan struct{})}
	server.SetBasePlaceSource(base, true)
	now := time.Now()
	server.suggestions.now = func() time.Time { return now }
	for _, city := range []string{"Fun", "Fun", "Port"} {
		request := httptest.NewRequest("GET", "/suggest?city_name="+city, nil)
		request.Header.Set("Fly-Client-IP", "192.0.2.1")
		status, _ := doRequest(t, server, request)
		require.Equal(t, 200, status)
	}
	select {
	case <-base.started:
	case <-time.After(time.Second):
		t.Fatal("first warmup did not start")
	}
	assert.Equal(t, 1, base.PlacesCalls())
	request := httptest.NewRequest("GET", "/suggest?city_name=Port", nil)
	request.Header.Set("Fly-Client-IP", "192.0.2.2")
	status, _ := doRequest(t, server, request)
	require.Equal(t, 200, status)
	select {
	case area := <-base.started:
		assert.Equal(t, "Porto", area.Name)
	case <-time.After(time.Second):
		t.Fatal("second client warmup did not start")
	}
	assert.Equal(t, 2, base.PlacesCalls())
	now = now.Add(baseWarmInterval)
	request = httptest.NewRequest("GET", "/suggest?city_name=Lis", nil)
	request.Header.Set("Fly-Client-IP", "192.0.2.1")
	status, _ = doRequest(t, server, request)
	require.Equal(t, 200, status)
	select {
	case area := <-base.started:
		assert.Equal(t, "Lisbon", area.Name)
	case <-time.After(time.Second):
		t.Fatal("warmup did not resume after two seconds")
	}
	assert.Equal(t, 3, base.PlacesCalls())
	close(base.release)
}

// As a visitor with the map disabled, I want search suggestions without a base query.
func TestSuggestPlaces_DoesNotWarmWhenMapDisabled(t *testing.T) {
	server, _, _ := createTestServer(gomock.NewController(t))
	server.SetPlaceSuggester(&fakePlaceSuggester{places: []application.PlaceSuggestion{{Name: "Funchal", Country: "Portugal", Lat: 32.6669, Lon: -16.9241}}})
	base := &writermaptest.FakeBaseSource{}
	server.SetBasePlaceSource(base, true)
	server.SetBasePlaceSource(base, false)
	assert.Nil(t, server.suggestions.baseSource)
	status, _ := doRequest(t, server, httptest.NewRequest("GET", "/suggest?city_name=Fun", nil))
	require.Equal(t, 200, status)
	assert.Equal(t, 0, base.PlacesCalls())
}

// As a visitor after a temporary source error, I want the same destination to be eligible for another warmup.
func TestSuggestionWarm_RetriesFailedQuery(t *testing.T) {
	cache := newSuggestionCache()
	place := application.PlaceSuggestion{Name: "Funchal", Country: "Portugal", Lat: 32.6669, Lon: -16.9241}
	area := writermap.Area{Name: place.Name, Country: place.Country, Lat: place.Lat, Lon: place.Lon}
	key := baseWarmKey{place.Name, place.Country, place.Lat, place.Lon}
	failed := &writermaptest.FakeBaseSource{Err: errors.New("offline")}
	cache.baseSource = failed
	cache.baseStarted[key] = true
	cache.loadBase(failed, key, area)
	assert.False(t, cache.baseStarted[key])

	retry := &blockingBaseSource{FakeBaseSource: &writermaptest.FakeBaseSource{}, started: make(chan writermap.Area, 1), release: make(chan struct{})}
	cache.baseSource = retry
	cache.warm("192.0.2.2", place)
	select {
	case got := <-retry.started:
		assert.Equal(t, area, got)
	case <-time.After(time.Second):
		t.Fatal("failed warmup was not retried")
	}
	assert.Equal(t, 1, retry.PlacesCalls())
	close(retry.release)
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
