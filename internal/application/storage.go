package application

import "time"

// Storage is the persistence needed by the dashboard and visit reporting. Adapters own
// database connections and serialization; application callers do not depend on SQLite.
type Storage interface {
	SaveCityData(city string, data map[string]any) error
	GetCityData(city string) (string, error)
	SaveVisit(visitor, path, city string) error
	GetVisitStats(since time.Time) (*VisitStats, error)
}
