package guides

import (
	"context"
	"log"
	"strings"
	"time"
)

// maxAttempts is how many times a city's intro may be regenerated after a validation rejection
// before the city is skipped.
const maxAttempts = 3

// promptCharLimit caps how much of a Wikivoyage article is sent to the model. The facts a
// ~120-word general intro needs (why go, what the areas are like, how to get around) sit near
// the top of these articles; staying well inside the model's context window keeps generation
// fast and grounded, rather than trailing off into district-by-district or "stay safe" listings
// a short intro has no room for anyway.
const promptCharLimit = 16000

// WikivoyageFetcher fetches a Wikivoyage article's plain text and revision. *WikivoyageAPI
// implements it; tests use a fake so no network is hit.
type WikivoyageFetcher interface {
	FetchText(ctx context.Context, title string) (WikivoyagePage, error)
}

// IntroGenerator asks a model for a city intro grounded in the given source text.
// *OllamaClient implements it; tests use a fake so no Ollama server is needed.
type IntroGenerator interface {
	GenerateIntro(ctx context.Context, cityName, sourceText string) (string, error)
}

// SkippedCity records why a city produced no guides.json entry this run.
type SkippedCity struct {
	Slug   string
	Reason string
}

// Run generates guides for every city in cities, starting from existing (a prior guides.json,
// or an empty map). It:
//   - skips regenerating any city whose stored revision already matches the current Wikivoyage
//     revision, keeping its existing entry as-is;
//   - retries a city's intro up to maxAttempts times when ValidateIntro rejects it, then skips
//     the city with a logged warning;
//   - otherwise writes a fresh Entry with the new intro, revision and generation time.
//
// It returns the full map to write (existing untouched entries included) and the cities it
// skipped this run, with why.
func Run(
	ctx context.Context,
	cities []City,
	existing map[string]Entry,
	fetcher WikivoyageFetcher,
	generator IntroGenerator,
	now func() time.Time,
) (map[string]Entry, []SkippedCity) {
	guides := make(map[string]Entry, len(existing))
	for slug, entry := range existing {
		guides[slug] = entry
	}

	var skipped []SkippedCity

	for _, city := range cities {
		slug := city.Slug()

		page, err := fetcher.FetchText(ctx, city.WikivoyageTitle())
		if err != nil {
			log.Printf("guides: %s: fetching wikivoyage text: %v", slug, err)

			skipped = append(skipped, SkippedCity{Slug: slug, Reason: err.Error()})

			continue
		}

		if prior, ok := existing[slug]; ok && prior.WikivoyageRevision == page.Revision {
			log.Printf("guides: %s: unchanged since revision %d, skipping", slug, page.Revision)

			continue
		}

		intro, ok := generateValidIntro(ctx, slug, city.Name, forPrompt(page.Text), generator)
		if !ok {
			skipped = append(skipped, SkippedCity{
				Slug:   slug,
				Reason: "intro kept naming places not in the source text after retries",
			})

			continue
		}

		guides[slug] = Entry{
			Intro:              intro,
			WikivoyageRevision: page.Revision,
			GeneratedAt:        now(),
		}
	}

	return guides, skipped
}

// generateValidIntro asks generator for an intro, retrying up to maxAttempts times whenever
// ValidateIntro rejects the result (or the generator itself errors). It logs each rejection so a
// human running the tool sees why a city needed a retry.
func generateValidIntro(ctx context.Context, slug, cityName, sourceText string, generator IntroGenerator) (string, bool) {
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		intro, err := generator.GenerateIntro(ctx, cityName, sourceText)
		if err != nil {
			log.Printf("guides: %s: attempt %d/%d: generating intro: %v", slug, attempt, maxAttempts, err)

			continue
		}

		if err := ValidateIntro(intro, sourceText); err != nil {
			log.Printf("guides: %s: attempt %d/%d: rejected: %v", slug, attempt, maxAttempts, err)

			continue
		}

		return intro, true
	}

	log.Printf("guides: %s: giving up after %d attempts", slug, maxAttempts)

	return "", false
}

// forPrompt trims text to at most promptCharLimit characters, breaking at the last whitespace
// inside the limit (when there is one) so a word isn't cut in half.
func forPrompt(text string) string {
	if len(text) <= promptCharLimit {
		return text
	}

	cut := text[:promptCharLimit]
	if i := strings.LastIndexAny(cut, " \n"); i > 0 {
		cut = cut[:i]
	}

	return cut
}
