package httpserver

import (
	"context"
	"errors"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"weatherservice/internal/adapters/sqlite"
	"weatherservice/internal/writermap"
	"weatherservice/internal/writermap/writermaptest"
)

// As a traveler, I want a destination and its day plan to remain usable when Wikidata is
// unavailable on a cold map load, with the map's failure reported instead of hidden.
func TestWriterMapResilience_ColdBaseFailureKeepsTilesAndDayPlan(t *testing.T) {
	store, err := sqlite.Open(filepath.Join(t.TempDir(), "writers.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	base := store.NewCachedBasePlaceSource(failedWikidata{})
	source := writermap.NewMapSource(base, noWriterMatches{}, store, nil, nil, writermap.Overrides{})
	_, err = source.Destination(context.Background(), writermap.Area{Name: "Mapdown", Country: "PT", Lat: 38.7223, Lon: -9.1393})
	require.ErrorContains(t, err, "Wikidata unavailable")
	areas, err := store.CachedDestinations(context.Background())
	require.NoError(t, err)
	assert.Empty(t, areas, "a failed cold load leaves no cache row")

	ctrl := gomock.NewController(t)
	server, weather, videos := createTestServer(ctrl)
	server.SetWriterMapSource(source, true)
	server.SetWriterMapTiles("https://tiles.example/{z}/{x}/{y}.png", "Test tiles")
	server.now = func() time.Time { return planNow }
	server.SetTripPlanner(fakeTripPlanner{build: planWithStays(nil, "")})
	info := createTestGeneralWeatherInfoFor("Mapdown")
	info.Country = "PT"
	weather.EXPECT().GenerateReport(gomock.Any(), "Mapdown").Return(info, nil)
	videos.EXPECT().GenerateReport(gomock.Any(), "Turistic places in Mapdown, PT").Return(createTestVideosStream(), nil)

	status, page := doRequest(t, server, httptest.NewRequest("GET", "/process-form/?city_name=Mapdown", nil))
	require.Equal(t, 200, status)
	assert.Contains(t, page, `data-map-tile-url="https://tiles.example/`)
	assert.Contains(t, page, `data-map-slot="status"`)
	assert.Contains(t, page, `role="status"`)
	assert.Contains(t, page, `data-map-url="/map/mapdown-pt.geojson`)

	status, _ = doRequest(t, server, httptest.NewRequest("GET", "/map/mapdown-pt.geojson?lat=38.7223&lon=-9.1393", nil))
	assert.Equal(t, 503, status, "the places request reports the missing data")

	status, plan := doRequest(t, server, planRequest(nil))
	require.Equal(t, 200, status)
	assert.Contains(t, plan, "Belem Tower")
}

// As a traveler, I want the writer's original post link even before an AI summary exists.
func TestWriterMapResilience_MissingSummariesKeepWriterLinks(t *testing.T) {
	for _, summaries := range []string{"empty table", "AI disabled"} {
		t.Run(summaries, func(t *testing.T) {
			source := writermaptest.NewMadeiraSource()
			if summaries == "empty table" {
				source.Summaries = map[string]writermap.Summary{}
			} else {
				source.Summaries = nil
			}
			server := NewAppServer(nil, nil, testStore)
			server.SetWriterMapSource(source, true)

			status, body := doRequest(t, server, httptest.NewRequest("GET", "/map/funchal-pt/Q473169?lat=32.65&lon=-16.91", nil))

			require.Equal(t, 200, status)
			assert.Contains(t, body, "Hiking Pico Ruivo")
			assert.Contains(t, body, "https://www.saltinourhair.com/madeira/pico-ruivo/")
			assert.NotContains(t, body, "AI summary")
		})
	}
}

type failedWikidata struct{}

func (failedWikidata) Places(context.Context, writermap.Area) (writermap.BasePlaces, error) {
	return writermap.BasePlaces{}, errors.New("Wikidata unavailable")
}

type noWriterMatches struct{}

func (noWriterMatches) WriterMatchRows(context.Context, []string) ([]writermap.MatchRow, error) {
	return nil, nil
}
