package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"
	"weatherservice/internal/writermap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriterIndex_ReindexReplacesRows(t *testing.T) {
	ctx := context.Background()
	store := writerIndexTestStore(t)
	post := writerIndexPost()
	require.NoError(t, store.UpsertWriterPost(ctx, post))
	require.NoError(t, store.UpsertWriterPost(ctx, post))
	assert.Equal(t, 1, writerIndexCount(t, store.db, "writer_posts"))
	assert.Equal(t, 2, writerIndexCount(t, store.db, "writer_names"))
	assert.Equal(t, 1, writerIndexCount(t, store.db, "writer_days"))

	post.Title = "Edited title"
	post.Extracted = writermap.ExtractedPost{Phrases: []writermap.NamePhrase{{Folded: "monte", OriginalCase: true, Start: 5, End: 10}}}
	require.NoError(t, store.UpsertWriterPost(ctx, post))
	assert.Equal(t, 1, writerIndexCount(t, store.db, "writer_posts"))
	assert.Equal(t, 1, writerIndexCount(t, store.db, "writer_names"))
	assert.Zero(t, writerIndexCount(t, store.db, "writer_days"))
	assert.Empty(t, writerIndexMatches(t, store, "funchal"))
	matches := writerIndexMatches(t, store, "monte")
	require.Len(t, matches, 1)
	assert.Equal(t, "Edited title", matches[0].Post.Title)
	assert.Equal(t, post.Extracted.Phrases[0], matches[0].Phrase)
}

