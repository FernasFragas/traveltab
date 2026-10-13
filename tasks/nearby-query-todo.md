# Smaller Wikidata nearby query — Tasks

Source of truth for progress. Read [the plan](nearby-query-plan.md) first, especially **Decisions**.

**How to read this file**
- Tasks run in order N1 → N6, by **one agent**: N2–N4 all edit `wikidatanearby.go`. N4 runs only if N1 says the fallback is needed.
- **Owns** = the only files the task may edit. 4a/4b, which owned these files, are done. This work runs **before** the 6a precision review.
- Start from an up-to-date `main` (`git fetch` and check `git status -sb`).
- TDD per [AGENTS.md](../AGENTS.md) §5: see each test fail for the intended reason first.
- Live Wikidata calls only in N1 and N6 (measurement), with the app's `UserAgent`, never personal details. `make test` stays offline.
- No commits or pushes. The owner commits.

**Targets:** cold query **< 5 s** and **< 5 MB** for Lisbon, Kyoto, London, Paris, New York and Tokyo. Funchal keeps its pins.

**User journeys**
- *As a traveller opening Lisbon, London, Paris, New York or Tokyo, I want the map's notable and writers' pins to appear, so that the map is useful in big cities too.*
- *As a traveller in Funchal, I want the same pins and writer matches as before, so that the fix costs me nothing.*

---

### N1. Measure and choose N (and whether the fallback is needed)
- [ ] Done

**Owns:** `docs/tdd/nearby-query.tdd.md` (new)

- Run the grouped query (no filter, then N = 1, 2, 3, 5) for Lisbon, Kyoto, London, Paris, New York, Tokyo and Funchal. Record items, response size and time.
- For each N, list which writer-matched places in the Madeira fixture and the 6a cases (São Jorge, Monte, Ponta do Sol, Teleférico do Funchal) would be dropped.
- **Pick N:** the largest N that drops no writer-matched place.
- **Fallback check:** if any city misses the targets at that N, time the two-query version (plan, **Pre-approved fallback**) at the same N. If that meets the targets, N4 runs. If not, **stop and ask the owner**.

**Verify:** the evidence doc has the table, the chosen N, and "one query" or "two queries", with the reason.

### N1b. Measure radius x N (option B)
- [x] Done. Result: **no setting meets the targets**; stopped, see `docs/tdd/nearby-query.tdd.md`

**Owns:** `docs/tdd/nearby-query.tdd.md`

- Owner chose option B: reduce the radius. Probe radii 10/15/20/25/40 km x N = 1, 2, 3 for Lisbon, Kyoto, London, Paris, New York, Tokyo, Funchal. Live calls, app `UserAgent`, sequential, generous client timeout so timings are real.
- **Pick ONE constant radius + N** meeting < 5 s and < 5 MB for all six cities and dropping no Madeira/6a writer match (N = 2 allowed). Per-destination radius only if simple; if no setting works, **stop and report**.

**Verify:** the evidence doc has the radius x N table and the chosen pair.

### N2. Group the query to one row per item
- [ ] Done

**Owns:** `internal/adapters/api/wikidatanearby.go`, `wikidatanearby_test.go`, `internal/adapters/api/testdata/wikidata_madeira.json` (re-recorded in the grouped shape)

- `GROUP BY ?item` with `GROUP_CONCAT(DISTINCT …; separator="\u001F")` for types, labels and aliases; `SAMPLE` for the rest. Opening hours as one concatenated field.
- `nearbyPlaces` splits the concatenated fields; the output `BasePlaces` stays the same.
- Tests: the existing `TestWikidataNearby_Places_MadeiraPinsAndGazetteer` assertions still hold on the re-recorded fixture; a name containing `|` or `,` survives the split; an item with several types, aliases and opening hours comes out as one place with all of them.

**Verify:** `go test ./internal/adapters/api/ -run 'WikidataNearby|NearbyPlaces'`

### N3. Filter by sitelinks and update the contract
- [ ] Done

**Owns:** `internal/adapters/api/wikidatanearby.go`, `wikidatanearby_test.go`, `docs/writers-map-contract.md` (the `BasePlaceSource` bullet only)

- Add `?item wikibase:sitelinks ?sitelinks FILTER(?sitelinks >= N)`, with N from N1b as a named constant next to `wikidataNearbyRadius`, and set the radius constant to the N1b value.
- Contract: `Gazetteer` keeps the nearby items with at least N sitelinks, not "the full nearby list".
- Test: the query sent contains the filter with that constant.

**Verify:** `go test ./internal/adapters/api/ -run WikidataNearby`

### N4. Split into two queries · *needed (owner option A)*
- [ ] Done. **Blocked 2026-10-12:** Q1 alone misses the targets, see `docs/tdd/nearby-query.tdd.md` (N1c). Owner decision needed.

**Owns:** `internal/adapters/api/wikidatanearby.go`, `wikidatanearby_test.go`, `internal/adapters/api/testdata/wikidata_*.json`, `docs/writers-map-contract.md` (the `BasePlaceSource` bullet only)

- Query 1: all filtered items, grouped, without area, enwiki article and opening hours.
- Query 2: `VALUES ?item { … }` over the 200 pins: area, enwiki article, opening hours.
- If query 2 fails, return the pins without those three fields rather than failing `Places` (same approach as `addPhotoCredits`).
- Contract: "from two Wikidata queries"; area, Wikipedia link and opening hours only for the top 200 pins.
- Tests: two queries sent; query 2 lists only pin QIDs; pins get the details; a non-pin gazetteer place has none; query 2 failing still returns pins.

**Verify:** `go test ./internal/adapters/api/ -run 'WikidataNearby|NearbyPlaces'`

### N5. Bump the base-place cache version
- [ ] Done

**Owns:** `internal/adapters/sqlite/wikidatanearby.go` (the `basePlaceCacheVersion` constant only), its test if one pins the value

- `v2` → `v3`. Destinations refresh lazily on their next visit; no warm-up job.

**Verify:** `go test ./internal/adapters/sqlite/`

### N6. Check it end to end and write the evidence
- [ ] Done

**Owns:** `docs/tdd/nearby-query.tdd.md`

- Run the app with `WRITERS_MAP=1` and a temp copy of the database. `curl` `/map/<dest>.geojson` cold for Lisbon, Kyoto, London, Paris, New York, Tokyo and Funchal: status, time, feature count.
- Compare Funchal's pins with the result cached before the change (the same QIDs, or the differences explained).
- Run the 6a matcher tests.
- Finish the TDD report: journeys, RED/GREEN excerpts, guarantees table, coverage, before/after timings.
- Add a line that the 6a precision review can now run on the new Madeira gazetteer.

**Verify:** every destination returns 200 within 5 s; `go test -cover ./internal/adapters/api/` ≥ 80%; `make test lint build` passes.
