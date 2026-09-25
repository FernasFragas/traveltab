# Destination photos: results

Implemented 2026-09-24 from the [plan](destination-photos-plan.md), work packages 1-5. Nothing was
committed, pushed or deployed; `weatherservice.db` and `.env` were not touched and the production
server was not run.

**Outcome:** every successful search (initial page, HTMX fragment, shared trip page) now shows a
live, attributed Wikimedia photo of the resolved destination, or an explicit "Photo unavailable"
state. Live lookups succeeded for 10 of 10 tested destinations; the limits are at the end.

## What changed

**New files**

| File | Purpose |
| --- | --- |
| `internal/application/destinationphoto.go` (+ test) | `DestinationIdentity`, `DestinationPhoto`, `DestinationPhotoSource`, identity normalization and matching (`DestinationMatchKM` = 25), `FoldName` |
| `internal/adapters/api/destinationphoto.go` (+ test) | Keyless Wikidata/Commons adapter: identity-checked city entity, Commons image and attribution, 4 s deadline |
| `internal/adapters/sqlite/destinationphoto.go` (+ test) | `Store.NewCachedDestinationPhotoSource`: identity-validated cache, 30-day and 1-hour lifetimes |
| `internal/adapters/httpserver/destinationphoto.go` (+ test) | `SetDestinationPhotoSource`, parallel lookup, the one presentation builder, page-cache identity validation |
| `cmd/design-preview/photos.go` (+ `photos_test.go`) | Deterministic photo fixtures, generated images, `-live-photos` support |
| `tasks/redesign/checks/destination-photos.mjs`, `destination-photos-live.mjs` | Offline and live browser checks |
| `tasks/artifacts/destination-photos/{fixture,live}/` | Screenshots and JSON |

**Edited:** `internal/adapters/api/wikimedia.go` (statement `Rank`, and `entitiesBatch` now reports the error object MediaWiki sends with a 200), `internal/adapters/httpserver/{server,trip,presentation}.go`, `views/weather_display.go.tpl`, `public/redesign/destination.css`, `cmd/web/main.go`, `cmd/design-preview/{fixture,main,server}.go`, `go.mod` (`golang.org/x/text` is now a direct dependency for diacritic folding), the `tasks/redesign/checks/*.mjs` scripts (http-smoke and interaction-regressions assert the photo states; browser-smoke gained an `OUTPUT_DIR` override), `README.md`, `docs/architecture.md`, `docs/design.md`, `docs/maintenance.md` ("Next up" item 2 marked done, limits added), `tasks/redesign/preview.md`, `CHANGELOG.md`. Files owned by the sitemap and slug work were not edited.

**Behavior**

