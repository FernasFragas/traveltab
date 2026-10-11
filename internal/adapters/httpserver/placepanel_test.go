package httpserver

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"weatherservice/internal/writermap"
	"weatherservice/internal/writermap/writermaptest"
)

// As a traveler, I want to see which writers mention a place and open their posts,
// so that I can read their full take on the writer's own site.
func TestPlacePanel_ListsEveryWriterPostOnceExpanded(t *testing.T) {
	body := placePanel(t, writermaptest.NewMadeiraSource(), "Q473169")

	writers := section(t, body, `<details class="map-panel-writers"`, `</details>`)
	assert.Contains(t, writers, "View writers' posts (2)")
	assert.Contains(t, writers, `href="https://www.saltinourhair.com/madeira/pico-ruivo/"`)
	assert.Contains(t, writers, "Hiking Pico Ruivo")
	assert.Contains(t, writers, "Salt in our Hair")
	assert.Contains(t, writers, `href="https://viagensasolta.com/pico-ruivo/"`)
	assert.Contains(t, writers, "Pico Ruivo na Madeira")
	assert.Contains(t, writers, `<span class="map-panel-language">pt</span>`)
	assert.Contains(t, writers, `bi-box-arrow-up-right`)
}

func TestPlacePanel_ShowsPartsInMockupOrder(t *testing.T) {
	body := placePanel(t, writermaptest.NewMadeiraSource(), "Q473169")

	order := []string{
		`class="map-panel-close"`, "<h3>Pico Ruivo</h3>", `class="map-panel-meta"`,
		"Mentioned by 2 writers", "AI summary", `class="map-panel-description"`,
		`data-save-qid="Q473169"`, "View writers' posts (2)",
	}
	last := -1
	for _, part := range order {
		index := strings.Index(body, part)
		require.NotEqual(t, -1, index, "missing %q", part)
		assert.Greater(t, index, last, "%q is out of order", part)
		last = index
	}
}

func TestPlacePanel_WriterPlaceWithoutSummaryHasNoSummaryBox(t *testing.T) {
	body := placePanel(t, writermaptest.NewMadeiraSource(), "Q1192102")

	assert.Contains(t, body, "Mentioned by 1 writer<")
	assert.NotContains(t, body, "AI summary")
	assert.NotContains(t, body, "map-panel-summary")
}

func TestPlacePanel_PlaceWithoutHoursOrWikipediaHasNoEmptyRows(t *testing.T) {
	body := placePanel(t, writermaptest.NewMadeiraSource(), "Q856866")

	assert.NotContains(t, body, "map-panel-hours")
	assert.NotContains(t, body, "View on Wikipedia")
}

func TestPlacePanel_ShowsOpeningHoursWhenPresent(t *testing.T) {
	body := placePanel(t, writermaptest.NewMadeiraSource(), "Q43179137")

	assert.Contains(t, body, `class="map-panel-hours"`)
	assert.Contains(t, body, "Open daily · 9:00 – 18:00")
}

func TestPlacePanel_LinksToWikipediaWhenPresent(t *testing.T) {
	body := placePanel(t, writermaptest.NewMadeiraSource(), "Q25444")

	assert.Contains(t, body, `href="https://en.wikipedia.org/wiki/Funchal"`)
	assert.Contains(t, body, "View on Wikipedia")
}

func TestPlacePanel_BaseOnlyDestinationSaysNoWriterMentions(t *testing.T) {
	source := &writermaptest.FakeSource{Result: writermap.DestinationResult{
		Places: []writermap.Place{{
			QID: "Q1", Names: []string{"Torre de Belém"}, Kind: "tower", Area: "Lisbon",
			Description: "A fortified tower.", PhotoURL: "https://example.org/belem.jpg",
			PhotoCredit: "Photo: A. Person, CC BY-SA 4.0",
		}},
	}}

	body := placePanel(t, source, "Q1")

	assert.Contains(t, body, "No writer mentions here yet")
	assert.Contains(t, body, "A fortified tower.")
	assert.Contains(t, body, `src="https://example.org/belem.jpg"`)
	assert.Contains(t, body, "Photo: A. Person, CC BY-SA 4.0")
	assert.Contains(t, body, `data-save-qid="Q1"`)
	assert.NotContains(t, body, "Mentioned by")
	assert.NotContains(t, body, "map-panel-writers")
}

func placePanel(t *testing.T, source *writermaptest.FakeSource, qid string) string {
	t.Helper()
	server := NewAppServer(nil, nil, testStore)
	server.SetWriterMapSource(source, true)
	status, body := doRequest(t, server, httptest.NewRequest("GET", "/map/funchal-pt/"+qid+"?lat=32.65&lon=-16.91", nil))
	require.Equal(t, 200, status)
	return body
}

func section(t *testing.T, body, start, end string) string {
	t.Helper()
	from := strings.Index(body, start)
	require.NotEqual(t, -1, from, "missing %q", start)
	to := strings.Index(body[from:], end)
	require.NotEqual(t, -1, to, "missing %q after %q", end, start)
	return body[from : from+to]
}
