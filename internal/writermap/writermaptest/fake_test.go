package writermaptest

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"weatherservice/internal/writermap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFakeSource_DestinationReturnsTheFixturePlaces(t *testing.T) {
	fixture := loadMadeiraFixture(t)
	source := &FakeSource{Result: fixture.DestinationResult}

	result, err := source.Destination(context.Background(), writermap.Area{Name: "Madeira"})

	require.NoError(t, err)
	assert.Len(t, result.Places, 15)
	assert.False(t, result.Complete)
}

func TestFakeSourceSwitches(t *testing.T) {
	fixture := loadMadeiraFixture(t)
	t.Run("partial", func(t *testing.T) {
		complete := fixture.DestinationResult
		complete.Complete = true
		source := &FakeSource{Result: complete, Partial: true}
		result, err := source.Destination(context.Background(), writermap.Area{})
		require.NoError(t, err)
		assert.False(t, result.Complete)
	})
	t.Run("empty", func(t *testing.T) {
		source := &FakeSource{Result: fixture.DestinationResult, Empty: true}
		result, err := source.Destination(context.Background(), writermap.Area{})
		require.NoError(t, err)
		assert.Empty(t, result.Places)
		assert.True(t, result.Complete)
	})
	t.Run("unavailable", func(t *testing.T) {
		source := &FakeSource{Unavailable: true}
		_, err := source.Destination(context.Background(), writermap.Area{})
		assert.ErrorIs(t, err, ErrFakeUnavailable)
	})
	t.Run("slow and cancellable", func(t *testing.T) {
		source := &FakeSource{Delay: time.Second}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := source.Destination(ctx, writermap.Area{})
		assert.ErrorIs(t, err, context.Canceled)
	})
}

func TestFakeSource_EmptyDestinationKeepsSourceStatuses(t *testing.T) {
	fixture := loadMadeiraFixture(t)
	source := &FakeSource{Result: fixture.DestinationResult, Empty: true}

	result, err := source.Destination(context.Background(), writermap.Area{})

	require.NoError(t, err)
	assert.Equal(t, fixture.SourceStatuses, result.SourceStatuses)
}

func TestFakeSource_DelayElapsesThenReturnsData(t *testing.T) {
	fixture := loadMadeiraFixture(t)
	source := &FakeSource{Result: fixture.DestinationResult, Delay: 10 * time.Millisecond}

	result, err := source.Destination(context.Background(), writermap.Area{})

	require.NoError(t, err)
	assert.Len(t, result.Places, 15)
}

func TestFakeSource_EditingAResultNameLeavesTheFakeUnchanged(t *testing.T) {
	fixture := loadMadeiraFixture(t)
	source := &FakeSource{Result: fixture.DestinationResult, Partial: true}
	result, err := source.Destination(context.Background(), writermap.Area{})
	require.NoError(t, err)

	result.Places[0].Names[0] = "changed"

	assert.Equal(t, "Madeira", source.Result.Places[0].Names[0])
}

func TestFakeSource_ReplacingAResultPlaceLeavesTheFakeUnchanged(t *testing.T) {
	fixture := loadMadeiraFixture(t)
	source := &FakeSource{Result: fixture.DestinationResult}
	result, err := source.Destination(context.Background(), writermap.Area{})
	require.NoError(t, err)

	result.Places[0] = writermap.Place{QID: "Q0"}

	assert.Equal(t, "Q30188", source.Result.Places[0].QID)
}

func TestFakeSourcePlace_ReturnsThePlaceWithItsSummary(t *testing.T) {
	fixture := loadMadeiraFixture(t)
	source := &FakeSource{Result: fixture.DestinationResult, Summaries: fixture.Summaries}

	place, summary, err := source.Place(context.Background(), writermap.Area{Name: "Madeira"}, "Q473169")

	require.NoError(t, err)
	assert.Equal(t, "Pico Ruivo", place.Names[0])
	require.NotNil(t, summary)
	assert.Contains(t, summary.EnglishText, "high mountain hike")
}

func TestFakeSourcePlace_ReturnsNilSummaryWhenNoneExists(t *testing.T) {
	fixture := loadMadeiraFixture(t)
	source := &FakeSource{Result: fixture.DestinationResult, Summaries: fixture.Summaries}

	_, summary, err := source.Place(context.Background(), writermap.Area{Name: "Madeira"}, "Q34799706")

	require.NoError(t, err)
	assert.Nil(t, summary)
}

