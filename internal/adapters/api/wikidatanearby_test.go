package api

import (
	"context"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"

	"weatherservice/internal/writermap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type nearbyTransport func(*http.Request) (*http.Response, error)

func (f nearbyTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestWikidataNearby_Places_MadeiraPinsAndGazetteer(t *testing.T) {
	fixture, err := os.ReadFile("testdata/wikidata_madeira.json")
	require.NoError(t, err)
	queries := 0
	client := &http.Client{Transport: nearbyTransport(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, UserAgent, req.Header.Get("User-Agent"))
		if strings.Contains(req.URL.Host, "query.wikidata.org") {
			queries++
			query := req.URL.Query().Get("query")
			assert.Contains(t, query, "wikibase:around")
			assert.Contains(t, query, "40")
			assert.Contains(t, query, "P31")
			assert.Contains(t, query, "P3025")
			assert.Contains(t, query, "?openTimeItem rdfs:label ?openTime")
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(fixture))), Header: make(http.Header)}, nil
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"query":{"pages":[]}}`)), Header: make(http.Header)}, nil
	})}

	got, err := NewWikidataNearbyAPI(client).Places(context.Background(), writermap.Area{Name: "Madeira", Lat: 32.75, Lon: -16.95})
	require.NoError(t, err)
	assert.Equal(t, 1, queries)
	require.Len(t, got.Pins, 2)
	assert.Equal(t, []string{"Q473169", "Q856866"}, []string{got.Pins[0].QID, got.Pins[1].QID})
	assert.ElementsMatch(t, []string{"Pico Ruivo", "Ruivo Peak", "Pico Ruivo de Santana"}, got.Pins[0].Names)
	assert.Equal(t, []string{"Q8502"}, got.Pins[0].Types)
	assert.Equal(t, "highest mountain on Madeira", got.Pins[0].Description)
	assert.Equal(t, "https://en.wikipedia.org/wiki/Pico_Ruivo", got.Pins[0].WikipediaURL)
	assert.Equal(t, "Open daily · 9:00 – 18:00", got.Pins[1].OpeningHours)
	assert.Empty(t, got.Pins[1].Photo)
	assert.Empty(t, got.Pins[1].WikipediaURL)
	require.Len(t, got.Gazetteer, 4)
	assert.Equal(t, "Q639161", got.Gazetteer[2].QID)
	assert.Equal(t, "Q623736", got.Gazetteer[3].QID)
	assert.Equal(t, []string{"Q13217644"}, got.Gazetteer[3].Types)
}

func TestWikidataNearby_Places_RadiusOverride(t *testing.T) {
	client := &http.Client{Transport: nearbyTransport(func(req *http.Request) (*http.Response, error) {
		assert.Contains(t, req.URL.Query().Get("query"), "80")
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"results":{"bindings":[]}}`)), Header: make(http.Header)}, nil
	})}
	got, err := NewWikidataNearbyAPI(client).Places(context.Background(), writermap.Area{RadiusKM: 80})
	require.NoError(t, err)
	assert.Empty(t, got.Pins)
	assert.Empty(t, got.Gazetteer)
}

func TestWikidataNearby_Places_CreditsCommonsImage(t *testing.T) {
	nearby, err := os.ReadFile("testdata/wikidata_madeira.json")
	require.NoError(t, err)
	commons, err := os.ReadFile("testdata/wikidata_commons.json")
	require.NoError(t, err)
	commonsCalls := 0
	client := &http.Client{Transport: nearbyTransport(func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Host, "commons.wikimedia.org") {
			commonsCalls++
			assert.Contains(t, req.URL.Query().Get("titles"), "File:Pico Ruivo.jpg")
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(commons))), Header: make(http.Header)}, nil
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(nearby))), Header: make(http.Header)}, nil
	})}

	got, err := NewWikidataNearbyAPI(client).Places(context.Background(), writermap.Area{})
	require.NoError(t, err)
	assert.Equal(t, 1, commonsCalls)
	assert.Equal(t, "https://upload.wikimedia.org/wikipedia/commons/a/aa/Pico_Ruivo.jpg", got.Pins[0].Photo)
	assert.Equal(t, "Ana · CC BY-SA 4.0", got.Pins[0].PhotoCredit)
	assert.Equal(t, "https://commons.wikimedia.org/wiki/File:Pico_Ruivo.jpg", got.Gazetteer[0].PhotoURL)
}

func TestWikidataNearby_Places_CommonsUnavailableLeavesPhotoEmpty(t *testing.T) {
	fixture, err := os.ReadFile("testdata/wikidata_madeira.json")
	require.NoError(t, err)
	client := &http.Client{Transport: nearbyTransport(func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Host, "commons.wikimedia.org") {
			return &http.Response{StatusCode: http.StatusServiceUnavailable, Body: io.NopCloser(strings.NewReader("unavailable")), Header: make(http.Header)}, nil
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(fixture))), Header: make(http.Header)}, nil
	})}
	got, err := NewWikidataNearbyAPI(client).Places(context.Background(), writermap.Area{})
	require.NoError(t, err)
	require.Len(t, got.Pins, 2)
	assert.Empty(t, got.Pins[0].Photo)
	assert.Empty(t, got.Gazetteer[0].Photo)
}

func TestWikidataNearby_Places_RetryAfterRespectsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	client := &http.Client{Transport: nearbyTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		cancel()
		return &http.Response{StatusCode: http.StatusTooManyRequests, Body: io.NopCloser(strings.NewReader("busy")), Header: http.Header{"Retry-After": {"30"}}}, nil
	})}
	_, err := NewWikidataNearbyAPI(client).Places(ctx, writermap.Area{})
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, calls)
}

func TestNearbyPlaces_CapsPinsButKeepsGazetteer(t *testing.T) {
	rows := make([]nearbyBinding, 205)
	for i := range rows {
		rows[i].Item.Value = "http://www.wikidata.org/entity/Q" + strconv.Itoa(i+1)
		rows[i].Coord.Value = "Point(-16.95 32.75)"
		rows[i].Type.Value = "http://www.wikidata.org/entity/Q8502"
		rows[i].EnLabel.Value = "Mountain " + strconv.Itoa(i+1)
		rows[i].Sitelinks.Value = strconv.Itoa(i)
	}
	got := nearbyPlaces(rows)
	require.Len(t, got.Pins, 200)
	require.Len(t, got.Gazetteer, 205)
	assert.Equal(t, "Q205", got.Pins[0].QID)
	assert.Equal(t, "Q6", got.Pins[199].QID)
}

func TestNearbyPlaces_DecodesP18FilePath(t *testing.T) {
	rows := []nearbyBinding{{}}
	rows[0].Item.Value = "http://www.wikidata.org/entity/Q1"
	rows[0].Coord.Value = "Point(-16.95 32.75)"
	rows[0].Type.Value = "http://www.wikidata.org/entity/Q8502"
	rows[0].EnLabel.Value = "Mountain"
	rows[0].Image.Value = "http://commons.wikimedia.org/wiki/Special:FilePath/Pico_Ruivo%20view.jpg"
	got := nearbyPlaces(rows)
	require.Len(t, got.Pins, 1)
	assert.Equal(t, "Pico Ruivo view.jpg", got.Pins[0].Photo)
}