func TestWriterIndex_LookupKeepsHostsAndPositionsDistinct(t *testing.T) {
	ctx := context.Background()
	store := writerIndexTestStore(t)
	first := writerIndexPost()
	second := writerIndexPost()
	second.Host = "other.example"
	second.Extracted.Phrases = []writermap.NamePhrase{{Folded: "funchal", OriginalCase: false, Start: 30, End: 37}}
	require.NoError(t, store.UpsertWriterPost(ctx, first))
	require.NoError(t, store.UpsertWriterPost(ctx, second))
	matches, err := store.FindWriterNames(ctx, []string{"funchal", "pico dos barcelos"})
	require.NoError(t, err)
	require.Len(t, matches, 3)
	assert.Equal(t, "example.com", matches[0].Post.Host)
	assert.Equal(t, "other.example", matches[2].Post.Host)
	assert.Equal(t, 30, matches[2].Phrase.Start)
	days, err := store.WriterPostDays(ctx, first.Host, first.PostID)
	require.NoError(t, err)
	assert.Equal(t, first.Extracted.Days, days)
	empty, err := store.FindWriterNames(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestWriterIndex_RepeatedPhraseIsOneRowWithEveryPosition(t *testing.T) {
	ctx := context.Background()
	store := writerIndexTestStore(t)
	post := writerIndexPost()
	post.Extracted.Phrases = append(post.Extracted.Phrases,
		writermap.NamePhrase{Folded: "funchal", OriginalCase: false, Start: 40, End: 47},
		writermap.NamePhrase{Folded: "funchal", OriginalCase: true, Start: 0, End: 7})
	require.NoError(t, store.UpsertWriterPost(ctx, post))
	assert.Equal(t, 2, writerIndexCount(t, store.db, "writer_names"), "one row per post and phrase")
	matches := writerIndexMatches(t, store, "funchal")
	require.Len(t, matches, 2, "a duplicate position is stored once")
	assert.Equal(t, writermap.NamePhrase{Folded: "funchal", OriginalCase: true, Start: 0, End: 7}, matches[0].Phrase)
	assert.Equal(t, writermap.NamePhrase{Folded: "funchal", OriginalCase: false, Start: 40, End: 47}, matches[1].Phrase)
	assert.Equal(t, post.Title, matches[1].Post.Title)
}

func TestWriterIndex_DeletePostKeepsOtherPosts(t *testing.T) {
	ctx := context.Background()
	store := writerIndexTestStore(t)
	first := writerIndexPost()
	second := writerIndexPost()
	second.PostID = 12
	require.NoError(t, store.UpsertWriterPost(ctx, first))
	require.NoError(t, store.UpsertWriterPost(ctx, second))
	require.NoError(t, store.DeleteWriterPost(ctx, first.Host, first.PostID))
	assert.Equal(t, 1, writerIndexCount(t, store.db, "writer_posts"))
	assert.Equal(t, 2, writerIndexCount(t, store.db, "writer_names"))
	assert.Equal(t, 1, writerIndexCount(t, store.db, "writer_days"))
	matches := writerIndexMatches(t, store, "funchal")
	require.Len(t, matches, 1)
	assert.Equal(t, second.PostID, matches[0].Post.PostID)
}

func TestWriterIndex_ListPostIDsForDeletionSweep(t *testing.T) {
	ctx := context.Background()
	store := writerIndexTestStore(t)
	first := writerIndexPost()
	second := writerIndexPost()
	second.PostID = 4
	third := writerIndexPost()
	third.Host = "other.example"
	require.NoError(t, store.UpsertWriterPost(ctx, first))
	require.NoError(t, store.UpsertWriterPost(ctx, second))
	require.NoError(t, store.UpsertWriterPost(ctx, third))
	ids, err := store.WriterPostIDs(ctx, first.Host)
	require.NoError(t, err)
	assert.Equal(t, []int64{4, 11}, ids)
}

func TestWriterIndex_FailedReindexKeepsPreviousPost(t *testing.T) {
	ctx := context.Background()
	store := writerIndexTestStore(t)
	post := writerIndexPost()
	require.NoError(t, store.UpsertWriterPost(ctx, post))
	post.Title = "Invalid edit"
	post.Extracted.Phrases = []writermap.NamePhrase{{Folded: "bad", Start: 9, End: 3}}
	assert.Error(t, store.UpsertWriterPost(ctx, post))
	matches := writerIndexMatches(t, store, "funchal")
	require.Len(t, matches, 1)
	assert.Equal(t, "Funchal guide", matches[0].Post.Title)
	assert.Equal(t, 2, writerIndexCount(t, store.db, "writer_names"))
}

func TestWriterIndex_PurgeHostDeletesEveryRowForIt(t *testing.T) {
	ctx := context.Background()
	store := writerIndexTestStore(t)
	first := writerIndexPost()
	second := writerIndexPost()
	second.Host = "other.example"
	require.NoError(t, store.UpsertWriterPost(ctx, first))
	require.NoError(t, store.UpsertWriterPost(ctx, second))
	require.NoError(t, store.SaveWriterSyncState(ctx, WriterSyncState{Host: first.Host, Status: "ready", Cursor: "2"}))
	require.NoError(t, store.SaveWriterSyncState(ctx, WriterSyncState{Host: second.Host, Status: "blocked"}))
	require.NoError(t, store.PurgeWriterHost(ctx, first.Host))
	for table, want := range map[string]int{"writer_posts": 1, "writer_names": 2, "writer_days": 1, "writer_sync": 1} {
		assert.Equal(t, want, writerIndexCount(t, store.db, table), table)
	}
	for _, table := range []string{"writer_posts", "writer_sync"} {
		var count int
		require.NoError(t, store.db.QueryRow("SELECT count(*) FROM "+table+" WHERE host = ?", first.Host).Scan(&count))
		assert.Zero(t, count, table)
	}
	matches := writerIndexMatches(t, store, "funchal")
	require.Len(t, matches, 1)
	assert.Equal(t, second.Host, matches[0].Post.Host)
}

func TestWriterIndex_SyncStateRoundTrips(t *testing.T) {
	ctx := context.Background()
	store := writerIndexTestStore(t)
	missing, err := store.GetWriterSyncState(ctx, "example.com")
	require.NoError(t, err)
	assert.Equal(t, WriterSyncState{Host: "example.com"}, missing)
	state := WriterSyncState{
		Host: "example.com", Status: "running", Cursor: "page:2",
		LastFull:  time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC),
		LastDelta: time.Date(2026, 10, 11, 8, 0, 0, 0, time.UTC),
	}
	require.NoError(t, store.SaveWriterSyncState(ctx, state))
	state.Status, state.Cursor = "ready", ""
	require.NoError(t, store.SaveWriterSyncState(ctx, state))
	got, err := store.GetWriterSyncState(ctx, state.Host)
	require.NoError(t, err)
	assert.Equal(t, state, got)
	assert.Equal(t, 1, writerIndexCount(t, store.db, "writer_sync"))
}

func TestWriterIndex_CreatesOnlyItsTablesInExistingDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "existing.db")
	conn, err := sql.Open("sqlite3", path)
	require.NoError(t, err)
	_, err = conn.Exec(`CREATE TABLE sentinel (value TEXT); INSERT INTO sentinel VALUES ('untouched');`)
	require.NoError(t, err)
	require.NoError(t, conn.Close())
	store, err := Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	require.NoError(t, store.CreateWriterIndexTables())
	require.NoError(t, store.CreateWriterIndexTables())
	for _, table := range []string{"writer_posts", "writer_names", "writer_days", "writer_sync"} {
		var name string
		require.NoError(t, store.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name))
		assert.Equal(t, table, name)
	}
	var value string
	require.NoError(t, store.db.QueryRow(`SELECT value FROM sentinel`).Scan(&value))
	assert.Equal(t, "untouched", value)
	var schema string
	require.NoError(t, store.db.QueryRow(`SELECT sql FROM sqlite_master WHERE name='writer_posts'`).Scan(&schema))
	assert.NotContains(t, schema, "body")
	assert.NotContains(t, schema, "content")
}

func writerIndexTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "writer-index.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	require.NoError(t, store.CreateWriterIndexTables())
	return store
}

func writerIndexPost() WriterPost {
	return WriterPost{
		Host: "example.com", PostID: 11, URL: "https://example.com/post", Title: "Funchal guide", Language: "en",
		Modified: time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC),
		Extracted: writermap.ExtractedPost{
			Phrases: []writermap.NamePhrase{
				{Folded: "funchal", OriginalCase: true, Start: 0, End: 7},
				{Folded: "pico dos barcelos", OriginalCase: true, Start: 10, End: 27},
			},
			Days:        []writermap.DaySection{{Number: 1, Start: 0, End: 50}},
			IsItinerary: true,
		},
	}
}

func writerIndexMatches(t *testing.T, store *Store, phrase string) []WriterNameMatch {
	t.Helper()
	matches, err := store.FindWriterNames(context.Background(), []string{phrase})
	require.NoError(t, err)
	return matches
}

func writerIndexCount(t *testing.T, conn *sql.DB, table string) int {
	t.Helper()
	var count int
	require.NoError(t, conn.QueryRow("SELECT count(*) FROM "+table).Scan(&count))
	return count
}
