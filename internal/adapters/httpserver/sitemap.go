package httpserver

import (
	"encoding/xml"
	"log"

	"github.com/gofiber/fiber/v2"
)

// recordSitemapSlug persists slug so /sitemap.xml lists it after a restart without another
// fetch. Storage inserts each slug as its own row, so concurrent visits cannot lose entries.
func (s *Server) recordSitemapSlug(slug string) {
	if s.storage == nil || slug == "" {
		return
	}
	if err := s.storage.RecordSitemapSlug(slug); err != nil {
		log.Printf("Error recording sitemap slug %s: %v", slug, err)
	}
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
// row in the cache, read from the persisted slug list - this never triggers a fresh fetch, so a
// crawl of this endpoint costs one cheap cache read.
func (s *Server) sitemap(ctx *fiber.Ctx) error {
	var slugs []string
	if s.storage != nil {
		var err error
		if slugs, err = s.storage.ListSitemapSlugs(); err != nil {
			log.Printf("Error listing sitemap slugs: %v", err) // serve what we have: an empty sitemap
		}
	}

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
