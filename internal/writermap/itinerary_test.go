package writermap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssembleItineraries_DaysFollowFirstMention(t *testing.T) {
	post := IndexedPost{Host: "writer.test", PostID: 4, URL: "https://writer.test/4", Title: "Madeira in two days", Language: "en",
		Extracted: ExtractedPost{IsItinerary: true, Days: []DaySection{{Number: 1, Start: 10, End: 100}, {Number: 2, Start: 100, End: 200}}}}
	mentions := map[string][]Mention{
		"Q1": {{WriterHost: post.Host, PostRef: "writer.test:4", Positions: []Position{{Start: 50, End: 55}, {Start: 130, End: 135}}}},
		"Q2": {{WriterHost: post.Host, PostRef: "writer.test:4", Positions: []Position{{Start: 30, End: 35}}}},
		"Q3": {{WriterHost: post.Host, PostRef: "writer.test:4", Positions: []Position{{Start: 120, End: 125}}}},
	}

	got := AssembleItineraries([]IndexedPost{post}, mentions)

	require.Len(t, got, 1)
	assert.Equal(t, []ItineraryDay{{QIDs: []string{"Q2", "Q1"}}, {QIDs: []string{"Q3", "Q1"}}}, got[0].Days)
	assert.Equal(t, post.URL, got[0].PostURL)
}

func TestAssembleItineraries_IgnoresNonItineraryAndUnmatchedDays(t *testing.T) {
	posts := []IndexedPost{
		{Host: "a.test", PostID: 1, Extracted: ExtractedPost{IsItinerary: false, Days: []DaySection{{Number: 1, Start: 0, End: 10}}}},
		{Host: "b.test", PostID: 2, Extracted: ExtractedPost{IsItinerary: true, Days: []DaySection{{Number: 1, Start: 0, End: 10}}}},
	}
	assert.Empty(t, AssembleItineraries(posts, nil))
}

func TestAssembleItineraries_FirstMentionCanStartAtZero(t *testing.T) {
	post := IndexedPost{Host: "writer.test", PostID: 5, Extracted: ExtractedPost{IsItinerary: true, Days: []DaySection{{Number: 1, Start: 0, End: 100}}}}
	mentions := map[string][]Mention{
		"Q1": {{PostRef: "writer.test:5", Positions: []Position{{Start: 0, End: 5}, {Start: 50, End: 55}}}},
		"Q2": {{PostRef: "writer.test:5", Positions: []Position{{Start: 20, End: 25}}}},
	}

	got := AssembleItineraries([]IndexedPost{post}, mentions)

	require.Len(t, got, 1)
	assert.Equal(t, []string{"Q1", "Q2"}, got[0].Days[0].QIDs)
}
