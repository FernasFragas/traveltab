# Fuller itinerary days: every day gets 2+ stops

**Status:** planned 2026-10-10, not started. Tasks: [fuller-days-todo.md](fuller-days-todo.md).

**Goal:** no generated day has a single stop. Aim at 3–4 stops a day (`MinStopsPerDay`–`MaxStopsPerDay`), never below 2.

**Runs in parallel with the Writers' Map, or after it.** It owns only `internal/planner/days.go`, `days_test.go`, `cities_test.go` and `docs/tdd/fuller-days.tdd.md`. Map 7e owns `types.go`, `build.go`, `build_test.go`; 7k owns `stays.go`. See [todo.md](todo.md).

## Context

**Pipeline today** (`internal/planner/`):
`Rank` (top 60 by sitelinks) → `GroupDays`: cut to **top 20** (`SunnyPoolSize`) → `selectCompact` → k-means with farthest-point seeds → `balanceClusters` (pulls within `PullReachKM` = 3 km) → `fillSingletons` → `WalkingLoop` → `Schedule` (rain swaps keep the stop count).

**Measured on the recorded fixtures** (5-day trips, `planNow` 2026-09-20):

| City | Stops per day |
|---|---|
| Lisbon | 4 · 3 · 4 · 3 · **2** |
| Kyoto | 4 · 3 · 4 · **2** · **2** |
| Tavira | 4 · 3 (only 2 days possible) |

## Why days are thin

1. **Top-20 cap: the main cause.** 5 days × 4 stops = 20 slots, the same as the pool, so nothing is left to fill gaps. Places ranked 21–60 sit unused next to thin days:
   - Arashiyama day: **Tenryū-ji** (`Q1759226`, rank 27), 0.3 km away
   - Fushimi Inari day: **Tōfuku-ji** (`Q2166743`, rank 20), 1.1 km away
   - Gulbenkian day: **Edward VII Park** (`Q2034183`, rank 32), 0.9 km away
2. **`fillSingletons` only fixes days with exactly 1 stop**, and only from the top 20. A 2-stop day is never topped up.
3. **It measures distance from `stops[0]` only**, not from the day's nearest stop.
4. **Farthest-point seeding builds days around remote places** (Mount Hiei: nothing within 2 km), which then stay thin.

## Decisions (owner, 2026-10-10)
- **Lone place → swap for another area.** If no area can fill the day, the trip gets fewer days and the existing short-trip note.
- **Target:** aim for 3–4, hard minimum 2.

## Design (all in `internal/planner/days.go`)

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

### 4. After map 7e (`Include`)
- Included places count as used, and are never swapped out or dropped.

## Verification
- Before/after stops-per-day table for the 3 fixture cities, 1–5 days: no day below 2, and the thin Kyoto and Lisbon days at 3+.
- `go test -cover ./internal/planner/...` ≥80%.
- `make test lint build`.
- Evidence in `docs/tdd/fuller-days.tdd.md`.
