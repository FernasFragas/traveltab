package httpserver

import (
	"strings"
	"sync"
	"time"
	"unicode/utf8"
	"weatherservice/internal/application"

	"github.com/gofiber/fiber/v2"
)

const suggestionTTL = 10 * time.Minute
const suggestionCacheLimit = 128

type suggestionEntry struct {
	places  []application.PlaceSuggestion
	expires time.Time
}

type suggestionCache struct {
	mu      sync.Mutex
	entries map[string]suggestionEntry
}

func newSuggestionCache() *suggestionCache {
	return &suggestionCache{entries: make(map[string]suggestionEntry)}
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
	return ctx.Render("search_suggestions", fiber.Map{"Suggestions": places})
}
