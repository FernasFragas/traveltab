package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// The 3D map data fixtures. Style assets and tiles are generated here, in Go, so the
// preview needs no external map hosts or binary fixture files in the repository:
//
//	/fixture-map/style.json                      MapLibre style (sources, glyphs and sprite on this origin)
//	/fixture-map/tiles/{z}/{x}/{y}.mvt           vector tiles: water, roads and extruded buildings around Funchal
//	/fixture-map/terrain/{z}/{x}/{y}.png         flat raster-dem tiles, terrarium encoding, sea level everywhere
//	/fixture-map/glyphs/{fontstack}/{range}.pbf  valid glyph files with no glyphs (the style has no text layers)
//	/fixture-map/sprite[@2x].json and .png       an empty sprite
//
// Tiles are computed from coordinates, so any z/x/y works and tiles away from Funchal are empty.
const (
	mapFixturePath = "/fixture-map/"
	mvtExtent      = 4096
	terrainSize    = 512
	glyphStack     = "Fixture Sans"
)

var (
	glyphRange  = regexp.MustCompile(`^\d+-\d+$`)
	terrainOnce sync.Once
	terrainPNG  []byte
	spriteOnce  sync.Once
	spritePNG   []byte
)

// fixtureMapAssetHandler routes the /fixture-map/ files. Responses are readable cross-origin,
// because MapLibre fetches them from the page on the preview port.
func fixtureMapAssetHandler(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, mapFixturePath)
	parts := strings.Split(rest, "/")
	switch {
	case rest == "style.json":
		writeMapAsset(w, "application/json", fixtureStyle("http://"+r.Host))
	case strings.HasPrefix(rest, "sprite") && strings.HasSuffix(rest, ".json") && len(parts) == 1:
		writeMapAsset(w, "application/json", []byte("{}"))
	case strings.HasPrefix(rest, "sprite") && strings.HasSuffix(rest, ".png") && len(parts) == 1:
		writeMapAsset(w, "image/png", fixtureSprite())
	case parts[0] == "glyphs" && len(parts) == 3 && strings.HasSuffix(parts[2], ".pbf") && glyphRange.MatchString(strings.TrimSuffix(parts[2], ".pbf")):
		writeMapAsset(w, "application/x-protobuf", fixtureGlyphs(parts[1], strings.TrimSuffix(parts[2], ".pbf")))
	case parts[0] == "tiles" && len(parts) == 4 && strings.HasSuffix(parts[3], ".mvt"):
		z, x, y, ok := tileCoordinates(parts[1], parts[2], strings.TrimSuffix(parts[3], ".mvt"))
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeMapAsset(w, "application/vnd.mapbox-vector-tile", fixtureVectorTile(z, x, y))
	case parts[0] == "terrain" && len(parts) == 4 && strings.HasSuffix(parts[3], ".png"):
		if _, _, _, ok := tileCoordinates(parts[1], parts[2], strings.TrimSuffix(parts[3], ".png")); !ok {
			http.NotFound(w, r)
			return
		}
		writeMapAsset(w, "image/png", fixtureTerrain())
	default:
		http.NotFound(w, r)
	}
}

func writeMapAsset(w http.ResponseWriter, contentType string, body []byte) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_, _ = w.Write(body)
}

// fixtureStyle is a small MapLibre style: land (the background), sea, two kinds of road and
// extruded buildings, from one vector source on origin.
func fixtureStyle(origin string) []byte {
	base := origin + mapFixturePath
	style := map[string]any{
		"version": 8,
		"name":    "TravelTab fixture style",
		"glyphs":  base + "glyphs/{fontstack}/{range}.pbf",
		"sprite":  base + "sprite",
		"sources": map[string]any{
			"fixture": map[string]any{
				"type":        "vector",
				"tiles":       []string{base + "tiles/{z}/{x}/{y}.mvt"},
				"minzoom":     0,
				"maxzoom":     14,
				"attribution": "Local fixture map data",
			},
		},
		"layers": []any{
			map[string]any{"id": "land", "type": "background", "paint": map[string]any{"background-color": "#202a32"}},
			map[string]any{"id": "water", "type": "fill", "source": "fixture", "source-layer": "water",
				"paint": map[string]any{"fill-color": "#000e29"}},
			map[string]any{"id": "road-secondary", "type": "line", "source": "fixture", "source-layer": "transportation",
				"filter": []any{"==", []any{"get", "class"}, "secondary"},
				"paint": map[string]any{"line-color": "#404860",
					"line-width":   []any{"interpolate", []any{"exponential", 1.55}, []any{"zoom"}, 10, 0.4, 13, 1.2, 20, 20},
					"line-opacity": []any{"interpolate", []any{"linear"}, []any{"zoom"}, 10, 0.35, 14, 0.9}}},
			map[string]any{"id": "road-primary", "type": "line", "source": "fixture", "source-layer": "transportation",
				"filter": []any{"==", []any{"get", "class"}, "primary"},
				"paint":  map[string]any{"line-color": "#4c576c", "line-width": 4}},
			map[string]any{"id": "building-3d", "type": "fill-extrusion", "source": "fixture", "source-layer": "building",
				"paint": map[string]any{
					"fill-extrusion-color":   "#2c3044",
					"fill-extrusion-height":  []any{"get", "render_height"},
					"fill-extrusion-base":    []any{"get", "render_min_height"},
					"fill-extrusion-opacity": 0.9,
				}},
			map[string]any{"id": "poi", "type": "symbol", "source": "fixture", "source-layer": "poi",
				"layout": map[string]any{"visibility": "none", "text-field": []any{"get", "name"}}},
		},
	}
	out, err := json.Marshal(style)
	if err != nil {
		panic(err)
	}
	return out
}

