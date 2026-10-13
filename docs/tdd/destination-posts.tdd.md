# Writer posts for new destinations — TDD and measurement

Source: [destination-posts-plan.md](../../tasks/destination-posts-plan.md), T1–T5 in [destination-posts-todo.md](../../tasks/destination-posts-todo.md).

Journeys:
- As a traveller opening Toronto for the first time, I want writers' posts about Toronto on the map, so that I see more than Wikidata's notable places.
- As a traveller opening Madeira, I want to keep the writer pins I have today, so that new cities don't push them out.
- As the owner, I want crawling and database size to stay bounded, so that visitors can't make the app hammer the blogs.

## T1. Measurements (2026-10-13)

**Size per post** (`launch-check.db`, 16,213 posts, `dbstat` over the `writer_*` tables and their indexes): 177.6 MB → **about 10.9 KB per post**. On the small T5 database: 2.0 MB for 135 posts, about 15 KB (page overhead). **T = 5,000 posts**, about 55–75 MB, well inside the 1 GB Fly volume.

**Top-up cost**, live, through the app's `WordPressClient` (its host limiter, robots.txt check and `UserAgent`), 11 active blogs, first search page only. "Destination posts" used only the destination's name as a known place, a lower bound (the app also uses the cached gazetteer names). Temporary test, deleted afterwards.

| Destination | Search hits | Picks (K = 100) | Requests | Time |
|---|---:|---:|---:|---:|
| Toronto | 184 | 18 | 22 (11 searches + 11 robots.txt, first contact) | 30.6 s |
| New York | 513 | 18 | 11 | 13.2 s |
| Tokyo | 221 | 24 | 11 | 13.6 s |
| Lisbon | 467 | 37 | 11 | 12.8 s |

Each top-up also fetches the picks: one `include=` request per blog with picks (≤100 IDs each). No destination reached K = 100.

**Deviation:** T1 planned to measure writer pins per destination with the matcher. That was measured end to end in T5 for Toronto and Funchal instead (Wikidata gazetteers for New York and Tokyo would have taken several minutes each).

## T2. Per-destination selection

`go test ./internal/writermap/ -run 'SelectPerDestination' -count=1`

RED:
```
internal/writermap/selection_test.go:219:30: undefined: writermap.DestinationCandidates
internal/writermap/selection_test.go:221:28: undefined: writermap.SelectPerDestination
FAIL	weatherservice/internal/writermap [build failed]
```

GREEN, `go test ./internal/writermap/ -run '^(TestSelectPerDestination|TestSelect_|TestTrim_|TestCountDestination)' -count=1 -v`:
```
--- PASS: TestSelectPerDestination_EachDestinationGetsItsOwnQuota (0.00s)
--- PASS: TestSelectPerDestination_OnePostPickedTwiceIsKeptOnce (0.00s)
--- PASS: TestSelectPerDestination_OverTheLimitDropsTheLowestRankedAcrossDestinations (0.00s)
--- PASS: TestSelectPerDestination_OneDestinationPicksLikeSelect (0.00s)
--- PASS: TestSelectPerDestination_NoRoomPicksNothing (0.00s)
ok  	weatherservice/internal/writermap	0.589s
```
The 14 existing `Select`, `Trim` and `CountDestinationPosts` tests passed in the same run.

## T3. Monthly run with per-destination picks

`go test ./internal/writermap/ -count=1`

RED 1 (the ceiling field): `job.MaxPosts undefined (type *writermap.SyncJob has no field or method MaxPosts)`.
RED 2, after adding only the field:
```
--- FAIL: TestSyncJob_EachDestinationGetsItsOwnQuota (0.00s)
        	Messages:   	Madeira keeps its 100 and Toronto adds its 30
        	Messages:   	Madeira's oldest 50 are still left out
FAIL
```
GREEN: `ok  	weatherservice/internal/writermap	0.472s`.

`TestSyncJob_PostsNoDestinationPicksArePurged` and `TestSyncJob_TheUnionStaysWithinTheCeiling` passed at RED 2 too: they guard behaviour the shared cap already had.

