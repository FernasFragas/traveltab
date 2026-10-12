package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// As a developer, I want the 3D map to load with no network, so the style, glyphs, sprite, vector
// tiles and terrain all come from the preview origin.
func TestPreview_ServesALocalMapStyle(t *testing.T) {
	origin := newMapOrigin(t)
	res, body := get(t, origin+"/fixture-map/style.json")
	assertContentType(t, res, "application/json")
	if res.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("style is not readable cross-origin: %v", res.Header)
	}
	var style struct {
		Version int    `json:"version"`
		Glyphs  string `json:"glyphs"`
		Sprite  string `json:"sprite"`
		Sources map[string]struct {
			Type  string   `json:"type"`
			Tiles []string `json:"tiles"`
		} `json:"sources"`
		Layers []json.RawMessage `json:"layers"`
	}
	if err := json.Unmarshal(body, &style); err != nil {
		t.Fatalf("style is not JSON: %v", err)
	}
	if style.Version != 8 || len(style.Sources) == 0 || len(style.Layers) == 0 {
		t.Fatalf("incomplete style: %s", body)
	}
	for _, u := range regexp.MustCompile(`https?://[^"]+`).FindAllString(string(body), -1) {
		if !strings.HasPrefix(u, origin+"/") {
			t.Errorf("style points off the preview origin: %s", u)
		}
	}
	for name, src := range style.Sources {
		if len(src.Tiles) == 0 || !strings.HasPrefix(src.Tiles[0], origin+"/fixture-map/tiles/") {
			t.Errorf("source %s tiles = %v", name, src.Tiles)
		}
	}
	if !strings.HasPrefix(style.Glyphs, origin+"/fixture-map/glyphs/") || !strings.HasPrefix(style.Sprite, origin+"/fixture-map/sprite") {
		t.Errorf("glyphs = %q, sprite = %q", style.Glyphs, style.Sprite)
	}
	if !strings.Contains(string(body), `"fill-extrusion"`) || !strings.Contains(string(body), `"render_height"`) {
		t.Error("style has no extruded buildings")
	}
}

func TestPreview_ServesGlyphsAndSprite(t *testing.T) {
	origin := newMapOrigin(t)
	res, body := get(t, origin+"/fixture-map/glyphs/Fixture%20Sans/0-255.pbf")
	assertContentType(t, res, "application/x-protobuf")
	if !strings.Contains(string(body), "Fixture Sans") || !strings.Contains(string(body), "0-255") {
		t.Errorf("glyph stub = %q", body)
	}
	res, body = get(t, origin+"/fixture-map/sprite.json")
	assertContentType(t, res, "application/json")
	if !json.Valid(body) {
		t.Errorf("sprite json = %q", body)
	}
	for _, path := range []string{"/fixture-map/sprite.png", "/fixture-map/sprite@2x.png", "/fixture-map/sprite@2x.json"} {
		res, body = get(t, origin+path)
		if strings.HasSuffix(path, ".png") {
			assertContentType(t, res, "image/png")
			if _, err := png.Decode(strings.NewReader(string(body))); err != nil {
				t.Errorf("%s is not a PNG: %v", path, err)
			}
		} else {
			assertContentType(t, res, "application/json")
		}
	}
}

func TestPreview_ServesAVectorTileOverFunchal(t *testing.T) {
	origin := newMapOrigin(t)
	x, y := tileXY(32.65, -16.91, 14)
	res, body := get(t, origin+"/fixture-map/tiles/14/"+strconv.Itoa(x)+"/"+strconv.Itoa(y)+".mvt")
	assertContentType(t, res, "application/vnd.mapbox-vector-tile")
	layers := mvtLayers(t, body)
	for _, name := range []string{"water", "transportation", "building"} {
		if layers[name] == 0 {
			t.Errorf("layer %q has no features; layers = %v", name, layers)
		}
	}
	if layers["building"] < 3 || layers["transportation"] < 2 {
		t.Errorf("want at least 3 buildings and 2 roads, got %v", layers)
	}
	if !strings.Contains(string(body), "render_height") {
		t.Error("buildings carry no render_height")
	}
}

func TestPreview_ServesAnEmptyVectorTileAwayFromFunchal(t *testing.T) {
	origin := newMapOrigin(t)
	res, body := get(t, origin+"/fixture-map/tiles/14/0/0.mvt")
	assertContentType(t, res, "application/vnd.mapbox-vector-tile")
	if len(mvtLayers(t, body)) != 0 {
		t.Errorf("tile away from Funchal has features: %v", mvtLayers(t, body))
	}
}

