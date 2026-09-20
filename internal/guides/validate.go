package guides

import (
	"fmt"
	"regexp"
	"strings"
)

// properNoun finds runs of capitalized words, e.g. "Alfama" or "Bairro Alto". This is a
// deliberately simple heuristic (per tasks/todo-v2.md: "don't over-engineer NLP") — it is the
// thing ValidateIntro checks against the source text, not a general-purpose place extractor.
var properNoun = regexp.MustCompile(`\b([A-Z][a-zA-Z'-]*(?:\s+[A-Z][a-zA-Z'-]*)*)\b`)

// commonSentenceStarters are capitalized words (and sentence-initial contractions) that show up
// at the start of sentences without naming a place. Skipping them avoids false positives like
// flagging "The" or "Don't" as an invented place.
var commonSentenceStarters = map[string]bool{
	"The": true, "This": true, "It": true, "Its": true, "A": true, "An": true, "In": true,
	"On": true, "At": true, "With": true, "From": true, "For": true, "And": true, "But": true,
	"Also": true, "Many": true, "Most": true, "Some": true, "There": true, "Here": true,
	"One": true, "Two": true, "Three": true, "You": true, "Your": true, "Whether": true,
	"While": true, "When": true, "If": true, "As": true, "Overall": true, "Visitors": true,
	"Travelers": true, "Travellers": true, "Beyond": true, "Despite": true,
	"Because": true, "Between": true, "Throughout": true, "Getting": true, "Around": true,
	"Whatever": true, "Both": true, "Each": true, "Every": true, "All": true,
	"Though": true, "Although": true, "Even": true, "Just": true, "Once": true, "Still": true,
	"Yet": true, "So": true, "Rather": true, "Instead": true, "Unlike": true, "Given": true,
	// Sentence-initial contractions: the properNoun pattern catches these the same way it would
	// a place name, but none of them is one.
	"Don't": true, "Doesn't": true, "Didn't": true, "Isn't": true, "Wasn't": true, "Aren't": true,
	"Weren't": true, "Won't": true, "Can't": true, "Couldn't": true, "Shouldn't": true,
	"Wouldn't": true, "Hasn't": true, "Haven't": true, "It's": true, "That's": true,
}

// ValidateIntro rejects an intro that names a place not found in the source Wikivoyage text.
// Every capitalized word or phrase in the intro (skipping common sentence-starting words) must
// appear as a substring of the source text; the first one that doesn't names the error.
func ValidateIntro(intro, sourceText string) error {
	for _, place := range extractPlaceNames(intro) {
		if !strings.Contains(sourceText, place) {
			return placeErr(place)
		}
	}

	return nil
}

// extractPlaceNames returns the candidate place names in text: capitalized words or phrases,
// minus common sentence starters, minus a lone capitalized word that merely opens a sentence,
// and minus any generic leading word glued onto a real name by capitalization alone.
//
// English capitalizes the first word of every sentence regardless of what it is, so a
// hand-maintained stopword list can never keep up with every ordinary word a model happens to
// put first ("Situated on seven hills, ..."). Worse, real model output routinely glues an
// ordinary word straight onto a real place name this way too — "Navigating Rome",
// "The Alfama district", "Though Barcelona" — which the properNoun pattern coalesces into one
// multi-word match that then fails a literal substring check even though the place itself is
// right there in the source. So a multi-word match has any leading stopword or gerund ("-ing")
// stripped off, however many times in a row that applies, before it's held to the check; what
// that leaves (if anything) is the real candidate.
func extractPlaceNames(text string) []string {
	var names []string

	for _, loc := range properNoun.FindAllStringIndex(text, -1) {
		match := text[loc[0]:loc[1]]
		startedAtSentenceBoundary := sentenceInitial(text, loc[0])

		words := stripGenericPrefix(strings.Fields(match))
		if len(words) == 0 {
			continue
		}

		name := strings.TrimSuffix(strings.Join(words, " "), "'s")
		if name == "" || commonSentenceStarters[name] {
			continue
		}

		// A single word, once any generic prefix is gone, still gets the sentence-initial pass:
		// there's no shorter phrase left to have proven it's a real name.
		if len(words) == 1 && startedAtSentenceBoundary {
			continue
		}

		names = append(names, name)
	}

	return names
}

// stripGenericPrefix drops leading words that are common stopwords or gerunds ("-ing"), leaving
// whatever (if anything) looks like it could be a real name.
func stripGenericPrefix(words []string) []string {
	for len(words) > 0 && isGenericWord(words[0]) {
		words = words[1:]
	}

	return words
}

// isGenericWord reports whether word is a recognized stopword or a gerund, and so never itself
// the start of an invented place name.
func isGenericWord(word string) bool {
	return commonSentenceStarters[word] || strings.HasSuffix(word, "ing")
}

// sentenceInitial reports whether position start in text is the very beginning of the text, or
// immediately follows a sentence-ending ".", "!" or "?" (skipping any whitespace in between).
func sentenceInitial(text string, start int) bool {
	before := strings.TrimRight(text[:start], " \t\n")
	if before == "" {
		return true
	}

	switch before[len(before)-1] {
	case '.', '!', '?':
		return true
	default:
		return false
	}
}

// placeErr formats ValidateIntro's rejection message.
func placeErr(place string) error {
	return fmt.Errorf("intro names %q, which does not appear in the source text", place)
}
