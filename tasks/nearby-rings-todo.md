# Ring-by-ring nearby loading — Tasks

Source of truth for progress. Read [the plan](nearby-rings-plan.md) first, especially **Decisions**.

**How to read this file**
- Tasks run in order R1 → R6 by **one agent**: R2 and R3 edit the same files.
- **Owns** = the only files the task may edit. Keep Owns disjoint from the Writers' Map tasks.
- Start from an up-to-date `main` (`git fetch` and check `git status -sb`).
- TDD per [AGENTS.md](../AGENTS.md) §5: see each test fail for the intended reason first.
- Live Wikidata calls only in R1 and R6 (measurement), sequential, with the app's `UserAgent`. `make test` stays offline.
- No commits or pushes. The owner commits.
- Start from the working tree as it is: the grouped query (old N2) is replaced in R2; the cache `v3` bump (old N5) stays.

**Targets:** first pins in **< 5 s**; each page **< 5 s and < 1 MB**; no request over 30 s. Funchal keeps its writer matches.

**User journeys**
- *As a traveller opening Lisbon, London, Paris, New York or Tokyo, I want the closest notable pins to appear first and more to fill in, so that the map is useful straight away.*
- *As a traveller in Funchal, I want the same writer matches as before, so that the fix costs me nothing.*
- *As a traveller who leaves the page mid-load, I want the load to carry on, so that the next visit is faster.*

---

### R1. Measure page time, page size and the floor
- [x] Done

**Owns:** `docs/tdd/nearby-rings.tdd.md` (new)

- For each city, time ring 1 (0–5 km) page 1 at 100 items, ordered by sitelinks. Record time, size, items.
- Time one deeper page (page 2) and the outermost ring page 1 for Paris and Tokyo.
- Try the sitelinks floor 1 and 2: how many pages until the 200 pins are filled?
- If a dense ring page takes over ~10 s, try 50 items, then labels in a second query (plan, pre-approved).

**Verify:** the evidence doc has the table, the chosen page size and floor, and the stop rule that fills 200 pins.

### R2. Ring page query
- [x] Done

**Owns:** `internal/adapters/api/wikidatanearby.go`, `wikidatanearby_test.go`, `internal/adapters/api/testdata/wikidata_*.json`

- Replace the grouped single query with a ring-page query: inner and outer radius, `ORDER BY DESC(?sitelinks), ?item`, `LIMIT` and `OFFSET`, sitelinks floor.
- Pages return plain rows; fold them into places with the existing merge logic.
- Constants next to `wikidataNearbyRadius`: ring step (5), page size, floor.
- Tests: the query sent has the radii, order, limit, offset and floor; a name with `|` or `,` survives; an item with several types, aliases and hours is one place; recorded fixtures only.

**Verify:** `go test ./internal/adapters/api/ -run 'WikidataNearby|NearbyPlaces'`

### R3. Ring loader and stop rule
- [x] Done

**Owns:** `internal/adapters/api/wikidatanearby.go`, `wikidatanearby_test.go`, `internal/writermap/source.go` (the `BasePlaceSource` interface only)

- Loop rings 5, 10 … 40 km and pages of 100, one at a time, nearest first.
- Stop rules: a short or empty page ends the ring; the 200 pins filled with notable items, or a thin ring, ends the load.
- Report progress after each page (partial `BasePlaces`) through a callback or channel on the source interface.
- Tests: ring order; page paging; each stop rule; a page failing mid-load returns what was loaded plus an error; 429/503 retry still applies.

**Verify:** `go test ./internal/adapters/api/ ./internal/writermap/ -run 'WikidataNearby|Rings|BasePlace'`

### R4. Partial results and background loading
- [x] Done

**Owns:** `internal/adapters/sqlite/wikidatanearby.go`, `wikidatanearby_test.go`, `internal/application/writermap.go`