**Existing tests:** the test helper sets `MaxPosts = 100`, the size of one destination's quota, so the blocked-blog, empty-search and overshoot tests keep their numbers and meaning (held rows count inside the ceiling). Three assertions now name `PostsPerDestination` instead of `MaxIndexedPosts`.

## T4. Top-up job and trigger

**New-destination hook** — `go test ./internal/adapters/sqlite/ -run OnNewDestination -count=1`
RED: `c.OnNewDestination undefined (type *cachedBasePlaces has no field or method OnNewDestination)`
GREEN: `go test ./internal/adapters/sqlite/ -count=1 -race` → `ok  	weatherservice/internal/adapters/sqlite	3.886s`

**One destination's recompute** — `go test ./internal/writermap/ -run RecomputeDestination -count=1`
RED: `source.RecomputeDestination undefined (type *MapSource has no field or method RecomputeDestination)`
GREEN: `ok  	weatherservice/internal/writermap	0.357s`

**Top-up, queue, budget, schedule** — `go test ./internal/writermap/ -count=1`
RED: `job.TopUp undefined (type *writermap.SyncJob has no field or method TopUp)`, `job.RecomputeDestination undefined`.
First GREEN run, `go test ./internal/writermap/ -count=1 -race`:
```
--- FAIL: TestSyncJob_ScheduleRetriesWhileATopUpRuns (0.00s)
        	Error:      	"[1 2 3]" should have 5 item(s), but has 3
        	Messages:   	the startup run happened after the top-up
```
The test was wrong, not the code: it didn't list Toronto as cached, so the monthly run correctly purged its posts. In the app a destination is always cached before its top-up (the hook fires after the store). The test now sets `job.Destinations`. Then:
```
go test ./internal/writermap/ -count=1 -race                          → ok  	weatherservice/internal/writermap	1.742s
go test ./internal/writermap/ -count=5 -race -run 'TopUp|Schedule'    → ok  	weatherservice/internal/writermap	1.711s
```

**Can these tests fail?** The schedule and budget tests first failed only by not compiling, so both behaviours were disabled temporarily (`if false && …`) and restored:
```
--- FAIL: TestSyncJob_TopUpsOverTheDailyBudgetWaitForTheNextDay (0.01s)
--- FAIL: TestSyncJob_ScheduleRetriesWhileATopUpRuns (0.00s)
```
After the restore: `ok  	weatherservice/internal/writermap	0.354s`.

**Added beyond the plan:** `Schedule` retries after a minute when a top-up holds the lock. Without it, a top-up running at startup or at 02:00 UTC would make `Run` return `ErrSyncRunning`, and the run would wait a whole day.

## Guarantees

| What is guaranteed | Test | Level | Result |
|---|---|---|---|
| Each destination gets its own quota; one doesn't take another's | `TestSelectPerDestination_EachDestinationGetsItsOwnQuota`, `TestSyncJob_EachDestinationGetsItsOwnQuota` | unit | PASS |
| A post picked twice is stored once | `TestSelectPerDestination_OnePostPickedTwiceIsKeptOnce` | unit | PASS |
| Over the ceiling, the lowest-ranked across destinations go | `TestSelectPerDestination_OverTheLimitDropsTheLowestRankedAcrossDestinations`, `TestSyncJob_TheUnionStaysWithinTheCeiling` | unit | PASS |
| One destination picks exactly as before | `TestSelectPerDestination_OneDestinationPicksLikeSelect` + existing `TestSelect_*`, `TestSyncJob_*` | unit | PASS |
| Posts no destination picks are purged | `TestSyncJob_PostsNoDestinationPicksArePurged` | unit | PASS |
| First cache fires the hook once, after the row is stored; hits, refreshes and failures don't | `TestWikidataNearbyCache_OnNewDestination_CalledOnceAfterTheFirstStore`, `…_NotCalledWhenTheLoadFails` | SQLite integration | PASS |
| A top-up fetches only new posts, doesn't count as a monthly run, recomputes the destination | `TestSyncJob_TopUp_IndexesOnlyTheNewDestinationsPosts` | unit | PASS |
| Top-ups rank with the monthly tally and never re-request comments; newest without a tally | `TestSyncJob_TopUp_RanksByTheLastMonthlyTally`, `TestSyncJob_TopUp_WithoutATallyTakesTheNewest` | unit | PASS |
| A top-up over the ceiling trims across destinations and logs blogs | `TestSyncJob_TopUp_OverTheCeilingTrimsTheLowestRankedAcrossDestinations` | unit | PASS |
| Blocked blogs aren't requested; a 403 marks a blog blocked; a failed blog doesn't stop the rest | `TestSyncJob_TopUp_SkipsBlockedBlogsAndMarksNewlyBlockedOnes`, `TestSyncJob_TopUp_FailedSearchKeepsTheOtherBlogs` | unit | PASS |
| Nothing new → no recompute | `TestSyncJob_TopUp_NothingNewSkipsTheRecompute` | unit | PASS |
| A top-up waits for a running sync | `TestSyncJob_TopUp_WaitsForARunningSync` | unit | PASS |
| Each destination is topped up once | `TestSyncJob_QueuedTopUpsRunOnceEach` | unit | PASS |
| The 21st top-up of a UTC day waits for the next day | `TestSyncJob_TopUpsOverTheDailyBudgetWaitForTheNextDay` | unit | PASS |
| A busy startup retries after a minute | `TestSyncJob_ScheduleRetriesWhileATopUpRuns` | unit | PASS |
| One destination's stored result can be recomputed alone | `TestMapSource_RecomputeDestinationReplacesOnlyThatDestination` | unit | PASS |