func TestPreview_ServesAFlatTerrainTile(t *testing.T) {
	origin := newMapOrigin(t)
	res, body := get(t, origin+"/fixture-map/terrain/12/1000/800.png")
	assertContentType(t, res, "image/png")
	img, err := png.Decode(strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("terrain is not a PNG: %v", err)
	}
	if img.Bounds().Dx() != 512 || img.Bounds().Dy() != 512 {
		t.Errorf("terrain size = %v", img.Bounds())
	}
	// Terrarium: metres = R*256 + G + B/256 - 32768, so sea level is R=128, G=0, B=0.
	for _, p := range [][2]int{{0, 0}, {256, 256}, {511, 511}} {
		r, g, b, _ := img.At(p[0], p[1]).RGBA()
		if r>>8 != 128 || g>>8 != 0 || b>>8 != 0 {
			t.Fatalf("pixel %v = %d,%d,%d, want sea level 128,0,0", p, r>>8, g>>8, b>>8)
		}
	}
}

func TestPreview_RejectsUnknownOrMalformedMapAssets(t *testing.T) {
	origin := newMapOrigin(t)
	for _, path := range []string{
		"/fixture-map/nope", "/fixture-map/tiles/x/1/1.mvt", "/fixture-map/tiles/14/1.mvt",
		"/fixture-map/terrain/1/1/bad.png", "/fixture-map/glyphs/OnlyOnePart", "/fixture-map/tiles/14/-1/2.mvt",
	} {
		if res, _ := get(t, origin+path); res.StatusCode != http.StatusNotFound {
			t.Errorf("%s status = %d", path, res.StatusCode)
		}
	}
}

// As a developer, I want the preview to hand the 3D basemap to the page, and to force the Leaflet
// fallback on request, so every map check can run on both engines.
func TestWriterMapPreview_SetsTheMapStyleAndTerrain(t *testing.T) {
	base, client := startPreviewProcess(t, "writers-map")
	body := pageBody(t, client, base+"/trip/madeira-pt")
	for _, want := range []string{
		`data-map-style-url="http://127.0.0.1:8088/fixture-map/style.json"`,
		`data-map-terrain-url="http://127.0.0.1:8088/fixture-map/terrain/%7bz%7d/%7bx%7d/%7by%7d.png"`,
		`data-map-tile-url=`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %s", want)
		}
	}
	if strings.Contains(body, "data-map-engine") {
		t.Error("the default scenario must not force an engine")
	}
}

func TestWriterMapPreview_LeafletScenarioForcesTheFallback(t *testing.T) {
	if !validScenario("writers-map-leaflet") {
		t.Fatal("scenario rejected")
	}
	base, client := startPreviewProcess(t, "writers-map-leaflet")
	body := pageBody(t, client, base+"/trip/madeira-pt")
	for _, want := range []string{`data-map-engine="leaflet"`, `data-map-tile-url=`} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %s", want)
		}
	}
}

func TestWriterMapPreview_LiveTilesLeavesTheProductionBasemap(t *testing.T) {
	base, client := startPreviewProcess(t, "writers-map", true)
	body := pageBody(t, client, base+"/trip/madeira-pt")
	for _, attr := range []string{"data-map-style-url", "data-map-terrain-url", "data-map-engine"} {
		if strings.Contains(body, attr) {
			t.Errorf("live-tiles page sets %s", attr)
		}
	}
}

func newMapOrigin(t *testing.T) string {
	t.Helper()
	server := httptest.NewServer(fixtureOriginHandler())
	t.Cleanup(server.Close)
	return server.URL
}

func get(t *testing.T, url string) (*http.Response, []byte) {
	t.Helper()
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res, body
}

func pageBody(t *testing.T, client *http.Client, url string) string {
	t.Helper()
	res, err := client.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK || !strings.Contains(string(body), "data-writers-map") {
		t.Fatalf("status = %d, body has no map shell", res.StatusCode)
	}
	return string(body)
}

func assertContentType(t *testing.T, res *http.Response, want string) {
	t.Helper()
	if res.StatusCode != http.StatusOK || !strings.HasPrefix(res.Header.Get("Content-Type"), want) {
		t.Fatalf("status = %d, content type = %q, want %q", res.StatusCode, res.Header.Get("Content-Type"), want)
	}
}

// tileXY is the standard slippy-map tile containing a coordinate.
func tileXY(lat, lon float64, z int) (int, int) {
	n := math.Exp2(float64(z))
	x := int((lon + 180) / 360 * n)
	rad := lat * math.Pi / 180
	y := int((1 - math.Log(math.Tan(rad)+1/math.Cos(rad))/math.Pi) / 2 * n)
	return x, y
}

