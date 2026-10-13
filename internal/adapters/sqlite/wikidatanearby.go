package sqlite

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"weatherservice/internal/application"
	"weatherservice/internal/writermap"
)

var (
	_ application.BasePlaceSource    = (*cachedBasePlaces)(nil)
	_ application.BasePlaceRefresher = (*cachedBasePlaces)(nil)
	_ writermap.PartialBaseSource    = (*cachedBasePlaces)(nil)
)

// basePlaceKeyFormat encodes the whole area, so CachedDestinations can read areas back from keys.
// v2: v1 rows stored a Commons page as the photo URL and the image URL as the photo.
const (
	basePlaceCacheVersion = "v3"
	basePlaceKeyFormat    = "wikidata:" + basePlaceCacheVersion + ":%q:%q:%g:%g:%g"
)

// basePartialKeyFormat holds a destination's half-loaded result. It has its own prefix, so a partial
// row is never read as a complete one and CachedDestinations never lists it.
const basePartialKeyFormat = "wikidatapartial:" + basePlaceCacheVersion + ":%q:%q:%g:%g:%g"

// A load whose page fails (Wikidata is often slow or busy) is tried basePlaceAttempts times,
// basePlaceRetryDelay apart, each resuming after the last stored page. If all fail, the next poll
// starts it again once basePlaceRetryAfter has passed.
const (
	basePlaceAttempts   = 3
	basePlaceRetryDelay = 5 * time.Second
	basePlaceRetryAfter = 30 * time.Second
)

// cachedBasePlaces stores both the visible pins and the unfiltered matching gazetteer in one row.
// Entries stay until Refresh replaces them. Loads run in the background, detached from the request
// that started them, one flight per destination. A paged source stores its progress after each page,
// so a failed or interrupted load resumes there; only the finished result is stored as complete.
type cachedBasePlaces struct {
	cache      *cached[writermap.BasePlaces]
	partials   *cached[writermap.BaseProgress]
	source     application.BasePlaceSource
	mu         sync.Mutex
	flights    map[string]*basePlaceFlight
	failures   map[string]basePlaceFailure
	retryDelay time.Duration
}

type basePlaceFlight struct {
	done     chan struct{}
	refresh  bool
	progress writermap.BaseProgress // guarded by cachedBasePlaces.mu
	result   writermap.BasePlaces
	err      error
}

type basePlaceFailure struct {
	err      error
	at       time.Time
	progress writermap.BaseProgress
}

// NewCachedBasePlaceSource wraps a Wikidata source with a permanent SQLite cache.
func (s *Store) NewCachedBasePlaceSource(source application.BasePlaceSource) application.BasePlaceSource {
	return &cachedBasePlaces{
		cache:    &cached[writermap.BasePlaces]{db: s.db, now: time.Now},
		partials: &cached[writermap.BaseProgress]{db: s.db, now: time.Now},
		source:   source,
		flights:  make(map[string]*basePlaceFlight),
		failures: make(map[string]basePlaceFailure),

		retryDelay: basePlaceRetryDelay,
	}
}

// Places returns the complete result, waiting for the background load if one is needed.
// A caller that gives up leaves the load running.
func (c *cachedBasePlaces) Places(ctx context.Context, dest writermap.Area) (writermap.BasePlaces, error) {
	key := basePlaceKey(dest)
	if places, _, ok := c.cache.load(key); ok {
		return places, nil
	}
	return c.fetch(ctx, dest, key, false)
}

// Partial returns the complete result when it is cached, otherwise starts the load if none is
// running and returns the places loaded so far. A load that failed less than basePlaceRetryAfter
// ago is not restarted; its error comes back with what it had loaded.
func (c *cachedBasePlaces) Partial(ctx context.Context, dest writermap.Area) (writermap.BaseProgress, error) {
	key := basePlaceKey(dest)
	c.mu.Lock()
	defer c.mu.Unlock()
	if places, _, ok := c.cache.load(key); ok {
		return writermap.BaseProgress{BasePlaces: places, Done: true}, nil
	}
	if flight := c.flights[key]; flight != nil {
		return flight.progress, nil
	}
	if failure, ok := c.failures[key]; ok && c.cache.now().Sub(failure.at) < basePlaceRetryAfter {
		return failure.progress, failure.err
	}
	return c.start(ctx, dest, key, false).progress, nil
}

