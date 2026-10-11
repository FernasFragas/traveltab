package httpserver

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
	"weatherservice/internal/application"
	"weatherservice/internal/writermap"

	"github.com/gofiber/fiber/v2"
)

const suggestionTTL = 10 * time.Minute
const suggestionCacheLimit = 128
const baseWarmInterval = 2 * time.Second

type suggestionEntry struct {
	places  []application.PlaceSuggestion
	expires time.Time
}

type suggestionCache struct {
	mu           sync.Mutex
	entries      map[string]suggestionEntry
	baseSource   application.BasePlaceSource
	baseStarted  map[baseWarmKey]bool
	clientWarmAt map[string]time.Time
	now          func() time.Time
}

func newSuggestionCache() *suggestionCache {
	return &suggestionCache{
		entries:      make(map[string]suggestionEntry),
		baseStarted:  make(map[baseWarmKey]bool),
		clientWarmAt: make(map[string]time.Time),
		now:          time.Now,
	}
}

type baseWarmKey struct {
	name, country string
	lat, lon      float64
}

func (c *suggestionCache) get(query string) ([]application.PlaceSuggestion, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[query]
	if !ok || time.Now().After(entry.expires) {
		delete(c.entries, query)
		return nil, false
	}
	return entry.places, true
}

func (c *suggestionCache) put(query string, places []application.PlaceSuggestion) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= suggestionCacheLimit {
		var oldest string
		var expires time.Time
		for key, entry := range c.entries {
			if oldest == "" || entry.expires.Before(expires) {
				oldest, expires = key, entry.expires
			}
		}
		delete(c.entries, oldest)
	}
	c.entries[query] = suggestionEntry{places: places, expires: time.Now().Add(suggestionTTL)}
}

func (s *Server) SetPlaceSuggester(source application.PlaceSuggester) {
	s.placeSuggester = source
}

func (s *Server) SetBasePlaceSource(source application.BasePlaceSource, enabled bool) {
	if !enabled {
		source = nil
	}
	s.suggestions.mu.Lock()
	s.suggestions.baseSource = source
	s.suggestions.mu.Unlock()
}

func validCountryCode(code string) bool {
	return len(code) == 2 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= 'Z'
}

func (s *Server) suggestPlaces(ctx *fiber.Ctx) error {
	query := strings.TrimSpace(strings.SplitN(ctx.Query("city_name"), ",", 2)[0])
	if utf8.RuneCountInString(query) < 2 || s.placeSuggester == nil {
		return ctx.SendString("")
	}
	key := strings.ToLower(query)
	places, ok := s.suggestions.get(key)
	if !ok {
		var err error
		places, err = s.placeSuggester.Suggest(ctx.Context(), query)
		if err != nil {
			return ctx.SendString("")
		}
		s.suggestions.put(key, places)
	}
	if len(places) > 0 {
		client := ctx.Get("Fly-Client-IP")
		if client == "" {
			client = ctx.IP()
		}
		s.suggestions.warm(client, places[0])
	}
	return ctx.Render("search_suggestions", fiber.Map{"Suggestions": places})
}

func (c *suggestionCache) warm(client string, place application.PlaceSuggestion) {
	if !validCoordinates(place.Lat, place.Lon) {
		return
	}
	key := baseWarmKey{place.Name, place.Country, place.Lat, place.Lon}
	c.mu.Lock()
	source := c.baseSource
	if source == nil {
		c.mu.Unlock()
		return
	}
	if c.baseStarted[key] {
		c.mu.Unlock()
		return
	}
	now := c.now()
	if last, ok := c.clientWarmAt[client]; ok && now.Sub(last) < baseWarmInterval {
		c.mu.Unlock()
		return
	}
	for ip, last := range c.clientWarmAt {
		if now.Sub(last) >= baseWarmInterval {
			delete(c.clientWarmAt, ip)
		}
	}
	c.clientWarmAt[strings.Clone(client)] = now
	c.baseStarted[key] = true
	c.mu.Unlock()
	go c.loadBase(source, key, writermap.Area{Name: place.Name, Country: place.Country, Lat: place.Lat, Lon: place.Lon})
}

func (c *suggestionCache) loadBase(source application.BasePlaceSource, key baseWarmKey, area writermap.Area) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := source.Places(ctx, area)
	if err != nil {
		c.mu.Lock()
		delete(c.baseStarted, key)
		c.mu.Unlock()
		log.Printf("Could not warm base places for %s, %s: %v", area.Name, area.Country, err)
	}
}
