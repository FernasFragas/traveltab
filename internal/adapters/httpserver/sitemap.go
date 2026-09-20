package httpserver

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"sort"

	"github.com/gofiber/fiber/v2"
)

// sitemapIndexKey is the pseudo "city" the sitemap's own index is stored under, reusing the
// existing city_data cache rather than a new table: application.Storage only exposes
// SaveCityData/GetCityData, and this stays inside that contract instead of widening it.
const sitemapIndexKey = "__sitemap_index__"

// recordSitemapSlug adds slug to the persisted list /sitemap.xml reads, so a city that has
// been searched once is still listed after a restart, and never triggers a fresh fetch just to
// be found again. It is called from the same place a city's own data gets cached, so it can
// itself race a concurrent writer; losing that race only delays a slug showing up in the
// sitemap by one more visit, never drops one that's already listed.
func (s *Server) recordSitemapSlug(slug string) {
	if s.storage == nil || slug == "" {
		return
	}

	slugs, _ := s.loadSitemapSlugs() // a missing or unreadable index just starts empty
	for _, existing := range slugs {
		if existing == slug {
			return
		}
	}

	slugs = append(slugs, slug)
	if err := s.storage.SaveCityData(sitemapIndexKey, map[string]any{"slugs": slugs}); err != nil {
		log.Printf("Error saving the sitemap index for slug %s: %v", slug, err)
	}
}

// loadSitemapSlugs reads the persisted list, or an empty list when there isn't one yet.
func (s *Server) loadSitemapSlugs() ([]string, error) {
	if s.storage == nil {
		return nil, fmt.Errorf("storage is not initialized")
	}

	raw, err := s.storage.GetCityData(sitemapIndexKey)
	if err != nil {
		return nil, err
	}

	var wrapper struct {
		Slugs []string `json:"slugs"`
	}
	if err := json.Unmarshal([]byte(raw), &wrapper); err != nil {
		return nil, err
	}
	return wrapper.Slugs, nil
}

type sitemapURL struct {
	Loc string `xml:"loc"`
}

type sitemapDocument struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

// sitemap answers GET /sitemap.xml with a /trip/:slug entry for every city that already has a
// row in the cache, read from the persisted index - this never triggers a fresh fetch, so a
// crawl of this endpoint costs one cheap cache read.
func (s *Server) sitemap(ctx *fiber.Ctx) error {
	slugs, _ := s.loadSitemapSlugs() // an empty or missing index is an empty sitemap, not an error
	sort.Strings(slugs)

	doc := sitemapDocument{XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	for _, slug := range slugs {
		doc.URLs = append(doc.URLs, sitemapURL{Loc: "/trip/" + slug})
	}

	body, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		log.Printf("Error building the sitemap: %v", err)
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}

	ctx.Set(fiber.HeaderContentType, fiber.MIMEApplicationXML)
	return ctx.Send(append([]byte(xml.Header), body...))
}
