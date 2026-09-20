# Tasks: v2 (shareable links, export, AI city intros)

Read [plan-v2.md](plan-v2.md) first for the decisions, architecture and file-ownership table. Background: [the v2 section](../docs/ideas/weather-aware-trip-planner.md#next-iteration-v2) of the one-pager.

**Method:** tests first, failing on an assertion before the code exists. See `tasks/todo.md`'s "How to work a task (TDD)" section for the house style — same rules apply here.

**Launch now, in parallel:** Lane A, Lane B.
**Queued:** Lane C, after Lane A reports its final route and slug pattern.

---

## Lane A: Shareable trip links

### A1. Determinism guarantee + city slugs

**Description:**
- `internal/adapters/httpserver/slug.go`: `Slug(city, country string) string` (lower-cased ASCII, spaces → hyphens, e.g. `Slug("Mexico City", "MX")` → `"mexico-city-mx"`) and `ParseSlug(slug string) (city, country string, ok bool)`, the inverse, splitting on the last hyphen-delimited two-letter segment.
- A new planner-level test proving the guarantee the whole feature depends on: **the same `Request` and `now` produce byte-identical `Plan.Days` stops and groupings.** Then, with a *later* `now` (so the forecast has refreshed), the **set of stops and which stops share a day stay the same**; only the **order of days** may change, and only when it does, `Plan.Note` explains it.

**Tests first (RED):**
- `TestSlug_BuildsCityDashCountry`
- `TestSlug_LowercasesAndHyphenatesSpaces`
- `TestParseSlug_RecoversCityAndCountry`
- `TestParseSlug_RejectsASlugWithNoCountrySuffix`
- `TestBuild_SameRequestAndNowGivesAnIdenticalPlan` (this one may already exist as `TestBuild_SameInputGivesTheSamePlan` — check before writing a duplicate)
- `TestBuild_LaterNowKeepsTheSameStopsAndGroupings`: same `Request`, two different `now` values a few days apart (use fakes so no real clock or network is involved); assert the same place IDs land in the same day-index groupings even if forecast-driven day *order* differs.
- `TestBuild_ReorderingDaysAddsANote`: when the day order does change between the two `now` values, `Plan.Note` contains a sentence saying so (agent picks exact wording; keep it short and factual, e.g. "Days reordered for the latest forecast.").

**Acceptance criteria:**
- [ ] `Slug`/`ParseSlug` round-trip for every ASCII city+country pair used in the existing test fixtures (Lisbon/pt, Tavira/pt, Kyoto/jp).
- [ ] The determinism test fails first against the current code if the guarantee doesn't already hold (it likely already holds from the MVP's determinism work — if so, this becomes a **regression guard**, not new behavior; say so plainly in your notes rather than claiming it as new).

**Files:** `internal/adapters/httpserver/slug.go`, `internal/adapters/httpserver/slug_test.go`, `internal/planner/build_test.go` (append only).

---

### A2. `GET /trip/:slug` route, with a shareable URL in the address bar

**Description:** A full page at `/trip/:slug` that works exactly like `/` (weather, videos, trip card), but:
- The slug fills the city/country search, exactly as if the visitor had typed it into the home page search box.
- When `?days=N&from=YYYY-MM-DD` are both present and valid, the page **renders the computed plan on first load** — no click needed to see it. Invalid or partial query params fall back to the pre-filled but unplanned form (same as visiting `/` after a search).
- **The URL stays shareable without JavaScript.** When `/plan` is called (from `/`, from `/trip/:slug`, wherever), the response carries an `HX-Push-Url` header set to the canonical `/trip/:slug?days=N&from=...` for the request that was just planned. HTMX reads that header and updates the browser's address bar on its own. Verify this is how the installed HTMX version (`views/index.go.tpl`'s `<script>` tag) behaves before relying on it; if it doesn't, use `hx-push-url` with a server-templated value instead, and say which approach you used and why.

