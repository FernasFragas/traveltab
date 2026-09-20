package httpserver

import (
	"sync"
	"time"
)

// defaultNewCitiesPerMinute is how many never-before-cached cities /trip/:slug will fully plan
// inside any rolling minute. A crawl burst across many different, uncached cities is the risk:
// it can hammer Wikimedia and Overpass. A repeat visit to an already-cached city is never
// limited, so this only ever slows down the automatic planning of genuinely new places.
const defaultNewCitiesPerMinute = 5

// newCityLimiter is a simple in-memory sliding-window counter: at most max Allow calls may
// succeed in any trailing minute, judged by whatever clock the caller hands in.
type newCityLimiter struct {
	max int

	mu   sync.Mutex
	hits []time.Time
}

func newNewCityLimiter(max int) *newCityLimiter {
	return &newCityLimiter{max: max}
}

// Allow reports whether one more new city may be planned as of now, and records the attempt
// when it does. Hits older than a minute before now are forgotten first, so the window slides
// rather than resetting on a fixed boundary.
func (l *newCityLimiter) Allow(now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := now.Add(-time.Minute)
	kept := l.hits[:0]
	for _, hit := range l.hits {
		if hit.After(cutoff) {
			kept = append(kept, hit)
		}
	}
	l.hits = kept

	if len(l.hits) >= l.max {
		return false
	}
	l.hits = append(l.hits, now)
	return true
}
