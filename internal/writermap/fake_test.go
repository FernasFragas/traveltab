package writermap

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFakeSource_DestinationReturnsTheFixturePlaces(t *testing.T) {
	fixture := loadMadeiraFixture(t)
	source := &FakeSource{Result: fixture.DestinationResult}

	result, err := source.Destination(context.Background(), Area{Name: "Madeira"})

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
		result, err := source.Destination(context.Background(), Area{})
		require.NoError(t, err)
		assert.False(t, result.Complete)
	})
	t.Run("empty", func(t *testing.T) {
		source := &FakeSource{Result: fixture.DestinationResult, Empty: true}
		result, err := source.Destination(context.Background(), Area{})
		require.NoError(t, err)
		assert.Empty(t, result.Places)
		assert.True(t, result.Complete)
	})
	t.Run("unavailable", func(t *testing.T) {
		source := &FakeSource{Unavailable: true}
		_, err := source.Destination(context.Background(), Area{})
		assert.ErrorIs(t, err, ErrFakeUnavailable)
	})
	t.Run("slow and cancellable", func(t *testing.T) {
		source := &FakeSource{Delay: time.Second}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := source.Destination(ctx, Area{})
		assert.ErrorIs(t, err, context.Canceled)
	})
}

func TestFakeSource_DelayElapsesThenReturnsData(t *testing.T) {
	fixture := loadMadeiraFixture(t)
	source := &FakeSource{Result: fixture.DestinationResult, Delay: 10 * time.Millisecond}

	result, err := source.Destination(context.Background(), Area{})

	require.NoError(t, err)
	assert.Len(t, result.Places, 15)
}

func TestFakeSource_EditingAResultNameLeavesTheFakeUnchanged(t *testing.T) {
	fixture := loadMadeiraFixture(t)
	source := &FakeSource{Result: fixture.DestinationResult, Partial: true}
	result, err := source.Destination(context.Background(), Area{})
	require.NoError(t, err)

	result.Places[0].Names[0] = "changed"

	assert.Equal(t, "Madeira", source.Result.Places[0].Names[0])
}

func TestFakeSource_ReplacingAResultPlaceLeavesTheFakeUnchanged(t *testing.T) {
	fixture := loadMadeiraFixture(t)
	source := &FakeSource{Result: fixture.DestinationResult}
	result, err := source.Destination(context.Background(), Area{})
	require.NoError(t, err)

	result.Places[0] = Place{QID: "Q0"}

	assert.Equal(t, "Q799", source.Result.Places[0].QID)
}

func TestFakeSourcePlace_ReturnsThePlaceWithItsSummary(t *testing.T) {
	fixture := loadMadeiraFixture(t)
	source := &FakeSource{Result: fixture.DestinationResult, Summaries: fixture.Summaries}

	place, summary, err := source.Place(context.Background(), Area{Name: "Madeira"}, "Q206626")

	require.NoError(t, err)
	assert.Equal(t, "Pico Ruivo", place.Names[0])
	require.NotNil(t, summary)
	assert.Contains(t, summary.EnglishText, "high mountain hike")
}

func TestFakeSourcePlace_ReturnsNilSummaryWhenNoneExists(t *testing.T) {
	fixture := loadMadeiraFixture(t)
	source := &FakeSource{Result: fixture.DestinationResult, Summaries: fixture.Summaries}

	_, summary, err := source.Place(context.Background(), Area{Name: "Madeira"}, "Q1792734")

	require.NoError(t, err)
	assert.Nil(t, summary)
}

func TestFakeSourcePlace_UnknownQIDReturnsNotFound(t *testing.T) {
	fixture := loadMadeiraFixture(t)
	source := &FakeSource{Result: fixture.DestinationResult}

	place, summary, err := source.Place(context.Background(), Area{}, "Q0")

	assert.ErrorIs(t, err, ErrPlaceNotFound)
	assert.Nil(t, place)
	assert.Nil(t, summary)
}

func TestFakeSourcePlace_EmptyDestinationReturnsNotFound(t *testing.T) {
	fixture := loadMadeiraFixture(t)
	source := &FakeSource{Result: fixture.DestinationResult, Empty: true}

	place, _, err := source.Place(context.Background(), Area{}, "Q206626")

	assert.ErrorIs(t, err, ErrPlaceNotFound)
	assert.Nil(t, place)
}

func TestFakeSourcePlace_UnavailableReturnsError(t *testing.T) {
	fixture := loadMadeiraFixture(t)
	source := &FakeSource{Result: fixture.DestinationResult, Unavailable: true}

	place, _, err := source.Place(context.Background(), Area{}, "Q206626")

	assert.ErrorIs(t, err, ErrFakeUnavailable)
	assert.Nil(t, place)
}

func TestFakeSummarizer_ReturnsConfiguredText(t *testing.T) {
	summarizer := &FakeSummarizer{Text: "A fixture summary."}

	text, err := summarizer.Summarize(context.Background(), Place{QID: "Q799"}, []Passage{{Text: "transient"}})

	require.NoError(t, err)
	assert.Equal(t, "A fixture summary.", text)
	assert.Equal(t, 1, summarizer.Calls)
}

func TestFakeSummarizer_ReturnsConfiguredError(t *testing.T) {
	wantErr := errors.New("summary unavailable")
	summarizer := &FakeSummarizer{Err: wantErr}

	_, err := summarizer.Summarize(context.Background(), Place{}, nil)

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

	assert.Equal(t, "en", fixture.Mentions["Q206626"][0].Language)
	assert.Equal(t, "pt", fixture.Mentions["Q206626"][1].Language)
}

func TestMadeiraFixture_SomeWriterPlacesHaveNoSummary(t *testing.T) {
	fixture := loadMadeiraFixture(t)

	assert.NotEmpty(t, fixture.Summaries["Q206626"].EnglishText)
	assert.Empty(t, fixture.Summaries["Q1792734"].EnglishText)
}

func TestMadeiraFixture_IsIncomplete(t *testing.T) {
	fixture := loadMadeiraFixture(t)

	assert.False(t, fixture.Complete)
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
	DestinationResult
	Summaries map[string]Summary `json:"summaries"`
}
