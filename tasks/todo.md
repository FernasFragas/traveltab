# Writers' Map (Always Ready) — PRs and Subtasks

Source of truth for progress. Read [the plan](plan.md) (especially **Delivery** and **Rules for Agents**) and [the one-pager](../docs/ideas/community-map.md) first.

**Status:** PR 1 subtasks 1a–1e done, plus the contract gaps found on October 8, 2026 (`:dest` in map URLs, the base-place ports and fake, the panel template owner). The owner approved the contract on October 8, 2026, so it is frozen and **PR 1 is done**; PR 3, PR 4 and the early 6a/9a can start once PR 1 is merged. PR 2 size and lookup measurements are recorded; coverage of the 55 research places remains unavailable and PR 5 stays gated. Updated October 6, 2026: summaries replace excerpts, AI in MVP (PR 9), monthly sync, caches never expire, "Plan with these places" (7e). Updated October 7, 2026: UI matches [the mockup](../docs/ideas/writers-map-mockup.png) — new subtasks 1e, 3d, 7f, 7g, 7h; 7a and 7c extended (see **Target Output** in the plan).

**How to read this file**
- Each **PR** merges on its own, CI green, with `WRITERS_MAP`, `WRITERS_SYNC` and `WRITERS_AI` **off**.
- Each **subtask** (1a, 1b …) can go to a different agent. Subtasks in one PR touch **disjoint files** unless marked *after*.
- **Owns** = the only files the subtask may edit. Shared files belong to the subtask marked **integrator**.
- Branches: `writers-map/pr<N>-<letter>` → merged into `writers-map/pr<N>` → PR to `main`.

**Conventions**
- Run from the repo root: `make build`, `make test`, `make lint`.
- **Tests never hit the network:** use the PR 1 fixture, the fake source, the fake summarizer, `httptest` and recorded responses.
- **No post text stored, logged or committed:** names, positions and AI summaries only.
- Past 2× an estimate or 5 files → split here first.

---

## PR 1 — Contracts, Data and Flags (≈5 h) · needs: none

**Goal:** freeze everything parallel work depends on. No user-visible change.

### 1a. Write the source data and policy — 1 h
- [x] Done

**Owns:** `writerdata/sources.json`, `writerdata/overrides.json`, `writerdata/embed.go`, `docs/maintenance.md`

- Sources: the 12 working blogs from [overlap results](blog-overlap-results.md), each with API base, language, `status` (`active`/`blocked`/`removed`) and date checked. VagaMundos, viajarentreviagens and lisbonlisboaportugal start `blocked`.
- Overrides: alias removals and merges (Clérigos ×3, the "Ribeira Grande" airport alias, "Nossa Senhora da Conceição"); destination areas for Madeira and São Miguel (default ≈40 km for cities); a per-QID summary hide list; a stoplist ("Fernandes", "Infante", "Achada" plus the research list).
- `docs/maintenance.md` gets a "Writers' map" section: user agent, limiter, robots.txt, 403 handling, names-only storage, AI summaries via LLM Gateway, monthly schedule, in-place updates, takedown steps.

**Verify:** `make build`; document review.

### 1b. Define the types, ports and contract — 1.5 h
- [x] Done

**Owns:** `internal/writermap/destinationdata.go`, `internal/application/writermap.go`, `docs/writers-map-contract.md`

- Types: `Area`, `Place` (QID, names, coordinates, kind, description, photo + credit, sitelinks), `Mention` (writer host, post ref/URL/title, language, positions), `Itinerary` (writer, post, days of QIDs), `Summary` (QID, English text, source post refs, model, generated at), `DestinationResult` (places, writer counts, itineraries, source statuses, complete flag).
- Port `WriterMapSource`: `Destination(ctx, dest)` and `Place(ctx, dest, qid)` (panel data incl. summary if any).
- Port `Summarizer`: `Summarize(ctx, place, passages []Passage) (string, error)`; passages carry language and post ref.
- Contract doc covers:
  - GeoJSON feature properties
  - panel fragment fields (summary optional; writer links with language label)
  - endpoint paths (`/map/:dest.geojson`, `/map/:dest/:qid`)
  - JS extension API: `TravelMap.registerLayer(name, {add, remove})`, events `travelmap:ready`, `travelmap:place-selected`, `travelmap:plan-updated`
  - template slots (`data-map-slot="panel|layers|saved|status|list"`)
  - save-button hook (`data-save-qid`)

