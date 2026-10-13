package main

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"weatherservice/internal/application"
	"weatherservice/internal/writermap"
	"weatherservice/internal/writermap/writermaptest"
)

var (
	fixtureTileOnce sync.Once
	fixtureTilePNG  []byte
)

// writerMapPreviewSource gives the preview three distinct destination states.
type writerMapPreviewSource struct {
	writers *writermaptest.FakeSource
	lisbon  *writermaptest.FakeSource
	base    *writermaptest.FakeSource
	empty   *writermaptest.FakeSource
}

var _ application.WriterMapSource = (*writerMapPreviewSource)(nil)

func newWriterMapPreviewSource() *writerMapPreviewSource {
	return &writerMapPreviewSource{
		writers: writermaptest.NewMadeiraSource(),
		lisbon:  newLisbonPreviewSource(),
		base: &writermaptest.FakeSource{Result: writermap.DestinationResult{
			Places: []writermap.Place{{QID: "Q208954", Names: []string{"Clérigos Tower"},
				Lat: 41.1457, Lon: -8.6148, Kind: "tower", Description: "A tower in Porto."}},
			WriterCounts: map[string]int{}, Complete: true,
		}},
		empty: &writermaptest.FakeSource{Result: writermap.DestinationResult{
			Places: []writermap.Place{}, WriterCounts: map[string]int{}, Complete: false,
			SourceStatuses: []writermap.SourceStatus{{Host: "fixture.base", Status: "unavailable"},
				{Host: "fixture.writers", Status: "unavailable"}},
		}},
	}
}

func (f *writerMapPreviewSource) Destination(ctx context.Context, area writermap.Area) (*writermap.DestinationResult, error) {
	return f.source(area).Destination(ctx, area)
}

func (f *writerMapPreviewSource) Place(ctx context.Context, area writermap.Area, qid string) (*writermap.Place, *writermap.Summary, error) {
	return f.source(area).Place(ctx, area, qid)
}

func (f *writerMapPreviewSource) source(area writermap.Area) *writermaptest.FakeSource {
	switch strings.ToLower(area.Name) {
	case "madeira", "funchal":
		return f.writers
	case "lisbon":
		return f.lisbon
	case "porto":
		return f.base
	default:
		return f.empty
	}
}

func newLisbonPreviewSource() *writermaptest.FakeSource {
	const host = "www.earthtrekkers.com"
	const ref = "www.earthtrekkers.com:lisbon-preview"
	const url = "https://www.earthtrekkers.com/one-day-in-lisbon-itinerary/"
	const title = "One Day in Lisbon, Portugal: 2 Detailed Itineraries"
	return &writermaptest.FakeSource{Result: writermap.DestinationResult{
		Places: []writermap.Place{
			{QID: "Q597", Names: []string{"Lisbon"}, Lat: 38.7223, Lon: -9.1393, Kind: "city", Area: "Lisbon"},
			{QID: "Q272781", Names: []string{"Jerónimos Monastery"}, Lat: 38.6979, Lon: -9.2064, Kind: "monastery", Area: "Belém"},
			{QID: "Q215003", Names: []string{"Belém Tower"}, Lat: 38.6916, Lon: -9.2160, Kind: "tower", Area: "Belém"},
		},
		Mentions: map[string][]writermap.Mention{
			"Q272781": {{WriterHost: host, BlogName: "Earth Trekkers", PostRef: ref, PostURL: url, PostTitle: title, Language: "en"}},
			"Q215003": {{WriterHost: host, BlogName: "Earth Trekkers", PostRef: ref, PostURL: url, PostTitle: title, Language: "en"}},
		},
		WriterCounts: map[string]int{"Q272781": 1, "Q215003": 1},
		Itineraries:  []writermap.Itinerary{{WriterHost: host, PostRef: ref, PostURL: url, PostTitle: title, Language: "en", Days: []writermap.ItineraryDay{{QIDs: []string{"Q272781", "Q215003", "Q597"}}}}},
		Complete:     true,
	}}
}

// fixtureTileHandler serves the same local, unmistakably synthetic tile at valid XYZ paths.
func fixtureTileHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/fixture-tiles/"), "/")
	if len(parts) != 3 || !strings.HasSuffix(parts[2], ".png") {
		http.NotFound(w, r)
		return
	}
	for _, value := range []string{parts[0], parts[1], strings.TrimSuffix(parts[2], ".png")} {
		if _, err := strconv.ParseUint(value, 10, 32); err != nil {
			http.NotFound(w, r)
			return
		}
	}
	fixtureTileOnce.Do(func() {
		img := image.NewRGBA(image.Rect(0, 0, 256, 256))
		for y := 0; y < 256; y++ {
			for x := 0; x < 256; x++ {
				shade := color.RGBA{0xc8, 0xdd, 0xc2, 0xff}
				if (x+y)%83 < 12 {
					shade = color.RGBA{0xef, 0xe6, 0xcf, 0xff}
				}
				img.SetRGBA(x, y, shade)
			}
		}
		var out bytes.Buffer
		if err := png.Encode(&out, img); err != nil {
			panic(err)
		}
		fixtureTilePNG = out.Bytes()
	})
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(fixtureTilePNG)
}