**Tests first (RED), in `internal/adapters/httpserver/trip_test.go`:**
- `TestTripPage_RendersTheWeatherAndPlanFromASlug`
- `TestTripPage_WithNoDatesShowsThePrefilledFormOnly`
- `TestTripPage_UnknownSlugFallsBackToTheCityQueryFlow` (whatever the existing `/`-style handler does for a city it hasn't seen)
- `TestPlanRoute_SetsHXPushURLToTheShareableLink`
- `TestPlanRoute_PushURLOmitsTheDateWhenThePlanFailed`

**Acceptance criteria:**
- [ ] `curl -s /trip/lisbon-pt?days=3&from=<tomorrow>` returns 200 with the plan already in the HTML (no HTMX round trip needed to see it) — verify by hand, not just via test doubles.
- [ ] Sharing a copied `/trip/...` URL in a fresh session (no cookies) reproduces the same plan.

**Files:** `internal/adapters/httpserver/trip.go`, `internal/adapters/httpserver/server.go` (add the route), `internal/adapters/httpserver/trip_test.go`, `views/index.go.tpl` (only if the page needs a small branch for "slug page" vs. plain `/`).

**Depends on:** A1.

---

### A3. Search basics: title, canonical URL, sitemap, and a crawler limit

**Description:**
- Page `<title>` and meta description for a `/trip/:slug` page, e.g. "3 days in Lisbon — TravelTab" / "A day-by-day plan for a 3-day trip to Lisbon, with weather-aware stops and places to stay."
- A `<link rel="canonical">` pointing at `/trip/:slug` **without** the `days`/`from` query params, so search engines don't index one URL per date combination.
- `GET /sitemap.xml`, listing only `/trip/:slug` entries for cities that already have a row in `city_data` (or the source cache) — never triggers a fresh fetch.
- A simple per-minute limit on how many **new** (uncached) cities `/trip/:slug` will fully plan. Beyond the limit, serve the pre-filled form with a short message instead of planning, so a crawl burst can't hammer Wikimedia/Overpass. Exact mechanism is the agent's call (an in-memory token bucket is enough; no new dependency needed) — keep it simple and tested.

**Tests first (RED):**
- `TestTripPage_HasATitleAndDescription`
- `TestTripPage_CanonicalLinkExcludesDaysAndFrom`
- `TestSitemap_ListsOnlyCachedCities`
- `TestSitemap_IsValidXML`
- `TestPlanRoute_LimitsNewCitiesPerMinute`: fake the clock/counter so the test doesn't need a real minute to pass.

**Acceptance criteria:**
- [ ] `curl -s /sitemap.xml` is well-formed XML and lists no city that hasn't been searched before.
- [ ] The rate limit is per **new** city, not per request — repeat visits to an already-planned city are never limited.

**Files:** `internal/adapters/httpserver/trip.go`, `internal/adapters/httpserver/sitemap.go` (new), `internal/adapters/httpserver/sitemap_test.go` (new), `internal/adapters/httpserver/ratelimit.go` (new, small), its test, `internal/adapters/httpserver/server.go` (route).

**Depends on:** A2.

**When Lane A finishes, report:** the exact route pattern and query-param names used, so Lane C can be launched against the real thing rather than the spec's example.

---

## Lane B: AI city intros (generate-only — do not wire into the app)

### B1. `cmd/guides`: Wikivoyage → Ollama → `guides/guides.json`

**Description:** A new command, replacing the empty `cmd/console` placeholder per the v2 spec (or living alongside it as `cmd/guides` — agent's call; the spec says "turn the empty `cmd/console` into `cmd/guides`", so prefer that unless something else already depends on the `console` name).

For each of the 20 starter cities (see plan-v2.md's Decisions):
1. Fetch the city's Wikivoyage page (the [MediaWiki Action API](https://en.wikivoyage.org/w/api.php), `action=query&prop=extracts|revisions`, plain text, no markup) and record the revision ID.
2. Ask a local Ollama model (`mistral:7b`, confirmed installed and pullable at `http://localhost:11434`) for a ~120-word general city intro — why go, what the areas are like, how to get around — **using only the fetched text**, with an explicit instruction not to invent places.
3. **Reject and retry** (a few attempts, then skip with a logged warning) any intro that names a place not found in the source Wikivoyage text. A simple substring/fuzzy check against place names extracted from the text is enough; don't over-engineer named-entity extraction.
4. Write `guides/guides.json`: `{"lisbon-pt": {"intro": "...", "wikivoyage_revision": 12345678, "generated_at": "2026-09-20T..."}, ...}`, keyed by the same slug format as Lane A (`city-country`, lower-cased, hyphenated) — coordinate the exact format with A1's `Slug` function if it has landed; otherwise implement the same rule independently and note the duplication for a later cleanup.
5. **Only regenerate a city whose Wikivoyage revision changed** since the last run — skip cities whose `guides.json` entry already has the current revision.

**Do NOT:** add `go:embed` anywhere, touch `views/trip_card.go.tpl`, or wire this into `cmd/web`. This command is stand-alone and its output is for human review.

**Tests first (RED), in `cmd/guides/*_test.go` (or `internal/guides` if the agent finds it cleaner to keep `cmd/guides/main.go` thin and put logic in an internal package — mirror how `cmd/placetypes` does this):**
- `TestFetchWikivoyageText_ReturnsPlainTextAndRevision` (stub the Wikivoyage HTTP call; no network in this test)
- `TestValidateIntro_RejectsAnInventedPlace`
- `TestValidateIntro_AcceptsPlacesFoundInTheSourceText`
- `TestWriteGuides_KeysBySlugAndRecordsRevision`
- `TestWriteGuides_SkipsACityWhoseRevisionHasNotChanged`
- Large, explicitly opt-in (`GENERATE_GUIDES=1`), not part of `make test`: an end-to-end run against the real Wikivoyage API and a real local Ollama call for **one** city, to prove the pipeline actually works before running all 20.

**Acceptance criteria:**
- [ ] `go run ./cmd/guides` (or `GENERATE_GUIDES=1 go run ./cmd/guides` if the agent gates the real run behind a flag/env var, which is reasonable given it needs Ollama and the network) produces `guides/guides.json` with entries for as many of the 20 cities as Wikivoyage and Ollama allow, and logs which ones were skipped and why.
- [ ] Read a sample of the generated intros yourself before reporting done, and quote 2–3 in your report so a human can sanity-check tone and accuracy without opening the file.
- [ ] `guides/guides.json` is valid JSON, and nothing in `internal/` or `views/` references it.
- [ ] `make test`, `make lint`, `make build` stay green with `cmd/guides` added.

**Files:** `cmd/guides/**` (new), `guides/**` (new, generated — not `.gitignore`d, since the point is that a human reviews the diff), `tasks/notes-guides.md` (new, your evidence file).

**Depends on:** nothing else in v2. Fully independent — this is why it runs in parallel with Lane A.

---

## Lane C: Export the trip

Lane A is done. The real route: `GET /trip/:slug` (slug = `city-country`, e.g. `lisbon-pt`), query params `days` (1–5) and `from` (`YYYY-MM-DD`). Both are required for a plan to exist. Slug helpers live in `internal/adapters/httpserver/slug.go` (`Slug`, `ParseSlug`) — reuse them, don't reimplement.

### C1. `.ics` calendar export

**Description:** `GET /trip/:slug.ics?days=N&from=YYYY-MM-DD` (same slug/param scheme as the page, `.ics` extension instead of the bare path). Only served when a full plan can be built (same validation as `/trip/:slug`; missing/invalid params → 404, not an empty calendar). One `VEVENT` per stop, at fixed daily times **10:00, 12:00, 15:00, 17:00**, each 1.5 hours, in the **forecast's IANA time zone** (`planner.Forecast.Timezone`, e.g. `Europe/Lisbon`) — when there's no forecast for a day, fall back to a sensible default (the request's local day boundary is fine; note your choice). Each event's `LOCATION` is the stop's name; `DESCRIPTION` links back to `/trip/:slug`. Import `time/tzdata` blank so the Docker image needs no system tzdata.

**Tests first (RED):**
- `TestICS_OneEventPerStopAtTheDefaultTimes`
- `TestICS_UsesTheForecastTimeZone`
- `TestICS_EscapesCommasAndSemicolonsInPlaceNames`
- `TestICS_WrapsLongLinesPerRFC5545` (75-octet folding — check whether a library already handles this before hand-rolling it; a small dependency is fine if it saves getting RFC 5545 folding subtly wrong)
- `TestICS_404sWithoutAFullPlan`
- `TestICS_ContentTypeIsTextCalendar`

**Acceptance criteria:**
- [ ] Generate one real `.ics` file (`curl` a real running plan) and **actually import it into Google Calendar** — confirm the events land on the right dates at the right times. Note plainly if Apple Calendar/Outlook weren't checked; don't claim you tested something you didn't.

**Files:** `internal/adapters/httpserver/export_ics.go` (new), `export_ics_test.go` (new), route added in `server.go`.

### C2. `.kml` export for Google My Maps

**Description:** `GET /trip/:slug.kml?days=N&from=YYYY-MM-DD`. One `<Folder>` per day, one `<Placemark>` per stop with its coordinates and name.

**Tests first (RED):**
- `TestKML_OneFolderPerDay`
- `TestKML_OnePlacemarkPerStopWithCoordinates`
- `TestKML_IsValidXML`
- `TestKML_404sWithoutAFullPlan`

**Acceptance criteria:**
- [ ] The file imports cleanly at mymaps.google.com (check by hand if you have a Google account available in this environment; otherwise validate structurally against the KML schema and say so).

**Files:** `internal/adapters/httpserver/export_kml.go` (new), `export_kml_test.go` (new), route added in `server.go`.

### C3. Bonus: "Open Day N in Google Maps" link

**Description:** A plain `https://www.google.com/maps/dir/?api=1&destination=...&waypoints=...` link per day, needing no API key. `MaxStopsPerDay` is 4, which fits Google's mobile 3-waypoint-plus-destination limit. Add a button/link per day in `views/trip_card.go.tpl`.

**Tests first (RED):**
- `TestGoogleMapsLink_BuildsWalkingDirectionsThroughTheDay`
- `TestGoogleMapsLink_RespectsTheWaypointLimit`

**Files:** `internal/adapters/httpserver/trip.go` (small addition — coordinate with what Lane A left there), `views/trip_card.go.tpl`.

**General constraints for Lane C:** same TDD rules as the other lanes. Don't touch `internal/adapters/httpserver/slug.go`, `sitemap.go`, `ratelimit.go`, or `internal/planner/build.go` — those are Lane A's finished, verified work. `gofmt`, `golangci-lint run ./...` at 0 issues, `make test` green, nothing committed.
