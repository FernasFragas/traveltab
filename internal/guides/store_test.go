package guides

import (
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