// mvtLayers returns the feature count per layer of a vector tile, with a minimal protobuf reader.
func mvtLayers(t *testing.T, tile []byte) map[string]int {
	t.Helper()
	layers := map[string]int{}
	for len(tile) > 0 {
		tag, n := binary.Uvarint(tile)
		if n <= 0 {
			t.Fatal("bad tag")
		}
		tile = tile[n:]
		if tag&7 != 2 {
			t.Fatalf("unexpected wire type in tile: %d", tag&7)
		}
		size, n := binary.Uvarint(tile)
		if n <= 0 || int(size) > len(tile)-n {
			t.Fatal("bad length")
		}
		msg := tile[n : n+int(size)]
		tile = tile[n+int(size):]
		if tag>>3 != 3 {
			continue
		}
		name, features := "", 0
		for len(msg) > 0 {
			ftag, n := binary.Uvarint(msg)
			msg = msg[n:]
			switch ftag & 7 {
			case 0:
				_, n = binary.Uvarint(msg)
				msg = msg[n:]
			case 2:
				l, n := binary.Uvarint(msg)
				body := msg[n : n+int(l)]
				msg = msg[n+int(l):]
				switch ftag >> 3 {
				case 1:
					name = string(body)
				case 2:
					features++
				}
			default:
				t.Fatalf("unexpected wire type in layer: %d", ftag&7)
			}
		}
		layers[name] = features
	}
	return layers
}

// As the owner, I want the map always dark, so the preview fixture style has a near-black land
// background and dark water, roads and buildings like the live OpenFreeMap dark style.
func TestPreview_MapStyleIsDark(t *testing.T) {
	var style struct {
		Layers []struct {
			ID    string         `json:"id"`
			Paint map[string]any `json:"paint"`
		} `json:"layers"`
	}
	if err := json.Unmarshal(fixtureStyle("http://example.test"), &style); err != nil {
		t.Fatal(err)
	}
	hexLuma := func(hex string) float64 {
		var r, g, b int
		if _, err := fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b); err != nil {
			t.Fatalf("not a hex colour: %q", hex)
		}
		return (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)) / 255
	}
	limits := map[string]struct {
		key string
		max float64
	}{
		"land": {"background-color", 0.2}, "water": {"fill-color", 0.12},
		"road-secondary": {"line-color", 0.4}, "road-primary": {"line-color", 0.5},
		"building-3d": {"fill-extrusion-color", 0.35},
	}
	for _, layer := range style.Layers {
		limit, ok := limits[layer.ID]
		if !ok {
			continue
		}
		colour, _ := layer.Paint[limit.key].(string)
		if luma := hexLuma(colour); luma > limit.max {
			t.Errorf("%s colour %s has luminance %.2f, want at most %.2f", layer.ID, colour, luma, limit.max)
		}
		delete(limits, layer.ID)
	}
	if len(limits) != 0 {
		t.Errorf("style lacks layers %v", limits)
	}
}

// As the owner, I want the preview map to share the Electroverse-like palette the live default
// style gets, so that what the browser checks see matches production.
func TestPreview_MapStyleUsesElectroversePalette(t *testing.T) {
	var style struct {
		Layers []struct {
			ID    string         `json:"id"`
			Paint map[string]any `json:"paint"`
		} `json:"layers"`
	}
	if err := json.Unmarshal(fixtureStyle("http://example.test"), &style); err != nil {
		t.Fatal(err)
	}
	want := map[string][2]string{
		"land":           {"background-color", "#18263b"},
		"water":          {"fill-color", "#000e29"},
		"road-secondary": {"line-color", "#404860"},
		"road-primary":   {"line-color", "#4c576c"},
		"building-3d":    {"fill-extrusion-color", "#2c3044"},
	}
	for _, layer := range style.Layers {
		if expected, ok := want[layer.ID]; ok {
			if got := layer.Paint[expected[0]]; got != expected[1] {
				t.Errorf("%s %s = %v, want %s", layer.ID, expected[0], got, expected[1])
			}
			delete(want, layer.ID)
		}
	}
	if len(want) != 0 {
		t.Errorf("style lacks layers %v", want)
	}
}

// As the owner, I want the preview's minor road to be as thin and dim at z10 to z13 as the live
// style's, so that checks see the same road weight. The fixture has no glyphs, so it has no label layers
// to mirror the place label changes.
func TestPreview_MapStyleThinsMinorRoads(t *testing.T) {
	var style struct {
		Layers []struct {
			ID    string         `json:"id"`
			Paint map[string]any `json:"paint"`
		} `json:"layers"`
	}
	if err := json.Unmarshal(fixtureStyle("http://example.test"), &style); err != nil {
		t.Fatal(err)
	}
	wantWidth := `["interpolate",["exponential",1.55],["zoom"],10,0.4,13,1.2,20,20]`
	wantOpacity := `["interpolate",["linear"],["zoom"],10,0.35,14,0.9]`
	for _, layer := range style.Layers {
		if layer.ID != "road-secondary" {
			continue
		}
		width, _ := json.Marshal(layer.Paint["line-width"])
		opacity, _ := json.Marshal(layer.Paint["line-opacity"])
		if string(width) != wantWidth || string(opacity) != wantOpacity {
			t.Errorf("road-secondary width %s opacity %s, want %s and %s", width, opacity, wantWidth, wantOpacity)
		}
		return
	}
	t.Error("style lacks road-secondary")
}
