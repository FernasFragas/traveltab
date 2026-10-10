package httpserver

import (
	"encoding/json"
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
			for _, slot := range []string{"panel", "layers", "saved", "status", "list"} {
				assert.Contains(t, body, `data-map-slot="`+slot+`"`)
			}
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
