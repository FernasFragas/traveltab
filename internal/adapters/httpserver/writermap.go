package httpserver

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"weatherservice/internal/application"
	"weatherservice/internal/writermap"

	"github.com/gofiber/fiber/v2"
)

type writerMapCardView struct {
	Destination string
	Lat, Lon    float64
	URL         string
	TileURL     string
	Attribution string
	StyleURL    string
	TerrainURL  string
	TerrainOff  bool
	Engine      string
}

// SetWriterMapSource enables the map only when the feature flag is on.
func (s *Server) SetWriterMapSource(source application.WriterMapSource, enabled bool) {
	s.writerMapSource = source
	s.writersMapEnabled = enabled && source != nil
}

// SetWriterMapTiles supplies a fixture or chosen basemap for flagged map pages.
func (s *Server) SetWriterMapTiles(tileURL, attribution string) {
	s.writerMapTileURL = tileURL
	s.writerMapTileAttr = attribution
}

// SetWriterMapBasemap sets the MapLibre style URL, the terrain tile URL template and the map
// engine. An empty value is left out of the page, so the browser default applies. Production sets
// none of them; the design preview points them at local fixtures.
func (s *Server) SetWriterMapBasemap(styleURL, terrainURL, engine string) {
	s.writerMapStyleURL = styleURL
	s.writerMapTerrainURL = terrainURL
	s.writerMapEngine = engine
}

// SetWriterMapTerrainOff renders an empty terrain URL, which turns terrain off in the browser.
func (s *Server) SetWriterMapTerrainOff() {
	s.writerMapTerrainOff = true
}

func (s *Server) writerMapCard(info application.GeneralWeatherInfo) *writerMapCardView {
	if !s.writersMapEnabled || !validCoordinates(info.Lat, info.Lon) {
		return nil
	}
	slug := Slug(info.City, info.Country)
	if _, _, ok := ParseSlug(slug); !ok {
		return nil
	}
	return &writerMapCardView{
		Destination: slug,
		Lat:         info.Lat,
		Lon:         info.Lon,
		URL:         "/map/" + url.PathEscape(slug) + ".geojson" + mapCoordinates(info.Lat, info.Lon),
		TileURL:     s.writerMapTileURL,
		Attribution: s.writerMapTileAttr,
		StyleURL:    s.writerMapStyleURL,
		TerrainURL:  s.writerMapTerrainURL,
		TerrainOff:  s.writerMapTerrainOff,
		Engine:      s.writerMapEngine,
	}
}

func (s *Server) writerMapGeoJSON(ctx *fiber.Ctx) error {
	area, query, ok := s.mapRequest(ctx)
	if !ok {
		return nil
	}
	result, loading, err := s.destinationSoFar(ctx, area)
	if err != nil || result == nil {
		return ctx.SendStatus(fiber.StatusServiceUnavailable)
	}
	features := make([]mapFeature, 0, len(result.Places))
	for _, place := range result.Places {
		if !validCoordinates(place.Lat, place.Lon) || place.QID == "" {
			continue
		}
		name := "" // a pin still waiting for its name; the browser shows it unlabelled
		if len(place.Names) > 0 {
			name = place.Names[0]
		}
		names := place.Names
		if names == nil {
			names = []string{}
		}
		properties := mapProperties{
			QID: place.QID, Name: name, Names: names, Kind: place.Kind,
			Description: place.Description, Sitelinks: place.Sitelinks,
			WriterCount: result.WriterCounts[place.QID],
			PanelURL:    "/map/" + url.PathEscape(ctx.Params("dest")) + "/" + url.PathEscape(place.QID) + query,
		}
		if validPlacePhotoURL(place.PhotoURL) {
			properties.Photo = place.Photo
			properties.PhotoURL = place.PhotoURL
			properties.PhotoCredit = place.PhotoCredit
		}
		features = append(features, mapFeature{
			Type:       "Feature",
			Geometry:   mapGeometry{Type: "Point", Coordinates: [2]float64{place.Lon, place.Lat}},
			Properties: properties,
		})
	}
	return ctx.JSON(mapCollection{
		Type: "FeatureCollection", Complete: result.Complete, Loading: loading, Features: features,
		Itineraries: mapItineraries(result.Itineraries),
	}, "application/geo+json; charset=utf-8")
}

