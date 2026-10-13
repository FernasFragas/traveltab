package sqlite

import (
	"context"
	"fmt"
	"strings"
	"time"

	"weatherservice/internal/writermap"
)

// The writers' map source reads the index and keeps its results through these methods.
var (
	_ writermap.MatchIndex       = (*Store)(nil)
	_ writermap.DestinationStore = (*Store)(nil)
)

// writerDestinationKeyFormat encodes the whole area, so WriterDestinations can read areas back.
const (
	writerDestinationKeyPrefix = "writermap:" + basePlaceCacheVersion + ":m3:"
	writerDestinationKeyFormat = writerDestinationKeyPrefix + "%q:%q:%g:%g:%g"
)

// WriterMatchRows returns indexed occurrences of the folded phrases, each with its post's day
// sections, which the matcher needs for day context and itineraries.
func (s *Store) WriterMatchRows(ctx context.Context, phrases []string) ([]writermap.MatchRow, error) {
	matches, err := s.FindWriterNames(ctx, phrases)
	if err != nil {
		return nil, err
	}
	days := make(map[writermap.PostRef][]writermap.DaySection)
	rows := make([]writermap.MatchRow, len(matches))
	for i, match := range matches {
		ref := writermap.PostRef(fmt.Sprintf("%s:%d", match.Post.Host, match.Post.PostID))
		postDays, ok := days[ref]
		if !ok {
			postDays, err = s.WriterPostDays(ctx, match.Post.Host, match.Post.PostID)
			if err != nil {
				return nil, err
			}
			days[ref] = postDays
		}
		post := match.Post
		rows[i] = writermap.MatchRow{
			Post: writermap.IndexedPost{
				Host: post.Host, PostID: post.PostID, URL: post.URL, Title: post.Title,
				Language: post.Language, Modified: post.Modified,
				Extracted: writermap.ExtractedPost{IsItinerary: post.Extracted.IsItinerary, Days: postDays},
			},
			Phrase: match.Phrase,
		}
	}
	return rows, nil
}

// WriterDestination reads a stored destination result. Results never expire. An unreadable
// row counts as missing, so the source recomputes it.
func (s *Store) WriterDestination(_ context.Context, dest writermap.Area) (writermap.DestinationResult, bool, error) {
	result, _, ok := s.writerDestinations().load(writerDestinationKey(dest))
	return result, ok, nil
}

// SaveWriterDestination upserts a destination result in place. A failed write is logged, as in
// the other caches: the result was computed, so the request can still be answered.
func (s *Store) SaveWriterDestination(_ context.Context, dest writermap.Area, result writermap.DestinationResult) error {
	s.writerDestinations().store(writerDestinationKey(dest), result)
	return nil
}

// WriterDestinations lists every stored destination, read back from the keys. Names and
// countries come back lowercased, as stored; they map to the same rows.
func (s *Store) WriterDestinations(ctx context.Context) ([]writermap.Area, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT key FROM source_cache WHERE key LIKE ?;`, writerDestinationKeyPrefix+"%")
	if err != nil {
		return nil, fmt.Errorf("list writer destinations: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var areas []writermap.Area
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("list writer destinations: %w", err)
		}
		var area writermap.Area
		if _, err := fmt.Sscanf(key, writerDestinationKeyFormat, &area.Name, &area.Country, &area.Lat, &area.Lon, &area.RadiusKM); err != nil {
			return nil, fmt.Errorf("parse writer destination %q: %w", key, err)
		}
		areas = append(areas, area)
	}
	return areas, rows.Err()
}

func (s *Store) writerDestinations() *cached[writermap.DestinationResult] {
	return &cached[writermap.DestinationResult]{db: s.db, now: time.Now}
}

func writerDestinationKey(dest writermap.Area) string {
	return fmt.Sprintf(writerDestinationKeyFormat,
		strings.ToLower(strings.TrimSpace(dest.Name)), strings.ToLower(strings.TrimSpace(dest.Country)),
		dest.Lat, dest.Lon, dest.RadiusKM)
}