## Coverage and gate

`go test -count=1 -cover ./internal/writermap/ ./internal/adapters/sqlite/` →
```
ok  	weatherservice/internal/writermap	0.639s	coverage: 94.5% of statements
ok  	weatherservice/internal/adapters/sqlite	2.072s	coverage: 83.4% of statements
```
`make test lint build` (golangci-lint from `/tmp/traveltab-lint-bin` on `PATH`) → exit 0, `0 issues.`, `go build -o bin/traveltab ./cmd/web`.

## T5. End to end (2026-10-13)

`DB_PATH=/private/tmp/destination-posts-e2e.db WRITERS_MAP=1 WRITERS_SYNC=1 ./bin/traveltab`, on a new, empty database.

| Time (UTC) | What happened |
|---|---|
| 20:01:06 | Startup sync starts (terms: Madeira, São Miguel; nothing cached yet) |
| 20:01:15 | Toronto map opened: `loading: true` |
| 20:04:22–20:04:45 | Startup sync indexes 75 posts across 11 blogs |
| 20:04:33 | Toronto loaded: **200 pins, 0 writer pins** (first visit, as planned) |
| 20:05:21 | Startup sync finishes; it refreshes the 1 cached destination (Toronto) |
| 20:05:47 | `writers top-up toronto: indexed 19 posts, trimmed 0` (ran right after the sync released the lock) |
| reload | Toronto: **200 pins, 98 writer pins** |

The CN Tower panel lists 11 posts, all Toronto guides (Alma de Viajante, The Planet D, Earth Trekkers, Nomadic Matt).

**Madeira didn't lose pins.** Funchal, first visit: **122 writer pins** from the startup sync's Madeira posts. These are the same posts the old shared cap picked: Madeira was the only searched destination with posts, and 75 is under 100. After `writers top-up funchal: indexed 41 posts, trimmed 0`: **148 writer pins**.

Index: 135 posts (75 + 19 + 41), 2.0 MB. The log contains no post text.

Lisbon wasn't loaded in T5: its Wikidata load takes several minutes. Its top-up search was measured in T1 (37 picks).

## Known gaps

- **Pins appear on the next visit,** not on the open map (owner decision).
- **Other destinations' stored results** wait for the monthly recompute after a top-up adds or trims posts (owner decision).
- **After a restart,** top-ups rank by newest until the next monthly run (no stored tally), and posts indexed before the restart rank lowest when a top-up trims.
- **A trim can't name destinations,** only blogs: the index doesn't record which destination picked a post.
- **"Once per process":** after a restart, a destination cached for the first time is still topped up once, but one cached before the restart isn't topped up again; the monthly run covers it.
