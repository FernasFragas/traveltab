# Writers' Map (Always Ready) — PRs and Subtasks

Source of truth for progress. Read [the plan](plan.md) (especially **Delivery** and **Rules for Agents**) and [the one-pager](../docs/ideas/community-map.md) first.

**Status:** PR 1 subtasks 1a–1e done, plus the contract gaps found on October 8, 2026 (`:dest` in map URLs, the base-place ports and fake, the panel template owner). The owner approved the contract on October 8, 2026, so it is frozen and **PR 1 is done**; PR 3, PR 4 and the early 6a/9a can start once PR 1 is merged. PR 2 is fully recorded (October 10, 2026): size and lookup pass, but coverage (24/49) and precision (31/40) fail their bars, The owner decided October 10, 2026: writer-mentioned places skip the type filter, precision ≥90% moves to a 6a acceptance, travellemming is `blocked`. **PR 5 is unblocked** (see [the results](writers-map-results.md)). **As of October 11, 2026:** PRs 3, 4 and 5 are done; **PR 6 is done** (6a precision review and 6b real-data run checked by the owner, October 13, 2026); **PR 7 is built (7a–7k) and integrated; `make lint` cannot run here (golangci-lint go1.26 vs module go1.27.2), and the open items are listed under PR 7**. Updated October 6, 2026: summaries replace excerpts, AI in MVP (PR 9), monthly sync, caches never expire, "Plan with these places" (7e). Updated October 7, 2026: UI matches [the mockup](../docs/ideas/writers-map-mockup.png) — new subtasks 1e, 3d, 7f, 7g, 7h; 7a and 7c extended. **Updated October 10, 2026: target is now [the Funchal mockup](<../docs/ideas/Funchal Travel Guide Dashboard.png>) for every city** — new 1f (contract change, approved by the owner October 10), 3e (satellite source, blocks 3d), 7i (hero + weather), 7k (travelers); 3d, 4a, 7a, 7c–7g reworked; 7h removed; hour estimates removed (see **Target Output** in the plan).

**How to read this file**
- Each **PR** merges on its own, CI green, with `WRITERS_MAP`, `WRITERS_SYNC` and `WRITERS_AI` **off**.
- Each **subtask** (1a, 1b …) can go to a different agent. Subtasks in one PR touch **disjoint files** unless marked *after*.
- **Owns** = the only files the subtask may edit. Shared files belong to the subtask marked **integrator**.
- Branches: `writers-map/pr<N>-<letter>` → merged into `writers-map/pr<N>` → PR to `main`.

**Conventions**
- Run from the repo root: `make build`, `make test`, `make lint`.
- **Tests never hit the network:** use the PR 1 fixture, the fake source, the fake summarizer, `httptest` and recorded responses.
- **No post text stored, logged or committed:** names, positions and AI summaries only.
- Past 5 files → split here first.

---

## PR 1 — Contracts, Data and Flags · needs: none

**Goal:** freeze everything parallel work depends on. No user-visible change.

### 1a. Write the source data and policy
- [x] Done

**Owns:** `writerdata/sources.json`, `writerdata/overrides.json`, `writerdata/embed.go`, `docs/maintenance.md`

- Sources: the 12 working blogs from [overlap results](blog-overlap-results.md), each with API base, language, `status` (`active`/`blocked`/`removed`) and date checked. VagaMundos, viajarentreviagens and lisbonlisboaportugal start `blocked`.
- Overrides: alias removals and merges (Clérigos ×3, the "Ribeira Grande" airport alias, "Nossa Senhora da Conceição"); destination areas for Madeira and São Miguel (default ≈40 km for cities); a per-QID summary hide list; a stoplist ("Fernandes", "Infante", "Achada" plus the research list).
- `docs/maintenance.md` gets a "Writers' map" section: user agent, limiter, robots.txt, 403 handling, names-only storage, AI summaries via LLM Gateway, monthly schedule, in-place updates, takedown steps.

**Verify:** `make build`; document review.

### 1b. Define the types, ports and contract
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

### 1c. Add the Madeira fixture, fake source and fake summarizer · *after 1b*
- [x] Done

**Owns:** `internal/writermap/writermaptest/fake.go`, `internal/writermap/writermaptest/fake_test.go`, `internal/writermap/writermaptest/testdata/madeira.json`

- About 15 places (base-only and writer pins), EN and PT writers, summaries on some writer pins and none on others, 2 itineraries, mixed source statuses.
- A fake `Summarizer` that returns a canned sentence or an error.
- The fake implements `WriterMapSource`, with switches for slow, partial, unavailable and empty.

**Verify:** `go test ./internal/writermap/...`.

### 1d. Add the feature flags · **integrator**
- [x] Done

**Owns:** `internal/config/env.go`, `internal/config/env_test.go`, `internal/config/flags.go`, `internal/config/flags_test.go`

- Add `WRITERS_MAP`, `WRITERS_SYNC` and `WRITERS_AI`, plus `LLM_GATEWAY_URL` / `LLM_GATEWAY_KEY` (unused until 9c); all default off/empty. Bring 1a–1c together in the working tree and run `make test lint build`; the owner commits and opens the PR.

