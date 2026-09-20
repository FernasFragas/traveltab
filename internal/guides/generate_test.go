package guides

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeFetcher answers FetchText from an in-memory map, keyed by title, and counts calls per
// title. No network is involved.
type fakeFetcher struct {
	pages map[string]WikivoyagePage
	calls map[string]int
}

func newFakeFetcher(pages map[string]WikivoyagePage) *fakeFetcher {
	return &fakeFetcher{pages: pages, calls: map[string]int{}}
}

func (f *fakeFetcher) FetchText(ctx context.Context, title string) (WikivoyagePage, error) {
	f.calls[title]++

	page, ok := f.pages[title]
	if !ok {
		return WikivoyagePage{}, errors.New("no fixture for " + title)
	}

	return page, nil
}

// fakeGenerator returns queued intros in order, one per call, and counts calls per city.
type fakeGenerator struct {
	intros []string
	next   int
	calls  int
	err    error
}

func (f *fakeGenerator) GenerateIntro(ctx context.Context, cityName, sourceText string) (string, error) {
	f.calls++

	if f.err != nil {
		return "", f.err
	}

	if f.next >= len(f.intros) {
		return f.intros[len(f.intros)-1], nil
	}

	intro := f.intros[f.next]
	f.next++

	return intro, nil
}

var fixedNow = func() time.Time { return time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC) }

func TestRun_WritesANewEntryWhenNoPriorRevisionExists(t *testing.T) {
	fetcher := newFakeFetcher(map[string]WikivoyagePage{
		"Lisbon": {Text: "Lisbon is a capital near Alfama.", Revision: 100},
	})
	generator := &fakeGenerator{intros: []string{"Lisbon is a capital near Alfama."}}

	guides, skipped := Run(context.Background(), []City{{"Lisbon", "pt"}}, nil, fetcher, generator, fixedNow)

	assert.Empty(t, skipped)
	require.Contains(t, guides, "lisbon-pt")
	assert.Equal(t, "Lisbon is a capital near Alfama.", guides["lisbon-pt"].Intro)
	assert.Equal(t, int64(100), guides["lisbon-pt"].WikivoyageRevision)
	assert.Equal(t, 1, generator.calls)
}

func TestRun_SkipsCityWhoseRevisionHasNotChanged(t *testing.T) {
	fetcher := newFakeFetcher(map[string]WikivoyagePage{
		"Lisbon": {Text: "Lisbon is a capital near Alfama.", Revision: 100},
	})
	generator := &fakeGenerator{intros: []string{"should not be used"}}

	existing := map[string]Entry{
		"lisbon-pt": {Intro: "old intro", WikivoyageRevision: 100, GeneratedAt: fixedNow()},
	}

	guides, skipped := Run(context.Background(), []City{{"Lisbon", "pt"}}, existing, fetcher, generator, fixedNow)

	assert.Empty(t, skipped)
	assert.Equal(t, "old intro", guides["lisbon-pt"].Intro)
	assert.Equal(t, 0, generator.calls, "the generator must not be asked when the revision is unchanged")
}

func TestRun_RegeneratesWhenRevisionChanged(t *testing.T) {
	fetcher := newFakeFetcher(map[string]WikivoyagePage{
		"Lisbon": {Text: "Lisbon is a capital near Alfama.", Revision: 200},
	})
	generator := &fakeGenerator{intros: []string{"Lisbon is a capital near Alfama, freshly rewritten."}}

	existing := map[string]Entry{
		"lisbon-pt": {Intro: "old intro", WikivoyageRevision: 100, GeneratedAt: fixedNow()},
	}

	guides, skipped := Run(context.Background(), []City{{"Lisbon", "pt"}}, existing, fetcher, generator, fixedNow)

	assert.Empty(t, skipped)
	assert.Equal(t, "Lisbon is a capital near Alfama, freshly rewritten.", guides["lisbon-pt"].Intro)
	assert.Equal(t, int64(200), guides["lisbon-pt"].WikivoyageRevision)
	assert.Equal(t, 1, generator.calls)
}

func TestRun_SkipsACityWhoseFetchFails(t *testing.T) {
	fetcher := newFakeFetcher(map[string]WikivoyagePage{})
	generator := &fakeGenerator{}

	guides, skipped := Run(context.Background(), []City{{"Nowhereville", "zz"}}, nil, fetcher, generator, fixedNow)

	assert.Empty(t, guides)
	require.Len(t, skipped, 1)
	assert.Equal(t, "nowhereville-zz", skipped[0].Slug)
}

func TestGenerateValidIntro_SucceedsOnRetryAfterARejectedAttempt(t *testing.T) {
	sourceText := "Lisbon is a capital near Alfama."
	generator := &fakeGenerator{intros: []string{
		"Don't miss the invented Xanadu Gardens.", // rejected: not in source
		"Lisbon rewards a wander through Alfama.", // accepted
	}}

	intro, ok := generateValidIntro(context.Background(), "lisbon-pt", "Lisbon", sourceText, generator)

	assert.True(t, ok)
	assert.Equal(t, "Lisbon rewards a wander through Alfama.", intro)
	assert.Equal(t, 2, generator.calls)
}

func TestForPrompt_LeavesShortTextUnchanged(t *testing.T) {
	assert.Equal(t, "Lisbon is a capital near Alfama.", forPrompt("Lisbon is a capital near Alfama."))
}

func TestForPrompt_TruncatesLongTextAtAWordBoundary(t *testing.T) {
	text := strings.Repeat("word ", promptCharLimit) // far longer than the limit

	truncated := forPrompt(text)

	assert.LessOrEqual(t, len(truncated), promptCharLimit)
	assert.False(t, strings.HasSuffix(truncated, "wor"), "must not cut a word in half")
}

func TestGenerateValidIntro_GivesUpAfterMaxAttempts(t *testing.T) {
	sourceText := "Lisbon is a capital near Alfama."
	generator := &fakeGenerator{intros: []string{"Don't miss the invented Xanadu Gardens."}}

	_, ok := generateValidIntro(context.Background(), "lisbon-pt", "Lisbon", sourceText, generator)

	assert.False(t, ok)
	assert.Equal(t, maxAttempts, generator.calls)
}
