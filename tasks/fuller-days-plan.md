# Fuller itinerary days: every day gets 2+ stops

**Status:** implemented and verified on 2026-10-12, including the owner's 15 km total daily walking limit. Recorded Lisbon, Kyoto and Tavira days have 3–4 stops and walks at most 15 km. `make test lint build` passes with a temporary Go 1.27-compatible linter; see [TDD evidence](../docs/tdd/fuller-days.tdd.md). Tasks: [fuller-days-todo.md](fuller-days-todo.md).

**Goal:** no generated day has a single stop. Aim at 3–4 stops a day (`MinStopsPerDay`–`MaxStopsPerDay`), never below 2.

**Independent of the remaining Writers' Map work.** Writers' Map 7e (`Include`) and 7k (travelers) are merged, and PRs 8–9 don't touch `internal/planner/`. This work owns `internal/planner/days.go`, `days_test.go`, `build.go` (`seedGroups` and `nearestUnused` only), `build_test.go`, `cities_test.go` and `docs/tdd/fuller-days.tdd.md`.

## Context

**Pipeline before this work** (`internal/planner/`). `Build` (`build.go`) ranks places, then calls `seedGroups(req.Include, ranked, days)`:
- **Nothing included → `GroupDays`** (`days.go`): `Rank` (top 60 by sitelinks) → cut to **top 20** (`SunnyPoolSize`) → `selectCompact` → k-means with farthest-point seeds → `balanceClusters` (then limited by `PullReachKM` = 3 km) → `fillSingletons` → `WalkingLoop`.
- **Saved places included (Writers' Map 7e) → `seedGroups`** (`build.go`): the included places are clustered into days (`clusterPlaces`, `capClusters`); each of those days is filled up to `MinStopsPerDay` by `nearestUnused`, **also from the top 20 only**; the remaining days come from `GroupDays(left, …)`.
- Then `Schedule` (rain swaps never lower the stop count).

**Measured on the recorded fixtures** (`planNow` 2026-09-20, re-run on 2026-10-12):

| City | 1 day | 2 days | 3 days | 4 days | 5 days |
|---|---|---|---|---|---|
| Lisbon | 4 | 4 4 | 4 4 4 | 4 3 4 3 | 4 3 4 3 **2** |
| Kyoto | 4 | 4 4 | 4 **2** 4 | 4 **2** 4 **2** | 4 3 4 **2** **2** |
| Tavira | 4 | 4 3 | 4 3 | 4 3 | 4 3 (only 2 days possible) |

## Why days are thin

1. **Top-20 cap: the main cause.** 5 days × 4 stops = 20 slots, the same as the pool, so nothing is left to fill gaps. Places ranked 21–60 sit unused next to thin days:
   - Arashiyama day: **Tenryū-ji** (`Q1759226`, rank 27), 0.3 km away
   - Fushimi Inari day: **Tōfuku-ji** (`Q2166743`, rank 20), 1.1 km away
   - Gulbenkian day: **Edward VII Park** (`Q2034183`, rank 32), 0.9 km away
2. **`fillSingletons` only fixes days with exactly 1 stop**, and only from the top 20. A 2-stop day is never topped up.
3. **It measures distance from `stops[0]` only**, not from the day's nearest stop.
4. **Farthest-point seeding builds days around remote places** (Mount Hiei: nothing within 2 km), which then stay thin.
5. **Days built around saved places have the same gaps.** `seedGroups` fills them with `nearestUnused` from the top 20 only, so a saved place far from other top-20 places becomes a 1- or 2-stop day. `nearestUnused` repeats `fillSingletons`'s nearest-within-reach search.

## Decisions (owner, 2026-10-10)
- **Lone place → swap for another area.** If no area can fill the day, the trip gets fewer days and the existing short-trip note.
- **Target:** aim for 3–4, hard minimum 2.

## Design (`internal/planner/days.go`, plus `seedGroups` in `build.go`)

### 1. `fillShortDays(clusters, ranked)` replaces `fillSingletons`
- Runs for **every day under `MinStopsPerDay`**, not only singletons.
- Draws from the **full ranked list** (≤60), which `GroupDays` already receives.
- Each round adds the most famous unused place within `compactRadiusKM` (1.5 km) of **any** stop; else the most famous within `PullReachKM`. Stops at `MinStopsPerDay` or when nothing is in reach.
- Reuse `nearestKM`, `moreFamous`.

### 2. `replaceLoneDays(clusters, ranked)` after the fill
- A day still at 1 stop drops that place and is rebuilt around the most famous unused ranked place, using the same walk as the fill, up to `MaxStopsPerDay`.
- Accept it at 2+ stops; else try the next seed; else drop the day. `Build` already adds the short-trip note when `len(groups) < req.Days`.
- **Exceptions:**
  - The day with the city's most famous place is never dropped (`TestGroupDays_AlwaysKeepsTheMostFamousPlace`).
  - A city with a single usable place still gets a 1-day plan, not an error (`TestGroupDays_OnePlaceGivesOneDay`).
- Deterministic: ties by ID through `moreFamous`. Reuse `mostFamousLeft`, `hasPlaceID`.

### 3. Unchanged
- `Rank`, `CandidatePoolSize`, k-means seeding, `Schedule`. `topUpRainyDays` swaps one-for-one, so it can't drop a day below 2.
- **Follow-up only if needed:** raising `CandidatePoolSize` above 60 would add Nison-in and Iwatayama Monkey Park to the fill pool.

### 4. Days built around saved places (`seedGroups` in `build.go`)
- Included places never reach `GroupDays`; they go through `seedGroups`. Change only that function:
  - Fill each included day with **`fillShortDays`**, drawing from `rest` (all ranked places not included), not the top-20 `pool`.
  - **Delete `nearestUnused`**; `fillShortDays` replaces it.
- **Never apply `replaceLoneDays` to an included day.** Included places are never swapped out or dropped, even if their day stays at 1 stop.
- The days after the included ones keep calling `GroupDays(left, …)`, so they get steps 1 and 2 automatically.
- Existing 7e tests in `build_test.go` stay green (all included places planned; extras go to "also suggested"; no `Include` → same plan as `GroupDays`).

## Verification
- Before/after stops-per-day table for the 3 fixture cities, 1–5 days: no day below 2, and the thin Kyoto and Lisbon days at 3+.
- A plan with an isolated included place gets its day topped up from beyond the top 20.
- `go test -cover ./internal/planner/...` ≥80%.
- `make test lint build`.
- Evidence in `docs/tdd/fuller-days.tdd.md`.

## Owner follow-up: 15 km daily walking limit (2026-10-12)

The owner changed the limit to **15 km for the complete walking loop of each day**, including the return. Initial cluster balancing still uses 3 km to keep areas compact. Filling may consider farther ranked places, but accepts one only if the resulting loop is at most 15 km. An overlong initial cluster loses its least famous stops. Saved places on an overlong day split across available days; those that cannot fit are returned in "also suggested". Rain swaps obey the same cap. See the follow-up section in the TDD evidence for the RED/GREEN checks and final city table.
