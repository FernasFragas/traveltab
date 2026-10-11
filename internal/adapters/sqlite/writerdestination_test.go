package sqlite

import (
	"context"
	"testing"
	"weatherservice/internal/writermap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriterMatchRows_JoinPostsWithTheirDays(t *testing.T) {
	ctx := context.Background()
	store := writerIndexTestStore(t)
	post := writerIndexPost()
	require.NoError(t, store.UpsertWriterPost(ctx, post))

	rows, err := store.WriterMatchRows(ctx, []string{"funchal", "pico dos barcelos"})

	require.NoError(t, err)
	require.Len(t, rows, 2)
	got := rows[0].Post
	assert.Equal(t, post.Host, got.Host)
	assert.Equal(t, post.PostID, got.PostID)
	assert.Equal(t, post.URL, got.URL)
	assert.Equal(t, post.Title, got.Title)
	assert.Equal(t, post.Language, got.Language)
	assert.True(t, got.Modified.Equal(post.Modified))
	assert.True(t, got.Extracted.IsItinerary)
	assert.Equal(t, post.Extracted.Days, got.Extracted.Days)
	assert.Empty(t, got.Extracted.Phrases)
	assert.Equal(t, post.Extracted.Phrases[0], rows[0].Phrase)
	assert.Equal(t, post.Extracted.Days, rows[1].Post.Extracted.Days)
}

func TestWriterDestination_UpsertsInPlaceAndListsAreas(t *testing.T) {
	ctx := context.Background()
	store := writerIndexTestStore(t)
	area := writermap.Area{Name: "Funchal", Country: "PT", Lat: 32.65, Lon: -16.91}

	_, ok, err := store.WriterDestination(ctx, area)
	require.NoError(t, err)
	assert.False(t, ok)

	first := writermap.DestinationResult{Places: []writermap.Place{{QID: "Q1", Names: []string{"Monte"}}}, WriterCounts: map[string]int{"Q1": 1}}
	second := writermap.DestinationResult{Places: []writermap.Place{{QID: "Q2", Names: []string{"Sé"}}}, WriterCounts: map[string]int{"Q2": 2}, Complete: true}
	require.NoError(t, store.SaveWriterDestination(ctx, area, first))
	require.NoError(t, store.SaveWriterDestination(ctx, area, second))

	got, ok, err := store.WriterDestination(ctx, writermap.Area{Name: " funchal ", Country: "pt", Lat: 32.65, Lon: -16.91})
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, second, got)
	assert.Equal(t, 1, writerIndexCount(t, store.db, "source_cache"))

	areas, err := store.WriterDestinations(ctx)
	require.NoError(t, err)
	assert.Equal(t, []writermap.Area{{Name: "funchal", Country: "pt", Lat: 32.65, Lon: -16.91}}, areas)
	cached, err := store.CachedDestinations(ctx)
	require.NoError(t, err)
	assert.Empty(t, cached)
}