func (c *cachedBasePlaces) Refresh(ctx context.Context, dest writermap.Area) error {
	_, err := c.fetch(ctx, dest, basePlaceKey(dest), true)
	return err
}

// fetch checks again under the flight lock so a caller that just missed the database cannot
// start a second query after the first query has stored its result and left the flight map.
func (c *cachedBasePlaces) fetch(ctx context.Context, dest writermap.Area, key string, refresh bool) (writermap.BasePlaces, error) {
	c.mu.Lock()
	flight := c.flights[key]
	if flight == nil {
		if !refresh {
			if places, _, ok := c.cache.load(key); ok {
				c.mu.Unlock()
				return places, nil
			}
		}
		flight = c.start(ctx, dest, key, refresh)
	}
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

// start registers a flight and loads in the background, so the load outlives the request that
// started it. The caller holds c.mu. A refresh ignores and does not write stored progress.
func (c *cachedBasePlaces) start(ctx context.Context, dest writermap.Area, key string, refresh bool) *basePlaceFlight {
	flight := &basePlaceFlight{done: make(chan struct{}), refresh: refresh}
	partialKey := basePartialKey(dest)
	if !refresh {
		if progress, _, ok := c.partials.load(partialKey); ok {
			flight.progress = progress
		}
	}
	c.flights[key] = flight
	go c.load(context.WithoutCancel(ctx), dest, key, partialKey, flight)
	return flight
}

func (c *cachedBasePlaces) load(ctx context.Context, dest writermap.Area, key, partialKey string, flight *basePlaceFlight) {
	var result writermap.BasePlaces
	var err error
	if paged, ok := c.source.(writermap.PagedBaseSource); ok {
		var final writermap.BaseProgress
		for attempt := 1; attempt <= basePlaceAttempts; attempt++ {
			c.mu.Lock()
			from := flight.progress
			c.mu.Unlock()
			final, err = paged.PlacesFrom(ctx, dest, from, func(progress writermap.BaseProgress) {
				c.mu.Lock()
				flight.progress = progress
				c.mu.Unlock()
				if !flight.refresh && !progress.Done {
					c.partials.store(partialKey, progress)
				}
			})
			c.mu.Lock()
			flight.progress = final
			c.mu.Unlock()
			if err == nil || attempt == basePlaceAttempts {
				break
			}
			log.Printf("base places for %s: attempt %d failed after %d places", dest.Name, attempt, len(final.Gazetteer))
			time.Sleep(c.retryDelay)
		}
		result = final.BasePlaces
	} else {
		result, err = c.source.Places(ctx, dest)
	}
	if err != nil {
		result = writermap.BasePlaces{}
		err = fmt.Errorf("fetch base places for %s: %w", dest.Name, err)
		log.Printf("fetch base places for %s failed after %d places", dest.Name, len(flight.progress.Gazetteer))
	} else {
		c.cache.store(key, result)
		c.removePartial(partialKey)
	}

	c.mu.Lock()
	flight.result, flight.err = result, err
	if err != nil {
		c.failures[key] = basePlaceFailure{err: err, at: c.cache.now(), progress: flight.progress}
	} else {
		delete(c.failures, key)
	}
	close(flight.done)
	delete(c.flights, key)
	c.mu.Unlock()
}

// CachedDestinations lists every destination with cached base places, read back from the cache
// keys. Names and countries come back lowercased, as stored; Refresh maps them to the same row.
func (s *Store) CachedDestinations(ctx context.Context) ([]writermap.Area, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT key FROM source_cache WHERE key LIKE ?;`, "wikidata:"+basePlaceCacheVersion+":%")
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

func basePartialKey(dest writermap.Area) string {
	return fmt.Sprintf(basePartialKeyFormat,
		strings.ToLower(strings.TrimSpace(dest.Name)), strings.ToLower(strings.TrimSpace(dest.Country)),
		dest.Lat, dest.Lon, dest.RadiusKM)
}

func (c *cachedBasePlaces) removePartial(key string) {
	if c.partials.db == nil {
		return
	}
	if _, err := c.partials.db.Exec(`DELETE FROM source_cache WHERE key = ?;`, key); err != nil {
		log.Printf("Error removing partial base places %s: %v", key, err)
	}
}
