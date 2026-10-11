package writermap

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractPost_NestedPhrasesAndRunePositions(t *testing.T) {
	result := ExtractPost("A guide", `<p>Visit <strong>Miradouro do Pico dos Barcelos</strong> &amp; São Vicente.</p>`)
	stream := "A guide\nVisit Miradouro do Pico dos Barcelos & São Vicente."

	assertPhrase(t, result.Phrases, "miradouro do pico dos barcelos", stream, "Miradouro do Pico dos Barcelos")
	assertPhrase(t, result.Phrases, "pico dos barcelos", stream, "Pico dos Barcelos")
	assertPhrase(t, result.Phrases, "sao vicente", stream, "São Vicente")
	assert.False(t, result.IsItinerary)
	assert.Empty(t, result.Days)
	encoded, err := json.Marshal(result)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "Visit")
	assert.NotContains(t, string(encoded), "São Vicente")
}

func TestExtractPost_EnglishDaySections(t *testing.T) {
	body, err := os.ReadFile("testdata/posts/en-itinerary.html")
	require.NoError(t, err)

	result := ExtractPost("Three days in Madeira", string(body))

	require.True(t, result.IsItinerary)
	require.Len(t, result.Days, 3)
	assert.Equal(t, []int{1, 2, 3}, []int{result.Days[0].Number, result.Days[1].Number, result.Days[2].Number})
	assert.Greater(t, result.Days[0].Start, utf8.RuneCountInString("Three days in Madeira\nStart in Funchal."))
	assert.Equal(t, result.Days[1].Start, result.Days[0].End)
	assert.Equal(t, result.Days[2].Start, result.Days[1].End)
	assert.Less(t, result.Days[2].Start, result.Days[2].End)
	assertPhraseWithinDay(t, result, "miradouro do pico dos barcelos", 0)
	assertPhraseWithinDay(t, result, "cabo girao", 1)
	assertPhraseWithinDay(t, result, "sao vicente", 2)
}

func TestExtractPost_PortugueseDaySections(t *testing.T) {
	body, err := os.ReadFile("testdata/posts/pt-itinerary.html")
	require.NoError(t, err)

	result := ExtractPost("Roteiro de dois dias", string(body))

	require.True(t, result.IsItinerary)
	require.Len(t, result.Days, 2)
	assert.Equal(t, []int{1, 2}, []int{result.Days[0].Number, result.Days[1].Number})
	assertPhraseWithinDay(t, result, "miradouro do pico dos barcelos", 0)
	assertPhraseWithinDay(t, result, "porto moniz", 1)
}

func TestExtractPost_ListicleIsNotItinerary(t *testing.T) {
	body, err := os.ReadFile("testdata/posts/listicle.html")
	require.NoError(t, err)

	result := ExtractPost("Top two sights in Madeira", string(body))

	assert.False(t, result.IsItinerary)
	assert.Empty(t, result.Days)
	assert.NotEmpty(t, result.Phrases)
}

func TestExtractPost_IgnoresMarkupAndHiddenCode(t *testing.T) {
	result := ExtractPost("", `<p>Cabo <em>Girão</em></p><script>Fake Place</script><style>.Fake Place{}</style>`)

	assert.Contains(t, foldedPhrases(result.Phrases), "cabo girao")
	assert.NotContains(t, foldedPhrases(result.Phrases), "fake place")
}

func TestExtractPost_LowercaseStartWithConnectorKeepsCaseFlag(t *testing.T) {
	result := ExtractPost("", `<p>Explore pico dos Barcelos today.</p>`)

	for _, phrase := range result.Phrases {
		if phrase.Folded == "pico dos barcelos" {
			assert.False(t, phrase.OriginalCase)
			assert.Equal(t, utf8.RuneCountInString("Explore "), phrase.Start)
			assert.Equal(t, utf8.RuneCountInString("Explore pico dos Barcelos"), phrase.End)
			return
		}
	}
	t.Fatal("missing lowercase-start phrase")
}

func TestExtractPost_SubspansHaveFiveWordCap(t *testing.T) {
	result := ExtractPost("", `<p>Miradouro do Pico dos Barcelos de Funchal</p>`)

	assert.Contains(t, foldedPhrases(result.Phrases), "miradouro do pico dos barcelos de funchal")
	assert.Contains(t, foldedPhrases(result.Phrases), "pico dos barcelos de funchal")
	for _, phrase := range result.Phrases {
		if phrase.Folded != "miradouro do pico dos barcelos de funchal" {
			assert.LessOrEqual(t, len(strings.Fields(phrase.Folded)), 5)
		}
	}
}

func assertPhrase(t *testing.T, phrases []NamePhrase, folded, stream, original string) {
	t.Helper()
	byteStart := strings.Index(stream, original)
	require.NotEqual(t, -1, byteStart)
	start := utf8.RuneCountInString(stream[:byteStart])
	end := start + utf8.RuneCountInString(original)
	for _, phrase := range phrases {
		if phrase.Folded == folded && phrase.Start == start && phrase.End == end {
			assert.True(t, phrase.OriginalCase)
			return
		}
	}
	t.Errorf("missing %q at rune span [%d,%d) in %+v", folded, start, end, phrases)
}

func assertPhraseWithinDay(t *testing.T, result ExtractedPost, folded string, dayIndex int) {
	t.Helper()
	day := result.Days[dayIndex]
	for _, phrase := range result.Phrases {
		if phrase.Folded == folded && phrase.Start >= day.Start && phrase.End <= day.End {
			return
		}
	}
	t.Errorf("missing %q within day %d", folded, day.Number)
}

func foldedPhrases(phrases []NamePhrase) []string {
	result := make([]string, len(phrases))
	for i, phrase := range phrases {
		result[i] = phrase.Folded
	}
	return result
}
