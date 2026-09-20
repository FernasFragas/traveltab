# TravelTab handover

**Updated:** 2026-09-20 (later the same day). **The trip planner MVP is complete, the follow-up round is done, and v2 (shareable links, export, AI city intros — generate-only) is now also complete.** Nothing is committed. Nothing is deployed.

## What was built

The weather-aware trip planner from `docs/ideas/weather-aware-trip-planner.md`, split into 17 tasks in `tasks/plan.md` and `tasks/todo.md`. All 17 are implemented, with evidence in `tasks/notes-*.md`. Original RED evidence is incomplete for some tasks; the completion pass documents this explicitly in `tasks/notes-completion.md`.

Enter a city, a start date and 1–5 days, and `GET /plan` returns a day-by-day plan: museums on rainy days, viewpoints on dry ones, plus places to stay near the middle of the plan. Every source is free and keyless: Wikipedia, Wikidata, Wikimedia Commons, Open-Meteo, OpenStreetMap.

## State: green

```
make build   ok
make lint    0 issues
make test    ok  internal adapters, config, placetypes and planner (race detector)
docker build ok  native ARM image
375 px UI    ok  HTMX, photos, stays, errors, no overflow, 7.41:1 subtitle contrast
```

The old Google Places hotels feature and the Foursquare itinerary are gone, along with their keys.

## Verified by hand, with real data

Lisbon, 3 days, from a real run of the app:

- Day 1 — Belém Tower, Padrão dos Descobrimentos, Jerónimos Monastery, Belém Palace (3.4 km)
- Day 2 — Alfama, São Vicente de Fora, Santa Engrácia, Castle of Saint George (1.7 km)
- Day 3 — Cathedral, Baixa, Santa Justa Lift, Teatro Nacional de São Carlos (1.7 km)
- Six real places to stay, with photos and Commons credits

**Timing:** ~3 ms cached. A city's first plan was ~25 s before the concurrency work and is now dominated by whether Overpass answers (see below). **Fallback proven:** pointing `OVERPASS_URLS` at a dead address still planned Porto, with the stay card showing "Places to stay are unavailable right now".

## Completion pass (2026-09-20)

Three agents reviewed planner behavior, city tests and UI; the coordinating session completed
remaining work when agent usage limits were reached. [Full evidence](tasks/notes-completion.md).

- Completed interrupted bounded Wikimedia concurrency and parallel place/forecast fetches.
- Added forecast outage/missing-data messages and indoor/outdoor labels.
- Rejected invalid coordinates and empty cities; improved planner contrast.
- Strengthened three vacuous city tests and verified they fail with an empty type map.
- Paired Kyoto's Arashiyama with the nearby bamboo grove from the unused ranked pool.
- Verified identical cached plans when all sources fail, including expired cache entries.
- Fixed the Docker architecture mismatch; native ARM image builds.
- Captured [mobile screenshots and browser results](tasks/notes-ui-validation.md).

## v2 (2026-09-20, same day): shareable links, export, AI city intros

Full spec and decisions in `tasks/plan-v2.md` / `tasks/todo-v2.md`. All three parts done:

- **Shareable trip links.** `GET /trip/:slug` (slug = `city-country`, e.g. `lisbon-pt`) renders a
  full plan on first load when `?days=N&from=YYYY-MM-DD` are both valid — no click needed. `/plan`
  sets an `HX-Push-Url` response header so the address bar updates itself, no JS. `/sitemap.xml`
  only lists already-cached cities, and a per-minute limit caps how many brand-new cities
  `/trip/:slug` will fully plan, so a crawl burst can't hammer Wikimedia/Overpass. One real
  planner gap got fixed along the way: `Plan.Note` now says "Days reordered for the latest
  forecast." when a link's day order shifts because the weather refreshed.
  ⚠️ `/sitemap.xml`'s city list is stored under a reserved pseudo-key in the existing `city_data`
  table, because `application.Storage` has no "list cached cities" method yet. Works, verified
  live, but is a workaround — a real `ListCachedCities()` method is a good small follow-up.
- **Export the trip.** `GET /trip/:slug.ics` and `.kml`, plus an "Open Day N in Google Maps" link
  needing no API key, on every plan. Real files were generated from a live run with real API data
  and independently parsed with Python's `icalendar` and `xml.etree` — not just asserted against
  in Go tests. Two real bugs were caught by running the tests, not by review: RFC 5545 line
  folding was one byte short on continuation lines, and the export URLs were first built with the
  file extension after the query string instead of on the path. ⚠️ No real import into Google
  Calendar/Apple Calendar/Outlook/Google My Maps was possible in this environment — only
  structural validation with real parsing libraries.
