# Implementation Plan: v2 (shareable links, export, AI city intros)

_Written 2026-09-20 from [the v2 section](../docs/ideas/weather-aware-trip-planner.md#next-iteration-v2) of the one-pager, after the MVP shipped as commit `4f27631` and the codebase moved to the `internal/` layout in [docs/architecture.md](../docs/architecture.md)._

## Status

- ✅ MVP shipped and committed.
- ✅ v2, part 1 (shareable links): Lane A complete, verified independently (full race suite, lint, build, live curl). See "Lane A's final route pattern" below.
- ✅ v2, part 3 (AI intros, generate-only): Lane B complete — all 20 starter cities generated to `guides/guides.json`, confirmed not wired into the app. See `tasks/notes-guides.md`.
- ✅ v2, part 2 (export): Lane C complete — `.ics`, `.kml` and a per-day Google Maps link, all validated against real generated files with independent libraries (Python `icalendar`, `xml.etree`), not just the Go test suite. See `tasks/notes-export.md`.

**All three v2 parts are done.** Nothing is committed.

## Lane A's final route pattern (for Lane C and anyone building on top of this)

- `GET /trip/:slug` — `slug` is `city-country` (e.g. `lisbon-pt`), built/parsed by `Slug`/`ParseSlug` in `internal/adapters/httpserver/slug.go`.
- Query params: `days` (1–5) and `from` (`YYYY-MM-DD`). Both must be present and valid, or the page falls back to the prefilled form.
- **Canonical URL has no query string**: `/trip/:slug`. That's what `/sitemap.xml` lists too.
- `GET /plan` is unchanged (`city`, `country`, `lat`, `lon`, `start`, `days` — note `start`, not `from`). On success it sets `HX-Push-Url: /trip/:slug?days=N&from=YYYY-MM-DD` so the address bar updates with no JS.
- Known compromise: `/sitemap.xml` reads its city list from a reserved pseudo-city key (`__sitemap_index__`) in the existing `city_data` table, because `application.Storage` has no "list cached cities" method yet and Lane A's agent was scoped away from touching `internal/application`/`internal/adapters/sqlite`. Functional and verified live; a real `ListCachedCities()` method is a good small follow-up, not a blocker.

## Decisions (2026-09-20)

Four open questions from the v2 spec, now answered:

- ✅ **AI intros generate, but do not wire into the app yet.** `cmd/guides` writes `guides/guides.json` as a reviewable artifact. It does **not** touch `views/trip_card.go.tpl`, does not `go:embed` the file, and the page shows no intro. Wiring it in is a follow-up task after a human reads the output.
- ✅ **Starter city list: the same 20 as `cmd/placetypes`.** Lisbon, Porto, Tavira, Funchal, Kyoto, Paris, Rome, Barcelona, London, New York, Tokyo, Istanbul, Prague, Amsterdam, Berlin, Seville, Florence, Vienna, Marrakesh, Mexico City.
- ✅ **City slug format: `city-country`**, both lower-cased ASCII with spaces as hyphens, country as the two-letter code the app already carries (`Request.Country`, e.g. `pt`, `jp`). `lisbon-pt`, `mexico-city-mx`. This disambiguates same-named cities and needs no new lookup table.
- ✅ **Default event times for calendar export: 10:00, 12:00, 15:00, 17:00**, 1.5 hours each, exactly as the one-pager proposed.

## Architecture

Everything lives under `internal/`, per `docs/architecture.md`. New v2 code follows the same boundaries:

- **`internal/planner`** stays free of HTTP, export formats and AI. It gets one addition: a determinism guarantee to test explicitly (see Task A1).
- **`internal/adapters/httpserver`** gets the new `/trip/:slug` route, the export routes, and the slug helpers.
- **`cmd/guides`** is a new, separate command (like `cmd/placetypes`), not part of `internal/`. It talks to Wikivoyage and to a local Ollama server, and writes `guides/guides.json` at the repo root. It must not be wired into `cmd/web`.

## Build order and parallel lanes

```
Lane A: Shareable trip links   ──┐
                                  ├──▶  Lane C: Export (queued until Lane A reports its final route/slug shape)
Lane B: AI city intros (generate-only, fully independent)
```

Lane A and Lane B run in parallel: they touch disjoint files and neither depends on the other. Lane C is queued because export URLs are specified to mirror the trip URL (`/trip/lisbon-pt.ics?…`), so it needs Lane A's actual route pattern, not a guess.

## Who owns which files

| File / directory | Lane |
|---|---|
| `internal/planner/build_test.go` (new determinism test only) | A |
| `internal/adapters/httpserver/trip.go`, `server.go`, `trip_test.go` | A (then C adds routes to the same files — **sequential**, not parallel) |
| `internal/adapters/httpserver/slug.go`, `slug_test.go` (new) | A |
| `views/index.go.tpl`, `views/trip_card.go.tpl` | A (SEO tags, push-url), then C (export buttons) |
| `public/sitemap.xml` route + generation | A |
| `cmd/guides/**` (new) | B |
| `guides/**` (new, generated data) | B |
| `internal/adapters/httpserver/export.go`, `export_test.go` (new) | C |

## Rules for every agent

Same as the MVP: **tests first**, failing on an assertion before the code exists; no network in normal tests (Lane B is the one place that legitimately needs the network and a local Ollama server — see its task); the dependency direction in `docs/architecture.md` holds (`planner` and `application` never import adapters); `gofmt` and `golangci-lint run ./...` stay at 0 issues; `make test` stays green; nothing committed by an agent.

## Risks

| Risk | Mitigation |
|---|---|
| A shareable link drifts over time as places or hotel data refresh | Task A1 pins the guarantee: same `Request` + `now` → identical stops and day groupings. Only the order of days may change, and only with a note. |
| Crawlers hitting `/trip/:slug` for uncached cities hammer Wikimedia/Overpass | Task A3 rate-limits how many *new* cities `/trip/:slug` will fully plan per minute; sitemap.xml only lists cities already cached, so crawlers mostly hit cheap cache reads. |
| AI intro invents a place that doesn't exist in the city | Task B rejects and retries any intro naming a place not found in the source Wikivoyage text; the task is not done if this check is missing. |
| AI intro ships to production without a human reading it | Explicitly out of scope for Lane B by decision above. The agent must not add `go:embed` or touch the trip template. |
| Export's URL scheme is decided twice (guessed now, then redone) | Lane C is queued, not launched, until Lane A's report gives the real pattern. |