- The cache decorator stores a **partial** result after each page (marked incomplete) and caches the **final** result as complete.
- The fetch runs **detached from the request context**, one flight per destination (reuse `flights`).
- A request during a load gets the partial result and a "loading" flag; an interrupted load resumes from the stored page on the next visit.
- Writer matching (`recompute`) runs once at the end, not on partial results.
- Tests: partial stored after each page; incomplete is never served as complete; closing the request doesn't abort the load; resume after a failure; one flight per destination.

**Verify:** `go test ./internal/adapters/sqlite/ ./internal/writermap/`

### R5. Map endpoint, page polling and contract
- [x] Done

**Owns:** `internal/adapters/httpserver/writermap.go`, `writermap_test.go`, `public/redesign/writers-map.js`, `tasks/redesign/checks/writers-map-core.test.mjs`, `docs/writers-map-contract.md` (the `BasePlaceSource` bullet only)

- `/map/<dest>.geojson` returns **200 with the pins so far** and `loading: true` while the load runs, `loading: false` when done. No 503 for a slow load.
- `writers-map.js`: poll while `loading` is true, add pins as they arrive, show "Loading more places…", then clear. Keep the existing error text for real failures.
- Contract: `Gazetteer` keeps nearby items with at least the floor sitelinks, loaded in 5 km rings; writer counts appear once loading finishes.
- Tests: Go handler test for `loading` true and false; JS test that polling adds pins and stops when done.
- Browser change: runtime check in `cmd/design-preview` (console clean, layout at 375 and 1440 px).

**Verify:** `go test ./internal/adapters/httpserver/`, the `writers-map-core` check, and the design-preview run.

### R6. Check it end to end and write the evidence
- [x] Done

**Owns:** `docs/tdd/nearby-rings.tdd.md`

- Run the app with `WRITERS_MAP=1` and a temp copy of the database. For Lisbon, Kyoto, London, Paris, New York, Tokyo and Funchal, `curl` `/map/<dest>.geojson` cold and poll until `loading` is false: time to first pins, time to finish, pins per poll, largest page time and size.
- Compare Funchal's final writer matches with the cache before the change.
- Run the 6a matcher tests.
- Finish the TDD report: journeys, RED/GREEN excerpts, guarantees table, coverage, before/after timings.
- Add a line that the 6a precision review can now run on the new Madeira gazetteer.

**Verify:** first pins < 5 s in every city; `go test -cover ./internal/adapters/api/ ./internal/adapters/sqlite/ ./internal/adapters/httpserver/` ≥ 80%; `make test lint build` passes.

---

### R7. Dense-city mode
- [x] Done

**Owns:** `internal/adapters/api/wikidatanearby.go`, `wikidatanearby_test.go`, `internal/writermap/source.go` (`BaseProgress.Mode`), `internal/adapters/sqlite/wikidatanearby_test.go`, `docs/writers-map-contract.md` (BasePlaceSource bullet), `docs/tdd/nearby-rings.tdd.md`

- [x] Measure items (2+ sitelinks) within 5 km for the seven cities; pick the threshold (1,500).
- [x] Capped count after the first page; mode `normal` or `dense` stored with the progress.
- [x] Dense: eight 1 km rings, end at 8 km; normal unchanged.
- [x] Tests: above/below threshold, 1 km rings and 8-round cap, resume keeps the mode, Funchal normal.
- [x] Re-measure London, Paris, New York, Tokyo; regression on Funchal and Lisbon.

### R8. Default view at the 200 m scale
- [x] Done

**Owns:** `public/redesign/writers-map.js`, `writers-map-leaflet.js`, `tasks/redesign/checks/writers-map-core.test.mjs`, `writers-map-core.browser.mjs`, `map-panel.browser.mjs`

- [x] Measure the real scale bar in London at 375 and 1440 px; choose the zoom (14.3, not 16.7).
- [x] Open centred at that zoom; remove the automatic fit-to-pins (`frameMap`) and its padding constants.
- [x] Tests: initial zoom and centre asserted, no fit on either engine; browser checks pan to the pins before clicking.
