package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"weatherservice/internal/writermap"
	"weatherservice/internal/writermap/writermaptest"
)

// A traveler opening a destination with the flag enabled can browse its places and
// retrieve details; without the flag, the existing Waze card remains intact.
func TestWriterMap_FlaggedPageAndEndpoints(t *testing.T) {
	server := NewAppServer(nil, nil, testStore)
	server.SetWriterMapSource(&writermaptest.FakeSource{Result: writermap.DestinationResult{
		Places:       []writermap.Place{{QID: "Q1", Names: []string{"Cliff"}, Lat: 32.66, Lon: -17, Kind: "cliff", Area: "Madeira"}},
		WriterCounts: map[string]int{"Q1": 2}, Complete: false,
	}}, true)

	resp, err := server.app.Test(httptest.NewRequest("GET", "/map/funchal-pt.geojson?lat=32.65&lon=-16.91", nil))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "application/geo+json; charset=utf-8", resp.Header.Get("Content-Type"))
	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	body := string(bodyBytes)
	var collection struct {
		Type     string `json:"type"`
		Complete bool   `json:"complete"`
		Features []struct {
			Geometry struct {
				Coordinates []float64 `json:"coordinates"`
			} `json:"geometry"`
			Properties struct {
				QID         string `json:"qid"`
				Name        string `json:"name"`
				PanelURL    string `json:"panel_url"`
				WriterCount int    `json:"writer_count"`
			} `json:"properties"`
		} `json:"features"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &collection))
	assert.Equal(t, "FeatureCollection", collection.Type)
	assert.False(t, collection.Complete)
	require.Len(t, collection.Features, 1)
	assert.Equal(t, []float64{-17, 32.66}, collection.Features[0].Geometry.Coordinates)
	assert.Equal(t, "Q1", collection.Features[0].Properties.QID)
	assert.Equal(t, "Cliff", collection.Features[0].Properties.Name)
	assert.Equal(t, 2, collection.Features[0].Properties.WriterCount)
	assert.Equal(t, "/map/funchal-pt/Q1?lat=32.65&lon=-16.91", collection.Features[0].Properties.PanelURL)

	status, body := doRequest(t, server, httptest.NewRequest("GET", collection.Features[0].Properties.PanelURL, nil))
	require.Equal(t, 200, status)
	assert.Contains(t, body, "Cliff")
	assert.Contains(t, body, `data-save-qid="Q1"`)
	assert.NotContains(t, body, "AI summary")
}

func TestWriterMap_InvalidRequests(t *testing.T) {
	server := NewAppServer(nil, nil, testStore)
	server.SetWriterMapSource(&writermaptest.FakeSource{}, true)
	for _, path := range []string{
		"/map/invalid.geojson?lat=1&lon=2",
		"/map/invalid/Q1?lat=1&lon=2",
	} {
		status, _ := doRequest(t, server, httptest.NewRequest("GET", path, nil))
		assert.Equal(t, 404, status, path)
	}
	for _, path := range []string{
		"/map/funchal-pt.geojson?lon=2",
		"/map/funchal-pt.geojson?lat=NaN&lon=2",
		"/map/funchal-pt.geojson?lat=91&lon=2",
		"/map/funchal-pt/Q1?lat=1&lon=181",
	} {
		status, _ := doRequest(t, server, httptest.NewRequest("GET", path, nil))
		assert.Equal(t, 400, status, path)
	}
	status, _ := doRequest(t, server, httptest.NewRequest("GET", "/map/funchal-pt/Q999?lat=1&lon=2", nil))
	assert.Equal(t, 404, status)
}

func TestWriterMap_PanelEscapesSourceText(t *testing.T) {
	server := NewAppServer(nil, nil, testStore)
	server.SetWriterMapSource(&writermaptest.FakeSource{Result: writermap.DestinationResult{
		Places: []writermap.Place{{QID: "Q1", Names: []string{"<script>alert(1)</script>"}, Description: "<img src=x onerror=alert(1)>"}},
	}}, true)
	status, body := doRequest(t, server, httptest.NewRequest("GET", "/map/funchal-pt/Q1?lat=1&lon=2", nil))
	require.Equal(t, 200, status)
	assert.NotContains(t, body, "<script>")
	assert.NotContains(t, body, "<img src=x")
	assert.Contains(t, body, "&lt;script&gt;")
}

func TestWriterMap_FixturePanel(t *testing.T) {
	server := NewAppServer(nil, nil, testStore)
	server.SetWriterMapSource(writermaptest.NewMadeiraSource(), true)
	status, body := doRequest(t, server, httptest.NewRequest("GET", "/map/funchal-pt/Q473169?lat=32.65&lon=-16.91", nil))
	require.Equal(t, 200, status)
	assert.Contains(t, body, "Pico Ruivo")
	assert.Contains(t, body, "AI summary")
	assert.Contains(t, body, "Hiking Pico Ruivo")
	assert.Contains(t, body, "Salt in our Hair")
	assert.Contains(t, body, `data-language="pt"`)
	assert.Contains(t, body, "Mentioned by 2 writers")
}

func TestWriterMap_PageFlagStates(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		ctrl := gomock.NewController(t)
		server, weather, videos := createTestServer(ctrl)
		server.SetWriterMapSource(&writermaptest.FakeSource{}, enabled)
		city := "Mapflagoff"
		if enabled {
			city = "Mapflagon"
		}
		info := createTestGeneralWeatherInfoFor(city)
		info.Country = "PT"
		weather.EXPECT().GenerateReport(gomock.Any(), city).Return(info, nil)
		videos.EXPECT().GenerateReport(gomock.Any(), "Turistic places in "+city+", PT").Return(createTestVideosStream(), nil)
		status, body := doRequest(t, server, httptest.NewRequest("GET", "/process-form/?city_name="+city, nil))
		require.Equal(t, 200, status)
		if enabled {
			for _, slot := range []string{"panel", "layers", "saved", "status"} {
				assert.Contains(t, body, `data-map-slot="`+slot+`"`)
			}
			assert.NotContains(t, body, `data-map-slot="list"`, "the place-tag list was removed")
			assert.Contains(t, body, `data-writers-map`)
			assert.Contains(t, body, `hx-trigger="load"`)
			assert.Contains(t, body, "/map/mapflagon-pt.geojson?lat=38.7223&amp;lon=-9.1393")
			assert.Contains(t, body, "View larger map")
			assert.NotContains(t, body, "Map by Waze")
		} else {
			assert.Contains(t, body, "Map by Waze")
			assert.Contains(t, body, "https://embed.waze.com/iframe?zoom=10&amp;lat=38.7223&amp;lon=-9.1393")
			assert.False(t, strings.Contains(body, "data-writers-map"))
		}
	}
}

func TestWriterMap_TripPageUsesTheFlaggedShell(t *testing.T) {
	ctrl := gomock.NewController(t)
	server, weather, videos := createTestServer(ctrl)
	server.SetWriterMapSource(writermaptest.NewMadeiraSource(), true)
	info := createTestGeneralWeatherInfoFor("Funchal")
	info.Country = "PT"
	info.Lat, info.Lon = 32.65, -16.91
	weather.EXPECT().GenerateReport(gomock.Any(), "funchal, pt").Return(info, nil)
	videos.EXPECT().GenerateReport(gomock.Any(), "Turistic places in Funchal, PT").Return(createTestVideosStream(), nil)
	status, body := doRequest(t, server, httptest.NewRequest("GET", "/trip/funchal-pt", nil))
	require.Equal(t, 200, status)
	assert.Contains(t, body, `data-writers-map`)
	assert.Contains(t, body, `/map/funchal-pt.geojson?lat=32.65&amp;lon=-16.91`)
	assert.NotContains(t, body, "Map by Waze")
}

// As the owner, I want the server to set the 3D basemap only when asked, so production keeps
// the browser defaults.
func TestWriterMap_ShellOmitsBasemapAttributesByDefault(t *testing.T) {
	body := mapShellBody(t, "Defaultcity", func(*Server) {})
	for _, attr := range []string{"data-map-style-url", "data-map-terrain-url", "data-map-engine"} {
		assert.NotContains(t, body, attr)
	}
}

func TestWriterMap_ShellCarriesBasemapAttributesWhenSet(t *testing.T) {
	body := mapShellBody(t, "Setcity", func(s *Server) {
		s.SetWriterMapBasemap("http://x/style.json", "http://x/{z}/{x}/{y}.png", "leaflet")
	})
	assert.Contains(t, body, `data-map-style-url="http://x/style.json"`)
	// html/template escapes braces in URL attributes, as it does for data-map-tile-url; the
	// browser core restores them.
	assert.Contains(t, body, `data-map-terrain-url="http://x/%7bz%7d/%7bx%7d/%7by%7d.png"`)
	assert.Contains(t, body, `data-map-engine="leaflet"`)
}

func TestWriterMap_ShellCanTurnTerrainOff(t *testing.T) {
	body := mapShellBody(t, "Offcity", func(s *Server) { s.SetWriterMapTerrainOff() })
	assert.Contains(t, body, `data-map-terrain-url=""`)
	assert.NotContains(t, body, "data-map-style-url")
	assert.NotContains(t, body, "data-map-engine")
}

func mapShellBody(t *testing.T, city string, configure func(*Server)) string {
	t.Helper()
	ctrl := gomock.NewController(t)
	server, weather, videos := createTestServer(ctrl)
	server.SetWriterMapSource(writermaptest.NewMadeiraSource(), true)
	configure(server)
	info := createTestGeneralWeatherInfoFor(city)
	info.Country = "PT"
	info.Lat, info.Lon = 32.65, -16.91
	weather.EXPECT().GenerateReport(gomock.Any(), strings.ToLower(city)+", pt").Return(info, nil)
	videos.EXPECT().GenerateReport(gomock.Any(), "Turistic places in "+city+", PT").Return(createTestVideosStream(), nil)
	status, body := doRequest(t, server, httptest.NewRequest("GET", "/trip/"+strings.ToLower(city)+"-pt", nil))
	require.Equal(t, 200, status)
	return body
}

// As a traveler, I want writers' routes on the map, so the browser receives each itinerary with
// its post link and days of stops, and never the post's internal reference.
func TestWriterMap_GeoJSONCarriesItineraries(t *testing.T) {
	server := NewAppServer(nil, nil, testStore)
	server.SetWriterMapSource(writermaptest.NewMadeiraSource(), true)

	status, body := doRequest(t, server, httptest.NewRequest("GET", "/map/funchal-pt.geojson?lat=32.65&lon=-16.91", nil))

	require.Equal(t, 200, status)
	var collection struct {
		Itineraries []struct {
			WriterHost string `json:"writer_host"`
			PostURL    string `json:"post_url"`
			PostTitle  string `json:"post_title"`
			Language   string `json:"language"`
			Days       []struct {
				QIDs []string `json:"qids"`
			} `json:"days"`
		} `json:"itineraries"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &collection))
	require.Len(t, collection.Itineraries, 2)
	first := collection.Itineraries[0]
	assert.Equal(t, "www.earthtrekkers.com", first.WriterHost)
	assert.Equal(t, "https://www.earthtrekkers.com/madeira-itinerary/", first.PostURL)
	assert.Equal(t, "Madeira Itinerary", first.PostTitle)
	assert.Equal(t, "en", first.Language)
	require.Len(t, first.Days, 3)
	assert.Equal(t, []string{"Q25444", "Q856866", "Q34799706"}, first.Days[0].QIDs)
	assert.NotContains(t, body, "post_ref")
}