**Verify:** `make build`; **owner reviews the contract doc** (approved October 8, 2026).

### 1c. Add the Madeira fixture, fake source and fake summarizer — 1 h · *after 1b*
- [x] Done

**Owns:** `internal/writermap/writermaptest/fake.go`, `internal/writermap/writermaptest/fake_test.go`, `internal/writermap/writermaptest/testdata/madeira.json`

- About 15 places (base-only and writer pins), EN and PT writers, summaries on some writer pins and none on others, 2 itineraries, mixed source statuses.
- A fake `Summarizer` that returns a canned sentence or an error.
- The fake implements `WriterMapSource`, with switches for slow, partial, unavailable and empty.

**Verify:** `go test ./internal/writermap/...`.

### 1d. Add the feature flags — 0.5 h · **integrator**
- [x] Done

**Owns:** `internal/config/env.go`, `internal/config/env_test.go`, `internal/config/flags.go`, `internal/config/flags_test.go`

- Add `WRITERS_MAP`, `WRITERS_SYNC` and `WRITERS_AI`, plus `LLM_GATEWAY_URL` / `LLM_GATEWAY_KEY` (unused until 9c); all default off/empty. Bring 1a–1c together in the working tree and run `make test lint build`; the owner commits and opens the PR.

**Verify:** `go test ./internal/config/...`; CI green.

### 1e. Add the fields the mockup needs — 1 h · *contract change, before the owner review*
- [x] Done — [TDD evidence](../docs/tdd/1e-mockup-fields.tdd.md)

**Owns:** `internal/writermap/destinationdata.go`, `internal/writermap/destinationdata_test.go`, `internal/writermap/writermaptest/fake.go`, `internal/writermap/writermaptest/fake_test.go`, `internal/writermap/writermaptest/testdata/madeira.json`, `writerdata/sources.json`, `writerdata/embed_test.go`, `docs/writers-map-contract.md`