- **AI city intros, generate-only.** All 20 starter cities are in `guides/guides.json`, each read
  by hand and checked against its Wikivoyage source for invented places. Confirmed **not** wired
  into the app (`grep -rn "guides" internal/ views/ cmd/web/` finds nothing) — by design, so a
  human reads the output before any of it reaches the page. The validator caught a real
  hallucination live ("Seville Airport", not in the source text) and the retry produced a clean
  intro. Wiring `guides/guides.json` into the page is a deliberate follow-up, not started.

**Both agents assigned to this round hit the usage-limit cutoff mid-task.** The Lane B (guides)
agent had 17 of 20 cities done and no build breakage; the coordinating session ran the missing 3
(`-only="London,Seville,Vienna"`) and read all three before calling it done. The Lane C (export)
agent left the build broken (one unused `net/url` import, no other persisted code despite ~250K
tokens spent); the coordinating session fixed the import and then implemented all of C1/C2/C3
itself, test-first, since a cold relaunch would have re-paid the same context cost with no
guarantee of a better outcome. All of this is independently verified: full race suite, 0 lint
issues, build, and the `planner`/`application` → no-adapters layering rule, all still hold.

## Remaining release steps and limits

- Human review of the generated place-type map and finished implementation, commit/PR, remote CI,
  then deployment approval. Nothing was committed, pushed or deployed by this pass.
- Browser validation used fixture providers. The map embed was deliberately blanked; production
  maps and the external Booking landing page were not checked. Commons photos and CDN HTMX loaded.
- Historical RED evidence is incomplete and cannot be reconstructed retroactively.
- Cold requests still depend on public API latency, particularly Overpass. See the separate
  performance notes; the completion pass did not repeat live timings.
- Walking distances are straight-line estimates. Some spread-out city days are long; a genuinely
  isolated stop may remain alone when no candidate is within the 3 km reach limit.
- v2 is now implemented (see the section above). Wiring `guides/guides.json` into the trip page
  is the one deliberately-deferred piece left: read the intros, then embed and render them.
- `application.Storage` needs a real `ListCachedCities()` method so `/sitemap.xml` can drop its
  pseudo-city-key workaround.

## Package reorganization

All application Go code is now under `internal/`; `cmd/` retains entry points. The complete API
folder and all source/city fixtures moved with their packages. SQLite connections are owned by
stores and injected into the HTTP adapter through `application.Storage`.

Validation: `make test`, `make lint`, `make build`, Docker build and the local fixture-server
HTTP smoke passed. The generator's help confirms its new default output path. The smoke checked
the full page, HTMX fragment, fixture cache reads and static CSS.

## Where things live

The code now lives under `internal/`, with command entry points under `cmd/`.
See [architecture and migration map](docs/architecture.md) for the package boundaries.

| What | Where |
|---|---|
| Planning rules | `internal/planner/` |
| Reports and trip service | `internal/application/` |
| Free API clients and fixtures | `internal/adapters/api/` |
| SQLite cache and analytics persistence | `internal/adapters/sqlite/` |
| Fiber routes and rendering | `internal/adapters/httpserver/` |
| Templates and styles | `views/`, `public/` |
| Environment configuration | `internal/config/` |
| Generated place types (421) | `internal/planner/placetypes_gen.go`, regenerated by `go run ./cmd/placetypes` |
| Offline generation logic | `internal/placetypes/` |
| Production wiring | `cmd/web/main.go` |
| Shareable links, export, sitemap, rate limit | `internal/adapters/httpserver/slug.go`, `trip.go`, `export_ics.go`, `export_kml.go`, `googlemaps.go`, `sitemap.go`, `ratelimit.go` |
| AI city intros (generate-only, not wired in) | `internal/guides/`, `cmd/guides/`, `guides/guides.json` |
| Original per-task evidence | `tasks/notes-*.md` (historical paths; use the migration map) |
| v2 spec and evidence | `tasks/plan-v2.md`, `tasks/todo-v2.md`, `tasks/notes-guides.md`, `tasks/notes-export.md` |

Application and planner packages must not import adapters. Normal tests remain offline.
