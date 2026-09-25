package guides

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteGuides_KeysBySlugAndRecordsRevision(t *testing.T) {
	path := filepath.Join(t.TempDir(), "guides.json")

	generatedAt := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	guides := map[string]Entry{
		"lisbon-pt": {
			Intro:              "Lisbon is a coastal capital of hills and trams.",
			WikivoyageRevision: 5364237,
			GeneratedAt:        generatedAt,
		},
	}

	require.NoError(t, WriteGuides(path, guides))

	loaded, err := LoadGuides(path)
	require.NoError(t, err)

	require.Contains(t, loaded, "lisbon-pt")
	entry := loaded["lisbon-pt"]
	assert.Equal(t, "Lisbon is a coastal capital of hills and trams.", entry.Intro)
	assert.Equal(t, int64(5364237), entry.WikivoyageRevision)
	assert.True(t, generatedAt.Equal(entry.GeneratedAt))
}

func TestLoadGuides_ReturnsEmptyMapWhenFileIsMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.json")

	loaded, err := LoadGuides(path)
	require.NoError(t, err)
	assert.Empty(t, loaded)
}

func TestLoadGuides_ReadsAFileWrittenBeforeTheReviewFieldsExisted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "guides.json")
	old := `{"lisbon-pt": {"intro": "Old intro.", "wikivoyage_revision": 5364237, "generated_at": "2026-09-20T18:08:01.230825+01:00"}}`
	require.NoError(t, os.WriteFile(path, []byte(old), 0o644))

	loaded, err := LoadGuides(path)
	require.NoError(t, err)

	entry := loaded["lisbon-pt"]
	assert.Equal(t, "Old intro.", entry.Intro)
	assert.False(t, entry.Reviewed)
	assert.Error(t, entry.Publishable())
}

func TestWriteGuides_OmitsReviewFieldsForAnUnreviewedEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "guides.json")
	entry := Entry{Intro: "Draft.", WikivoyageRevision: 1, GeneratedAt: time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)}

	require.NoError(t, WriteGuides(path, map[string]Entry{"lisbon-pt": entry}))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "reviewed", "the generator's output format has no review fields")
	assert.NotContains(t, string(data), "wikivoyage_title")
}
