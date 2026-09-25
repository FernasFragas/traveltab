package application

import "time"

// Storage is the persistence needed by the dashboard and visit reporting. Adapters own
// database connections and serialization; application callers do not depend on SQLite.
type Storage interface {
	SaveCityData(city string, data map[string]any) error
	GetCityData(city string) (string, error)
	// RecordSitemapSlug persists a /trip/:slug slug for the sitemap; recording a slug twice is
	// a no-op. ListSitemapSlugs returns all recorded slugs sorted, never calling a provider.
	RecordSitemapSlug(slug string) error
	ListSitemapSlugs() ([]string, error)
	SaveVisit(visitor, path, city string) error
	GetVisitStats(since time.Time) (*VisitStats, error)
}
