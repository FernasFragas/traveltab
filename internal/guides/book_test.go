package guides

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	guidedata "weatherservice/guides"
	"weatherservice/internal/slug"
)

func reviewedEntry() Entry {
	return Entry{
		Intro:              "Lisbon is the capital of Portugal.\n\nIt is built on seven hills.",
		WikivoyageRevision: 5364237,
		GeneratedAt:        time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC),
		WikivoyageTitle:    "Lisbon",
		Reviewed:           true,
		ReviewedAt:         "2026-09-24",
		ReviewNote:         "checked",
	}
}

func TestEntry_Publishable(t *testing.T) {
	for name, tc := range map[string]struct {
		change  func(*Entry)
		problem string
	}{
		"reviewed and complete": {func(*Entry) {}, ""},
		"not reviewed":          {func(e *Entry) { e.Reviewed = false }, "not reviewed"},
		"blank intro":           {func(e *Entry) { e.Intro = " \n " }, "empty intro"},
		"no revision":           {func(e *Entry) { e.WikivoyageRevision = 0 }, "revision"},
		"no title":              {func(e *Entry) { e.WikivoyageTitle = "" }, "title"},
		"padded title":          {func(e *Entry) { e.WikivoyageTitle = " Lisbon" }, "title"},
		"control character":     {func(e *Entry) { e.WikivoyageTitle = "Lis\nbon" }, "title"},
		"no review date":        {func(e *Entry) { e.ReviewedAt = "" }, "reviewed_at"},
		"badly formed date":     {func(e *Entry) { e.ReviewedAt = "24/09/2026" }, "reviewed_at"},
	} {
		t.Run(name, func(t *testing.T) {
			entry := reviewedEntry()
			tc.change(&entry)

			err := entry.Publishable()
			if tc.problem == "" {
				assert.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.problem)
		})
	}
}

func TestBook_GuideMatchesCityAndCountryTogether(t *testing.T) {
	book := NewBook(map[string]Entry{"lisbon-pt": reviewedEntry()})

	guide, ok := book.Guide("Lisbon", "PT")
	require.True(t, ok)
	assert.Equal(t, "Lisbon", guide.SourceTitle)
	assert.Equal(t, int64(5364237), guide.SourceRevision)
	assert.Equal(t, "2026-09-24", guide.ReviewedOn)
	assert.True(t, strings.HasPrefix(guide.Intro, "Lisbon is the capital"))

	for name, tc := range map[string]struct{ city, country string }{
		"same name, other country": {"Lisbon", "US"},
		"other city, same country": {"Porto", "PT"},
		"country given as a name":  {"Lisbon", "Portugal"},
		"blank city":               {"", "PT"},
		"blank country":            {"Lisbon", ""},
		"local spelling":           {"Lisboa", "PT"},
	} {
		_, ok := book.Guide(tc.city, tc.country)
		assert.False(t, ok, name)
	}

	// Case and stray whitespace do not matter, as in trip slugs.
	_, ok = book.Guide("  lisbon ", "pt")
	assert.True(t, ok)
}

func TestBook_NeverPublishesAnUnreviewedOrIncompleteEntry(t *testing.T) {
	unreviewed := reviewedEntry()
	unreviewed.Reviewed = false

	untitled := reviewedEntry()
	untitled.WikivoyageTitle = ""

	book := NewBook(map[string]Entry{"lisbon-pt": unreviewed, "porto-pt": untitled})

	assert.Zero(t, book.Len())

	_, ok := book.Guide("Lisbon", "PT")
	assert.False(t, ok)
	_, ok = book.Guide("Porto", "PT")
	assert.False(t, ok)
}

func TestBook_NilHasNoGuides(t *testing.T) {
	var book *Book

	_, ok := book.Guide("Lisbon", "PT")
	assert.False(t, ok)
	assert.Zero(t, book.Len())
}

func TestParseBook_RejectsMalformedJSON(t *testing.T) {
	_, err := ParseBook([]byte(`{"lisbon-pt": `))
	assert.Error(t, err)
}

// The shipped file is what the site serves. These checks keep hand edits from publishing
// something the attribution rules would not allow, without needing the network.
func TestShippedGuides_AreReviewedAttributedAndCanonical(t *testing.T) {
	entries, err := LoadGuides(filepath.Join("..", "..", "guides", "guides.json"))
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	starter := map[string]bool{}
	for _, city := range Cities {
		starter[city.Slug()] = true
	}

	book, err := ParseBook(guidedata.JSON)
	require.NoError(t, err, "the embedded file must decode")

	published := 0

	for key, entry := range entries {
		city, country, ok := slug.Parse(key)
		require.True(t, ok, "key %q is not a city-country slug", key)
		assert.True(t, starter[key], "key %q is not one of the starter cities", key)
		assert.NotEmpty(t, entry.WikivoyageTitle, key)
		assert.Positive(t, entry.WikivoyageRevision, key)

		if !entry.Reviewed {
			// An unreviewed entry must stay unpublished.
			_, shown := book.Guide(city, country)
			assert.False(t, shown, "%s is unreviewed but published", key)

			continue
		}

		require.NoError(t, entry.Publishable(), key)
		assert.NotEmpty(t, entry.ReviewNote, key)
		assert.Less(t, len(strings.Fields(entry.Intro)), 260, "%s: an intro should stay short", key)

		guide, shown := book.Guide(city, country)
		require.True(t, shown, "%s is reviewed but not published", key)
		assert.Equal(t, entry.WikivoyageRevision, guide.SourceRevision, key)

		published++
	}

	assert.Equal(t, published, book.Len(), "the embedded book and the file on disk must agree")
}

func TestShippedGuides_FileIsWhatTheGeneratorWouldWrite(t *testing.T) {
	path := filepath.Join("..", "..", "guides", "guides.json")

	entries, err := LoadGuides(path)
	require.NoError(t, err)

	rewritten := filepath.Join(t.TempDir(), "guides.json")
	require.NoError(t, WriteGuides(rewritten, entries))

	want, err := os.ReadFile(path)
	require.NoError(t, err)

	got, err := os.ReadFile(rewritten)
	require.NoError(t, err)

	assert.Equal(t, string(want), string(got), "guides.json must round-trip through the generator's writer unchanged")
}
