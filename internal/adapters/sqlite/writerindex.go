package sqlite

import (
	"context"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"weatherservice/internal/writermap"
)

// WriterPost is the metadata and extracted positions retained for one source post.
type WriterPost struct {
	Host      string
	PostID    int64
	URL       string
	Title     string
	Language  string
	Modified  time.Time
	Extracted writermap.ExtractedPost
}

// WriterNameMatch connects an indexed phrase to its source post.
type WriterNameMatch struct {
	Post   WriterPost
	Phrase writermap.NamePhrase
}

// WriterSyncState is the resumable progress and latest status of one source.
type WriterSyncState struct {
	Host      string
	Status    string
	LastFull  time.Time
	LastDelta time.Time
	Cursor    string
}

// CreateWriterIndexTables adds the writers index to a new or existing database.
// The web integrator calls it from Store.initialize.
func (s *Store) CreateWriterIndexTables() error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin writer index schema: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, statement := range []string{
		`CREATE TABLE IF NOT EXISTS writer_posts (
			id INTEGER PRIMARY KEY, host TEXT NOT NULL, post_id INTEGER NOT NULL, url TEXT NOT NULL,
			title TEXT NOT NULL, lang TEXT NOT NULL, modified TEXT NOT NULL,
			is_itinerary INTEGER NOT NULL, UNIQUE (host, post_id)
		)`,
		// One row per post and phrase, with every occurrence packed into positions (see
		// encodePositions): about a fifth of the size of one row per occurrence.
		`CREATE TABLE IF NOT EXISTS writer_names (
			folded TEXT NOT NULL, post_ref INTEGER NOT NULL, positions BLOB NOT NULL,
			PRIMARY KEY (folded, post_ref)
		) WITHOUT ROWID`,
		`CREATE INDEX IF NOT EXISTS writer_names_post ON writer_names (post_ref)`,
		`CREATE TABLE IF NOT EXISTS writer_days (
			post_ref INTEGER NOT NULL, day_number INTEGER NOT NULL,
			start INTEGER NOT NULL, end INTEGER NOT NULL,
			PRIMARY KEY (post_ref, day_number, start, end), CHECK (end > start)
		) WITHOUT ROWID`,
		`CREATE TABLE IF NOT EXISTS writer_sync (
			host TEXT PRIMARY KEY, status TEXT NOT NULL, last_full TEXT,
			last_delta TEXT, cursor TEXT NOT NULL
		)`,
	} {
		if _, err := tx.Exec(statement); err != nil {
			return fmt.Errorf("create writer index: %w", err)
		}
	}
	return tx.Commit()
}

// UpsertWriterPost atomically replaces a post and its extracted positions.
func (s *Store) UpsertWriterPost(ctx context.Context, post WriterPost) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin writer post upsert: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var postRef int64
	err = tx.QueryRowContext(ctx, `INSERT INTO writer_posts
		(host, post_id, url, title, lang, modified, is_itinerary) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(host, post_id) DO UPDATE SET url=excluded.url, title=excluded.title,
		lang=excluded.lang, modified=excluded.modified, is_itinerary=excluded.is_itinerary
		RETURNING id`,
		post.Host, post.PostID, post.URL, post.Title, post.Language,
		writerIndexTime(post.Modified), post.Extracted.IsItinerary).Scan(&postRef)
	if err != nil {
		return fmt.Errorf("upsert writer post: %w", err)
	}
	for _, table := range []string{"writer_names", "writer_days"} {
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+table+" WHERE post_ref=?", postRef); err != nil {
			return fmt.Errorf("replace writer post positions: %w", err)
		}
	}
	byPhrase, err := groupPhrases(post.Extracted.Phrases)
	if err != nil {
		return err
	}
	for folded, phrases := range byPhrase {
		_, err := tx.ExecContext(ctx, `INSERT INTO writer_names (folded, post_ref, positions) VALUES (?, ?, ?)`,
			folded, postRef, encodePositions(phrases))
		if err != nil {
			return fmt.Errorf("insert writer name: %w", err)
		}
	}
	for _, day := range post.Extracted.Days {
		_, err := tx.ExecContext(ctx, `INSERT INTO writer_days
			(post_ref, day_number, start, end) VALUES (?, ?, ?, ?)
			ON CONFLICT DO NOTHING`,
			postRef, day.Number, day.Start, day.End)
		if err != nil {
			return fmt.Errorf("insert writer day: %w", err)
		}
	}
	return tx.Commit()
}