func (s *Server) writerMapPlace(ctx *fiber.Ctx) error {
	area, _, ok := s.mapRequest(ctx)
	if !ok {
		return nil
	}
	qid := ctx.Params("qid")
	place, summary, err := s.writerMapSource.Place(ctx.Context(), area, qid)
	if errors.Is(err, writermap.ErrPlaceNotFound) {
		return ctx.SendStatus(fiber.StatusNotFound)
	}
	if err != nil || place == nil {
		return ctx.SendStatus(fiber.StatusServiceUnavailable)
	}
	result, _, err := s.destinationSoFar(ctx, area)
	if err != nil || result == nil {
		return ctx.SendStatus(fiber.StatusServiceUnavailable)
	}
	name := "Loading name…"
	if len(place.Names) > 0 {
		name = place.Names[0]
	}
	visiblePlace := *place
	if !validPlacePhotoURL(place.PhotoURL) {
		visiblePlace.Photo = ""
		visiblePlace.PhotoURL = ""
		visiblePlace.PhotoCredit = ""
	}
	return ctx.Render("place_panel", fiber.Map{
		"Place": &visiblePlace, "Name": name, "Summary": summary,
		"WriterCount": result.WriterCounts[qid], "Mentions": result.Mentions[qid],
	})
}

// destinationSoFar returns the destination, or while its places are still loading the pins found so
// far, so a cold big city answers at once instead of waiting for the last ring.
func (s *Server) destinationSoFar(ctx *fiber.Ctx, area writermap.Area) (*writermap.DestinationResult, bool, error) {
	if source, ok := s.writerMapSource.(application.ProgressiveWriterMapSource); ok {
		return source.DestinationProgress(ctx.Context(), area)
	}
	result, err := s.writerMapSource.Destination(ctx.Context(), area)
	return result, false, err
}

func (s *Server) mapRequest(ctx *fiber.Ctx) (writermap.Area, string, bool) {
	if !s.writersMapEnabled {
		_ = ctx.SendStatus(fiber.StatusNotFound)
		return writermap.Area{}, "", false
	}
	city, country, ok := ParseSlug(ctx.Params("dest"))
	if !ok {
		_ = ctx.SendStatus(fiber.StatusNotFound)
		return writermap.Area{}, "", false
	}
	lat, latErr := strconv.ParseFloat(ctx.Query("lat"), 64)
	lon, lonErr := strconv.ParseFloat(ctx.Query("lon"), 64)
	if latErr != nil || lonErr != nil || !validCoordinates(lat, lon) {
		_ = ctx.SendStatus(fiber.StatusBadRequest)
		return writermap.Area{}, "", false
	}
	return writermap.Area{Name: city, Country: country, Lat: lat, Lon: lon}, mapCoordinates(lat, lon), true
}

func mapCoordinates(lat, lon float64) string {
	return fmt.Sprintf("?lat=%s&lon=%s", strconv.FormatFloat(lat, 'f', -1, 64), strconv.FormatFloat(lon, 'f', -1, 64))
}

func validPlacePhotoURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "upload.wikimedia.org" || host == "thumb.wikimedia.org"
}

type mapCollection struct {
	Type        string         `json:"type"`
	Complete    bool           `json:"complete"`
	Loading     bool           `json:"loading"` // more places are still being loaded; poll again
	Features    []mapFeature   `json:"features"`
	Itineraries []mapItinerary `json:"itineraries"`
}

type mapFeature struct {
	Type       string        `json:"type"`
	Geometry   mapGeometry   `json:"geometry"`
	Properties mapProperties `json:"properties"`
}

type mapGeometry struct {
	Type        string     `json:"type"`
	Coordinates [2]float64 `json:"coordinates"`
}

type mapProperties struct {
	QID         string   `json:"qid"`
	Name        string   `json:"name"`
	Names       []string `json:"names"`
	Kind        string   `json:"kind"`
	Description string   `json:"description"`
	Photo       string   `json:"photo,omitempty"`
	PhotoCredit string   `json:"photo_credit,omitempty"`
	PhotoURL    string   `json:"photo_url,omitempty"`
	Sitelinks   int      `json:"sitelinks"`
	WriterCount int      `json:"writer_count"`
	PanelURL    string   `json:"panel_url"`
}

// mapItinerary is a writer route as the browser sees it; the post's internal reference stays on the server.
type mapItinerary struct {
	WriterHost string                   `json:"writer_host"`
	PostURL    string                   `json:"post_url"`
	PostTitle  string                   `json:"post_title"`
	Language   string                   `json:"language"`
	Days       []writermap.ItineraryDay `json:"days"`
}

func mapItineraries(itineraries []writermap.Itinerary) []mapItinerary {
	routes := make([]mapItinerary, 0, len(itineraries))
	for _, itinerary := range itineraries {
		routes = append(routes, mapItinerary{
			WriterHost: itinerary.WriterHost, PostURL: itinerary.PostURL, PostTitle: itinerary.PostTitle,
			Language: itinerary.Language, Days: itinerary.Days,
		})
	}
	return routes
}
