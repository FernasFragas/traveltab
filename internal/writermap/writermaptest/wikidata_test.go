package writermaptest

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"weatherservice/internal/writermap"
)

// TestMadeiraFixture_PlacesMatchWikidata checks every fixture QID against Wikidata, as recorded in
// testdata/wikidata-places.json. RECORD_FIXTURES=1 refreshes that file from the live API.
func TestMadeiraFixture_PlacesMatchWikidata(t *testing.T) {
	fixture := loadMadeiraFixture(t)
	if os.Getenv("RECORD_FIXTURES") == "1" {
		recordWikidataPlaces(t, fixture.Places)
	}
	recorded := loadWikidataPlaces(t)

	for _, place := range fixture.Places {
		entity, ok := recorded[place.QID]
		if !assert.True(t, ok, "%s is not a Wikidata item (or redirects elsewhere)", place.QID) {
			continue
		}
		assert.Contains(t, place.Names, entity.Label, place.QID)
		assert.InDelta(t, entity.Lat, place.Lat, 0.05, "%s latitude", place.QID)
		assert.InDelta(t, entity.Lon, place.Lon, 0.05, "%s longitude", place.QID)
	}
}

const wikidataPlacesFile = "testdata/wikidata-places.json"

type wikidataPlace struct {
	Label string  `json:"label"`
	Lat   float64 `json:"lat"`
	Lon   float64 `json:"lon"`
}

func loadWikidataPlaces(t *testing.T) map[string]wikidataPlace {
	t.Helper()
	data, err := os.ReadFile(wikidataPlacesFile)
	require.NoError(t, err, "record it with RECORD_FIXTURES=1")
	var places map[string]wikidataPlace
	require.NoError(t, json.Unmarshal(data, &places))
	return places
}

// recordWikidataPlaces saves each item's English label and coordinates, keyed by the item's own
// ID, so a QID that redirects to another item is missing from the file.
func recordWikidataPlaces(t *testing.T, places []writermap.Place) {
	t.Helper()
	ids := make([]string, 0, len(places))
	for _, place := range places {
		ids = append(ids, place.QID)
	}
	query := url.Values{
		"action":    {"wbgetentities"},
		"ids":       {strings.Join(ids, "|")},
		"props":     {"labels|claims"},
		"languages": {"en"},
		"format":    {"json"},
	}
	request, err := http.NewRequest(http.MethodGet, "https://www.wikidata.org/w/api.php?"+query.Encode(), nil)
	require.NoError(t, err)
	request.Header.Set("User-Agent", "TravelTab-fixtures/0.1 (writers map test fixture)")
	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()
	require.Equal(t, http.StatusOK, response.StatusCode)

	var body struct {
		Entities map[string]struct {
			ID     string `json:"id"`
			Labels map[string]struct {
				Value string `json:"value"`
			} `json:"labels"`
			Claims struct {
				P625 []struct {
					Mainsnak struct {
						Datavalue struct {
							Value struct {
								Latitude  float64 `json:"latitude"`
								Longitude float64 `json:"longitude"`
							} `json:"value"`
						} `json:"datavalue"`
					} `json:"mainsnak"`
				} `json:"P625"`
			} `json:"claims"`
		} `json:"entities"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&body))

	recorded := map[string]wikidataPlace{}
	for _, entity := range body.Entities {
		place := wikidataPlace{Label: entity.Labels["en"].Value}
		if len(entity.Claims.P625) > 0 {
			coordinates := entity.Claims.P625[0].Mainsnak.Datavalue.Value
			place.Lat, place.Lon = coordinates.Latitude, coordinates.Longitude
		}
		recorded[entity.ID] = place
	}
	data, err := json.MarshalIndent(recorded, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(wikidataPlacesFile, append(data, '\n'), 0o644))
}
