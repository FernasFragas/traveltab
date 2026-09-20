package sqlite

import (
	"fmt"
	"strings"
	"time"
	"weatherservice/internal/application"
)

func (s *Store) createVisitsTable() error {
	createTableSQL := `CREATE TABLE IF NOT EXISTS visits (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        visited_at TIMESTAMP NOT NULL,
        visitor TEXT NOT NULL,
        path TEXT NOT NULL,
        city TEXT
    );
    CREATE INDEX IF NOT EXISTS idx_visits_visited_at ON visits (visited_at);`

	if _, err := s.db.Exec(createTableSQL); err != nil {
		return fmt.Errorf("error creating visits table: %w", err)
	}

	return nil
}

// SaveVisit stores a single page view.
func (s *Store) SaveVisit(visitor, path, city string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("database is not initialized")
	}
	insertSQL := `INSERT INTO visits (visited_at, visitor, path, city) VALUES (?, ?, ?, ?);`

	_, err := s.db.Exec(insertSQL, time.Now().UTC(), visitor, path, strings.ToLower(strings.TrimSpace(city)))
	if err != nil {
		return fmt.Errorf("error saving visit: %w", err)
	}

	return nil
}

// GetVisitStats aggregates the visits recorded since the given time.
func (s *Store) GetVisitStats(since time.Time) (*application.VisitStats, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("database is not initialized")
	}

	// Visits are stored as UTC text, so the comparison only works against UTC.
	since = since.UTC()

	stats := &application.VisitStats{Since: since, Daily: []application.DailyStats{}, TopCities: []application.CityStats{}}

	totalsSQL := `SELECT COUNT(DISTINCT visitor), COUNT(*), COUNT(NULLIF(city, ''))
        FROM visits WHERE visited_at >= ?;`
	if err := s.db.QueryRow(totalsSQL, since).Scan(&stats.UniqueVisitors, &stats.PageViews, &stats.Searches); err != nil {
		return nil, fmt.Errorf("error querying visit totals: %w", err)
	}

	dailySQL := `SELECT substr(visited_at, 1, 10) AS day, COUNT(DISTINCT visitor), COUNT(*)
        FROM visits WHERE visited_at >= ? GROUP BY day ORDER BY day;`
	rows, err := s.db.Query(dailySQL, since)
	if err != nil {
		return nil, fmt.Errorf("error querying daily visits: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var day application.DailyStats
		if err := rows.Scan(&day.Day, &day.UniqueVisitors, &day.PageViews); err != nil {
			return nil, fmt.Errorf("error reading daily visits: %w", err)
		}
		stats.Daily = append(stats.Daily, day)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading daily visits: %w", err)
	}

	citiesSQL := `SELECT city, COUNT(*) AS searches FROM visits
        WHERE visited_at >= ? AND city != '' GROUP BY city ORDER BY searches DESC LIMIT 10;`
	cityRows, err := s.db.Query(citiesSQL, since)
	if err != nil {
		return nil, fmt.Errorf("error querying top cities: %w", err)
	}
	defer func() { _ = cityRows.Close() }()

	for cityRows.Next() {
		var city application.CityStats
		if err := cityRows.Scan(&city.City, &city.Searches); err != nil {
			return nil, fmt.Errorf("error reading top cities: %w", err)
		}
		stats.TopCities = append(stats.TopCities, city)
	}
	if err := cityRows.Err(); err != nil {
		return nil, fmt.Errorf("error reading top cities: %w", err)
	}

	return stats, nil
}
