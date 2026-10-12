package sqlite

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"weatherservice/internal/application"
	"weatherservice/internal/writermap"
)

var (
	_ application.BasePlaceSource    = (*cachedBasePlaces)(nil)
	_ application.BasePlaceRefresher = (*cachedBasePlaces)(nil)
)

// basePlaceKeyFormat encodes the whole area, so CachedDestinations can read areas back from keys.
// v2: v1 rows stored a Commons page as the photo URL and the image URL as the photo.
const (
	basePlaceCacheVersion = "v2"
	basePlaceKeyFormat    = "wikidata:" + basePlaceCacheVersion + ":%q:%q:%g:%g:%g"
)

// cachedBasePlaces stores both the visible pins and the unfiltered matching gazetteer in one row.
// Entries stay until Refresh replaces them. flights coordinates queries for each destination.
type cachedBasePlaces struct {
	cache   *cached[writermap.BasePlaces]
	source  application.BasePlaceSource
	mu      sync.Mutex
	flights map[string]*basePlaceFlight
}

type basePlaceFlight struct {
	done    chan struct{}
	refresh bool
	result  writermap.BasePlaces
	err     error
}

// NewCachedBasePlaceSource wraps a Wikidata source with a permanent SQLite cache.
func (s *Store) NewCachedBasePlaceSource(source application.BasePlaceSource) application.BasePlaceSource {
	return &cachedBasePlaces{
		cache:   &cached[writermap.BasePlaces]{db: s.db, now: time.Now},
		source:  source,
		flights: make(map[string]*basePlaceFlight),
	}
}

func (c *cachedBasePlaces) Places(ctx context.Context, dest writermap.Area) (writermap.BasePlaces, error) {
	key := basePlaceKey(dest)
	if places, _, ok := c.cache.load(key); ok {
		return places, nil
	}
	return c.fetch(ctx, dest, key, false)
}

func (c *cachedBasePlaces) Refresh(ctx context.Context, dest writermap.Area) error {
	_, err := c.fetch(ctx, dest, basePlaceKey(dest), true)
	return err
}

// fetch checks again under the flight lock so a caller that just missed the database cannot
// start a second query after the first query has stored its result and left the flight map.
func (c *cachedBasePlaces) fetch(ctx context.Context, dest writermap.Area, key string, refresh bool) (writermap.BasePlaces, error) {
	c.mu.Lock()
	if flight := c.flights[key]; flight != nil {
		c.mu.Unlock()
		select {
		case <-ctx.Done():
			return writermap.BasePlaces{}, ctx.Err()
		case <-flight.done:
			if refresh && !flight.refresh {
				return c.fetch(ctx, dest, key, true)
			}
			return flight.result, flight.err
		}
	}
	if !refresh {
		if places, _, ok := c.cache.load(key); ok {
			c.mu.Unlock()
			return places, nil
		}
	}
	flight := &basePlaceFlight{done: make(chan struct{}), refresh: refresh}
	c.flights[key] = flight
	c.mu.Unlock()

	result, err := c.source.Places(ctx, dest)
	if err != nil {
		result = writermap.BasePlaces{}
		err = fmt.Errorf("fetch base places for %s: %w", dest.Name, err)
	} else {
		c.cache.store(key, result)
	}

	c.mu.Lock()
	flight.result, flight.err = result, err
	close(flight.done)
	delete(c.flights, key)
	c.mu.Unlock()
	return result, err
}

// CachedDestinations lists every destination with cached base places, read back from the cache
// keys. Names and countries come back lowercased, as stored; Refresh maps them to the same row.
func (s *Store) CachedDestinations(ctx context.Context) ([]writermap.Area, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT key FROM source_cache WHERE key LIKE 'wikidata:v2:%';`)
	if err != nil {
		return nil, fmt.Errorf("list cached destinations: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var areas []writermap.Area
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("list cached destinations: %w", err)
		}
		var area writermap.Area
		if _, err := fmt.Sscanf(key, basePlaceKeyFormat, &area.Name, &area.Country, &area.Lat, &area.Lon, &area.RadiusKM); err != nil {
			return nil, fmt.Errorf("parse cached destination %q: %w", key, err)
		}
		areas = append(areas, area)
	}
	return areas, rows.Err()
}

func basePlaceKey(dest writermap.Area) string {
	return fmt.Sprintf(basePlaceKeyFormat,
		strings.ToLower(strings.TrimSpace(dest.Name)), strings.ToLower(strings.TrimSpace(dest.Country)),
		dest.Lat, dest.Lon, dest.RadiusKM)
}
