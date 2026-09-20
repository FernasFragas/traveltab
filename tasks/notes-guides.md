# Notes: Lane B — AI city intros (`cmd/guides`)

Evidence for `tasks/todo-v2.md`'s "Lane B: AI city intros" task. Read `tasks/plan-v2.md`'s
Decisions section first for why this lane generates but does not wire anything in.

## Environment check

- `curl -s http://localhost:11434/api/tags` and `ollama list` both reached a local Ollama
  server with `mistral:7b` installed (4.4 GB, Q4_K_M). The live pipeline ran against the real
  Wikivoyage API and the real local Ollama server — nothing here is fabricated.
- `en.wikivoyage.org/w/api.php` with `action=query&prop=extracts|revisions&explaintext=1&rvprop=ids&redirects=1`
  answers with plain text and a revision ID, confirmed by hand with `curl` before writing any
  code. `redirects=1` matters: "Marrakesh" redirects to the article titled "Marrakech", and "New
  York" alone is a short disambiguation-style stub — the real article is "New York City" — so
  `internal/guides/city.go` carries an explicit Wikivoyage-title override for that one city.

## Package layout

- `internal/guides/`: all the logic, mirroring how `internal/placetypes` backs `cmd/placetypes`.
  - `wikivoyage.go` / `wikivoyage_test.go` — `WikivoyageAPI.FetchText`, stubbed with the same
    `roundTripFunc` fake-transport pattern as `internal/adapters/api/openweather_test.go`.
  - `ollama.go` / `ollama_test.go` — `OllamaClient.GenerateIntro`, POSTs to
    `http://localhost:11434/api/generate` with `stream:false`, stubbed the same way.
  - `validate.go` / `validate_test.go` — `ValidateIntro`, the invented-place check.
  - `store.go` / `store_test.go` — `Entry`, `LoadGuides`, `WriteGuides`.
  - `generate.go` / `generate_test.go` — `Run` (the orchestration: fetch, skip-if-unchanged,
    retry-then-skip, write) and `generateValidIntro`, using `fakeFetcher`/`fakeGenerator` so no
    network or Ollama call happens in the normal test suite.
  - `city.go` / `city_test.go` — the 20 starter cities (same list, same order as
    `internal/placetypes/generator.go`'s `cities`), each with the two-letter country code the
    slug needs.
  - `slug.go` / `slug_test.go` — `Slug(city, country) string`.
- `cmd/guides/main.go` — thin flag-parsing wrapper, mirroring `cmd/placetypes/main.go`. Flags:
  `-out` (default `guides/guides.json`), `-ollama`, `-model`, `-only` (comma-separated city names,
  used for the one-city smoke test below), `-timeout`.
- `cmd/console` was removed and replaced by `cmd/guides`, per the todo's explicit preference
  ("turn the empty `cmd/console` into `cmd/guides`"). Nothing in the module imported
  `cmd/console`; `README.md` mentions it as a placeholder (`cmd/console`, "Console client... is
  an empty placeholder") but that's documentation outside this lane's ownership, not code — worth
  a follow-up edit when a human reviews this.

## Slug duplication (as instructed)

Lane A landed `internal/adapters/httpserver/slug.go` and `internal/adapters/httpserver/trip.go`
etc. *during* this run (visible in `git status` partway through). Per the task's explicit
instruction, `internal/guides/slug.go` does **not** import `internal/adapters/httpserver` — a
`cmd/`-adjacent package importing the HTTP adapter package would cross the layer boundary
`docs/architecture.md` sets up. `internal/guides.Slug` independently implements the same rule
(lower-case ASCII, spaces to hyphens, `city-country`) and is covered by its own tests
(`TestSlug_BuildsCityDashCountry`, `TestSlug_LowercasesAndHyphenatesSpaces`,
`TestSlug_MultiWordCityWithUppercaseCountry`). **This is duplicated logic**, flagged here for a
later cleanup: once both lanes are stable, hoisting `Slug`/`ParseSlug` into a small
dependency-free package both sides import would remove the duplication without violating the
layering (neither side would depend on the other's adapter package).

## TDD: RED then GREEN

Every function was written with a zero-value stub first (`return nil`, `return "", nil`, etc.,
per this repo's house TDD style in `tasks/todo.md`), the named tests were run against the stub,
and every test that should fail did fail **on an assertion**, not a compile error. Full command:

```
CGO_ENABLED=1 go test -race -count=1 -v ./internal/guides/...
```

RED run (18 failing on assertions, 6 passing trivially — see below):

```
--- FAIL: TestCity_Slug_MatchesCityDashCountry
--- FAIL: TestRun_SkipsCityWhoseRevisionHasNotChanged
--- FAIL: TestRun_SkipsACityWhoseFetchFails
--- FAIL: TestGenerateValidIntro_SucceedsOnRetryAfterARejectedAttempt (assert.True(t, ok) failed; intro "")
--- FAIL: TestGenerateValidIntro_GivesUpAfterMaxAttempts (expected 3 calls, got 0)
--- FAIL: TestSlug_BuildsCityDashCountry (expected "lisbon-pt", got "")
--- FAIL: TestSlug_LowercasesAndHyphenatesSpaces (expected "mexico-city-mx", got "")
--- FAIL: TestSlug_MultiWordCityWithUppercaseCountry (expected "new-york-us", got "")
--- FAIL: TestWriteGuides_KeysBySlugAndRecordsRevision (map does not contain "lisbon-pt")
--- FAIL: TestValidateIntro_RejectsAnInventedPlace (expected an error, got nil)
--- FAIL: TestOllamaClient_GenerateIntro_PostsThePromptAndModel (0 requests recorded)
--- FAIL: TestOllamaClient_GenerateIntro_ReturnsErrorOnHTTPFailure (expected an error, got nil)
--- FAIL: TestOllamaClient_GenerateIntro_ReturnsErrorOnEmptyResponse (expected an error, got nil)
--- FAIL: TestFetchWikivoyageText_ReturnsPlainTextAndRevision (0 requests recorded)
--- FAIL: TestFetchWikivoyageText_ReturnsErrorForAMissingPage (expected an error, got nil)
--- FAIL: TestFetchWikivoyageText_ReturnsErrorOnHTTPFailure (expected an error, got nil)
--- FAIL: TestRun_WritesANewEntryWhenNoPriorRevisionExists
--- FAIL: TestRun_RegeneratesWhenRevisionChanged
```

(`TestCities_...`, `TestCity_WikivoyageTitle_...` x2, `TestValidateIntro_AcceptsPlacesFoundInTheSourceText`,
`TestValidateIntro_IgnoresCommonSentenceStarters`, `TestLoadGuides_ReturnsEmptyMapWhenFileIsMissing`
passed immediately: the first three test plain data/lookup logic that was already implemented
outside the TDD loop — the city list and its Wikivoyage-title override — and the other three are
positive-path assertions against a stub that trivially returns "no error"/"empty map", which is
expected and non-diagnostic; the corresponding *negative*-path tests above are the real RED
evidence for those functions.)

After implementing each function for real (GREEN), the full run was clean:

```
$ CGO_ENABLED=1 go test -race -count=1 ./internal/guides/...
ok  	weatherservice/internal/guides	1.469s
```

### A refactor found by the live run, done properly (RED again, then fixed)

While validating the real Lisbon output (see below), the validator rejected the model's first two
attempts for a reason that had nothing to do with an invented place: it flagged ordinary
capitalized words that merely *open a sentence* ("Situated on seven hills, ..."), because English
capitalizes a sentence's first word regardless of what it is, and a hand-maintained stopword list
can never keep up with every word a model might pick. Wrote two new tests first —
`TestValidateIntro_IgnoresALoneCapitalizedWordStartingASentence` (failed on the "Situated"
assertion, confirmed RED) and `TestValidateIntro_StillRejectsAMultiWordInventedPlaceAtASentenceStart`
(already passed) — then changed `extractPlaceNames` to skip a *lone* capitalized word right after
a sentence boundary, while still holding a *multi-word* capitalized run there to the substring
check (ordinary prose essentially never produces two capitalized words back to back by grammar
alone, so a multi-word run at a sentence's start is still almost certainly a real name).
Re-ran: all green, `golangci-lint run` still 0 issues.

Also added `forPrompt` (`internal/guides/generate.go`) to cap what's sent to the model at 8000
characters, breaking on a word boundary: several of the 20 articles (Paris, London, Berlin, New
York City, Mexico City) run 130,000–190,000 characters, far more than a short intro needs and
more than is sensible to hand a CPU-bound 7B model each of 20 times. Covered by
`TestForPrompt_LeavesShortTextUnchanged` and `TestForPrompt_TruncatesLongTextAtAWordBoundary`.

## Live pipeline run

Ran a single city first, as instructed, before the full 20:

```
$ go run ./cmd/guides -only=Lisbon -out=guides/guides.json
guides: wrote guides/guides.json (1 cities, 0 skipped)
```

Generated intro (`guides/guides.json`, `lisbon-pt`, Wikivoyage revision 5364237):

> Lisbon, Portugal's vibrant capital, nestled at the Tagus river's mouth, offers an alluring blend
> of history and contemporary culture. With its distinctive white-bleached limestone buildings,
> intimate alleyways, and relaxed ambiance, the city enchants visitors year-round. Lisbon's unique
> charm lies in its contrasts—from elegant squares and broad avenues to hilly, narrow, winding
> streets, and from upscale hotels to unassuming eateries hidden in Bairro Alto's modest streets.
> The city is built on seven hills, making exploration a workout, but the iconic trams, efficient
> metro network, and excellent public transportation links make navigating a breeze. The city's
> rich heritage includes the UNESCO World Heritage Site of Sintra, seaside resorts, world-class
> museums, and the hilltop Cristo Rei statue in Almada. Despite its size, Lisbon's pace is less
> frantic than other million-city destinations, creating a welcoming atmosphere for travelers.

Read by hand against the Wikivoyage source: every named place (Bairro Alto, Sintra, Almada,
Cristo Rei) is in the source text; nothing is invented; ~135 words, single paragraph, no
headings — close enough to the ~120-word target and good enough to hand to a human reviewer.
This cleared the bar, so the full 20-city run followed (see the end of this file for its results,
filled in once it finished).

### Full 20-city run — completed 2026-09-20 (coordinating session, after a usage-limit cutoff)

The agent that ran this got cut off by a session limit partway through a re-run of the full
batch (it had decided to regenerate all 20 fresh rather than rely on revision-skipping, to get
consistent quality — reasonable, but not necessary). What it left on disk: **17 of 20 cities
already generated and passing validation** in `guides/guides.json` (amsterdam-nl, barcelona-es,
berlin-de, florence-it, funchal-pt, istanbul-tr, kyoto-jp, lisbon-pt, marrakesh-ma, mexico-city-mx,
new-york-us, paris-fr, porto-pt, prague-cz, rome-it, tavira-pt, tokyo-jp). Missing: London,
Seville, Vienna.

Rather than re-run all 20 again, the coordinating session ran only the missing three with
`-only="London,Seville,Vienna"`, since the tool already had 17 good entries on disk and
re-running them would just re-spend Ollama time for no reason:

```
$ go run ./cmd/guides -only="London,Seville,Vienna" -out=guides/guides.json
guides: seville-es: attempt 1/3: rejected: intro names "Seville Airport", which does not appear in the source text
guides: wrote guides/guides.json (20 cities, 0 skipped)
```

The Seville rejection is the validator working exactly as designed: the model's first attempt
named an airport that Seville's Wikivoyage article never mentions, and the retry produced a clean
intro with no invented places. All **20 of 20 starter cities now have an entry**.

Read by hand (the same bar the single-city Lisbon check used: every named place present in the
source text, ~100–150 words, no headings):

- **London**: mentions the Thames, Greater London's 32 boroughs, the City of London, the
  Underground ("the Tube") — all present in the source. No invented places.
- **Seville**: mentions the Guadalquivir river and the old town — present in the source (after
  the retry). No invented places.
- **Vienna**: mentions the 23 districts and the historic first district — present in the source.
  No invented places.

All cleared review. `CGO_ENABLED=1 go test -race -count=1 ./internal/guides/...` and
`golangci-lint run ./...` (whole repo) still pass after the run. `guides/guides.json` is the
complete artifact for human review; nothing in `internal/` or `views/` references it — confirmed
with `grep -rn "guides" internal/ views/ cmd/web/`, which finds nothing.

**Lane B is done.** Wiring `guides/guides.json` into the page is a follow-up task, by design —
see the decision in `tasks/plan-v2.md`.
