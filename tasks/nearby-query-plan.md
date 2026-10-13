# Smaller Wikidata nearby query: pins load in dense cities

**Status:** SUPERSEDED 2026-10-12 by [nearby-rings-plan.md](nearby-rings-plan.md): no radius, N or two-query setting met the targets (measurements in `docs/tdd/nearby-query.tdd.md`). Original status: plan only, not started. Owner decisions recorded 2026-10-12 (see **Decisions**). Tasks: [nearby-query-todo.md](nearby-query-todo.md).

**Goal:** the base-place query (`WikidataNearbyAPI.Places`) finishes in **under 5 s** with a response **under 5 MB**, cold, for Lisbon, Kyoto, London, Paris, New York and Tokyo, so notable and writers' pins load. Funchal and Madeira keep the pins and writer matches they have today.

**Runs before the 6a precision review.** 4a and 4b (which owned these files) are done. This work owns `internal/adapters/api/wikidatanearby.go`, `wikidatanearby_test.go`, `internal/adapters/api/testdata/wikidata_*.json`, `internal/adapters/sqlite/wikidatanearby.go` (the cache version constant only), the `BasePlaceSource` lines of `docs/writers-map-contract.md` and `docs/tdd/nearby-query.tdd.md`.

## The problem (measured 2026-10-12)

`nearby()` asks for **every Wikidata item with coordinates within 40 km**, with 11 `OPTIONAL` fields and no limit. Each extra type, alias, label or opening-hours value **multiplies** an item's rows.

| City | Items within 40 km | Items with ≥3 sitelinks |
|---|---|---|
| Lisbon | 5,923 | 1,377 |
| Kyoto | 8,148 | not measured |
| London | 65,132 | 7,152 |
| Paris | 102,186 | 7,801 |
| New York, Tokyo | not measured (N1) | not measured (N1) |

- **Lisbon, the full query:** 18,885 rows for 5,819 items, **27 MB**, **~24 s** to download. The app's `http.Client` timeout is **30 s, including the body**, so `/map/lisbon-pt.geojson` returns **503 after 30 s**. A failure is not cached, so every page load repeats it.
- **London and Paris** have 11–17× as many items. Grouping rows alone can't fix them; the number of items must drop too.

## Fix: one grouped, filtered query

Keep **one query** and the `BasePlaces` shape. Change two things:

1. **One row per item.** `GROUP BY ?item` with `GROUP_CONCAT(DISTINCT …)` for the fields that can have several values (types, en/pt labels, en/pt aliases) and `SAMPLE(…)` for the single ones (coordinates, type label, description, area, image, sitelinks, enwiki article). Opening hours become one concatenated field, parsed in Go.
   - Lisbon goes from 18,885 rows to ~5,800.
   - `nearbyPlaces` then reads one row per item instead of merging many.
2. **Keep only notable items:** `?item wikibase:sitelinks ?s FILTER(?s >= N)`. This cuts London and Paris from 65k–102k items to ~7–8k. **N is chosen in N1 from measurements** (see **Decisions**).

**Contract update.** `docs/writers-map-contract.md` says `Gazetteer` "keeps the full nearby list". After this fix it keeps the nearby items with **at least N sitelinks**. Update that line in N3.

**Pre-approved fallback: two queries.** If, at the N from N1, any city still misses 5 s / 5 MB:
- **Query 1** (all filtered items, grouped): QID, coordinates, types, type label, labels, aliases, description, image, sitelinks.
- **Query 2** (the 200 pins only, `VALUES ?item { … }`): area, enwiki article, opening hours.
- **Cost:** places that writers promote to pins (not in the top 200) have **no area, Wikipedia link or opening hours** in their panel.
- Update the contract line "from one Wikidata query" to "from two Wikidata queries" and say which fields only pins have.

**Radius (owner, option B, 2026-10-12):** live probes showed the grouped query and the two-query fallback both time out for Lisbon at 40 km, so the **search radius shrinks** (chosen in N1b from measurements) on top of the sitelinks filter. This supersedes "40 km unchanged".

**Not changing:** the 200-pin cap, the **30 s** client timeout, `addPhotoCredits`, the retry on 429/503.

## Decisions (owner, 2026-10-12)

| Decision | Choice |
|---|---|
| Threshold N | **Measure it.** The largest N that drops **no** writer-matched place in the Madeira fixture or the 6a cases (São Jorge, Monte, Ponta do Sol, Teleférico do Funchal) and keeps every city under the targets. |
| If grouped + filtered is still too slow or too big | **Two queries, pre-approved** (above). No need to stop and ask. |
| Order with 6a | **This fix first**, then the 6a precision review runs on the new gazetteer. |
| Cached destinations | **Bump `v2` → `v3`, refresh lazily:** each destination re-runs the new query on its next visit. No warm-up job. |
| Targets | **< 5 s and < 5 MB**, cold. |
| Cities measured | Lisbon, Kyoto, London, Paris, **New York, Tokyo**, Funchal. |
| 30 s timeout | **Unchanged.** |
| Option A (2026-10-12) | **Two queries, 40 km, N <= 2.** Q1 = all items (QID, coords, types, type label, labels, description, image, sitelinks); Q2 = top-200 pins only (area, enwiki article, hours, aliases if dropped). Q2 failure returns pins without those fields. Supersedes option B. |
| Radius (superseded) | **Option B: reduce it** for dense areas (supersedes "40 km unchanged"). Prefer ONE constant radius + N; per-destination radius only if simple, else stop and report. |
| Threshold N | **N = 2 allowed.** Exact same 200 Funchal pins no longer required; Madeira/Funchal writer matches and the 6a cases must still match. |

**If no N works** (every N small enough for the targets drops a writer place, even with two queries): stop and ask the owner. This is the only case that needs a new decision.

## Done when

- Lisbon, Kyoto, London, Paris, New York and Tokyo each: cold query < 5 s, response < 5 MB, `/map/<dest>.geojson` returns 200 with notable pins.
- Funchal and Madeira: the same pins, and the 6a writer-pin cases still pass.
- The contract line matches the new behaviour.
- Unit tests use recorded fixtures only (no live calls in `make test`); `make test lint build` passes.
- Evidence in `docs/tdd/nearby-query.tdd.md`, including before/after timings.
- **Then:** the 6a precision review can run on the new Madeira gazetteer.
