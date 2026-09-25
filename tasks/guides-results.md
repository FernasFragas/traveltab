# City guides: results

Run 2026-09-24. The per-city review is in [guides-review.md](guides-review.md).

## Outcome

**Reviewed city intros now show on the destination page, with Wikivoyage attribution. Every other city gets an explicit "no reviewed guide yet" card. Serving a request never calls Ollama or the network.**

| Item | Result |
| --- | --- |
| Starter cities reviewed and published | 20 of 20 |
| Rewritten from the generated draft | 20 of 20 |
| Left unreviewed or dropped | 0 |
| Draft statements corrected / removed | 26 (in 17 cities) / 56 |

## What was built

- **Data.** `guides/guides.json` entries gained optional `wikivoyage_title`, `reviewed`, `reviewed_at` and `review_note`. Old files still load; an unreviewed entry serialises exactly as before. The generator records the title and never sets `reviewed`, so a regenerated entry is hidden until reviewed (kept unchanged if its revision is unchanged).
- **Read side.** `guides.Book` keeps only publishable entries (reviewed, intro, revision, valid title, `YYYY-MM-DD` date) and looks them up in memory by the shared `internal/slug` key, so city and country must both match. `application.CityGuideSource` is the contract; `guides/embed.go` (package `guidedata`) compiles the file into the binary; `cmd/web` wires it with `Server.SetCityGuideSource`.
- **Presentation.** `destinationPresentation` (now a `Server` method, the one builder for the initial page, HTMX fragment and shared trip page) sets `.Presentation.Guide`. `views/guide_card.go.tpl` renders it inside `section#overview` after the map/planner grid; `public/redesign/guide.css` (`.guide-*`, registered last in `index.go.tpl`) styles it. Text is escaped, no `template.HTML`. Section IDs and the `/plan` out-of-band contract are unchanged. A guide lacking a title or revision is treated as missing.
- **Two states:** reviewed (intro, "Reviewed" badge, article, revision permalink, authors, CC BY-SA 4.0, "Summarised and reworded", review date) and missing (plain sentence plus a Wikivoyage search link). The credits footer also names Wikivoyage.
- **Preview and checks.** Fixtures: reviewed Lisbon, Porto, Paris; namesake Paris, Texas and Coimbra without a guide; awkward `Longguide`; the `fallback` scenario has no guides. New `tasks/redesign/checks/guides.mjs`; `http-smoke.mjs` and `interaction-regressions.mjs` assert guide states.
- **Docker.** No Dockerfile change was needed: `COPY . .` puts `guides/` in the build (it is not in `.dockerignore`) so `go:embed` compiles it in, and `views/` and `public/` are already copied.

## Verification (all run 2026-09-24)

| Check | Outcome |
| --- | --- |
| `go test -race -count=1 ./...` | Passed, all packages |
| `go test -race -count=5` on `internal/adapters/httpserver` and `internal/guides` | Passed |
| `make lint` | 0 issues |
| `make build` | Passed |
| `git diff --check` | Clean (tracked files); new untracked files checked for trailing whitespace by hand, none |
| `docker build -t traveltab .` | Built. The image contains `/app/public/redesign/guide.css` (with the 900px rule), `/app/views/guide_card.go.tpl`, the `guide.css` link in `index.go.tpl`, and the binary holds the embedded guides (`wikivoyage_title` present) |
| `node --check` | Passed for all `tasks/redesign/checks/*.mjs` and `public/redesign/*.js` |
| `http-smoke.mjs` (default preview) | Passed, including guide states |
| `interaction-regressions.mjs` | Passed, including Porto/Coimbra guide swaps |
| `destination-photos.mjs` (output to a scratch folder) | Passed |
| `guides.mjs` at 375 px and 1440 px, plus fallback scenario | Passed, no JavaScript exceptions; screenshots and `browser-results.json` in [tasks/artifacts/guides/](artifacts/guides/) |
| Guide text contrast (both states) | 5.81:1 for the lowest colour, against a 4.5:1 requirement |

The browser runs used Brave headless with a temporary profile and its own debug port (9341), not the user's browser; the preview servers and browser were stopped afterwards. `browser-smoke.mjs` was not re-run: it overwrites the historical September evidence and this work did not change it.

New Go tests cover the book (publishability, city-plus-country matching, unreviewed entries never published, nil book), the shipped file (reviewed fields, keys are starter-city slugs, embedded book equals the file, round-trips through the generator's writer byte for byte), generator title and review-reset behaviour, the HTTP states on all three routes, namesakes, escaping, attribution required, planner response without a guide, and the preview fixtures.

## Unrelated fix

`TestDestinationPhoto_NamesakeCacheEntriesNeverDetermineTheDestination` (from the photo work) failed in about three of four full-package runs, with and without the guide changes: subtests shared one city key, so a background cache save turned the next subtest's cache miss into a hit. Each subtest now has its own key; six consecutive package runs and a `-race -count=5` run passed.

## Unresolved

- **The review was done by an AI.** A person should skim the 20 intros before a wider launch; only hard figures had a second source (Wikipedia).
- **Local spellings miss.** A guide needs the resolved name and country to match, so a search resolving to "Lisboa" shows the missing state. I could not check what OpenWeather returns for each starter city, because that needs the API key in `.env`, which I left alone.
- **`cmd/web` wiring has no test.** It compiles, and the identical wiring is exercised through the design preview, but a real search against live providers was not run.
- **`cmd/loadtest-server` has no guide source**, so load tests render the missing card; its recorded results were not re-run.
- Five Wikivoyage pages had newer revisions by the review date. The intros are pinned to the reviewed revision (linked as a permalink); the differences do not touch anything kept.
- The generator's proper-name heuristic is still not a factual check; the review is the factual check.
- Other unrelated open items remain in [maintenance.md](../docs/maintenance.md).