func TestWriterMap_GeoJSONHasEmptyItinerariesWhenThereAreNone(t *testing.T) {
	server := NewAppServer(nil, nil, testStore)
	server.SetWriterMapSource(&writermaptest.FakeSource{Empty: true}, true)

	status, body := doRequest(t, server, httptest.NewRequest("GET", "/map/funchal-pt.geojson?lat=32.65&lon=-16.91", nil))

	require.Equal(t, 200, status)
	assert.Contains(t, body, `"itineraries":[]`)
}

// loadingSource is a writers' map source that is still loading base places.
type loadingSource struct {
	*writermaptest.FakeSource
	result  writermap.DestinationResult
	loading bool
	err     error
}

func (l *loadingSource) DestinationProgress(context.Context, writermap.Area) (*writermap.DestinationResult, bool, error) {
	if l.err != nil {
		return nil, false, l.err
	}
	result := l.result
	return &result, l.loading, nil
}

// As a traveller opening a big city cold, I want the map to answer straight away with the pins
// found so far and say it is still loading, so that I see places instead of an error.
func TestWriterMap_GeoJSONServesPinsSoFarWhileLoading(t *testing.T) {
	server := NewAppServer(nil, nil, testStore)
	server.SetWriterMapSource(&loadingSource{
		FakeSource: &writermaptest.FakeSource{},
		result: writermap.DestinationResult{
			Places:       []writermap.Place{{QID: "Q1", Names: []string{"Cliff"}, Lat: 32.66, Lon: -17}},
			WriterCounts: map[string]int{},
		},
		loading: true,
	}, true)

	status, body := doRequest(t, server, httptest.NewRequest("GET", "/map/funchal-pt.geojson?lat=32.65&lon=-16.91", nil))

	require.Equal(t, 200, status, "a slow load is not a 503")
	var collection struct {
		Loading  bool `json:"loading"`
		Complete bool `json:"complete"`
		Features []struct {
			Properties struct {
				QID string `json:"qid"`
			} `json:"properties"`
		} `json:"features"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &collection))
	assert.True(t, collection.Loading)
	assert.False(t, collection.Complete)
	require.Len(t, collection.Features, 1)
	assert.Equal(t, "Q1", collection.Features[0].Properties.QID)
}

func TestWriterMap_GeoJSONSaysLoadingFalseWhenTheLoadIsDone(t *testing.T) {
	server := NewAppServer(nil, nil, testStore)
	server.SetWriterMapSource(&loadingSource{FakeSource: &writermaptest.FakeSource{}, result: writermap.DestinationResult{WriterCounts: map[string]int{}, Complete: true}}, true)

	status, body := doRequest(t, server, httptest.NewRequest("GET", "/map/funchal-pt.geojson?lat=32.65&lon=-16.91", nil))

	require.Equal(t, 200, status)
	assert.Contains(t, body, `"loading":false`)
}

func TestWriterMap_GeoJSONWithAPlainSourceIsNeverLoading(t *testing.T) {
	server := NewAppServer(nil, nil, testStore)
	server.SetWriterMapSource(&writermaptest.FakeSource{Empty: true}, true)

	status, body := doRequest(t, server, httptest.NewRequest("GET", "/map/funchal-pt.geojson?lat=32.65&lon=-16.91", nil))

	require.Equal(t, 200, status)
	assert.Contains(t, body, `"loading":false`)
}

func TestWriterMap_GeoJSONStillFailsWhenNothingLoaded(t *testing.T) {
	server := NewAppServer(nil, nil, testStore)
	server.SetWriterMapSource(&loadingSource{FakeSource: &writermaptest.FakeSource{}, err: errors.New("wikidata down")}, true)

	status, _ := doRequest(t, server, httptest.NewRequest("GET", "/map/funchal-pt.geojson?lat=32.65&lon=-16.91", nil))

	assert.Equal(t, 503, status)
}

func TestWriterMap_PanelOpensForAPinWhileLoading(t *testing.T) {
	server := NewAppServer(nil, nil, testStore)
	place := writermap.Place{QID: "Q1", Names: []string{"Cliff"}, Lat: 32.66, Lon: -17}
	server.SetWriterMapSource(&loadingSource{
		FakeSource: &writermaptest.FakeSource{Result: writermap.DestinationResult{Places: []writermap.Place{place}}},
		result:     writermap.DestinationResult{Places: []writermap.Place{place}, WriterCounts: map[string]int{}},
		loading:    true,
	}, true)

	status, body := doRequest(t, server, httptest.NewRequest("GET", "/map/funchal-pt/Q1?lat=32.65&lon=-16.91", nil))

	require.Equal(t, 200, status)
	assert.Contains(t, body, "Cliff")
}

// As a traveller, I never want to see a raw QID as a place name, so a pin whose name has not
// loaded yet has an empty name for the browser to treat as "loading".
func TestWriterMap_GeoJSONNamelessPinHasAnEmptyNameNotItsQID(t *testing.T) {
	server := NewAppServer(nil, nil, testStore)
	server.SetWriterMapSource(&loadingSource{
		FakeSource: &writermaptest.FakeSource{},
		result:     writermap.DestinationResult{Places: []writermap.Place{{QID: "Q42", Lat: 32.66, Lon: -17}}, WriterCounts: map[string]int{}},
		loading:    true,
	}, true)

	status, body := doRequest(t, server, httptest.NewRequest("GET", "/map/funchal-pt.geojson?lat=32.65&lon=-16.91", nil))

	require.Equal(t, 200, status)
	assert.Contains(t, body, `"name":""`)
	assert.Contains(t, body, `"names":[]`)
}

func TestWriterMap_PanelForANamelessPinDoesNotShowTheQID(t *testing.T) {
	server := NewAppServer(nil, nil, testStore)
	place := writermap.Place{QID: "Q42", Lat: 32.66, Lon: -17}
	server.SetWriterMapSource(&loadingSource{
		FakeSource: &writermaptest.FakeSource{Result: writermap.DestinationResult{Places: []writermap.Place{place}}},
		result:     writermap.DestinationResult{Places: []writermap.Place{place}, WriterCounts: map[string]int{}},
		loading:    true,
	}, true)

	status, body := doRequest(t, server, httptest.NewRequest("GET", "/map/funchal-pt/Q42?lat=32.65&lon=-16.91", nil))

	require.Equal(t, 200, status)
	assert.Contains(t, body, "Loading name")
}