- The photo lookup starts once the weather step resolves the destination, in parallel with the cache read and the video request, and the response always waits for it. Lisbon's curated photo skips it.
- The adapter: `wbsearchentities` (name or alias in any language, at most 5 entities) → entity claims → accept only a populated place within 25 km, in the same country (ISO code read from the country entity, kept in memory); ambiguous matches are refused; a nearby-article search is the fallback for namesakes the name search ranks too low → Commons `imageinfo` for at most 3 files (P18, then the article's page image) at a 1280 px thumbnail. Flags, logos, maps, non-bitmap, portrait, under-800 px, non-free, restricted and unattributed files are rejected; HTML credits become plain text; every URL is validated (https, allowed Wikimedia hosts).
- Cache: key `destphoto:v1:<folded name>:<ISO country>:<lat 3 dp>:<lon 3 dp>` in `source_cache`; the entry repeats its identity and is validated on read. 30 days for a photo, 1 hour for "none found". Expired photo: refresh live, keep the old one if the refresh fails or finds nothing. Errors, unreadable entries and incomplete metadata are never cached or trusted; a broken database still returns the live photo.
- Page-cache identity: a cached city page is used only when name, country and coordinates (25 km) match the freshly resolved destination; otherwise data is refetched and the fresh weather is displayed. Entries written before this change keep working when they match; entries without a country or coordinates are refetched.
- Template: `data-photo-state` = `ready`, `unavailable` (server) or `failed` (image `error`); the fallback "Photo unavailable" fills the same frame and the failed image's credit and caption are hidden. Inline `onload`/`onerror` on the image, so nothing is registered twice after swaps or history restoration.

## Commands and results

All run from the repository root on 2026-09-24. Localhost listeners and the Docker build needed to run outside the sandbox.

| Command | Result |
| --- | --- |
| `go test -race -count=1 ./...` | **Pass**, every package (cmd/design-preview, api, httpserver, sqlite, application, config, guides, placetypes, planner, slug) |
| `make lint` | **Pass** (`0 issues`; gofmt clean) |
| `make build` | **Pass** (`bin/traveltab`, git-ignored) |
| `git diff --check` | **Clean** (after removing a trailing blank line in `preview.md`) |
| `node --check` on all five `tasks/redesign/checks/*.mjs` | **Pass** |
| `docker build -t traveltab:photo-check .` | **Pass** (image removed afterwards) |
| `http-smoke.mjs` (default preview) | **Pass**, including the photo states |
| `interaction-regressions.mjs` | **Pass** (Porto swaps in its fixture photo; Coimbra shows no stale photo) |
| `destination-photos.mjs` with a `-slow` preview | **Pass** at 375 px and 1440 px, 0 JavaScript exceptions |
| `browser-smoke.mjs` default and `fallback` (`OUTPUT_DIR` in scratch) | **Pass**; the first `fallback` run failed once at the screenshot helper's `window.scrollY === 0` wait and passed on the next two runs, so treat that wait as flaky (it is not photo related) |
| Negative control: image `onerror` reverted to `this.hidden = true` | `destination-photos.mjs` **fails** as intended (template restored, verified by diff) |

New offline tests cover: identity normalization and matching; the adapter (city match, aliases and diacritics, namesakes across and within a country, wrong country, beyond 25 km, missing coordinates, non-settlements, ambiguity, nearby fallback, image rejection rules, landscape preference, page-image fallback, Commons redirects, unsafe URLs, HTML metadata, 429/500/malformed/200-with-error at every request, deadline, cancellation, request bounds, User-Agent, no key); the cache (cold, warm, expiry, stale retention, negative expiry, country and coordinate isolation, corrupt and mismatched entries, incomplete metadata, storage failure, errors not cached); and the HTTP layer (uncached and cached destinations on all three routes, namesake cache entries, legacy entries, Lisbon override, every unavailable case, escaping, sequential and concurrent searches, one lookup per response). Fixture tests use counting stubs and assert no URL points at Wikimedia. No offline test calls a network.

## Browser acceptance (offline fixture)

Isolated Brave 153 (headless, temporary profile, its own debug port; the regular browser was never used) against `go run ./cmd/design-preview` with HTMX 1.9.11 pinned. At 375 px and 1440 px:

- initial Lisbon (curated) → HTMX Paris → Porto → Paris, Texas: the photo, credit and identity are replaced together; the image had `complete` and `naturalWidth > 0` (1280) each time; one hero, one credit; hero frame unchanged; no horizontal overflow;
- no-photo, provider-failure and unlisted destinations: explicit "Photo unavailable", weather and planner intact;
- rapid five-search sequence (10 observations per width) and, on a `-slow` preview, two overlapping searches (14 observations): every observation paired a city with its own photo and credit;
- shared `/trip/paris-fr` and `/trip/paris-us`, and reload; plan → Back → Forward keeps the loaded photo;
- broken image URL: initial load, HTMX swap and history restore all end in the fallback with the credit and caption hidden.

Screenshots and JSON: `tasks/artifacts/destination-photos/fixture/` (`375-*`, `1440-*`, `browser-results.json`). The fixture run proves page behavior only, not provider coverage.

## Live checks (real Wikimedia APIs)

Run separately from the offline suite: `LIVE_API_TESTS=1 LIVE_RUNS=3 go test -v -run TestDestinationPhotoLive ./internal/adapters/api/` (fresh client per run, so each is a cold lookup; only Wikimedia's own caches are warm). Every selected image was downloaded (HTTP 200, `image/*`). Identity was checked against the resolved coordinates and ISO country.

| Destination | Entity | Credit | License | Image | Cold latency, 3 runs |
| --- | --- | --- | --- | --- | --- |
| Paris, FR | Q90 | Yann Caradec from Paris, France | CC BY-SA 2.0 | 1280x800, 335 KB | 1.75, 2.03, 1.38 s |
| Paris, US (Texas) | Q830149 | Adavyd | CC BY-SA 3.0 | 1280x857, 247 KB | 2.00, 1.48, 1.41 s |
| Porto, PT | Q36433 | Rititaneves | CC BY-SA 3.0 | 1280x857, 386 KB | 1.29, 1.16, 1.10 s |
| Tokyo, JP | Q1490 | Morio | CC BY-SA 3.0 | 1280x699, 271 KB | 1.48, 1.40, 1.45 s |
| Tavira, PT (small town) | Q372840 | Digitalsignal | CC BY-SA 3.0 | 1280x854, 248 KB | 1.04, 1.03, 1.17 s |
| Óbidos, PT (small town) | Q275862 | Lacobrigo | CC BY-SA 4.0 | 1280x854, 425 KB | 1.40, 0.98, 1.33 s |
| "Lisboa", PT (local name) | Q597 | Lacobrigo | CC BY-SA 4.0 | 1280x786, 1.77 MB PNG | 1.33, 1.32, 1.46 s |
| Valencia, VE (nearby fallback) | Q54880 | Ccmaracay2 | Public domain | 1280x960, 355 KB | 2.79, 2.19, 2.23 s |
| Springfield, US (Illinois) | Q28515 | Carol M. Highsmith | Public domain | 1280x841, 225 KB | 1.52, 1.43, 1.43 s |
| Kandersteg, CH | Q66218 | Earth explorer | Public domain | 1280x962, 361 KB | 1.23, 1.29, 1.19 s |

The two Parises resolve to different entities and photographs. Public-domain files have no license URL, so the page shows the license name as text. License links are the Creative Commons URLs Commons supplies. Extra one-off runs: Sintra, Zürich, Gothenburg, Perth AU and Perth GB (both resolved correctly, 1.0-2.3 s), "Atlantis" (nothing found, `(nil, nil)`, 1.1 s), and San Juan, PR: **no photo, 3.1 s**, see the limits.

**Through the cache** (`TestDestinationPhotoCacheLive`, empty SQLite database, no prewarming): cold 1.24-1.84 s, the repeat 116-232 µs; each destination went live exactly once.

**End to end in a browser** (`design-preview -live-photos`, real photos, fixture weather; `destination-photos-live.mjs`): Paris, Paris Texas, Porto, Tokyo and Tavira at 1440 px and 375 px, 10 of 10 passed. The cold search response took 0.75-1.70 s; each real image loaded (`naturalWidth` 1280) with the credit and license links shown. Screenshots and JSON: `tasks/artifacts/destination-photos/live/`.

**Latency:** cold lookups took 1.0-2.8 s (median about 1.4 s); the slowest is the nearby-article fallback. The 4-second deadline was never hit, but San Juan's fallback path took 3.1 s, so the margin is not large. The lookup overlaps the video request, so the added latency is at most the difference. I did not change the budget.

**Failures recorded honestly:** no live lookup errored. Two destinations produced no photo, as designed: San Juan, PR (Wikidata's country is the United States, the ISO code differs) and the invented "Atlantis". One `browser-smoke.mjs` run flaked (above).

## Remaining limits

- A photograph needs a Wikidata entity whose name, coordinates (25 km) and country ISO code match. Territories whose ISO code differs from Wikidata's country (Puerto Rico), places without a matching entity or free image, and genuinely ambiguous names show "Photo unavailable". A lookup has never chosen a wrong-country photograph in these tests, but "no wrong photo" is only as strong as the coordinates the weather step returns; a wrong geocode gets the wrong city's photo consistently.
- During a Wikimedia outage every uncached search waits up to 4 s before the unavailable state (in parallel with the video request). Errors are not cached and there is no circuit breaker.
- Only metadata is cached. The browser downloads from Commons, and some selected files are large (the "Lisboa" PNG is 1.77 MB). Caching image bytes was left out per the plan.
- Photos are whatever the entity's Wikidata image is, sometimes a landmark rather than a skyline. The caption is generic ("View of <city>"), and Commons descriptions are not used for alt text.
- Concurrent identical cold searches are not de-duplicated; each does its own lookup.
- Live results depend on Wikimedia at the time of the run. Live checks are not in CI, and the fixture browser checks say nothing about provider coverage.
- The redesign plan's verification gaps (accessibility audit, real map and video checks) are unchanged.
