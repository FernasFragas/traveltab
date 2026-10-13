# Writer posts for new destinations: per-destination quota and top-up

**Status:** done (2026-10-13). T = 5,000 from T1. End to end, Toronto went from 0 to 98 writer pins after its top-up, and Funchal from 122 to 148 (see `docs/tdd/destination-posts.tdd.md`). Owner decisions recorded 2026-10-13 (see **Decisions**). Tasks: [destination-posts-todo.md](destination-posts-todo.md).

**Goal:** a destination first visited after the monthly sync gets its own writer posts within minutes, without waiting a month and without taking posts away from other destinations.

**Owns:** `internal/writermap/selection.go`, `selection_test.go`, `internal/writermap/sync.go`, `sync_test.go`, `internal/adapters/sqlite/wikidatanearby.go` (the new-destination hook only), `wikidatanearby_test.go`, `cmd/web/main.go` (wiring only), `internal/writermap/source.go` (`RecomputeDestination` only, added during T4), `source_test.go`, the writers' map section of `docs/maintenance.md`, `docs/tdd/destination-posts.tdd.md`. The writer-context files (`match.go`, `writerdestination.go`) are not touched.

## The problem (measured 2026-10-13)

The app on `/tmp/tt-cold.db` loaded Toronto: **200 pins, 0 writer pins**.
- **One shared cap:** the index holds `MaxIndexedPosts = 100` posts in total, across all blogs and all destinations (`selection.go`).
- **Picked once a month:** the sync searches each blog for the destinations known **at that moment**: the override areas plus cached cities. The first run happens at startup; after that, only once a month in the off-peak window (02:00–06:00 UTC).
- **Toronto came too late:** it was cached after the 19:26 UTC run, so no blog was searched for it. The index has 2 posts that mention Toronto, both Portugal visa posts.
- **A later run wouldn't help much:** Toronto would compete with every other destination for the same 100 posts, and most configured blogs are about Portugal.

For comparison, the old uncapped `launch-check.db` index had 234 posts mentioning Toronto.

## Fix

### 1. A quota per destination instead of one shared cap

- Each destination gets up to **K posts** (**K = 100**, the size of today's whole cap), chosen with today's rules: destination posts only (the title or slug names a known place), shared across blogs proportionally, ranked by recent comments, then newest.
- The index holds the **union** of the destinations' picks. A post picked for two destinations is stored once.
- A **total ceiling T** bounds the index (starting at **5,000** posts, set from T1's size measurement). When the index would go over T, the **lowest-ranked posts across all destinations are trimmed** (fewest recent comments, then oldest; today's `Trim`) until it fits. Each trim is logged with the destinations it touched.
- The monthly run re-picks every destination's K and purges posts no destination picked, as today.

### 2. Top-up when a destination is first cached

- **Trigger:** when a destination's base places are stored for the **first** time (not on a refresh), the cache calls `OnNewDestination(area)`.
- **Job:** `SyncJob.TopUp(ctx, area)`:
  1. Searches each active blog for the destination's name (1 request per blog, as the monthly run does).
  2. Ranks the hits with the blog's last comment tally (see **Comment tallies**).
  3. Selects up to K, fetches only the posts not already indexed, and indexes them.
  4. Recomputes that destination's stored writer result, so its next map request shows the pins.
- **One at a time:** top-ups wait in a de-duplicated queue and share the monthly run's lock, so Wikidata and the blogs are never hit by two jobs at once.
- **Budget:** at most **D top-ups per day** (**D = 20**). Over the budget, the destination waits in the queue for the next day. This stops a visitor browsing many random cities from setting off many blog crawls.
- **Blocked or removed blogs** are skipped, as in the monthly run.

### Comment tallies

The monthly run fetches up to 10 pages of recent comments per blog (about 110 requests across all blogs). Doing that again for each top-up is too costly.
- Keep each blog's last tally in memory from the monthly run and use it for top-ups. After a restart with no tally yet, rank by newest only, the same fallback blogs with no readable comments use today.

### What the traveller sees

- **First visit:** pins load as today, with no writer pins. The top-up runs in the background (T1 measures how long).
- **Next visit or reload:** writer pins appear.
- No live update on the open page; that would need the map to poll for writer results, which is a separate change.

### Other destinations

New posts can mention other cached destinations (a Toronto post naming Montreal). Their stored results aren't recomputed by the top-up; they pick up those posts at the next monthly refresh.

## Decisions (owner, 2026-10-13)

| Decision | Choice |
|---|---|
| K, posts per destination | **100** |
| T, total ceiling | **Set from T1**, starting at 5,000 posts |
| Over T | **Trim the lowest-ranked posts across all destinations** until it fits; log it |
| D, top-ups per day | **20** |
| Comment tally for top-ups | **Reuse the monthly tally in memory**; newest-first fallback |
| Live update of the open map | **No**; pins show on the next visit |
| Other destinations' results | **Wait for the monthly refresh** |
| Existing indexes | **First startup after the deploy** (owner, 2026-10-13, replacing "next monthly run"): `WRITERS_RESYNC_BEFORE = "2026-10-13T20:00:00Z"` in `fly.toml` makes older sources due at once. Next time, only that line changes. |

## Risks

- **More crawling.** Each top-up makes about 1 search per blog plus the post fetches (≤ K). The daily budget and the shared lock bound it; T1 measures the real count.
- **Bigger database.** The index stores names and positions only, never post text. T1 measures bytes per post to set T.
- **Thin destinations stay thin.** If no blog writes about a city, the top-up finds nothing. That's correct, and it's logged per destination.
- **Trimming touches other destinations.** A top-up over T can delete another destination's lowest-ranked posts. Its stored result keeps listing them until the monthly refresh (the links still work, as public posts). T1's T keeps this rare.
- **Coverage shifts for existing destinations.** Madeira and Lisbon move from a share of 100 to their own K. T5 checks their writer pins don't drop.

## Done when

- On a cold database, visiting Toronto starts one top-up. After it finishes, the next Toronto map request has writer pins.
- Madeira/Funchal and Lisbon keep at least today's writer pins after the monthly run.
- The index never goes over T: going over trims the lowest-ranked posts. The D budget and the shared lock are covered by tests.
- No top-up runs on a refresh, or for a destination already cached.
- `make test lint build` passes; evidence in `docs/tdd/destination-posts.tdd.md`.
