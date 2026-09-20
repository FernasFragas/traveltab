package guides

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const lisbonSourceText = `Lisbon is the capital of Portugal. The historic Alfama district sits ` +
	`below São Jorge Castle, and the nightlife of Bairro Alto draws visitors after dark. Trams ` +
	`climb the hills between them.`

func TestValidateIntro_AcceptsPlacesFoundInTheSourceText(t *testing.T) {
	intro := "Lisbon rewards wandering. Alfama's narrow lanes lead up to São Jorge Castle, and " +
		"Bairro Alto comes alive after dark."

	err := ValidateIntro(intro, lisbonSourceText)
	assert.NoError(t, err)
}

func TestValidateIntro_RejectsAnInventedPlace(t *testing.T) {
	intro := "Lisbon rewards wandering. Don't miss the famous Xanadu Gardens on the waterfront."

	err := ValidateIntro(intro, lisbonSourceText)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Xanadu Gardens")
}

func TestValidateIntro_IgnoresCommonSentenceStarters(t *testing.T) {
	intro := "The city is easy to explore. There are trams everywhere. Also, Alfama is lovely."

	err := ValidateIntro(intro, lisbonSourceText)
	assert.NoError(t, err)
}

// Real generated text routinely opens a sentence with an ordinary capitalized word that has
// nothing to do with a place ("Situated on seven hills, ..."). A hand-maintained stopword list
// can never cover every such word, so ValidateIntro instead treats a lone capitalized word right
// after a sentence boundary as ordinary capitalization, not a place name, and only holds a
// multi-word capitalized phrase to the substring check even there.
func TestValidateIntro_IgnoresALoneCapitalizedWordStartingASentence(t *testing.T) {
	intro := "Situated on seven hills, Lisbon draws visitors year round. Overlooking the river, " +
		"the historic Alfama district rewards wandering."

	err := ValidateIntro(intro, lisbonSourceText)
	assert.NoError(t, err)
}

func TestValidateIntro_StillRejectsAMultiWordInventedPlaceAtASentenceStart(t *testing.T) {
	intro := "Lisbon rewards wandering. Xanadu Gardens is a must-see stop near the waterfront."

	err := ValidateIntro(intro, lisbonSourceText)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Xanadu Gardens")
}

// Real model output routinely glues an ordinary word straight onto a real place name —
// "Navigating Rome", "The Alfama district", "Though Barcelona" — because both words happen to be
// capitalized and adjacent. The properNoun pattern coalesces those into one two-word match, which
// then fails a literal substring check even though the place itself ("Rome", "Alfama",
// "Barcelona") is right there in the source. ValidateIntro strips a recognized generic leading
// word (a stopword, or a gerund ending "-ing") off a multi-word match before checking it, however
// many times in a row that applies.
func TestValidateIntro_StripsAGenericGerundPrefixFromAMultiWordPhrase(t *testing.T) {
	intro := "Navigating Lisbon is easy thanks to its trams."

	err := ValidateIntro(intro, lisbonSourceText)
	assert.NoError(t, err)
}

func TestValidateIntro_StripsTheArticleFromAMultiWordPhrase(t *testing.T) {
	intro := "The Alfama district is Lisbon's most historic corner."

	err := ValidateIntro(intro, lisbonSourceText)
	assert.NoError(t, err)
}

func TestValidateIntro_StripsADiscourseWordFromAMultiWordPhrase(t *testing.T) {
	intro := "Though Lisbon draws crowds, it never feels overrun."

	err := ValidateIntro(intro, lisbonSourceText)
	assert.NoError(t, err)
}

func TestValidateIntro_StillRejectsAnInventedPlaceAfterStrippingAGenericPrefix(t *testing.T) {
	intro := "Exploring Xanadu Gardens is the highlight of any trip to Lisbon."

	err := ValidateIntro(intro, lisbonSourceText)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Xanadu Gardens")
}