**Verify:** `go test ./internal/config/...`; CI green.

### 1e. Add the fields the mockup needs · *contract change, before the owner review*
- [x] Done — [TDD evidence](../docs/tdd/1e-mockup-fields.tdd.md)

**Owns:** `internal/writermap/destinationdata.go`, `internal/writermap/destinationdata_test.go`, `internal/writermap/writermaptest/fake.go`, `internal/writermap/writermaptest/fake_test.go`, `internal/writermap/writermaptest/testdata/madeira.json`, `writerdata/sources.json`, `writerdata/embed_test.go`, `docs/writers-map-contract.md`

- GeoJSON properties gain `writers` (host array, for the "All writers" filter) and `has_summary` (for the Summaries layer).
- Place panel gains `area` (Wikidata P131 label, shown as *kind · area*) and per-writer `blog_name`, from a new `name` field in `writerdata/sources.json`.
- Itinerary days gain `distance_km` and `walk_minutes` (for the writer's walk card, which shows one day). Distance is straight-line between consecutive stops, as in the planner; real values come from 6a, the fixture fills them now.
- Plan stops gain a `writers_pick` flag (for 7g's "Writers' pick" chip): true when the stop's QID has a writer count of 1 or more. Defined in the contract; 7g builds it. *(Removed by 1f.)*
- Fixture filled for all new fields.

**Verify:** `go test ./internal/writermap/... ./writerdata/`.

### 1f. Add the fields the Funchal mockup needs · *contract change, owner review*
- [x] Done — [TDD evidence](../docs/tdd/1f-funchal-fields.tdd.md); contract approved October 10, 2026

**Owns:** `internal/writermap/destinationdata.go`, `internal/writermap/destinationdata_test.go`, `internal/writermap/writermaptest/fake.go`, `internal/writermap/writermaptest/fake_test.go`, `internal/writermap/writermaptest/testdata/madeira.json`, `docs/writers-map-contract.md`

- Place panel gains `opening_hours` (optional text, from Wikidata P3025/P8626/P8627) and `wikipedia_url` (optional).
- Plan stops gain `distance_km` from the previous stop (straight line, as in the planner; 0 for the first stop).
- **Remove the 1e fields the Funchal layout no longer uses:**
  - `writers_pick` on plan stops (the "Writers' pick" chip is gone)
  - GeoJSON `writers` (the "All writers" filter is gone) and `has_summary` (the Summaries chip is gone)
  - itinerary-day `distance_km` and `walk_minutes` (the walk card is gone)
- Fixture filled: some places with opening hours, some without.
- The contract was frozen on October 8, so this lands as its own small contract PR after owner review.

**Verify:** `go test ./internal/writermap/...`; **owner reviewed and approved the contract change October 10, 2026**.

**PR 1 done when:** contract doc (with 1e) approved; fixture and fakes build; flags default off. **Done:** contract approved October 8, 2026. The separate 1f contract change was approved October 10, 2026.

---

## PR 2 — Index Spike · needs: none · *parallel with PR 1*

### 2a. Measure index size, lookup speed and precision · single agent
- [x] Done

**Owns:** `tasks/writers-map-spike/` (throwaway), `tasks/writers-map-results.md`

- Crawl two full blogs (viajecomigo PT, saltinourhair EN), build the names-only tables in a temp SQLite file, run the Madeira Wikidata nearby query, match.

**Acceptance — record in the results doc:**
- [x] MB and rows per 1,000 posts, and the projection for ~20,400 posts (proposed bar ≤300 MB).
- [x] Madeira lookup time (proposed ≤200 ms).
- [x] 40-match precision with Wikidata aliases, against the research's 37/40 (proposed ≥90%). *Recorded October 10: 31/40, fails the bar.*
- [x] How many of the 55 research places (≥3 blogs) Wikidata nearby plus filters still returns. *Recorded October 10: the re-run finds 49; 24 pass the type filter, 22 are inside the cap.*
- [x] **Verdict:** keep the schema, cap sub-spans, or add OSM names. **PR 5 waits for this.**

**Verify:** results document reviewed by the owner (approved October 10, 2026).

---

## PR 3 — Map Shell Behind the Flag · needs: PR 1

**Goal:** with `WRITERS_MAP=1`, every destination shows the new map with fixture data. With it off, Waze is unchanged.

### 3a. Write the core map JS and CSS
- [x] Done — [TDD evidence](../docs/tdd/3a-map-core.tdd.md)

**Owns:** `public/redesign/writers-map.js`, `public/redesign/writers-map.css`

- Leaflet (unpkg + SRI) and OSM tiles with attribution; tile URL read from a data attribute.
- One map instance across HTMX swaps.
- Layer registry and events as in the contract.
- List alternative reachable by keyboard.
- Status area for loading, partial and unavailable states.

**Verify:** exercised by 3c.

### 3b. Wire the template and fixture endpoints · **integrator**
- [x] Done — [TDD evidence](../docs/tdd/3b-map-shell.tdd.md)

**Owns:** `views/map_card.go.tpl`, `views/index.go.tpl`, `views/place_panel.go.tpl` (minimal; 7a builds it out), `internal/adapters/httpserver/writermap.go`, `internal/adapters/httpserver/writermap_test.go`, `internal/adapters/httpserver/server.go`, `cmd/web/main.go`

- Flag on: the map shell with all contract slots, plus `hx-trigger="load"`. Flag off: today's Waze markup, byte-for-byte.
- Contract endpoints, served from the PR 1 fake. `:dest` is the trip slug plus `?lat=…&lon=…` (see **Destination in map URLs** in the contract); the shell's data attributes carry the full URLs.
- Minimal `place_panel.go.tpl`: the contract's panel fields and writer links, no mockup styling.
- Keep "View larger map".

**Verify:** `go test ./internal/adapters/httpserver/...` (flag on and off; bad slug → 404; bad coordinates → 400; escaping).

### 3c. Add the design-preview scenario
- [x] Done

**Owns:** `cmd/design-preview/writersmap.go`, `cmd/design-preview/writersmap_test.go`

- Scenarios: Madeira (writers), a base-only city, sources unavailable. Local fixture tiles.

**Acceptance**
- [x] 375 and 1440 px.
- [x] Two destination swaps leave one map.
- [x] Keyboard can reach the list.

**Verify:** `go test ./cmd/design-preview`.

### 3e. Choose a free satellite/terrain basemap · **blocks 3d**
- [x] Done

**Implementation (2026-10-11):** EOX 2016 imagery selected and verified in the preview; terms are
recorded in `docs/maintenance.md` and accepted by the owner on 2026-10-11.

**Owns:** `docs/maintenance.md` (basemap section), `public/redesign/writers-map.js` (tile URL default and attribution only, after 3a)

- Check the terms of candidate imagery sources (Esri World Imagery, EOX Sentinel-2 cloudless, others found) for free use on this site: licence, attribution, request limits, key needed or not.
- Free sources only: no paid plans, no keys tied to billing.
- Record the choice and its attribution in `docs/maintenance.md`; the tile URL stays configurable with OSM as the fallback.
- **No source fits → stop and ask the owner.** Never ship a source without clear terms.

**Verify:** owner reviews the recorded terms; preview shows the imagery with its attribution.

### 3d. Lay out the card like the Funchal mockup · *after 3a, 3b, 3e*
- [x] Done

**Owns:** `public/redesign/writers-map-layout.css`, `views/map_card.go.tpl` (layout markup only, with 3b's agreement)

- Header: "Writers' Map" with an **info icon** (tooltip explains the four layers). No status pill: the `status` slot is a short inline message, shown only for partial or unavailable data.
- ≥1024 px: map ≈65% and `panel` slot side by side, equal height. <1024 px: panel stacks under the map as a card.
- Legend-chip row top-right above the map (`layers` slot).
- Basemap from 3e. Map chrome: zoom and scale bar only.
- Empty panel state: "Pick a place on the map".

**Acceptance**
- [x] Preview at 375 and 1440 px matches the Funchal mockup's layout (not its photos).
- [x] No horizontal scroll at 375 px.

**Verify:** preview check side by side with the Funchal mockup.

**PR 3 done when:** both flag states pass tests; preview shows the shell in the Funchal layout on the satellite basemap.

---

## PR 4 — Base Layer · needs: PR 1

### 4a. Write the Wikidata nearby adapter
- [x] Done — [TDD evidence](../docs/tdd/4a.tdd.md)

**Owns:** `internal/adapters/api/wikidatanearby.go`, `wikidatanearby_test.go`, `internal/adapters/api/testdata/wikidata_*.json`

- Implements `application.BasePlaceSource`.
- SPARQL `wikibase:around`: ≈40 km for cities (overrides first for islands/regions), visitable-class filter, infrastructure excluded, capped at ~200 ranked by sitelinks (writer count re-ranks later in 6a).
- The type filter and cap apply to **base pins only**. The unfiltered nearby list (P31 types kept) is also returned as the gazetteer for writer matching, so 6a can keep writer-mentioned places of any type (owner decision, October 10, 2026; see [the results](writers-map-results.md)).
- Visitable classes come from the planner's generated list (`planner.PlaceTypes`, skipping `never`), not a second list.
- Returns en/pt labels and aliases, English description, P18 image with Commons credit (reuse `wikimedia.go` parsing), sitelinks, the English Wikipedia URL, and opening hours when present (P3025 open days with P8626/P8627 opening and closing times, as one short line like "Open daily · 9:00 – 18:00").
- Retry-After honored; identified user agent.

**Acceptance**
- [x] Madeira fixture's base pins include Pico Ruivo and Cabo Girão and exclude the airport; Câmara de Lobos (a municipality/freguesia) is in the gazetteer but not a base pin.
- [x] Missing image, description, Wikipedia link or opening hours → field left empty.

**Verify:** `go test ./internal/adapters/api/...`.

### 4b. Add the cache decorator
- [x] Done — [TDD evidence](../docs/tdd/4b.tdd.md)

**Owns:** `internal/adapters/sqlite/wikidatanearby.go`, `wikidatanearby_test.go`

- Implements `application.BasePlaceSource` and `application.BasePlaceRefresher`.
- Cache in `source_cache` with **no expiry**, following `sourcecache.go`: cached → no call; error with no cache → empty result plus an error the caller can log.
- `Refresh(ctx, dest)` re-runs the query and overwrites the row in place (called by the monthly job in 5d).
- Singleflight per destination.

**Verify:** `go test -race ./internal/adapters/sqlite/...` (20 concurrent callers → 1 query).

### 4c. Warm from search suggestions · **integrator**
- [x] Done — [TDD evidence](../docs/tdd/4c.tdd.md)

**Owns:** `internal/adapters/httpserver/suggest.go`, `suggest_test.go`, `cmd/web/main.go` (wiring only)

- Top suggestion → background base query: deduplicated, at most one per client per 2 s, only when `WRITERS_MAP` is on, never delaying the suggestion response.

**Verify:** `go test ./internal/adapters/httpserver/...` with `writermaptest.FakeBaseSource`, which counts calls.

**PR 4 done when:** cached base data is available behind the flag; suggestion latency is unchanged.

---

## PR 5 — Writers Index and Monthly Sync · needs: PR 1, **PR 2 verdict**

### 5a. Create the index tables and store
- [x] Done — [TDD evidence](../docs/tdd/5a.tdd.md)

**Owns:** `internal/adapters/sqlite/writerindex.go`, `writerindex_test.go`

- Tables `writer_posts`, `writer_names`, `writer_days` and `writer_sync`, using the schema from the PR 2 verdict, created with `CREATE TABLE IF NOT EXISTS` from a function the integrator calls.
- Store methods: upsert a post with its names and days in one transaction, delete a post, purge a host, look up by phrases, read and write the sync cursor.

**Acceptance**
- [x] Re-indexing a post replaces its rows and never duplicates them.
- [x] Purging a host removes every row for it.
- [x] An existing database file is untouched otherwise.

**Verify:** `go test ./internal/adapters/sqlite/...`.

### 5b. Extract name phrases and day headings
- [x] Done — [TDD evidence](../docs/tdd/5b.tdd.md)

**Owns:** `internal/writermap/extract.go`, `extract_test.go`, `internal/writermap/testdata/posts/`

- Pure function: HTML in → phrases (folded, original-case flag, positions, sub-spans of up to 5 words), day sections, and an itinerary flag out. No text is returned.

**Acceptance**
- [x] "Miradouro do Pico dos Barcelos" yields both the full name and "Pico dos Barcelos".
- [x] The EN and PT itinerary fixtures split into the right days; a listicle isn't flagged as an itinerary.

**Verify:** `go test ./internal/writermap/...`.

### 5c. Write the WordPress client and host limiter
- [x] Done — [TDD evidence](../docs/tdd/5c.tdd.md)

**Owns:** `internal/adapters/api/wordpress.go`, `wordpress_test.go`, `internal/adapters/api/hostlimiter.go`, `hostlimiter_test.go`

- Listing by page with `modified_after`, an id-only sweep, fetch by id (`_fields=content`), robots.txt check (cached).
- Per-host limiter ≥1.5 s, shared by the sync and the summary job (PR 9).
- 403 returns a typed `ErrBlocked`; one retry with backoff on timeout.

**Verify:** `go test -race` with `httptest` covering pages, 400 past the last page, 403, timeout and robots disallow.

### 5d. Build the monthly sync job · *after 5a–5c* · **integrator**
- [x] Done — [TDD evidence](../docs/tdd/5d.tdd.md)

**Owns:** `internal/writermap/sync.go`, `sync_test.go`, `internal/adapters/sqlite/database.go` (table creation call), `cmd/web/main.go` (start only when `WRITERS_SYNC` is on)

- Initial crawl, then one **monthly** run: changed posts (`modified_after`), deleted-posts sweep, Wikidata `Refresh` for cached destinations, then a hook for the summary step (PR 9). Single-run lock, resumable cursor, one page in memory at a time, off-peak.
- All writes are upserts in place; no history rows.
- `ErrBlocked` → source marked blocked and skipped; `removed` sources purged.
- One progress line per host per run.

**Acceptance**
- [x] An interrupted crawl resumes from its cursor.
- [x] Edited posts are re-indexed; deleted posts are dropped after the sweep.
- [x] Two consecutive runs over unchanged data leave row counts and file size unchanged.

**Verify:** `go test -race ./internal/writermap/...` with the 5c fake server; CI green.

**PR 5 done when:** a two-blog sync runs locally with `WRITERS_SYNC=1` and resumes after being killed.

---

## PR 6 — Matching and the Real Source · needs: PR 4, PR 5

> **6a can start right after PR 1** against the fixture, and waits in its branch until PR 6 opens. On-demand excerpts were dropped on October 6 — summaries (PR 9) replace them.

### 6a. Write the matcher and itinerary assembly · *early*
- [x] Done — code merged (#31), [TDD evidence](../docs/tdd/6a.tdd.md); precision review checked by the owner (2026-10-13)

**Owns:** `internal/writermap/match.go`, `match_test.go`, `internal/writermap/itinerary.go`, `itinerary_test.go`

- Pure function: places (aliases) plus index rows go in; mentions per place, writer counts and itineraries come out.
- Rules from the research: case rule, stoplist, nested-name suppression by position, overrides.
- Re-rank places: writer count first, then sitelinks; keep the ~200 cap.
- **Writer-mentioned places skip the type filter**, like planner boosts; a Deny type still drops them (owner decision, October 10, 2026).
- **Destination context rule:** a mention counts only when its post (or section) is about the destination, not just a same-name place elsewhere in Portugal. Use indexed destination phrases (e.g. "Madeira", "Funchal") per post.

**Acceptance**
- [x] "São Pedro de Alcântara" doesn't count as "Alcântara".
- [x] "praia" in running text doesn't match.
- [x] The three Clérigos entries merge into one.
- [x] Itinerary days are ordered by first mention.
- [x] São Jorge, Monte, Ponta do Sol and Teleférico do Funchal show as writer pins although their types aren't in `PlaceTypes`.
- [x] "Bom Jesus do Monte" in a post about Braga doesn't count for Monte (Madeira).
- [x] **Precision:** 40 random first mentions on the real Madeira gazetteer, hand-reviewed, ≥90% right place (2a baseline: 31/40, 34/40 with the alias fixes).

**Verify:** `go test ./internal/writermap/...`.

### 6b. Compose the real source and wire it · **integrator**
- [x] Done — code merged (#31), [TDD evidence](../docs/tdd/6b.tdd.md); real-data Madeira run checked by the owner (2026-10-13; see also [post-limit L4](post-limit-results.md#coverage-l4) and `docs/tdd/nearby-rings.tdd.md` R6)

**Owns:** `internal/writermap/source.go`, `source_test.go`, `internal/adapters/sqlite/writerdestination.go`, `internal/adapters/httpserver/writermap.go`, `internal/adapters/httpserver/server.go`, `cmd/web/main.go`

- `WriterMapSource` built from base (PR 4), index store (PR 5) and matcher; `Place()` reads `writer_summaries` if the table has a row (empty until PR 9).
- Per-destination result stored with no expiry, upserted in place, recomputed after each monthly run.
- Replaces the fake when `WRITERS_MAP` is on.

**Acceptance**
- [x] No blog request during `Destination()` or `Place()` (test asserts zero HTTP calls).
- [x] The fake is still available to the preview.

**Verify:** `go test -race ./...`; local run with both flags on shows Madeira on real data.

**PR 6 done when:** Madeira works end to end locally on real data (links, no summaries yet).

---

## PR 7 — Map UI Features · needs: PR 3 · *runs on the fake source, parallel with PRs 4–6*

### 7a. Build pins and the place panel
- [x] Done — [TDD evidence](../docs/tdd/7a.tdd.md)

**Owns:** `public/redesign/map-panel.js`, `public/redesign/map-panel.css`, `views/place_panel.go.tpl` (builds out 3b's minimal template), `internal/adapters/httpserver/placepanel_test.go`, `views/index.go.tpl` (the `map-panel` script and stylesheet tags only)

- Writer pins drawn above base pins.
- **Pin look:** writer pins orange with their writer count, notable (base) places blue; big places labelled. Plan pins (7b) are green numbered and itinerary pins (7c) purple numbered.
- **Panel order:** photo + close · name · *kind · area* · writer monograms (initials, colour from host; "+N") with "Mentioned by N writers" · **AI summary** (labelled) · description · **opening hours** (when present) · **View on Wikipedia** · **Save to plan** (primary, `data-save-qid`) · **View writers' posts (N)**.
- "View writers' posts (N)" expands the full list: post title, blog name, language chip, external-link icon. **No quoted blog text.**
- Base-only panel: description, photo with credit, opening hours, Wikipedia link, Save to plan.

**Acceptance**
- [x] Every writer is listed with a post link once the list is expanded.
- [x] A writer pin without a summary shows no empty summary box.
- [x] A place without opening hours or Wikipedia link shows no empty row.
- [x] A base-only destination shows "No writer mentions here yet".

**Verify:** `go test ./internal/adapters/httpserver/ -run PlacePanel`; preview at 375 and 1440 px.

### 7b. Draw the day plan layer
- [x] Done — [TDD evidence](../docs/tdd/7b.tdd.md)

**Owns:** `public/redesign/map-dayplan.js`, `views/itinerary_body.go.tpl`, `views/trip_card.go.tpl`, `views/index.go.tpl` (the `map-dayplan` script tag only)

- The plan response emits stop coordinates per day; the script fires `travelmap:plan-updated` and draws green numbered pins per day with a day toggle.

**Acceptance**
- [x] Generating, regenerating and clearing a plan leave no duplicate layers. *(`map-dayplan.browser.mjs`, `pr7-acceptance.browser.mjs`)*
- [x] Works on base-only destinations. *(`pr7-acceptance.browser.mjs`, Porto at 375 and 1440 px)*

**Verify:** template test for the emitted data; preview check.

### 7c. Add writers' itinerary layers
- [x] Done — [TDD evidence](../docs/tdd/7c.tdd.md)

**Owns:** `public/redesign/map-itineraries.js`, `public/redesign/map-itineraries.css`, `views/index.go.tpl` (the `map-itineraries` script and stylesheet tags only)

- **Data:** the GeoJSON's top-level `itineraries` (`writer_host`, `post_url`, `post_title`, `language`, `days[].qids`), served since the contract change approved October 11, 2026. Register the layer as `itineraries`.
- One writer's itinerary is drawn at a time: purple numbered pins in order and a line between them, labelled as the writer's route and linked to the post.
- The **Writers' itineraries** chip (7f) shows the layer; when there are several, a small picker in the `layers` slot chooses one. Hidden when there are none.

**Verify:** preview check with fixture itineraries.

### 7d. Add "Save to plan"
- [x] Done — [TDD evidence](../docs/tdd/7d.tdd.md)

**Owns:** `public/redesign/map-saved.js`, `views/index.go.tpl` (the `map-saved` script tag only)

- Binds `data-save-qid` buttons; saved QIDs kept in `localStorage` (try/catch). The button toggles to "Saved".
- On the next plan request the saved QIDs are added as `include` (via `htmx:configRequest`), so 7e seeds them into the days.
- A line next to the plan form: "N saved places will be included", with a clear link. *Built in the map card's `saved` slot, since the form is in another card (owner decision, October 11, 2026); 7g styles it.*

**Acceptance**
- [x] Saves survive a reload. *(`map-saved.test.mjs`, `pr7-integration.browser.mjs`)*
- [x] Blocked storage hides the buttons and the line. *(`pr7-acceptance.browser.mjs`, `localStorage` made to throw)*
- [x] Saving makes no server call; only the plan request carries the QIDs. *(`pr7-acceptance.browser.mjs`: 0 requests on save, `include` only on `/plan`)*

**Verify:** preview check, including a private window.

### 7e. Seed saved places into the plan (`Include`)
- [x] Done — [TDD evidence](../docs/tdd/7e.tdd.md)
- [x] Follow-up: .ics/.kml links and handlers carry `include` (exports only; push URL and `/trip/:slug` page unchanged) — [TDD evidence](../docs/tdd/exports-include.tdd.md)

**Owns:** `internal/planner/types.go`, `internal/planner/build.go` (seeding only), `internal/planner/build_test.go`, `internal/adapters/httpserver/trip.go` (query parsing in `planTrip`), `trip_test.go`, `views/itinerary_body.go.tpl` (the "also suggested" block only; after 7b, before 7g)

- `planner.Request` gains `Include []Place` (max ~12); `Build` places them into days first (nearest together where possible), then fills with nearby places as today.
- `/plan` accepts the QIDs in the query string (`include`, repeated: `include=Q1&include=Q2`, as 7d sends them), resolved through `WriterMapSource.Place` (the fake in PR 7, the real source after PR 6); unknown QIDs are ignored. The URL is shareable.
- With `WRITERS_MAP` off, `include` is ignored and plans are unchanged.

**Acceptance**
- [x] All included places appear in the plan; requests without `Include` produce today's plans byte-for-byte. *(`TestBuild_IncludedPlacesAppearInTheDays`, `TestBuild_WithoutIncludeGivesTheUnseededPlan`)*
- [x] More included places than days can hold → extra ones listed as "also suggested", not dropped silently. *(`TestBuild_MoreIncludedPlacesThanDaysCanHoldAreAlsoSuggested`; preview: `pr7-acceptance.browser.mjs`)*

**Verify:** `go test ./internal/planner/... ./internal/adapters/httpserver/...`; preview check.

### 7f. Add the legend chips · *after 7a*
- [x] Done — [TDD evidence](../docs/tdd/7f.tdd.md)

**Owns:** `public/redesign/map-layers.js`, `public/redesign/map-layers.css`, `views/index.go.tpl` (the `map-layers` script and stylesheet tags only)

- Four chips that toggle their layer: **Notable places** (blue) · **Writers' pins** (orange) · **Your day plan** (green; disabled until a plan exists) · **Writers' itineraries** (purple; hidden when there are none).
- Chip state is buttons with `aria-pressed`; remembered per tab in `sessionStorage` (try/catch).

**Acceptance**
- [x] Every chip toggles its layer without redrawing the others. *(`pr7-integration.browser.mjs`)*
- [x] All chips off → empty map, no error. *(`pr7-integration.browser.mjs`)*

**Verify:** preview check.

### 7g. Rework the itinerary rows · *after 7b, 7e, 7i*
- [x] Done — [TDD evidence](../docs/tdd/7g.tdd.md)
- Gap closed: short stop description (Wikidata English description, or the writers' map's) — [evidence](../docs/tdd/descriptions.tdd.md).

**Owns:** `views/itinerary_body.go.tpl` (after 7b), `internal/adapters/httpserver/itinerary_test.go`, `public/redesign/map-saved.css`, `views/index.go.tpl` (the `map-saved` stylesheet tag only)

- Buttons: **Export to calendar** (existing ICS) and **View on map** (scrolls to the map and turns on **Your day plan**).
- Per day: number, date, **max temperature and condition** (from 7i's daily forecast), km walking, collapse toggle.
- Per stop: number, Commons photo, name, *kind · distance from the previous stop* (`distance_km`, 1f), short description.
- Style 7d's saved line (`.writers-map-saved` in the map card's `saved` slot) in `map-saved.css`.
- **View on map** clicks the **Your day plan** chip when it isn't pressed, so 7f's remembered state stays in step. Don't call `TravelMap.setLayerEnabled` directly: 7f re-applies its own state on the next sync.

**Acceptance**
- [x] Days without forecast data show no temperature, not "0°". *(`TestItinerary_DaysWithoutForecastDataShowNoTemperature`)*
- [x] The first stop of a day shows no distance. *(`TestItinerary_FirstStopShowsNoDistance...`)*

**Verify:** `go test ./internal/adapters/httpserver/ -run Itinerary`; preview check.

### 7i. Build the hero and the richer weather card
- [x] Done — [TDD evidence](../docs/tdd/7i.tdd.md)
- Gap closed: hero line falls back to the city's Wikidata description when there is no tagline — [evidence](../docs/tdd/descriptions.tdd.md).
- Phone: the hero line shows below the photo, above the weather card — [evidence](../docs/tdd/hero-phone-line.tdd.md).
- Hero contrast: narrower desktop wash (flat to 40%), dark label restored on phones, 2-line clamp dropped on the hero line and stop descriptions — [evidence](../docs/tdd/hero-overlay-clamp.tdd.md).

**Owns:** `views/weather_display.go.tpl`, `views/weather_card.go.tpl`, `public/redesign/destination.css`, `views/index.go.tpl` (header order only), `internal/adapters/api/openweather.go`, `openweather_test.go`, `internal/adapters/api/openmeteo_forecast.go`, `openmeteo_forecast_test.go`, `internal/adapters/httpserver/presentation.go`, `presentation_test.go`

- **Header:** brand · search · tabs (search moves before the tabs).
- **Hero:** full-width photo; city, *region, country* (region from the geocoder's `admin1`), one-line description (manifest tagline, else the Wikidata description).
- **Weather card over the hero:** temperature, condition, **feels like** and **wind** (already parsed in `openweather.go`), humidity, **sunset**.
- **4-day forecast:** day, icon, high/low from Open-Meteo daily (`temperature_2m_max`, `temperature_2m_min`, `weather_code`; free), added next to today's `precipitation_sum`.
- Phone: the weather card stacks under the photo.

**Acceptance**
- [x] Any city shows region and country; a city without a region shows country only. *(`TestDestinationPresentation_ShowsRegionAndCountryFromTheGeocoder`, `..._NoRegionShowsCountryOnly`)*
- [x] Forecast unavailable → the card shows current weather only, no empty row. *(`TestWeatherDisplay_NoForecastAndNoSunsetRenderNoEmptyRow`)*

**Verify:** `go test ./internal/adapters/api/... ./internal/adapters/httpserver/...`; preview at 375 and 1440 px.

### 7k. Add travelers to the plan form · *after 7b, 7e*
- [x] Done — [TDD evidence](../docs/tdd/7k.tdd.md)

**Owns:** `views/trip_card.go.tpl` (travelers field only, after 7b), `internal/planner/stays.go`, `internal/planner/stays_test.go`, `internal/adapters/httpserver/trip.go` (travelers parsing only, after 7e)

- **Travelers** select (1–8, default 2) next to Start date and Days. The form has no end date; keep Days (owner decision, October 11, 2026).
- Passed to `BookingURL` as `group_adults`; kept in the shareable `/plan` URL.

**Acceptance**
- [x] The Booking link carries the chosen number of travelers. *(`TestBookingURL_CarriesTheTravelers`; preview shows `group_adults=2`)*
- [x] A missing or invalid value falls back to 2. *(`TestPlanRoute_MissingOrInvalidTravelersFallBackToTwo`)*

**Verify:** `go test ./internal/planner/... ./internal/adapters/httpserver/...`.

**Integrator for PR 7:** whoever lands last adds the script and style tags to `views/index.go.tpl` and runs the full preview journey on fixture data.

**Integrator result (October 11, 2026):** evidence in [pr7-integration.tdd.md](../docs/tdd/pr7-integration.tdd.md). Full journey runs clean on fixture data at 375 and 1440 px. Not in this PR (open): the hero falls back to the manifest tagline only (no Wikidata description), stops show no short description (`planner.Place` has none), and `include` / `travelers` are not carried on the `/trip/:slug` re-plan or the `.ics` / `.kml` exports.

**Follow-up (October 11, 2026):** the descriptions and the exports gap are closed (links under 7e, 7g, 7i); `/trip/:slug` still re-plans without `include` (owner decision). Browser regressions found after integration (stale `scenarios.mjs` weather check, hero contrast, mobile-menu tab order, clamped text under text spacing) are fixed against a HEAD baseline — [evidence](../docs/tdd/pr7-browser-regressions.tdd.md).

**PR 7 done when:** with `WRITERS_MAP=1`, the full UI works on fixture data and matches the Funchal mockup's layout.

---

## PR 8 — Verify and Launch · needs: PR 6, PR 7

### 8a. Add resilience tests
- [x] Done

**Owns:** `internal/writermap/resilience_test.go`, `internal/adapters/httpserver/writermap_resilience_test.go`

**Acceptance**
- [x] Wikidata down with no cache → tiles, day plan and an honest message.
- [x] Summary table empty or `WRITERS_AI` off → panels show links only.
- [x] A sync killed mid-crawl → it resumes, and searches keep working.
- [x] 20 concurrent searches for a new destination → one Wikidata query.

**Verify:** `go test -race ./...`. [Evidence](../docs/tdd/8a.tdd.md).

### 8b. Measure live behavior
- [ ] Partial measurements recorded in [PR 8 live verification](writers-map-results.md#pr-8-live-verification); an unchanged-data rerun, current 40-match precision, 20 itinerary posts and Fly-sized VM verification remain open

**Owns:** `tasks/writers-map-results.md`

- Local build limited to the Fly machine's memory, against real sources.

**Acceptance — record:**
- [ ] Full sync: time, bandwidth, peak memory, index size.
- [ ] Monthly delta size; database size after two runs (should not grow on unchanged data).
- [x] Search latency warm and cold for Madeira, Lisbon, a non-Portuguese city, and a city with no writers. *Lisbon's full cold time exceeded the 180-second probe; see results.*
- [x] Panel open time (local read only).
- [ ] Live 40-match precision.
- [x] Share of Madeira and Lisbon places with Wikidata opening hours.
- [ ] 20 itinerary posts parsed.
- [ ] The owner sets the targets and the launch date.

**Verify:** results document.

### 8c. Run journeys and write docs
- [x] Done — [TDD and browser evidence](../docs/tdd/8c.tdd.md); `make test lint build` passes with a temporary Go 1.27-built linter

**Owns:** `cmd/design-preview/fixture.go`, `README.md`, `docs/architecture.md`, `CHANGELOG.md`

- Journey at 375 and 1440 px and keyboard-only, on at least two cities: search → hero + weather → pins → panel → writers' posts → save to plan → plan with travelers → day layer → writer itinerary.
- Final screenshots next to [the Funchal mockup](<../docs/ideas/Funchal Travel Guide Dashboard.png>); differences other than the planned ones (plan, **Target Output**) are fixed or listed.

**Verify:** `go test ./cmd/design-preview`; `make test lint build`.

### 8d. Turn the flags on and release — owner · *after 8a–8c*
- [ ] Done

**Owns:** deploy configuration (`fly.toml` env or secrets)

- [ ] The owner reviews the release candidate.
- [ ] `WRITERS_SYNC=1` first; wait for the initial crawl to finish; then `WRITERS_MAP=1`.
- [ ] `WRITERS_AI` stays off until PR 9d.
- [ ] Sign-off recorded in `tasks/writers-map-results.md`.

---

## PR 9 — AI Summaries via LLM Gateway · needs: PR 5, PR 6 · **9c needs the LLM Gateway project**

**Goal:** an English 2–3 sentence "what writers say" summary per writer pin, made monthly on the server. Not on the launch path: 9a–9b merge with `WRITERS_AI` off; 9c–9d wait for LLM Gateway.

### 9a. Cut passages around mentions · *early, after PR 1*
- [ ] Done

**Owns:** `internal/writermap/passages.go`, `passages_test.go`, `internal/writermap/testdata/posts/` (add only)

- Pure function: post HTML + a place's aliases → passages (the sentence with the mention ± one sentence), language, post ref. Skips address, hotel and price-list lines. Max ~1,500 chars per post. Nothing is persisted.

**Acceptance**
- [ ] An "Address: … Chiado" line is skipped.
- [ ] A mention no longer in the post → no passage.
- [ ] Works on the EN and PT fixtures.

**Verify:** `go test ./internal/writermap/...`.

### 9b. Build the summary job and store · *after 5d, 9a* · **integrator**
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

### 9c. Write the LLM Gateway adapter · ***blocked on the LLM Gateway project***
- [ ] Done

**Owns:** `internal/adapters/api/llmgateway.go`, `llmgateway_test.go`

- Implements `Summarizer` against the gateway API (`LLM_GATEWAY_URL`, `LLM_GATEWAY_KEY`). Fixed prompt: "2–3 sentences in English; use only these passages; no facts not in them."
- Timeouts, one retry, request size cap.

**Verify:** `go test` with `httptest` covering success, timeout, 4xx/5xx.

### 9d. Check quality and switch AI on · *after 9c* · owner sign-off
- [ ] Done

**Owns:** `tasks/writers-map-results.md`

**Acceptance — record:**
- [ ] 20 Madeira summaries hand-checked against their posts: invented facts (target 0), PT sources summarized correctly.
- [ ] Monthly run time and number of gateway calls.
- [ ] `docs/maintenance.md` updated with the gateway setup and how to hide a bad summary.
- [ ] Owner switches `WRITERS_AI=1`.

**PR 9 done when:** summaries show on live writer pins and the first monthly run is recorded.
