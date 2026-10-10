package writermap

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSourceStatus_NeverCheckedLeavesOutCheckedAt(t *testing.T) {
	status := SourceStatus{Host: "example.com", Status: "active"}

	data, err := json.Marshal(status)

	require.NoError(t, err)
	assert.JSONEq(t, `{"host":"example.com","status":"active"}`, string(data))
}

func TestSourceStatus_CheckedEncodesCheckedAt(t *testing.T) {
	checkedAt := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)
	status := SourceStatus{Host: "example.com", Status: "active", CheckedAt: &checkedAt}

	data, err := json.Marshal(status)

	require.NoError(t, err)
	assert.JSONEq(t, `{"host":"example.com","status":"active","checked_at":"2026-10-05T00:00:00Z"}`, string(data))
}

func TestPlace_EncodesArea(t *testing.T) {
	place := Place{QID: "Q856866", Names: []string{"Cabo Girão"}, Area: "Câmara de Lobos"}

	data, err := json.Marshal(place)

	require.NoError(t, err)
	assert.JSONEq(t, `{"qid":"Q856866","names":["Cabo Girão"],"lat":0,"lon":0,"area":"Câmara de Lobos"}`, string(data))
}

func TestPlace_EncodesOpeningHoursAndWikipediaURL(t *testing.T) {
	place := Place{QID: "Q25444", OpeningHours: "Open daily · 9:00 – 18:00", WikipediaURL: "https://en.wikipedia.org/wiki/Funchal"}

	data, err := json.Marshal(place)

	require.NoError(t, err)
	assert.JSONEq(t, `{"qid":"Q25444","names":null,"lat":0,"lon":0,"opening_hours":"Open daily · 9:00 – 18:00","wikipedia_url":"https://en.wikipedia.org/wiki/Funchal"}`, string(data))
}

func TestPlace_OmitsUnknownOpeningHoursAndWikipediaURL(t *testing.T) {
	place := Place{QID: "Q473169"}

	data, err := json.Marshal(place)

	require.NoError(t, err)
	assert.NotContains(t, string(data), "opening_hours")
	assert.NotContains(t, string(data), "wikipedia_url")
}

func TestMention_EncodesBlogName(t *testing.T) {
	mention := Mention{WriterHost: "www.portugalist.com", BlogName: "Portugalist"}

	data, err := json.Marshal(mention)

	require.NoError(t, err)
	assert.JSONEq(t, `{"writer_host":"www.portugalist.com","blog_name":"Portugalist","post_ref":"","post_url":"","post_title":"","language":"","positions":null}`, string(data))
}

func TestItinerary_DayEncodesOnlyStops(t *testing.T) {
	itinerary := Itinerary{Days: []ItineraryDay{{QIDs: []string{"Q25444", "Q856866"}}}}

	data, err := json.Marshal(itinerary.Days)

	require.NoError(t, err)
	assert.JSONEq(t, `[{"qids":["Q25444","Q856866"]}]`, string(data))
}
