package httpserver

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"

	"weatherservice/internal/planner"
)

// exportKML answers GET /trip/:slug.kml with the trip as a KML file for Google My Maps: one
// Folder per day, one Placemark per stop.
func (s *Server) exportKML(ctx *fiber.Ctx) error {
	plan, ok := s.resolveTripPlan(ctx)
	if !ok {
		return ctx.SendStatus(fiber.StatusNotFound)
	}

	ctx.Set(fiber.HeaderContentType, "application/vnd.google-earth.kml+xml; charset=utf-8")
	ctx.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s.kml"`, ctx.Params("slug")))

	return ctx.SendString(buildKML(plan))
}

// buildKML renders plan as a complete KML document.
func buildKML(plan *planner.Plan) string {
	var b strings.Builder

	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<kml xmlns="http://www.opengis.net/kml/2.2">` + "\n")
	b.WriteString("<Document>\n")

	for i, day := range plan.Days {
		fmt.Fprintf(&b, "<Folder>\n<name>Day %d</name>\n", i+1)
		for _, stop := range day.Stops {
			fmt.Fprintf(&b,
				"<Placemark>\n<name>%s</name>\n<Point>\n<coordinates>%s,%s</coordinates>\n</Point>\n</Placemark>\n",
				xmlEscape(stop.Name), formatCoord(stop.Lon), formatCoord(stop.Lat))
		}
		b.WriteString("</Folder>\n")
	}

	b.WriteString("</Document>\n</kml>\n")

	return b.String()
}

// xmlEscape escapes the five characters XML text content requires escaped.
func xmlEscape(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return replacer.Replace(s)
}

// formatCoord renders a coordinate the way KML expects: plain decimal, no exponent notation,
// trailing zeros trimmed.
func formatCoord(v float64) string {
	s := fmt.Sprintf("%.6f", v)
	s = strings.TrimRight(s, "0")
	s = strings.TrimSuffix(s, ".")
	return s
}