// DeleteWriterPost removes one post and all its extracted positions.
func (s *Store) DeleteWriterPost(ctx context.Context, host string, postID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin writer post delete: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, statement := range []string{
		"DELETE FROM writer_names WHERE post_ref IN (SELECT id FROM writer_posts WHERE host=? AND post_id=?)",
		"DELETE FROM writer_days WHERE post_ref IN (SELECT id FROM writer_posts WHERE host=? AND post_id=?)",
		"DELETE FROM writer_posts WHERE host=? AND post_id=?",
	} {
		if _, err := tx.ExecContext(ctx, statement, host, postID); err != nil {
			return fmt.Errorf("delete writer post: %w", err)
		}
	}
	return tx.Commit()
}

// PurgeWriterHost removes all indexed data and sync progress for a source.
func (s *Store) PurgeWriterHost(ctx context.Context, host string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin writer host purge: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, statement := range []string{
		"DELETE FROM writer_names WHERE post_ref IN (SELECT id FROM writer_posts WHERE host=?)",
		"DELETE FROM writer_days WHERE post_ref IN (SELECT id FROM writer_posts WHERE host=?)",
		"DELETE FROM writer_posts WHERE host=?",
		"DELETE FROM writer_sync WHERE host=?",
	} {
		if _, err := tx.ExecContext(ctx, statement, host); err != nil {
			return fmt.Errorf("purge writer host: %w", err)
		}
	}
	return tx.Commit()
}

