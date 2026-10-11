# Writers index: cap at 100 posts — Tasks

Source of truth for progress. Read [the plan](post-limit-plan.md) first.

**Status:** not started (2026-10-11).

**How to read this file**
- Order: **L1 → owner decision → L2 → L3 → L4**. L2 and L3 can run in parallel after L1, because their files are disjoint.
- **Owns** = the only files the task may edit. None overlap with open Writers' Map tasks in [todo.md](todo.md).
- TDD per [AGENTS.md](../AGENTS.md) §5. **Tests never hit the network:** `httptest` and recorded responses only.
- **No post text stored, logged or committed:** names, IDs and counts only.
- No commits or pushes. The owner commits.

**User journeys**
- *As the owner, I want the index capped at 100 posts, so that syncs are light on the blogs and on our storage.*
- *As a traveller, I want the posts behind the pins to be each writer's best-known work, so that the links are worth opening.*

---

### L1. Pick the popularity signal · spike · **owner decides**
- [ ] Done

**Owns:** `tasks/post-limit-results.md` (new), scratch scripts in a gitignored or temporary folder (not committed)

- For each of the 11 active blogs in `writerdata/sources.json`, record which signals work:
  - **recent comments tally:** comments open? how many in the last 1,000?
  - **sticky posts:** how many?
  - **destination search:** `search=` on Madeira, Açores, Lisboa and Porto. How many hits?
- Use the app's user agent, honour robots.txt, and space requests about 1 per second. Record **counts and post IDs only**.
- For each signal, list the 9 posts it would pick per blog, and how many of those mention a Portuguese destination (from titles only).
- Recommend:
  - one signal and its fallback order
  - whether to rank only among destination-matching posts (see the plan's **Main risk**)
  - the share rule, if even share doesn't fit
- [ ] **Owner decision recorded** in the plan's **Owner decisions**.

**Verify:** results table per blog in `tasks/post-limit-results.md`; owner sign-off.

### L2. Rank a blog's posts by the chosen signal · *after L1*
- [ ] Done

**Owns:** `internal/adapters/api/wordpress.go`, `internal/adapters/api/wordpress_test.go`, recorded fixtures in `internal/adapters/api/testdata/wordpress/`

- [ ] RED `TestWordPress_RankedPostIDs_OrdersByTheChosenSignal`, from a recorded comments/sticky fixture.
- [ ] RED `TestWordPress_RankedPostIDs_FallsBackWhenTheSignalIsEmpty`: closed comments lead to the next signal in L1's order.
- [ ] RED `TestWordPress_RankedPostIDs_StopsAtTheRequestedCount`: no extra pages are fetched.
- [ ] RED: robots-disallowed and 403 return the existing `ErrDisallowed`/`ErrBlocked`.
- [ ] GREEN: the new method uses the existing limiter, user agent and `listPage`.

**Verify:** `go test -cover ./internal/adapters/api/` ≥80%.

### L3. Cap the sync at 100 and purge drop-outs · *after L1*
- [ ] Done

**Owns:** `internal/writermap/sync.go`, `internal/writermap/sync_test.go`, `cmd/web/main.go` (the `wordPressPosts` adapter only)

- [ ] RED `TestSync_IndexesAtMost100PostsInTotal`: 11 fake blogs × 50 posts → 100 indexed.
- [ ] RED `TestSync_SplitsTheCapEvenlyAcrossActiveBlogs`: 9 each, with the leftover passed on in allowlist order. Blocked and removed sources get no share.
- [ ] RED `TestSync_GivesAnUnusedShareToTheNextBlogs`: a blog with 3 posts → the others take the rest.
- [ ] RED `TestSync_PurgesPostsThatDropOutOfTheTop`: month 2 ranks differently → old posts deleted, new ones indexed.
- [ ] RED `TestSync_FetchesContentOnlyForChosenPosts`: the fake counts content requests.
- [ ] GREEN: replace the cursor crawl with "rank, choose, upsert, purge". Keep the blocked/removed handling and the `SummaryStep` call.
- [ ] Remove code this change orphans (cursor and overlap helpers), and only that.

**Verify:** `go test -race -cover ./internal/writermap/` ≥80%.

### L4. Measure coverage and document · *after L2, L3*
- [ ] Done

**Owns:** `docs/maintenance.md` (writers' map section), `docs/tdd/post-limit.tdd.md` (new), `tasks/post-limit-results.md` (coverage section)

- Local run: `DB_PATH=post-limit.db WRITERS_MAP=1 WRITERS_SYNC=1 go run ./cmd/web`.
- Record:
  - indexed post count (≤100)
  - posts per blog
  - Madeira places with writer pins, against PR 2's 24/49
  - total requests and time for one full run
- Update `docs/maintenance.md`: the cap, how posts are picked, purge on drop-out.
- Evidence doc per AGENTS.md: journeys, RED/GREEN per behaviour, guarantees table, coverage.

**Verify:** `make test lint build`; results reviewed by the owner before 8b.

**Done when:** a full sync indexes ≤100 posts, chosen by the agreed signal, re-picked monthly with drop-outs purged, and the owner has seen the coverage trade-off.
