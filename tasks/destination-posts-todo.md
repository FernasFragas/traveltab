# Writer posts for new destinations — Tasks

Source of truth for progress. Read [the plan](destination-posts-plan.md) first.

**Status (2026-10-13):** T1–T5 done, plus the owner's follow-up: `WRITERS_RESYNC_BEFORE` re-picks existing indexes at the first startup after a deploy. See `docs/tdd/destination-posts.tdd.md`.

**How to read this file**
- Tasks run in order T1 → T5. T2–T4 edit `sync.go` and must not run concurrently.
- **Owns** = the only files the task may edit.
- Start from an up-to-date branch (`git fetch`, `git status -sb`).
- TDD per [AGENTS.md](../AGENTS.md) §5: see each test fail for the intended reason first.
- Live calls only in T1 and T5, through the app's WordPress client (its limiter, robots.txt check and `UserAgent`), never personal details. `make test` stays offline.
- No commits or pushes. The owner commits.

**User journeys**
- *As a traveller opening Toronto for the first time, I want writers' posts about Toronto on the map, so that I see more than Wikidata's notable places.*
- *As a traveller opening Madeira, I want to keep the writer pins I have today, so that new cities don't push them out.*
- *As the owner, I want crawling and database size to stay bounded, so that visitors can't make the app hammer the blogs.*

---

### T1. Measure a top-up
- [x] Done (2026-10-13): ~11 KB/post → T = 5,000; top-up ≈ 11 searches, 13–31 s, 18–37 picks. Writer pins measured in T5 instead.

**Owns:** `docs/tdd/destination-posts.tdd.md`, a temporary measuring test (deleted afterwards)

- For **Toronto, New York, Tokyo** (new) and **Lisbon** (known), with the configured blogs:
  - search hits per blog, destination posts after the title/slug filter, posts K would pick;
  - requests and time for a full top-up;
  - writer pins the destination would get (matcher on a temp database copy).
- Bytes per indexed post (`writer_names` + `writer_posts` + `writer_days`) from `launch-check.db`, to set T.

**Verify:** the evidence doc has the table and the chosen T, with the bytes-per-post reasoning.

### T2. Per-destination selection
- [x] Done (2026-10-13): `SelectPerDestination`, `MergeBlogs`, `PostsPerDestination = 100`, `MaxIndexedPosts = 5000`.

**Owns:** `internal/writermap/selection.go`, `selection_test.go`

- `SelectPerDestination(destinations, blogs, places, k)`: today's `Select` per destination, then the union, with each pick tagged with the destinations that chose it.
- `MaxIndexedPosts` becomes the per-destination K; add the total ceiling T.
- Tests (RED first):
  - two destinations each get up to K, regardless of how many posts the other has;
  - a post picked for both is stored once;
  - a union over T is cut to T, lowest-ranked across all destinations first (reusing `Trim`);
  - one destination's rules match today's `Select` (no regression).

**Verify:** `go test ./internal/writermap/ -run 'Select|Trim'`

### T3. Monthly run with per-destination picks
- [x] Done (2026-10-13): per-term candidates, `SelectPerDestination`, `SyncJob.MaxPosts`, tally and hits kept for top-ups.

**Owns:** `internal/writermap/sync.go`, `sync_test.go`

- `syncSources` searches each destination per blog, as today, and selects with `SelectPerDestination`.
- Keeps the blog's comment tally in memory for top-ups.
- The purge removes only posts no destination picked. Blocked-blog rows still count in T.
- Tests (RED first): adding a destination doesn't drop another destination's posts; posts no destination picks are purged; the ceiling holds; the existing sync tests still pass.

**Verify:** `go test ./internal/writermap/ -run Sync`

### T4. Top-up job and trigger
- [x] Done (2026-10-13): `TopUp`, `QueueTopUp`, `WorkTopUps`, `TopUpsPerDay = 20`, `OnNewDestination` hook, `MapSource.RecomputeDestination`, `Schedule` retries a busy start after a minute.

**Owns:** `internal/writermap/sync.go`, `sync_test.go`, `internal/adapters/sqlite/wikidatanearby.go` (hook only), `wikidatanearby_test.go`, `cmd/web/main.go` (wiring), `internal/writermap/source.go` (`RecomputeDestination` only), `source_test.go`

- `SyncJob.TopUp(ctx, area)` and a de-duplicated queue worker sharing `running` with `Run`.
- The daily budget D; over budget, the destination waits for the next day.
- Fetch only posts not already indexed; recompute that destination's stored writer result.
- `cachedBasePlaces` calls `OnNewDestination` after the **first** successful store of a key, never on `Refresh`.
- Wire it in `main.go` only when `WRITERS_SYNC=1`.
- Tests (RED first):
  - the first cache of a destination queues one top-up; a refresh or a second visit doesn't;
  - two queued requests for the same destination run once;
  - a top-up waits while a monthly run holds the lock;
  - the (D+1)th top-up in a day waits for the next day;
  - a top-up that takes the index over T trims the lowest-ranked posts across destinations, and logs which destinations lost posts;
  - a blocked blog isn't requested;
  - after the top-up, the destination's stored result includes the new posts' writer pins.

**Verify:** `go test ./internal/writermap/ ./internal/adapters/sqlite/`; `go test -cover` ≥ 80% for both.

### T5. Check it end to end and write the evidence
- [x] Done (2026-10-13): Toronto 0 → 98 writer pins after its top-up; Funchal 122 → 148; `docs/maintenance.md` updated; `make test lint build` passed.

**Owns:** `docs/tdd/destination-posts.tdd.md`, the writers' map section of `docs/maintenance.md`

- Run the app on a new temp database (`DB_PATH=…`, `WRITERS_MAP=1`, `WRITERS_SYNC=1`). Let the first run finish, then open Toronto: no writer pins at first; after the logged top-up, reload and see writer pins.
- Compare Madeira/Funchal and Lisbon writer pins with today's: no drop.
- Record requests, time, and index size against T.
- Document K, T, D and the top-up in `docs/maintenance.md`.

**Verify:** the plan's **Done when** holds; `make test lint build` passes.