// mapLayer is one vector-tile layer being built; tags are interned per layer as the format needs.
type mapLayer struct {
	name     string
	features [][]byte
	keys     []string
	values   [][]byte
}

// mapShape is a fixture feature in longitude/latitude.
type mapShape struct {
	layer string
	kind  int // 2 line, 3 polygon
	ring  [][2]float64
	tags  map[string]any
}

// fixtureShapes is the Funchal scene: sea to the south, two roads, five buildings.
func fixtureShapes() []mapShape {
	building := func(lon, lat float64, height uint64) mapShape {
		const d = 0.0005
		return mapShape{"building", 3, [][2]float64{{lon, lat}, {lon + d, lat}, {lon + d, lat + d}, {lon, lat + d}},
			map[string]any{"render_height": height, "render_min_height": uint64(0)}}
	}
	return []mapShape{
		{"water", 3, [][2]float64{{-17.3, 32.40}, {-16.5, 32.40}, {-16.5, 32.642}, {-17.3, 32.642}}, nil},
		{"transportation", 2, [][2]float64{{-16.93, 32.646}, {-16.91, 32.650}, {-16.89, 32.6505}}, map[string]any{"class": "primary"}},
		{"transportation", 2, [][2]float64{{-16.91, 32.644}, {-16.91, 32.656}}, map[string]any{"class": "secondary"}},
		building(-16.9125, 32.6485, 18),
		building(-16.9105, 32.6490, 45),
		building(-16.9090, 32.6500, 27),
		building(-16.9120, 32.6510, 12),
		building(-16.9100, 32.6515, 33),
	}
}

// fixtureVectorTile returns the Mapbox Vector Tile for z/x/y: every fixture shape that overlaps
// the tile, with coordinates in tile units. A tile with no shapes is a valid, empty tile.
func fixtureVectorTile(z, x, y int) []byte {
	layers := map[string]*mapLayer{}
	var order []string
	for _, shape := range fixtureShapes() {
		points, visible := projectToTile(shape.ring, z, x, y)
		if !visible {
			continue
		}
		layer := layers[shape.layer]
		if layer == nil {
			layer = &mapLayer{name: shape.layer}
			layers[shape.layer] = layer
			order = append(order, shape.layer)
		}
		layer.features = append(layer.features, encodeFeature(layer, shape, points))
	}
	var tile []byte
	for _, name := range order {
		tile = appendBytes(tile, 3, encodeLayer(layers[name]))
	}
	return tile
}

// projectToTile converts longitude/latitude to tile units (y down) and reports whether the
// shape comes within a 256-unit buffer of the tile.
func projectToTile(ring [][2]float64, z, x, y int) ([][2]int, bool) {
	scale := math.Exp2(float64(z))
	points := make([][2]int, len(ring))
	minX, minY, maxX, maxY := math.MaxInt32, math.MaxInt32, math.MinInt32, math.MinInt32
	for i, p := range ring {
		sin := math.Sin(p[1] * math.Pi / 180)
		worldX := (p[0] + 180) / 360 * scale
		worldY := (0.5 - math.Log((1+sin)/(1-sin))/(4*math.Pi)) * scale
		px := int(math.Round((worldX - float64(x)) * mvtExtent))
		py := int(math.Round((worldY - float64(y)) * mvtExtent))
		points[i] = [2]int{px, py}
		minX, maxX = min(minX, px), max(maxX, px)
		minY, maxY = min(minY, py), max(maxY, py)
	}
	const buffer = 256
	visible := maxX >= -buffer && minX <= mvtExtent+buffer && maxY >= -buffer && minY <= mvtExtent+buffer
	return points, visible
}

func encodeFeature(layer *mapLayer, shape mapShape, points [][2]int) []byte {
	var feature []byte
	var tags []byte
	for _, key := range sortedKeys(shape.tags) {
		tags = binary.AppendUvarint(tags, uint64(layer.internKey(key)))
		tags = binary.AppendUvarint(tags, uint64(layer.internValue(shape.tags[key])))
	}
	if len(tags) > 0 {
		feature = appendBytes(feature, 2, tags)
	}
	feature = appendVarint(feature, 3, uint64(shape.kind))
	if shape.kind == 3 && signedArea(points) < 0 {
		for i, j := 0, len(points)-1; i < j; i, j = i+1, j-1 {
			points[i], points[j] = points[j], points[i]
		}
	}
	return appendBytes(feature, 4, encodeGeometry(shape.kind, points))
}