// FindWriterNames returns all indexed occurrences of the supplied folded phrases.
func (s *Store) FindWriterNames(ctx context.Context, phrases []string) ([]WriterNameMatch, error) {
	if len(phrases) == 0 {
		return nil, nil
	}
	marks := strings.TrimSuffix(strings.Repeat("?,", len(phrases)), ",")
	args := make([]any, len(phrases))
	for i, phrase := range phrases {
		args[i] = phrase
	}
	rows, err := s.db.QueryContext(ctx, `SELECT p.host, p.post_id, p.url, p.title, p.lang,
		p.modified, p.is_itinerary, n.folded, n.positions
		FROM writer_names n JOIN writer_posts p ON p.id=n.post_ref
		WHERE n.folded IN (`+marks+`)`, args...)
	if err != nil {
		return nil, fmt.Errorf("find writer names: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var matches []WriterNameMatch
	for rows.Next() {
		var post WriterPost
		var modified, folded string
		var positions []byte
		if err := rows.Scan(&post.Host, &post.PostID, &post.URL,
			&post.Title, &post.Language, &modified, &post.Extracted.IsItinerary,
			&folded, &positions); err != nil {
			return nil, fmt.Errorf("scan writer name: %w", err)
		}
		if modified != "" {
			post.Modified, err = time.Parse(time.RFC3339Nano, modified)
			if err != nil {
				return nil, fmt.Errorf("parse writer post modified: %w", err)
			}
		}
		phrases, err := decodePositions(folded, positions)
		if err != nil {
			return nil, err
		}
		for _, phrase := range phrases {
			matches = append(matches, WriterNameMatch{Post: post, Phrase: phrase})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(matches, func(i, j int) bool {
		a, b := matches[i], matches[j]
		if a.Post.Host != b.Post.Host {
			return a.Post.Host < b.Post.Host
		}
		if a.Post.PostID != b.Post.PostID {
			return a.Post.PostID < b.Post.PostID
		}
		if a.Phrase.Start != b.Phrase.Start {
			return a.Phrase.Start < b.Phrase.Start
		}
		if a.Phrase.End != b.Phrase.End {
			return a.Phrase.End < b.Phrase.End
		}
		return a.Phrase.Folded < b.Phrase.Folded
	})
	return matches, nil
}

// WriterPostDays returns a post's day sections in source order.
func (s *Store) WriterPostDays(ctx context.Context, host string, postID int64) ([]writermap.DaySection, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT d.day_number, d.start, d.end FROM writer_days d
		JOIN writer_posts p ON p.id=d.post_ref
		WHERE p.host=? AND p.post_id=? ORDER BY d.start, d.end`, host, postID)
	if err != nil {
		return nil, fmt.Errorf("find writer days: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var days []writermap.DaySection
	for rows.Next() {
		var day writermap.DaySection
		if err := rows.Scan(&day.Number, &day.Start, &day.End); err != nil {
			return nil, fmt.Errorf("scan writer day: %w", err)
		}
		days = append(days, day)
	}
	return days, rows.Err()
}

// WriterPostIDs lists the stored IDs for a host's deleted-post sweep.
func (s *Store) WriterPostIDs(ctx context.Context, host string) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT post_id FROM writer_posts WHERE host=? ORDER BY post_id`, host)
	if err != nil {
		return nil, fmt.Errorf("list writer post IDs: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan writer post ID: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// GetWriterSyncState returns an empty state for a host that has not synced yet.
func (s *Store) GetWriterSyncState(ctx context.Context, host string) (WriterSyncState, error) {
	state := WriterSyncState{Host: host}
	var lastFull, lastDelta sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT status, last_full, last_delta, cursor FROM writer_sync WHERE host=?`, host).
		Scan(&state.Status, &lastFull, &lastDelta, &state.Cursor)
	if errors.Is(err, sql.ErrNoRows) {
		return state, nil
	}
	if err != nil {
		return state, fmt.Errorf("read writer sync state: %w", err)
	}
	if lastFull.Valid {
		state.LastFull, err = time.Parse(time.RFC3339Nano, lastFull.String)
		if err != nil {
			return state, fmt.Errorf("parse writer last full sync: %w", err)
		}
	}
	if lastDelta.Valid {
		state.LastDelta, err = time.Parse(time.RFC3339Nano, lastDelta.String)
		if err != nil {
			return state, fmt.Errorf("parse writer last delta sync: %w", err)
		}
	}
	return state, nil
}

// SaveWriterSyncState updates status, timestamps and cursor in one row.
func (s *Store) SaveWriterSyncState(ctx context.Context, state WriterSyncState) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO writer_sync (host, status, last_full, last_delta, cursor)
		VALUES (?, ?, ?, ?, ?) ON CONFLICT(host) DO UPDATE SET status=excluded.status,
		last_full=excluded.last_full, last_delta=excluded.last_delta, cursor=excluded.cursor`,
		state.Host, state.Status, writerIndexNullableTime(state.LastFull),
		writerIndexNullableTime(state.LastDelta), state.Cursor)
	if err != nil {
		return fmt.Errorf("save writer sync state: %w", err)
	}
	return nil
}

func writerIndexTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func writerIndexNullableTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return writerIndexTime(value)
}

// groupPhrases groups a post's phrases by folded text, dropping repeated positions, and rejects
// empty or reversed spans so a bad extraction leaves the previous version of the post in place.
func groupPhrases(phrases []writermap.NamePhrase) (map[string][]writermap.NamePhrase, error) {
	byPhrase := make(map[string][]writermap.NamePhrase)
	seen := make(map[[2]int]map[string]bool)
	for _, phrase := range phrases {
		if phrase.End <= phrase.Start || phrase.Start < 0 {
			return nil, fmt.Errorf("invalid writer name span %d-%d", phrase.Start, phrase.End)
		}
		span := [2]int{phrase.Start, phrase.End}
		if seen[span][phrase.Folded] {
			continue
		}
		if seen[span] == nil {
			seen[span] = make(map[string]bool)
		}
		seen[span][phrase.Folded] = true
		byPhrase[phrase.Folded] = append(byPhrase[phrase.Folded], phrase)
	}
	return byPhrase, nil
}

// encodePositions packs each occurrence as two uvarints: start<<1 | original case, then length.
func encodePositions(phrases []writermap.NamePhrase) []byte {
	var packed []byte
	for _, phrase := range phrases {
		start := uint64(phrase.Start) << 1
		if phrase.OriginalCase {
			start |= 1
		}
		packed = binary.AppendUvarint(packed, start)
		packed = binary.AppendUvarint(packed, uint64(phrase.End-phrase.Start))
	}
	return packed
}

func decodePositions(folded string, packed []byte) ([]writermap.NamePhrase, error) {
	var phrases []writermap.NamePhrase
	for len(packed) > 0 {
		start, n := binary.Uvarint(packed)
		if n <= 0 {
			return nil, fmt.Errorf("decode writer name positions for %q", folded)
		}
		length, m := binary.Uvarint(packed[n:])
		if m <= 0 {
			return nil, fmt.Errorf("decode writer name positions for %q", folded)
		}
		packed = packed[n+m:]
		phrases = append(phrases, writermap.NamePhrase{
			Folded: folded, OriginalCase: start&1 == 1,
			Start: int(start >> 1), End: int(start>>1 + length),
		})
	}
	return phrases, nil
}
