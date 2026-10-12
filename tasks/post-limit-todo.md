# Writers index: cap at 100 posts — Tasks

Source of truth for progress. Read [the plan](post-limit-plan.md) first.

**Status:** L1–L4 done (2026-10-11), including the decision 6 and 7 follow-ups and the orphan cleanup. One live sync indexed 100 posts; owner to review the coverage before 8b ([results](post-limit-results.md#coverage-l4)).

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
- [x] Done — [results](post-limit-results.md); 178 requests, no host blocked

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
- [x] **Owner decision recorded** in the plan's **Owner decisions** (2026-10-11).

**Verify:** results table per blog in `tasks/post-limit-results.md`; owner sign-off.

### L2. Client reads: search, comment counts, posts by ID · *after L1*
- [x] Done

**Owns:** `internal/adapters/api/wordpress.go`, `internal/adapters/api/wordpress_test.go`, recorded fixtures in `internal/adapters/api/testdata/wordpress/`

- [x] RED `TestWordPress_SearchPosts_ReturnsIDsTitlesAndSlugs`: `_fields` limited, no content requested.
- [x] RED `TestWordPress_RecentCommentCounts_TalliesCommentsPerPost`, over 2 pages, stopping at the page limit.
- [x] RED `TestWordPress_RecentCommentCounts_NotFoundMeansNoComments`: 404 → empty map, no error.
- [x] RED `TestWordPress_PostsByID_FetchesOnlyTheGivenIDs` (`include=`).
- [x] RED: robots-disallowed and 403 return the existing `ErrDisallowed`/`ErrBlocked` for all three.
- [x] GREEN: reuse the limiter, user agent and `listPage`.

**Verify:** `go test -cover ./internal/adapters/api/` ≥80%.

### L3. Select ≤100 posts and wire the sync · *after L1; parallel with L2 against a fake*
- [x] Done

**Owns:** `internal/writermap/selection.go` (new), `internal/writermap/selection_test.go` (new), `internal/writermap/sync.go`, `internal/writermap/sync_test.go`, `cmd/web/main.go` (the `wordPressPosts` adapter only)

**Selection (pure):**
- [x] RED `TestSelect_KeepsOnlyPostsWhoseTitleOrSlugNamesAPlace`: a body-only search hit is dropped; accents and case are ignored ("Açores"/"acores").
- [x] RED `TestSelect_SplitsProportionallyWithLargestRemainder`: totals exactly 100 when there are enough candidates.
- [x] RED `TestSelect_GivesEveryBlogWithADestinationPostAtLeastOne`.
- [x] RED `TestSelect_RanksByCommentsThenNewest`.
- [x] RED `TestSelect_BlogWithoutCommentsTakesItsNewest`.
- [x] RED `TestSelect_FewerCandidatesThanTheCapKeepsThemAll`.

**Sync:**
- [x] RED `TestSync_IndexesOnlyTheSelectedPosts`: the fake counts content fetches = selected count ≤100.
- [x] RED `TestSync_PurgesPostsThatDropOut`: month 2 ranks differently → old posts deleted, new ones indexed.
- [x] RED `TestSync_SkipsBlockedAndRemovedSources` (keeps today's purge on `removed`).
- [x] GREEN: replace the cursor crawl and deleted sweep with search → counts → `Select` → fetch → upsert → purge. Keep `SummaryStep`.
- [x] Remove only the code this change orphans (cursor, overlap and sweep helpers).

**Verify:** `go test -race -cover ./internal/writermap/` ≥80%.

### L4. Measure coverage and document · *after L2, L3*
- [x] Done — 100 posts, 145 requests, 3 min 39 s; Madeira 45/49 (different method from PR 2's 24/49)

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

### L3 follow-ups (owner decisions 6 and 7) and cleanup
- [x] RED/GREEN: a blocked blog's rows stay and count inside the 100 (30 held rows, active blogs get 70).
- [x] RED/GREEN: a blog with indexed posts and a sudden empty search keeps its rows (inside the cap), logs a warning with the host, is not finished and is retried.
- [x] RED/GREEN (decision 9): the keep-rows guard triggers on 0 destination candidates after the title/URL filter, not 0 raw hits (20 rows kept, other blog gets 80).
- [x] RED/GREEN (decision 10): after a run over 100 (a failed blog kept old rows), trim the lowest-ranked rows of the updated blogs to exactly 100; log the count.
- [x] Removed the unused `ListPosts`/`PostIDs` client methods and their tests; coverage stays at or above 80%.

**Done when:** a full sync indexes ≤100 posts, chosen by the agreed signal, re-picked monthly with drop-outs purged, and the owner has seen the coverage trade-off.
