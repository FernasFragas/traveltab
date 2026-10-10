package httpserver

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
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
}

// SetWriterMapSource enables the fixture-backed map only when the feature flag is on.
func (s *Server) SetWriterMapSource(source application.WriterMapSource, enabled bool) {
	s.writerMapSource = source
	s.writersMapEnabled = enabled && source != nil
}

// SetWriterMapTiles supplies a fixture or chosen basemap for flagged map pages.
func (s *Server) SetWriterMapTiles(tileURL, attribution string) {
	s.writerMapTileURL = tileURL
	s.writerMapTileAttr = attribution
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
	}
}

func (s *Server) writerMapGeoJSON(ctx *fiber.Ctx) error {
	area, query, ok := s.mapRequest(ctx)
	if !ok {
		return nil
	}
	result, err := s.writerMapSource.Destination(ctx.Context(), area)
	if err != nil || result == nil {
		return ctx.SendStatus(fiber.StatusServiceUnavailable)
	}
	features := make([]mapFeature, 0, len(result.Places))
	for _, place := range result.Places {
		if !validCoordinates(place.Lat, place.Lon) || place.QID == "" {
			continue
		}
		name := place.QID
		if len(place.Names) > 0 {
			name = place.Names[0]
		}
		names := place.Names
		if names == nil {
			names = []string{}
		}
		features = append(features, mapFeature{
			Type:     "Feature",
			Geometry: mapGeometry{Type: "Point", Coordinates: [2]float64{place.Lon, place.Lat}},
			Properties: mapProperties{
				QID: place.QID, Name: name, Names: names, Kind: place.Kind,
				Description: place.Description, Photo: place.Photo, PhotoCredit: place.PhotoCredit,
				PhotoURL: place.PhotoURL, Sitelinks: place.Sitelinks,
				WriterCount: result.WriterCounts[place.QID],
				PanelURL:    "/map/" + url.PathEscape(ctx.Params("dest")) + "/" + url.PathEscape(place.QID) + query,
			},
		})
	}
	return ctx.JSON(mapCollection{Type: "FeatureCollection", Complete: result.Complete, Features: features}, "application/geo+json; charset=utf-8")
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
	result, err := s.writerMapSource.Destination(ctx.Context(), area)
	if err != nil || result == nil {
		return ctx.SendStatus(fiber.StatusServiceUnavailable)
	}
	name := place.QID
	if len(place.Names) > 0 {
		name = place.Names[0]
	}
	return ctx.Render("place_panel", fiber.Map{
		"Place": place, "Name": name, "Summary": summary,
		"WriterCount": result.WriterCounts[qid], "Mentions": result.Mentions[qid],
	})
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

type mapCollection struct {
	Type     string       `json:"type"`
	Complete bool         `json:"complete"`
	Features []mapFeature `json:"features"`
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
	Photo       string   `json:"photo"`
	PhotoCredit string   `json:"photo_credit"`
	PhotoURL    string   `json:"photo_url"`
	Sitelinks   int      `json:"sitelinks"`
	WriterCount int      `json:"writer_count"`
	PanelURL    string   `json:"panel_url"`
}
