package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"
	"weatherservice/internal/planner"
)

// How long each kind of answer stays fresh. Famous places and hotels barely move, forecasts do.
const (
	placesCacheTTL   = 30 * 24 * time.Hour
	staysCacheTTL    = 30 * 24 * time.Hour
	forecastCacheTTL = 3 * time.Hour
)

// cached wraps one planner source with the source_cache table. A fresh entry is served from the
// database, an expired one is refreshed, and when the refresh fails the old entry is served
// instead: a stale list of museums beats no trip plan. Only an empty cache returns the error.
type cached[T any] struct {
	db     *sql.DB
	source string
	ttl    time.Duration
	now    func() time.Time
	fetch  func(context.Context, float64, float64) (T, error)
}

// key rounds the coordinates to 3 decimals (about 110 m), so two people asking about the same
// city share an entry, and names the source, so places and stays never mix.
func (c *cached[T]) key(lat, lon float64) string {
	return fmt.Sprintf("%s:%.3f:%.3f", c.source, lat, lon)
}

func (c *cached[T]) get(ctx context.Context, lat, lon float64) (T, error) {
	var zero T
	key := c.key(lat, lon)

	old, fetchedAt, hit := c.load(key)
	if hit && c.now().Sub(fetchedAt) < c.ttl {
		return old, nil
	}

	fresh, err := c.fetch(ctx, lat, lon)
	if err != nil {
		if hit {
			log.Printf("Refreshing %s failed, serving data cached at %s: %v",
				key, fetchedAt.Format(time.RFC3339), err)
			return old, nil
		}
		return zero, fmt.Errorf("error fetching %s data with nothing cached: %w", c.source, err)
	}

	c.store(key, fresh)
	return fresh, nil
}

// load reads an entry. A missing row, a database that isn't initialized, or a blob that can't be
// read counts as a miss, so a broken cache never breaks a source.
func (c *cached[T]) load(key string) (T, time.Time, bool) {
	var zero T
	if c.db == nil {
		return zero, time.Time{}, false
	}

	var (
		fetchedAt      int64
		compressedData []byte
	)
	err := c.db.QueryRow(`SELECT fetched_at, data FROM source_cache WHERE key = ?;`, key).
		Scan(&fetchedAt, &compressedData)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Printf("Error reading cached data for %s: %v", key, err)
		}
		return zero, time.Time{}, false
	}

	decompressedJSON, err := decompressBlob(compressedData)
	if err != nil {
		log.Printf("Error decompressing cached data for %s: %v", key, err)
		return zero, time.Time{}, false
	}

	var value T
	if err := json.Unmarshal(decompressedJSON, &value); err != nil {
		log.Printf("Error unmarshaling cached data for %s: %v", key, err)
		return zero, time.Time{}, false
	}

	return value, time.Unix(fetchedAt, 0).UTC(), true
}

// store saves an entry. Failures are logged and never reach the caller: the data was fetched, so
// the request can be answered whether or not it could be cached.
func (c *cached[T]) store(key string, value T) {
	if c.db == nil {
		return
	}

	compressedData, err := compressJSON(value)
	if err != nil {
		log.Printf("Error compressing data for %s: %v", key, err)
		return
	}

	insertSQL := `REPLACE INTO source_cache (key, fetched_at, data) VALUES (?, ?, ?);`
	if _, err := c.db.Exec(insertSQL, key, c.now().Unix(), compressedData); err != nil {
		log.Printf("Error saving cached data for %s: %v", key, err)
	}
}

type cachedPlaces struct{ *cached[[]planner.Place] }
type cachedForecast struct{ *cached[*planner.Forecast] }
type cachedStays struct{ *cached[[]planner.Stay] }

// NewCachedPlaceSource wraps a place source with the 30-day cache.
func (s *Store) NewCachedPlaceSource(src planner.PlaceSource) planner.PlaceSource {
	return &cachedPlaces{&cached[[]planner.Place]{
		db: s.db, source: "places", ttl: placesCacheTTL, now: time.Now, fetch: src.PlacesNear,
	}}
}

// NewCachedForecastSource wraps a forecast source with the 3-hour cache.
func (s *Store) NewCachedForecastSource(src planner.ForecastSource) planner.ForecastSource {
	return &cachedForecast{&cached[*planner.Forecast]{
		db: s.db, source: "forecast", ttl: forecastCacheTTL, now: time.Now, fetch: src.DailyForecast,
	}}
}

// NewCachedStaySource wraps a stay source with the 30-day cache.
func (s *Store) NewCachedStaySource(src planner.StaySource) planner.StaySource {
	return &cachedStays{&cached[[]planner.Stay]{
		db: s.db, source: "stays", ttl: staysCacheTTL, now: time.Now, fetch: src.StaysNear,
	}}
}

func (c *cachedPlaces) PlacesNear(ctx context.Context, lat, lon float64) ([]planner.Place, error) {
	return c.get(ctx, lat, lon)
}
func (c *cachedForecast) DailyForecast(ctx context.Context, lat, lon float64) (*planner.Forecast, error) {
	return c.get(ctx, lat, lon)
}
func (c *cachedStays) StaysNear(ctx context.Context, lat, lon float64) ([]planner.Stay, error) {
	return c.get(ctx, lat, lon)
}
