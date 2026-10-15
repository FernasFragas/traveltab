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
		writers: newMadeiraPreviewSource(),
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

// newMadeiraPreviewSource adds panel and category examples without changing the shared fixture.
func newMadeiraPreviewSource() *writermaptest.FakeSource {
	source := writermaptest.NewMadeiraSource()
	for i := range source.Result.Places {
		place := &source.Result.Places[i]
		if place.QID == "Q25444" {
			place.PhotoURL = "https://upload.wikimedia.org/wikipedia/commons/thumb/0/0d/Funchal_Madeira_Portugal.jpg/640px-Funchal_Madeira_Portugal.jpg"
			place.PhotoCredit = "Wikimedia Commons"
			place.OpeningHours = "Visitor information: daily 09:00–18:00"
		}
	}
	source.Result.Mentions["Q25444"] = []writermap.Mention{
		{WriterHost: "www.earthtrekkers.com", BlogName: "Earth Trekkers", PostURL: "https://www.earthtrekkers.com/madeira-itinerary/", PostTitle: "Madeira Itinerary", Language: "en"},
		{WriterHost: "viagensasolta.com", BlogName: "Viagens à Solta", PostURL: "https://viagensasolta.com/roteiro-madeira/", PostTitle: "Roteiro de 5 dias na Madeira", Language: "pt"},
	}
	source.Result.WriterCounts["Q25444"] = 2
	source.Result.Places = append(source.Result.Places,
		writermap.Place{QID: "Q900001", Names: []string{"Funchal museum"}, Lat: 32.651, Lon: -16.912, Kind: "museum", Area: "Funchal", Description: "Preview museum."},
		writermap.Place{QID: "Q900002", Names: []string{"Funchal café"}, Lat: 32.649, Lon: -16.908, Kind: "cafe", Area: "Funchal", Description: "Preview café."},
	)
	return source
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
			{QID: "Q215003", Names: []string{"Belém Tower"}, Lat: 38.6916, Lon: -9.2160, Kind: "tower", Area: "Belém", Description: "A fortified tower overlooking the Tagus, with Manueline stonework and riverside views.", OpeningHours: "Tu-Su 10:00-18:30", WikipediaURL: "https://en.wikipedia.org/wiki/Bel%C3%A9m_Tower", PhotoURL: "https://upload.wikimedia.org/fixture-photos/lisbon-landmark.png", PhotoCredit: "Synthetic preview image"},
			{QID: "Q272781", Names: []string{"Jerónimos Monastery"}, Lat: 38.6979, Lon: -9.2064, Kind: "monastery", Area: "Belém", Description: "An ornate monastery with vaulted cloisters beside the gardens of Belém.", PhotoURL: "https://upload.wikimedia.org/fixture-photos/lisbon-landmark.png", PhotoCredit: "Synthetic preview image"},
			{QID: "Q900013", Names: []string{"Alfama"}, Lat: 38.7118, Lon: -9.1293, Kind: "neighborhood", Area: "Alfama", Description: "Narrow lanes, tiled houses and evening fado in Lisbon's oldest neighborhood.", PhotoURL: "https://upload.wikimedia.org/fixture-photos/lisbon-landmark.png", PhotoCredit: "Synthetic preview image"},
			{QID: "Q597", Names: []string{"Lisbon"}, Lat: 38.7223, Lon: -9.1393, Kind: "city", Area: "Lisbon", Description: "Portugal's hilltop capital on the Tagus River.", PhotoURL: "https://upload.wikimedia.org/fixture-photos/lisbon-landmark.png", PhotoCredit: "Synthetic preview image"},
			{QID: "Q900014", Names: []string{"National Tile Museum"}, Lat: 38.7244, Lon: -9.1130, Kind: "museum", Area: "Xabregas", Description: "Historic azulejo tiles displayed in a former convent.", PhotoURL: "https://upload.wikimedia.org/fixture-photos/lisbon-museum.png", PhotoCredit: "Synthetic preview image"},
			{QID: "Q900015", Names: []string{"MAAT"}, Lat: 38.6958, Lon: -9.1945, Kind: "museum", Area: "Belém", Description: "Contemporary art, architecture and technology on the waterfront.", PhotoURL: "https://upload.wikimedia.org/fixture-photos/lisbon-museum.png", PhotoCredit: "Synthetic preview image"},
			{QID: "Q900016", Names: []string{"Pastéis de Belém"}, Lat: 38.6975, Lon: -9.2032, Kind: "cafe", Area: "Belém", Description: "A traditional bakery known for warm custard tarts.", PhotoURL: "https://upload.wikimedia.org/fixture-photos/lisbon-food.png", PhotoCredit: "Synthetic preview image"},
			{QID: "Q900017", Names: []string{"Time Out Market Lisboa"}, Lat: 38.7071, Lon: -9.1459, Kind: "market", Area: "Cais do Sodré", Description: "A lively food hall inside the historic Ribeira market.", PhotoURL: "https://upload.wikimedia.org/fixture-photos/lisbon-food.png", PhotoCredit: "Synthetic preview image"},
			{QID: "Q900018", Names: []string{"Miradouro da Senhora do Monte"}, Lat: 38.7191, Lon: -9.1328, Kind: "viewpoint", Area: "Graça", Description: "A hilltop panorama across the castle, city roofs and river.", PhotoURL: "https://upload.wikimedia.org/fixture-photos/lisbon-nature.png", PhotoCredit: "Synthetic preview image"},
			{QID: "Q900019", Names: []string{"Miradouro de Santa Catarina"}, Lat: 38.7091, Lon: -9.1474, Kind: "viewpoint", Area: "Santa Catarina", Description: "A relaxed terrace above the Tagus with sunset views.", PhotoURL: "https://upload.wikimedia.org/fixture-photos/lisbon-nature.png", PhotoCredit: "Synthetic preview image"},
			{QID: "Q900020", Names: []string{"Estrela Garden"}, Lat: 38.7148, Lon: -9.1592, Kind: "garden", Area: "Estrela", Description: "Leafy paths, ponds and shaded benches beside the basilica.", PhotoURL: "https://upload.wikimedia.org/fixture-photos/lisbon-nature.png", PhotoCredit: "Synthetic preview image"},
			{QID: "Q900021", Names: []string{"Eduardo VII Park"}, Lat: 38.7291, Lon: -9.1524, Kind: "park", Area: "Avenidas Novas", Description: "A green hillside with formal hedges and views towards the river.", PhotoURL: "https://upload.wikimedia.org/fixture-photos/lisbon-nature.png", PhotoCredit: "Synthetic preview image"},
		},
		Mentions: map[string][]writermap.Mention{
			"Q272781": {{WriterHost: host, BlogName: "Earth Trekkers", PostRef: ref, PostURL: url, PostTitle: title, Language: "en"}},
			"Q215003": {
				{WriterHost: host, BlogName: "Earth Trekkers", PostRef: ref, PostURL: url, PostTitle: title, Language: "en"},
				{WriterHost: "fixture.writer", BlogName: "Lisbon Notebook (preview)", PostRef: "fixture.writer:belem", PostURL: "https://example.com/preview/belem", PostTitle: "A morning beside the Tagus", Language: "en"},
			},
		},
		WriterCounts: map[string]int{"Q272781": 1, "Q215003": 2},
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
