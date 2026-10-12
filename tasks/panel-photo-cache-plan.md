# Panel photo still broken (stale destination cache)

**Status:** fixed locally 2026-10-12 with **B + C** (D1); cold Lisbon recompute exceeded 35 s. Tests and runtime evidence: [panel-photo-cache.tdd.md](../docs/tdd/panel-photo-cache.tdd.md). Tasks: [panel-photo-cache-todo.md](panel-photo-cache-todo.md).

**Symptom:** the place panel for *Condes de Castro Guimarães Palace* (Lisbon) shows a broken image with the credit "Diego Delso · CC BY-SA 4.0". The earlier fix ([panel-photo-fix.tdd.md](../docs/tdd/panel-photo-fix.tdd.md)) is in the running code.

## Root cause
There are **two caches**, and the earlier fix only invalidated the inner one.

| Cache key | Holds | Bumped by the fix? |
|---|---|---|
| `wikidata:v1` → `wikidata:v2` (`internal/adapters/sqlite/wikidatanearby.go`) | base places from Wikidata | **yes** |
| `writermap:v1` (`internal/adapters/sqlite/writerdestination.go`) | the whole `DestinationResult`, **including each place's photo fields**, and it never expires | **no** |

The writers' map source reads `writermap:v1` first, so it never reaches the new `wikidata:v2` lookup. The panel keeps getting the old v1 place, whose `PhotoURL` is the **Commons page** (`https://commons.wikimedia.org/wiki/File:…`). An HTML page in `<img src>` renders as a broken image.

## Evidence (local, 2026-10-12)
- **Server:** `go run ./cmd/web` started 16:45. The fix files are dated 15:49, so the server runs fixed code.
- **Database:** `source_cache` has **no** `wikidata:v2` rows. It has `wikidata:v1` and `writermap:v1` rows for Lisbon and Funchal, written 2026-10-11 22:29.
- **Live response:** `GET /map/lisbon-pt.geojson?lat=38.7078&lon=-9.1366` comes back in **3 ms** (from cache). Q10344066 has:
  - `photo_url`: `https://commons.wikimedia.org/wiki/File:Palacio_de_los_condes_…DD_12.jpg`, the page and not an image
  - `photo`: the full-size original URL with `utm_*` parameters, which is the old v1 shape
- **Production is probably the same:** Fly's database was filled by the same code, so a deploy alone won't fix it.

## Side effects to handle
1. **Live recompute may fail.** Once the cache is invalidated, the first visit runs the live Wikidata nearby query. For Lisbon that took about **80 s, against a 30 s timeout** (security item 10). If it times out, the map is empty instead of merely missing photos.
2. **The monthly refresh loses destinations.** Both `CachedDestinations` (v2 only) and `WriterDestinations` list areas from the keys. After a bump, old destinations drop out of the refresh until someone visits them again.
3. **Stale rows stay in the database.** Old v1 rows are never read again, but they keep taking space (Lisbon alone is about 476 KB compressed).

## Fix options (owner decision: D1 in the todo)
| Option | Cost | Benefit |
|---|---|---|
| **A. Bump `writermap:v1` → `v2`** (same as the last fix) | One-line change, plus a test. Side effects 1–3 remain. | Simplest, and consistent with the previous fix. |
| **B. A plus a guard that ties the versions** (recommended): the destination key includes the base-place version, so bumping one invalidates both | A little more code and one test | Stops this bug class from coming back |
| **C. Startup cleanup:** delete `wikidata:v1%` and `writermap:v1%` rows | One idempotent statement plus a test | Frees space. It doesn't stop a repeat on its own, so it goes with A or B. |

**Recommendation:** **B + C**. Before deploying, either warm the destinations through the refresh job or wait for the SPARQL timeout work (PR 8) to land.