func encodeLayer(layer *mapLayer) []byte {
	out := appendVarint(nil, 15, 2)
	out = appendBytes(out, 1, []byte(layer.name))
	for _, feature := range layer.features {
		out = appendBytes(out, 2, feature)
	}
	for _, key := range layer.keys {
		out = appendBytes(out, 3, []byte(key))
	}
	for _, value := range layer.values {
		out = appendBytes(out, 4, value)
	}
	return appendVarint(out, 5, mvtExtent)
}

// encodeGeometry writes MoveTo, LineTo and (for polygons) ClosePath commands with zigzag deltas.
func encodeGeometry(kind int, points [][2]int) []byte {
	command := func(id, count uint64) uint64 { return id | count<<3 }
	zigzag := func(n int) uint64 { return uint64(uint32((n << 1) ^ (n >> 31))) }
	out := binary.AppendUvarint(nil, command(1, 1))
	out = binary.AppendUvarint(out, zigzag(points[0][0]))
	out = binary.AppendUvarint(out, zigzag(points[0][1]))
	out = binary.AppendUvarint(out, command(2, uint64(len(points)-1)))
	for i := 1; i < len(points); i++ {
		out = binary.AppendUvarint(out, zigzag(points[i][0]-points[i-1][0]))
		out = binary.AppendUvarint(out, zigzag(points[i][1]-points[i-1][1]))
	}
	if kind == 3 {
		out = binary.AppendUvarint(out, command(7, 1))
	}
	return out
}

// signedArea is positive for the clockwise-on-screen exterior rings that MVT requires (y is down).
func signedArea(points [][2]int) int {
	area := 0
	for i := range points {
		next := points[(i+1)%len(points)]
		area += points[i][0]*next[1] - next[0]*points[i][1]
	}
	return area
}

func (l *mapLayer) internKey(key string) int {
	for i, existing := range l.keys {
		if existing == key {
			return i
		}
	}
	l.keys = append(l.keys, key)
	return len(l.keys) - 1
}

func (l *mapLayer) internValue(value any) int {
	var encoded []byte
	switch v := value.(type) {
	case string:
		encoded = appendBytes(nil, 1, []byte(v))
	case uint64:
		encoded = appendVarint(nil, 5, v)
	}
	for i, existing := range l.values {
		if bytes.Equal(existing, encoded) {
			return i
		}
	}
	l.values = append(l.values, encoded)
	return len(l.values) - 1
}

// fixtureGlyphs is a valid glyph file (one font stack, no glyphs).
func fixtureGlyphs(stack, glyphRange string) []byte {
	inner := appendBytes(nil, 1, []byte(stack))
	inner = appendBytes(inner, 2, []byte(glyphRange))
	return appendBytes(nil, 1, inner)
}

// fixtureSprite is a 1x1 transparent PNG, enough for the empty sprite sheet.
func fixtureSprite() []byte {
	spriteOnce.Do(func() {
		spritePNG = encodePNG(image.NewRGBA(image.Rect(0, 0, 1, 1)))
	})
	return spritePNG
}

// fixtureTerrain is a flat terrarium tile: metres = R*256 + G + B/256 - 32768, so R=128, G=0, B=0
// is sea level.
func fixtureTerrain() []byte {
	terrainOnce.Do(func() {
		img := image.NewRGBA(image.Rect(0, 0, terrainSize, terrainSize))
		for y := 0; y < terrainSize; y++ {
			for x := 0; x < terrainSize; x++ {
				img.SetRGBA(x, y, color.RGBA{R: 128, A: 255})
			}
		}
		terrainPNG = encodePNG(img)
	})
	return terrainPNG
}

// tileCoordinates parses z/x/y and rejects values outside the tile grid.
func tileCoordinates(zs, xs, ys string) (z, x, y int, ok bool) {
	z, errZ := strconv.Atoi(zs)
	x, errX := strconv.Atoi(xs)
	y, errY := strconv.Atoi(ys)
	if errZ != nil || errX != nil || errY != nil || z < 0 || z > 22 || x < 0 || y < 0 || x >= 1<<z || y >= 1<<z {
		return 0, 0, 0, false
	}
	return z, x, y, true
}

func encodePNG(img image.Image) []byte {
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		panic(err)
	}
	return out.Bytes()
}

func appendVarint(dst []byte, field int, value uint64) []byte {
	dst = binary.AppendUvarint(dst, uint64(field)<<3)
	return binary.AppendUvarint(dst, value)
}

func appendBytes(dst []byte, field int, value []byte) []byte {
	dst = binary.AppendUvarint(dst, uint64(field)<<3|2)
	dst = binary.AppendUvarint(dst, uint64(len(value)))
	return append(dst, value...)
}

func sortedKeys(tags map[string]any) []string {
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