- GeoJSON properties gain `writers` (host array, for the "All writers" filter) and `has_summary` (for the Summaries layer).
- Place panel gains `area` (Wikidata P131 label, shown as *kind · area*) and per-writer `blog_name`, from a new `name` field in `writerdata/sources.json`.
- Itinerary days gain `distance_km` and `walk_minutes` (for the writer's walk card, which shows one day). Distance is straight-line between consecutive stops, as in the planner; real values come from 6a, the fixture fills them now.
- Plan stops gain a `writers_pick` flag (for 7g's "Writers' pick" chip): true when the stop's QID has a writer count of 1 or more. Defined in the contract; 7g builds it.
- Fixture filled for all new fields.

**Verify:** `go test ./internal/writermap/... ./writerdata/`.

**PR 1 done when:** contract doc (with 1e) approved; fixture and fakes build; flags default off. **Done:** contract approved October 8, 2026.

---

## PR 2 — Index Spike (≈3 h) · needs: none · *parallel with PR 1*

### 2a. Measure index size, lookup speed and precision — 3 h · single agent
- [ ] Done

**Owns:** `tasks/writers-map-spike/` (throwaway), `tasks/writers-map-results.md`

- Crawl two full blogs (viajecomigo PT, saltinourhair EN), build the names-only tables in a temp SQLite file, run the Madeira Wikidata nearby query, match.

**Acceptance — record in the results doc:**
- [x] MB and rows per 1,000 posts, and the projection for ~20,400 posts (proposed bar ≤300 MB).
- [x] Madeira lookup time (proposed ≤200 ms).
- [ ] 40-match precision with Wikidata aliases, against the research's 37/40 (proposed ≥90%).
- [ ] How many of the 55 research places (≥3 blogs) Wikidata nearby plus filters still returns.
- [x] **Verdict:** keep the schema, cap sub-spans, or add OSM names. **PR 5 waits for this.**

**Verify:** results document reviewed by the owner.

---

## PR 3 — Map Shell Behind the Flag (≈6 h) · needs: PR 1

**Goal:** with `WRITERS_MAP=1`, every destination shows the new map with fixture data. With it off, Waze is unchanged.

### 3a. Write the core map JS and CSS — 1.5 h
- [ ] Done

**Owns:** `public/redesign/writers-map.js`, `public/redesign/writers-map.css`

- Leaflet (unpkg + SRI) and OSM tiles with attribution; tile URL read from a data attribute.
- One map instance across HTMX swaps.
- Layer registry and events as in the contract.
- List alternative reachable by keyboard.
- Status area for loading, partial and unavailable states.

**Verify:** exercised by 3c.

### 3b. Wire the template and fixture endpoints — 1.5 h · **integrator**
- [ ] Done

**Owns:** `views/map_card.go.tpl`, `views/index.go.tpl`, `views/place_panel.go.tpl` (minimal; 7a builds it out), `internal/adapters/httpserver/writermap.go`, `internal/adapters/httpserver/writermap_test.go`, `internal/adapters/httpserver/server.go`, `cmd/web/main.go`

- Flag on: the map shell with all contract slots, plus `hx-trigger="load"`. Flag off: today's Waze markup, byte-for-byte.
- Contract endpoints, served from the PR 1 fake. `:dest` is the trip slug plus `?lat=…&lon=…` (see **Destination in map URLs** in the contract); the shell's data attributes carry the full URLs.
- Minimal `place_panel.go.tpl`: the contract's panel fields and writer links, no mockup styling.
- Keep "View larger map".

**Verify:** `go test ./internal/adapters/httpserver/...` (flag on and off; bad slug → 404; bad coordinates → 400; escaping).

### 3c. Add the design-preview scenario — 1 h
- [ ] Done

**Owns:** `cmd/design-preview/writersmap.go`, `cmd/design-preview/writersmap_test.go`

- Scenarios: Madeira (writers), a base-only city, sources unavailable. Local fixture tiles.

**Acceptance**
- [ ] 375 and 1440 px.
- [ ] Two destination swaps leave one map.
- [ ] Keyboard can reach the list.

**Verify:** `go test ./cmd/design-preview`.

### 3d. Lay out the card like the mockup — 2 h · *after 3a, 3b*
- [ ] Done

**Owns:** `public/redesign/writers-map-layout.css`, `views/map_card.go.tpl` (layout markup only, with 3b's agreement)

- Header: "Writers' Map", subtitle, green **"Always ready"** pill in the `status` slot (turns amber for partial, grey for unavailable).
- ≥1024 px: map ≈60% and `panel` slot ≈40% side by side, equal height. <1024 px: panel stacks under the map as a card.
- Chip row overlaid top-left of the map (`layers` slot); "All writers" select top-right.
- Map chrome: zoom, north arrow, scale bar, basemap thumbnail toggle (OSM / satellite). **Check first** that the imagery source's terms allow free use on this site (Esri World Imagery is not confirmed); if no free source fits, leave the satellite option out.
- Empty panel state: "Pick a place on the map".

**Acceptance**
- [ ] Preview at 375 and 1440 px matches the mockup's layout (not its photos).
- [ ] No horizontal scroll at 375 px.

**Verify:** preview check side by side with the mockup.

**PR 3 done when:** both flag states pass tests; preview shows the shell in the mockup layout.

---

## PR 4 — Base Layer (≈5 h) · needs: PR 1

### 4a. Write the Wikidata nearby adapter — 2.5 h
- [ ] Done

**Owns:** `internal/adapters/api/wikidatanearby.go`, `wikidatanearby_test.go`, `internal/adapters/api/testdata/wikidata_*.json`

- Implements `application.BasePlaceSource`.
- SPARQL `wikibase:around`: ≈40 km for cities (overrides first for islands/regions), visitable-class filter, infrastructure excluded, capped at ~200 ranked by sitelinks (writer count re-ranks later in 6a).
- Visitable classes come from the planner's generated list (`planner.PlaceTypes`, skipping `never`), not a second list.
- Returns en/pt labels and aliases, English description, P18 image with Commons credit (reuse `wikimedia.go` parsing), and sitelinks.
- Retry-After honored; identified user agent.

**Acceptance**
- [ ] Madeira fixture includes Pico Ruivo, Cabo Girão and Câmara de Lobos, and excludes the airport.
- [ ] Missing image or description → field left empty.

**Verify:** `go test ./internal/adapters/api/...`.

### 4b. Add the cache decorator — 1.5 h
- [ ] Done

**Owns:** `internal/adapters/sqlite/wikidatanearby.go`, `wikidatanearby_test.go`

- Implements `application.BasePlaceSource` and `application.BasePlaceRefresher`.
- Cache in `source_cache` with **no expiry**, following `sourcecache.go`: cached → no call; error with no cache → empty result plus an error the caller can log.
- `Refresh(ctx, dest)` re-runs the query and overwrites the row in place (called by the monthly job in 5d).
- Singleflight per destination.

**Verify:** `go test -race ./internal/adapters/sqlite/...` (20 concurrent callers → 1 query).

### 4c. Warm from search suggestions — 1 h · **integrator**
- [ ] Done

**Owns:** `internal/adapters/httpserver/suggest.go`, `suggest_test.go`, `cmd/web/main.go` (wiring only)

- Top suggestion → background base query: deduplicated, at most one per client per 2 s, only when `WRITERS_MAP` is on, never delaying the suggestion response.

**Verify:** `go test ./internal/adapters/httpserver/...` with `writermaptest.FakeBaseSource`, which counts calls.

**PR 4 done when:** cached base data is available behind the flag; suggestion latency is unchanged.

---

## PR 5 — Writers Index and Monthly Sync (≈9 h) · needs: PR 1, **PR 2 verdict**

### 5a. Create the index tables and store — 2 h
- [ ] Done

**Owns:** `internal/adapters/sqlite/writerindex.go`, `writerindex_test.go`

- Tables `writer_posts`, `writer_names`, `writer_days` and `writer_sync`, using the schema from the PR 2 verdict, created with `CREATE TABLE IF NOT EXISTS` from a function the integrator calls.
- Store methods: upsert a post with its names and days in one transaction, delete a post, purge a host, look up by phrases, read and write the sync cursor.

**Acceptance**
- [ ] Re-indexing a post replaces its rows and never duplicates them.
- [ ] Purging a host removes every row for it.
- [ ] An existing database file is untouched otherwise.

**Verify:** `go test ./internal/adapters/sqlite/...`.

### 5b. Extract name phrases and day headings — 2.5 h
- [ ] Done

**Owns:** `internal/writermap/extract.go`, `extract_test.go`, `internal/writermap/testdata/posts/`

- Pure function: HTML in → phrases (folded, original-case flag, positions, sub-spans of up to 5 words), day sections, and an itinerary flag out. No text is returned.

**Acceptance**
- [ ] "Miradouro do Pico dos Barcelos" yields both the full name and "Pico dos Barcelos".
- [ ] The EN and PT itinerary fixtures split into the right days; a listicle isn't flagged as an itinerary.

**Verify:** `go test ./internal/writermap/...`.

### 5c. Write the WordPress client and host limiter — 2 h
- [ ] Done

**Owns:** `internal/adapters/api/wordpress.go`, `wordpress_test.go`, `internal/adapters/api/hostlimiter.go`, `hostlimiter_test.go`

- Listing by page with `modified_after`, an id-only sweep, fetch by id (`_fields=content`), robots.txt check (cached).
- Per-host limiter ≥1.5 s, shared by the sync and the summary job (PR 9).
- 403 returns a typed `ErrBlocked`; one retry with backoff on timeout.

**Verify:** `go test -race` with `httptest` covering pages, 400 past the last page, 403, timeout and robots disallow.

### 5d. Build the monthly sync job — 2.5 h · *after 5a–5c* · **integrator**
- [ ] Done

**Owns:** `internal/writermap/sync.go`, `sync_test.go`, `internal/adapters/sqlite/database.go` (table creation call), `cmd/web/main.go` (start only when `WRITERS_SYNC` is on)

- Initial crawl, then one **monthly** run: changed posts (`modified_after`), deleted-posts sweep, Wikidata `Refresh` for cached destinations, then a hook for the summary step (PR 9). Single-run lock, resumable cursor, one page in memory at a time, off-peak.
- All writes are upserts in place; no history rows.
- `ErrBlocked` → source marked blocked and skipped; `removed` sources purged.
- One progress line per host per run.

**Acceptance**
- [ ] An interrupted crawl resumes from its cursor.
- [ ] Edited posts are re-indexed; deleted posts are dropped after the sweep.
- [ ] Two consecutive runs over unchanged data leave row counts and file size unchanged.

**Verify:** `go test -race ./internal/writermap/...` with the 5c fake server; CI green.

**PR 5 done when:** a two-blog sync runs locally with `WRITERS_SYNC=1` and resumes after being killed.

---

## PR 6 — Matching and the Real Source (≈5.5 h) · needs: PR 4, PR 5

> **6a can start right after PR 1** against the fixture, and waits in its branch until PR 6 opens. On-demand excerpts were dropped on October 6 — summaries (PR 9) replace them.

### 6a. Write the matcher and itinerary assembly — 2.5 h · *early*
- [ ] Done

**Owns:** `internal/writermap/match.go`, `match_test.go`, `internal/writermap/itinerary.go`, `itinerary_test.go`

- Pure function: places (aliases) plus index rows go in; mentions per place, writer counts and itineraries come out.
- Rules from the research: case rule, stoplist, nested-name suppression by position, overrides.
- Re-rank places: writer count first, then sitelinks; keep the ~200 cap.

**Acceptance**
- [ ] "São Pedro de Alcântara" doesn't count as "Alcântara".
- [ ] "praia" in running text doesn't match.
- [ ] The three Clérigos entries merge into one.
- [ ] Itinerary days are ordered by first mention.

**Verify:** `go test ./internal/writermap/...`.

### 6b. Compose the real source and wire it — 3 h · **integrator**
- [ ] Done

**Owns:** `internal/writermap/source.go`, `source_test.go`, `internal/adapters/sqlite/writerdestination.go`, `internal/adapters/httpserver/writermap.go`, `internal/adapters/httpserver/server.go`, `cmd/web/main.go`

- `WriterMapSource` built from base (PR 4), index store (PR 5) and matcher; `Place()` reads `writer_summaries` if the table has a row (empty until PR 9).
- Per-destination result stored with no expiry, upserted in place, recomputed after each monthly run.
- Replaces the fake when `WRITERS_MAP` is on.

**Acceptance**
- [ ] No blog request during `Destination()` or `Place()` (test asserts zero HTTP calls).
- [ ] The fake is still available to the preview.

**Verify:** `go test -race ./...`; local run with both flags on shows Madeira on real data.

**PR 6 done when:** Madeira works end to end locally on real data (links, no summaries yet).

---

## PR 7 — Map UI Features (≈22.5 h) · needs: PR 3 · *runs on the fake source, parallel with PRs 4–6*

### 7a. Build pins and panels — 4 h
- [ ] Done

**Owns:** `public/redesign/map-panel.js`, `public/redesign/map-panel.css`, `views/place_panel.go.tpl` (builds out 3b's minimal template), `internal/adapters/httpserver/placepanel_test.go`

- Writer pins drawn above base pins and sized by writer count.
- Base panel: description, photo with credit, Wikipedia link.
- Writer panel adds "Mentioned by N writers", the summary labelled *AI summary* when present, and one link per writer's post with a language label. **No quoted blog text.**
- Save button slot `data-save-qid`.
- **Mockup look:** writer pins terracotta with pen icon, base pins green; top places labelled; clusters with a count.
- **Panel order:** photo + close · name · *kind · area* · Want to visit (heart) · writer monograms (initials, colour from host; "+N") with "Mentioned by N writers" · description · **AI summary** card · **From Travel Writers** (post title, blog name, language chip, external-link icon; first 2, then "View all (N)") · "Plan with these places".

**Acceptance**
- [ ] Every writer is listed with a post link.
- [ ] A writer pin without a summary shows links only, with no empty box.
- [ ] A base-only destination shows "No writer mentions here yet".

**Verify:** `go test ./internal/adapters/httpserver/ -run PlacePanel`; preview at 375 and 1440 px.

### 7b. Draw the day plan layer — 3 h
- [ ] Done

**Owns:** `public/redesign/map-dayplan.js`, `views/itinerary_body.go.tpl`, `views/trip_card.go.tpl`

- The plan response emits stop coordinates per day; the script fires `travelmap:plan-updated` and draws numbered pins per day with a day toggle.

**Acceptance**
- [ ] Generating, regenerating and clearing a plan leave no duplicate layers.
- [ ] Works on base-only destinations.

**Verify:** template test for the emitted data; preview check.

### 7c. Add writers' itinerary layers and the walk card — 3 h
- [ ] Done

**Owns:** `public/redesign/map-itineraries.js`, `public/redesign/map-itineraries.css`

- Lists itineraries in the `layers` slot; one is drawn at a time, with ordered pins and lines, labelled as the writer's route and linked to the post; hidden when there are none.
- Route drawn as a dashed terracotta line.
- **Walk card** bottom-left of the map: route name, from → to, km and walking time, "View full route" (fits the map to the route).

**Verify:** preview check with fixture itineraries.

### 7d. Add "Want to visit" — 2 h
- [ ] Done

**Owns:** `public/redesign/map-saved.js`

- Binds `data-save-qid` buttons; "Saved" list in the `saved` slot; `localStorage` wrapped in try/catch.

**Acceptance**
- [ ] Saves survive a reload.
- [ ] Blocked storage hides the buttons.
- [ ] No server calls.

**Verify:** preview check, including a private window.

### 7e. Add "Plan with these places" — 3.5 h
- [ ] Done

**Owns:** `internal/planner/types.go`, `internal/planner/build.go` (seeding only), `internal/planner/build_test.go`, `internal/adapters/httpserver/trip.go` (query parsing in `planTrip`), `trip_test.go`, `public/redesign/map-itineraries.js` (button only, after 7c)

- `planner.Request` gains `Include []Place` (max ~12); `Build` places them into days first (by writer day where possible), then fills with nearby places as today.
- `/plan` accepts the QIDs in the query string, resolved through `WriterMapSource.Place` (the fake in PR 7, the real source after PR 6); unknown QIDs are ignored. The URL is shareable.
- Button on each writer's itinerary layer.

**Acceptance**
- [ ] All included places appear in the plan; requests without `Include` produce today's plans byte-for-byte.
- [ ] More included places than days can hold → extra ones listed as "also suggested", not dropped silently.

**Verify:** `go test ./internal/planner/... ./internal/adapters/httpserver/...`; preview check.

### 7f. Add layer chips and the writer filter — 2 h · *after 7a*
- [ ] Done

**Owns:** `public/redesign/map-layers.js`, `public/redesign/map-layers.css`

- Chips: **Base places**, **Writers' places**, **Summaries** (only places with `has_summary`; hidden while `WRITERS_AI` is off), **My plan** (badge with day count; disabled until a plan exists).
- "All writers" select lists blog names; picking one shows only that writer's pins.
- Chip state is buttons with `aria-pressed`; remembered per tab in `sessionStorage` (try/catch).

**Acceptance**
- [ ] Every chip toggles its layer without redrawing the others.
- [ ] Writer filter + Writers' chip off → no writer pins, no error.

**Verify:** preview check.

### 7g. Enrich the itinerary cards — 3 h · *after 7b, 7e*
- [ ] Done

**Owns:** `internal/planner/schedule.go`, `internal/planner/schedule_test.go`, `internal/adapters/api/openmeteo_forecast.go`, `openmeteo_forecast_test.go` (daily max temperature and weather code only), `views/itinerary_body.go.tpl` (after 7b)

- Per day: **max temperature and condition** (Open-Meteo `temperature_2m_max`, `weather_code`; free) next to the walking km.
- Per stop: **time slot** from a 9:00 start, a visit length per kind and walking time; **walking minutes to the next stop** (4.5 km/h on straight-line distance × 1.3).
- Tag chip: **Writers' pick** when writers mention the stop (via `WriterMapSource`), else the kind (Historic site, Museum …).
- **View on map** button: scrolls to the map and turns on **My plan**.

**Acceptance**
- [ ] Plans without forecast data show no temperature, not "0°".
- [ ] Time slots never pass 19:00; extra stops move to "also suggested".

**Verify:** `go test ./internal/planner/... ./internal/adapters/...`; preview check.

### 7h. Add "Suggested based on writers" chips — 2 h · *after 7e*
- [ ] Done

**Owns:** `views/trip_card.go.tpl` (chip row only, after 7b), `internal/adapters/httpserver/trip_themes.go`, `trip_themes_test.go`

- Up to 4 theme chips under the plan form (e.g. Iconic sites, Local neighbourhoods, Great food, Scenic views), built from the kinds of the destination's top writer places.
- Picking chips adds those places to `Include` (7e) before "Generate personalized plan".
- Hidden on base-only destinations.

**Verify:** `go test ./internal/adapters/httpserver/ -run Themes`; preview check.

**Integrator for PR 7:** whoever lands last adds the script and style tags to `views/index.go.tpl` and runs the full preview journey on fixture data.

**PR 7 done when:** with `WRITERS_MAP=1`, the full UI works on fixture data.

---

## PR 8 — Verify and Launch (≈9 h) · needs: PR 6, PR 7

### 8a. Add resilience tests — 3 h
- [ ] Done

**Owns:** `internal/writermap/resilience_test.go`, `internal/adapters/httpserver/writermap_resilience_test.go`

**Acceptance**
- [ ] Wikidata down with no cache → tiles, day plan and an honest message.
- [ ] Summary table empty or `WRITERS_AI` off → panels show links only.
- [ ] A sync killed mid-crawl → it resumes, and searches keep working.
- [ ] 20 concurrent searches for a new destination → one Wikidata query.

**Verify:** `go test -race ./...`.

### 8b. Measure live behavior — 3 h
- [ ] Done

**Owns:** `tasks/writers-map-results.md`

- Local build limited to the Fly machine's memory, against real sources.

**Acceptance — record:**
- [ ] Full sync: time, bandwidth, peak memory, index size.
- [ ] Monthly delta size; database size after two runs (should not grow on unchanged data).
- [ ] Search latency warm and cold for Madeira, Lisbon, a non-Portuguese city, and a city with no writers.
- [ ] Panel open time (local read only).
- [ ] Live 40-match precision.
- [ ] 20 itinerary posts parsed.
- [ ] The owner sets the targets and the launch date.

**Verify:** results document.

### 8c. Run journeys and write docs — 3 h
- [ ] Done

**Owns:** `cmd/design-preview/fixture.go`, `README.md`, `docs/architecture.md`, `CHANGELOG.md`

- Journey at 375 and 1440 px and keyboard-only: search → pins → panel → post link → save → plan → day layer → writer itinerary → plan with these places.
- Final screenshots next to [the mockup](../docs/ideas/writers-map-mockup.png); differences other than the planned ones (plan, **Target Output**) are fixed or listed.

**Verify:** `go test ./cmd/design-preview`; `make test lint build`.

### 8d. Turn the flags on and release — owner · *after 8a–8c*
- [ ] Done

**Owns:** deploy configuration (`fly.toml` env or secrets)

- [ ] The owner reviews the release candidate.
- [ ] `WRITERS_SYNC=1` first; wait for the initial crawl to finish; then `WRITERS_MAP=1`.
- [ ] `WRITERS_AI` stays off until PR 9d.
- [ ] Sign-off recorded in `tasks/writers-map-results.md`.

---

## PR 9 — AI Summaries via LLM Gateway (≈9 h) · needs: PR 5, PR 6 · **9c needs the LLM Gateway project**

**Goal:** an English 2–3 sentence "what writers say" summary per writer pin, made monthly on the server. Not on the launch path: 9a–9b merge with `WRITERS_AI` off; 9c–9d wait for LLM Gateway.

### 9a. Cut passages around mentions — 2 h · *early, after PR 1*
- [ ] Done

**Owns:** `internal/writermap/passages.go`, `passages_test.go`, `internal/writermap/testdata/posts/` (add only)

- Pure function: post HTML + a place's aliases → passages (the sentence with the mention ± one sentence), language, post ref. Skips address, hotel and price-list lines. Max ~1,500 chars per post. Nothing is persisted.

**Acceptance**
- [ ] An "Address: … Chiado" line is skipped.
- [ ] A mention no longer in the post → no passage.
- [ ] Works on the EN and PT fixtures.

**Verify:** `go test ./internal/writermap/...`.

### 9b. Build the summary job and store — 3 h · *after 5d, 9a* · **integrator**
- [ ] Done

**Owns:** `internal/writermap/summaries.go`, `summaries_test.go`, `internal/adapters/sqlite/writersummaries.go`, `writersummaries_test.go`, `internal/adapters/sqlite/database.go` (table creation call), `cmd/web/main.go` (`wireSummaries`, only when `WRITERS_AI` is on)

- Table `writer_summaries` (QID, text, source post refs, model, generated at), upserted in place.
- Runs as the last step of the monthly job: places whose writer posts changed or that have no summary; per-run cap; resumable; posts fetched by id through the host limiter; text dropped after the call.
- Removed hosts and the override hide list delete or suppress summaries.

**Acceptance**
- [ ] With the fake summarizer, Madeira pins get summaries; a second run with no changes makes zero summarizer calls.
- [ ] Summarizer error → previous summary kept; run continues.

**Verify:** `go test -race ./internal/writermap/... ./internal/adapters/sqlite/...`.

**PR 9a–9b done when:** merged with `WRITERS_AI` off; the fake summarizer fills Madeira locally.

### 9c. Write the LLM Gateway adapter — 2 h · ***blocked on the LLM Gateway project***
- [ ] Done

**Owns:** `internal/adapters/api/llmgateway.go`, `llmgateway_test.go`

- Implements `Summarizer` against the gateway API (`LLM_GATEWAY_URL`, `LLM_GATEWAY_KEY`). Fixed prompt: "2–3 sentences in English; use only these passages; no facts not in them."
- Timeouts, one retry, request size cap.

**Verify:** `go test` with `httptest` covering success, timeout, 4xx/5xx.

### 9d. Check quality and switch AI on — 2 h · *after 9c* · owner sign-off
- [ ] Done

**Owns:** `tasks/writers-map-results.md`

**Acceptance — record:**
- [ ] 20 Madeira summaries hand-checked against their posts: invented facts (target 0), PT sources summarized correctly.
- [ ] Monthly run time and number of gateway calls.
- [ ] `docs/maintenance.md` updated with the gateway setup and how to hide a bad summary.
- [ ] Owner switches `WRITERS_AI=1`.

**PR 9 done when:** summaries show on live writer pins and the first monthly run is recorded.