func TestFakeSourcePlace_UnknownQIDReturnsNotFound(t *testing.T) {
	fixture := loadMadeiraFixture(t)
	source := &FakeSource{Result: fixture.DestinationResult}

	place, summary, err := source.Place(context.Background(), writermap.Area{}, "Q0")

	assert.ErrorIs(t, err, writermap.ErrPlaceNotFound)
	assert.Nil(t, place)
	assert.Nil(t, summary)
}

func TestFakeSourcePlace_EmptyDestinationReturnsNotFound(t *testing.T) {
	fixture := loadMadeiraFixture(t)
	source := &FakeSource{Result: fixture.DestinationResult, Empty: true}

	place, _, err := source.Place(context.Background(), writermap.Area{}, "Q473169")

	assert.ErrorIs(t, err, writermap.ErrPlaceNotFound)
	assert.Nil(t, place)
}

func TestFakeSourcePlace_UnavailableReturnsError(t *testing.T) {
	fixture := loadMadeiraFixture(t)
	source := &FakeSource{Result: fixture.DestinationResult, Unavailable: true}

	place, _, err := source.Place(context.Background(), writermap.Area{}, "Q473169")

	assert.ErrorIs(t, err, ErrFakeUnavailable)
	assert.Nil(t, place)
}

func TestFakeSummarizer_ReturnsConfiguredText(t *testing.T) {
	summarizer := &FakeSummarizer{Text: "A fixture summary."}

	text, err := summarizer.Summarize(context.Background(), writermap.Place{QID: "Q30188"}, []writermap.Passage{{Text: "transient"}})

	require.NoError(t, err)
	assert.Equal(t, "A fixture summary.", text)
	assert.Equal(t, 1, summarizer.Calls)
}

func TestFakeSummarizer_ReturnsConfiguredError(t *testing.T) {
	wantErr := errors.New("summary unavailable")
	summarizer := &FakeSummarizer{Err: wantErr}

	_, err := summarizer.Summarize(context.Background(), writermap.Place{}, nil)

	assert.ErrorIs(t, err, wantErr)
	assert.Equal(t, 1, summarizer.Calls)
}

func TestMadeiraFixture_HasFifteenPlacesAndTwoItineraries(t *testing.T) {
	fixture := loadMadeiraFixture(t)

	assert.Len(t, fixture.Places, 15)
	assert.Len(t, fixture.Itineraries, 2)
}

func TestMadeiraFixture_PlaceHasAliases(t *testing.T) {
	fixture := loadMadeiraFixture(t)

	assert.Contains(t, fixture.Places[2].Names, "Pico Ruivo")
	assert.Contains(t, fixture.Places[2].Names, "Pico Ruivo de Santana")
}

func TestMadeiraFixture_PlaceIsMentionedInEnglishAndPortuguese(t *testing.T) {
	fixture := loadMadeiraFixture(t)

	assert.Equal(t, "en", fixture.Mentions["Q473169"][0].Language)
	assert.Equal(t, "pt", fixture.Mentions["Q473169"][1].Language)
}

func TestMadeiraFixture_SomeWriterPlacesHaveNoSummary(t *testing.T) {
	fixture := loadMadeiraFixture(t)

	assert.NotEmpty(t, fixture.Summaries["Q473169"].EnglishText)
	assert.Empty(t, fixture.Summaries["Q34799706"].EnglishText)
}

func TestMadeiraFixture_IsIncomplete(t *testing.T) {
	fixture := loadMadeiraFixture(t)

	assert.False(t, fixture.Complete)
}

func TestMadeiraFixture_SourceStatusesUseContractValues(t *testing.T) {
	fixture := loadMadeiraFixture(t)

	for _, status := range fixture.SourceStatuses {
		assert.Contains(t, []string{"active", "partial", "blocked", "removed"}, status.Status, status.Host)
	}
}

func loadMadeiraFixture(t *testing.T) madeiraFixture {
	t.Helper()
	data, err := os.ReadFile("testdata/madeira.json")
	require.NoError(t, err)
	var fixture madeiraFixture
	require.NoError(t, json.Unmarshal(data, &fixture))
	return fixture
}

type madeiraFixture struct {
	writermap.DestinationResult
	Summaries map[string]writermap.Summary `json:"summaries"`
}
