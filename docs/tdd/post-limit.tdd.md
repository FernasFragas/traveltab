# Post limit: TDD Evidence

## L2. Client reads: search, comment counts, posts by ID

**Source task:** `tasks/post-limit-todo.md` L2. Files: `internal/adapters/api/wordpress.go`, `wordpress_test.go`. Responses are inline synthetic JSON (short fake titles), no network.

### User journeys

- As the owner, I want the client to search a blog by destination, tally recent comments and fetch chosen posts by ID, so that the sync can pick the 100 best-known posts and download only those.
- As a blog host, I want robots.txt, the shared limiter and the user agent respected on every new read.

### RED and GREEN

RED: the tests reference the missing methods. `go test ./internal/adapters/api/ -run 'SearchPosts|RecentComment|PostsByID|NewReads'` gave `testWordPressClient(server).SearchPosts undefined (type *WordPressClient has no field or method SearchPosts)`, `undefined: WordPressHit`, and the same for `RecentCommentCounts` and `PostsByID`.

GREEN: the same command gave `ok ... api` with all 11 new tests `--- PASS`. `go test -cover ./internal/adapters/api/` gave `coverage: 83.0% of statements`. `go vet` and `gofmt -l` were clean. `make test` passed for every package; `make lint` could not run (`golangci-lint` built with go1.26, config targets 1.27.2).

### Guarantees

| What is guaranteed | Test name | Type | Result |
| --- | --- | --- | --- |
| Search sends `search`, `orderby=date`, `order=desc`, `_fields=id,link,slug,title,date_gmt,date` (no content), decodes entities, falls back to `date` | `TestWordPress_SearchPosts_ReturnsIDsTitlesAndSlugs` | integration | PASS |
| A page past the last one ends cleanly | `TestWordPress_SearchPosts_PastLastPageEndsCleanly` | integration | PASS |
| Comments are tallied per post over 2 pages and stop at `maxPages`, with `_fields=post` | `TestWordPress_RecentCommentCounts_TalliesCommentsPerPost` | integration | PASS |
| Stops at the last page | `TestWordPress_RecentCommentCounts_StopsAtTheLastPage` | integration | PASS |
| 404 on comments gives an empty map and no error | `TestWordPress_RecentCommentCounts_NotFoundMeansNoComments` | integration | PASS |
| Only the given IDs are requested (`include=`) | `TestWordPress_PostsByID_FetchesOnlyTheGivenIDs` | integration | PASS |
| IDs go in batches of 100; none means no request | `TestWordPress_PostsByID_BatchesOfAHundred`, `TestWordPress_PostsByID_NoIDsMakesNoRequest` | integration | PASS |
| robots-disallowed returns `ErrDisallowed` with no API call; 403 returns `ErrBlocked`; other statuses are errors, for all three reads | `TestWordPress_NewReads_RobotsDisallowReturnsErrDisallowed`, `..._403ReturnsErrBlocked`, `..._OtherStatusIsError` | integration | PASS |

### Known gaps

- Limiter and user agent reuse `do`, already covered by `TestWordPress_EveryRequestWaitsForTheHostLimiter`.
- The new 404 case in `getOnce` is also what `PostContent` now returns for a missing post (a plain error, as before).

## L3. Select at most 100 posts and wire the sync

**Source task:** `tasks/post-limit-todo.md` L3. Files: `internal/writermap/selection.go`, `selection_test.go`, `sync.go`, `sync_test.go`, and `cmd/web/main.go` (`wireWriterSync`, `cachedPlaceNames`, the `wordPressPosts` adapter). Tests use a fake `PostSource` that counts searches and fetches; the blocked-host test uses `httptest` with the real client. No network.

### User journeys

- As the owner, I want the index capped at 100 posts, so that syncs are light on the blogs and on our storage.
- As a traveller, I want the posts behind the pins to be each writer's best-known work about the destination, so that the links are worth opening.

### Design choices

- **Destination post:** a search hit whose title or slug, folded with the matcher's `foldName` (accent- and case-insensitive), contains a known place name as whole words.
- **Known place names:** the search terms (override area names plus cached destination names) plus the matcher's names for every cached destination's base places. `cmd/web` builds the second list with `MatchLookupPhrases` (so the stoplist and 5-letter minimum apply) and passes it as `SyncJob.PlaceNames`.
- **Search terms:** override areas plus cached destinations, de-duplicated by folded name, 1 search (page 1) per term per blog. `Area` has no aliases, so a name is its only spelling.
- **Share:** proportional, largest remainder, at least 1 per blog with a destination post, never above a blog's candidates or the total cap. With more blogs than slots, the biggest blogs keep the slots.
- **"Due" rule (no cursor):** a run is due when any non-blocked source has never finished (`LastFull` zero, any hour) or, in the off-peak window, its last finished run was in an earlier month. The cap is shared, so one due source re-picks every blog. `LastFull` is set to now when a source finishes; `Cursor` and `LastDelta` are no longer written.
- **Failure rule:** an error while gathering candidates stops the run before any write, so a half-seen set never purges rows; the sources stay due and the next window retries. A blocked source is marked blocked and left out; its existing rows stay. A fetch error keeps that blog's rows and leaves it due.
- **Summary step:** runs after a finished run if a post was added or removed.

### RED and GREEN

RED, selection: `go test ./internal/writermap/ -run TestSelect` failed to compile with `undefined: writermap.BlogCandidates` (and `Select`, `SearchHit`, `Pick`), the missing code the tests reference.

RED, sync: `go vet ./internal/writermap` gave `sync_test.go:687:40: undefined: writermap.PostSource`.

GREEN: after `selection.go` the selection tests gave `ok ... coverage: 14.7%` (only that file ran). After the `sync.go` rewrite one test failed because of a test setup mistake (a post the Madeira search could not find); fixed the fixture, then `go test -race -cover ./internal/writermap/` gave `ok ... coverage: 94.7% of statements`. `go vet ./...` and `gofmt -l internal cmd` were clean. `make test build` passed; `make lint` was not run (golangci-lint go1.26 / go1.27.2 mismatch).

### Guarantees

| What is guaranteed | Test name | Type | Result |
| --- | --- | --- | --- |
| Only hits whose title or slug names a place count; accents and case are ignored; body-only hits are dropped | `TestSelect_KeepsOnlyPostsWhoseTitleOrSlugNamesAPlace` | unit | PASS |
| Duplicate hits (several terms) count once | `TestSelect_DropsDuplicateHitsFromSeveralSearchTerms` | unit | PASS |
| Shares are proportional with largest remainder and total exactly 100 | `TestSelect_SplitsProportionallyWithLargestRemainder` | unit | PASS |
| Every blog with a destination post gets at least 1 | `TestSelect_GivesEveryBlogWithADestinationPostAtLeastOne` | unit | PASS |
| A small blog is not given more than it has; the total stays 100 | `TestSelect_NeverGivesABlogMoreThanItsCandidates` | unit | PASS |
| Comments descending, then newest | `TestSelect_RanksByCommentsThenNewest` | unit | PASS |
| No comment data: newest | `TestSelect_BlogWithoutCommentsTakesItsNewest` | unit | PASS |
| Fewer candidates than the cap: all kept | `TestSelect_FewerCandidatesThanTheCapKeepsThemAll` | unit | PASS |
| No places or blogs: nothing; more blogs than slots: the cap holds | `TestSelect_NoPlacesOrNoBlogsPicksNothing`, `TestSelect_MoreBlogsThanTheCapKeepsTheCapAndTheBiggestBlogs` | unit | PASS |
| Only the 100 chosen posts are fetched and stored | `TestSyncJob_IndexesOnlyTheSelectedPosts` | integration | PASS |
| One search per distinct term per blog; a post found twice is fetched once | `TestSyncJob_SearchesEachTermOncePerBlogAndFetchesEachPostOnce` | integration | PASS |
| Matcher place names make a title a destination title | `TestSyncJob_PlaceNamesFromTheMatcherCountAsDestinationNames` | integration | PASS |
| Name-listing or destination-listing errors stop the run before any request | `TestSyncJob_PlaceNamesErrorStopsTheRunBeforeAnyRequest`, `TestSyncJob_ListingCachedDestinationsFailsTheRunBeforeAnyRequest` | integration | PASS |
| Month 2 re-picks: dropped posts are deleted, new ones indexed, 100 total | `TestSyncJob_PurgesPostsThatDropOut` | integration | PASS |
| The cap is shared across blogs by destination posts (75/25) | `TestSyncJob_SharesTheCapAcrossBlogsByTheirDestinationPosts` | integration | PASS |
| Comment ranking and the newest fallback work through the sync | `TestSyncJob_UsesCommentCountsToRankAndNoCommentsNewest` | integration | PASS |
| Due only in a new month and the off-peak window; one due source re-picks all | `TestSyncJob_RunWaitsForNewMonthAndOffPeak`, `TestSyncJob_DueRunSearchesEveryActiveBlogEvenWhenOnlyOneIsDue` | integration | PASS |
| A failed run writes nothing and retries | `TestSyncJob_FailedRunWritesNothingAndRetriesNextWindow`, `TestSyncJob_CommentErrorFailsTheRun`, `TestSyncJob_FetchErrorKeepsThatBlogsRowsAndRetries` | integration | PASS |
| 403 and robots disallow mark the host blocked, no further requests | `TestSyncJob_BlockedOrDisallowedHostIsMarkedAndSkipped` | integration | PASS |
| `removed` purges the host; allowlist-blocked is skipped | `TestSyncJob_SkipsBlockedAndRemovedSources` | integration | PASS |
| One run at a time; refresh then summary order; summary skipped when nothing changed or nothing is due | `TestSyncJob_ConcurrentRunIsRejected`, `..._RefreshesCachedDestinationsThenRunsSummaryStep`, `..._SummaryStepIsSkippedWhenNothingWasIndexed`, `..._NothingDueSkipsRefreshAndSummaryStep` | integration | PASS |
| One log line per host, no post text | `TestSyncJob_LogsOneLinePerHostWithoutPostText` | integration | PASS |
| Schedule retries a failed startup run at 02:00; unchanged runs keep SQLite counts and size | `TestSyncJob_ScheduleRunsAtStartupThenEachOffPeakWindow`, `..._DefaultSleepStopsWhenContextEnds`, `..._TwoUnchangedRunsKeepRowCountsAndFileSize` | integration | PASS |

### Coverage and known gaps

`go test -race -cover ./internal/writermap/` gave 94.7%. `cmd/web` has no tests; `cachedPlaceNames` and the adapter are compile-checked only (the adapter mirrors the tested `clientSource`). Not measured: real request counts per run (L4).

### L3 follow-up: owner decisions 6 and 7

**Source:** `tasks/post-limit-plan.md` Owner decisions 6 and 7. Files: `internal/writermap/selection.go`, `sync.go` and their tests.

**Journeys**

- As the owner, I want a temporarily blocked blog's pins to survive, and the 100-post cap to hold anyway, so that a 403 neither loses data nor lets the index grow.
- As the owner, I want a blog whose search suddenly returns nothing to keep its rows and be retried, so that one odd response does not wipe its pins.

**Design choices**

- **Held rows:** the rows of every blocked blog (allowlist `blocked`, state `blocked`, or blocked during the run) and of every blog with a sudden empty search are counted with `SyncIndex.PostIDs`. `Select` gets `MaxIndexedPosts` minus that count (`Select` returns nothing for a limit of 0 or less), so the active blogs share what is left.
- **Sudden empty search:** the blog's search hits are empty (all terms, no error) and it has indexed posts. The run logs `writers sync <host>: warning: search returned no posts, keeping N indexed posts, retrying next run`, does not save its state (not finished, so it stays due), and holds its rows. A blog with no indexed posts and no hits goes through the normal path: no picks, state saved, no warning.
- **Not covered:** a blog that has hits but none naming a place still drops all its rows (the check is on hits, not destination posts).

**RED:** `go test ./internal/writermap/ -run 'Blocked|SuddenEmpty|NeverHad'` failed for the intended reason: `should have 35 item(s), but has 50` (blocked cases, active blogs ignored the 30 held rows), and in the empty-search test `should have 30 item(s), but has 0` ("the rows are kept") and `should have 70 item(s), but has 100` ("and count inside the cap"). The never-had-posts test passed at once (it pins today's behaviour).

**GREEN:** the same command passed. `go test -race -cover ./internal/writermap/` gave `coverage: 94.6% of statements`.

| What is guaranteed | Test name | Type | Result |
| --- | --- | --- | --- |
| A blocked blog (allowlist or earlier run) keeps 30 rows; two active blogs get 35 each (total 100) | `TestSyncJob_BlockedBlogsRowsStayAndCountInsideTheCap` | integration | PASS |
| A blog that returns 403 during the run keeps its rows inside the cap | `TestSyncJob_BlogBlockedDuringTheRunKeepsItsRowsInsideTheCap` | integration | PASS |
| Sudden empty search: rows kept (30), other blog gets 70, warning names the host, state not advanced; next run re-picks (50/50) | `TestSyncJob_SuddenEmptySearchKeepsRowsWarnsAndRetries` | integration | PASS |
| A blog that never had posts and finds none is skipped without a warning, and counts as finished | `TestSyncJob_BlogThatNeverHadPostsAndFindsNoneIsJustSkipped` | integration | PASS |
| No room left (limit 0 or less) picks nothing | `TestSelect_NoRoomLeftPicksNothing` | unit | PASS |

### Orphan cleanup (Part B)

`WordPressClient.ListPosts` and `WordPressClient.PostIDs` (the old cursor crawl and deleted-posts sweep reads) had no caller outside their own tests (`grep` over `cmd/`, `internal/`). Both and their 7 tests were removed. Tests of the shared behaviour (robots cache, 403, limiter, missing robots) that used `PostIDs` now call `SearchPosts`; one new test, `TestWordPress_FirstPage400IsError`, keeps the first-page 400 case that only `ListPosts` covered. `listPage`, `wordpressPosts`, `morePages` and `wordpressListFields` are still used by `SearchPosts`, `RecentCommentCounts` and `PostsByID`. `SyncIndex.PostIDs` (the SQLite read) is still used by the sync. `go test -race -cover ./internal/adapters/api/` gave `coverage: 82.8% of statements`.

## L4. Measure coverage and document

**Source task:** `tasks/post-limit-todo.md` L4. Files: `docs/maintenance.md` (writers' map section), `tasks/post-limit-results.md` (`## Coverage (L4)`), this section. No production code.

### User journeys

- As the owner, I want to see what one real sync indexes and how much of Madeira it covers, so that I can accept the cap before 8b.
- As an operator, I want `docs/maintenance.md` to describe the cap, the picking rules and the blocked and empty-search handling.

### Commands and results (real, 2026-10-11)

| Step | Command | Result |
| --- | --- | --- |
| Cache Madeira first (sync off) | `DB_PATH=<scratch>/post-limit.db WRITERS_MAP=1 ./web`, then `curl /trip/funchal-pt`, `/map/funchal-pt.geojson?...`, `/map/madeira-pt.geojson?...` | 200 each; `source_cache` held `wikidata:v1:"funchal"...` and `"madeira"...` |
| Live sync | `DB_PATH=<scratch>/post-limit.db WRITERS_MAP=1 WRITERS_SYNC=1 ./web` (scratch copy on port 18080; 8080 was in use) | 11 active blogs synced; `select count(*) from writer_posts` gave `100` |
| Requests | counting transport in the scratch copy | 145 requests: 141 x 200, 1 x 302, 3 x 404; no 403, 429 or robots disallow |
| Time | server log | `20:53:53` to `20:57:32` (3 min 39 s) for the blogs |
| Coverage | `curl /map/madeira-pt.geojson?lat=32.75&lon=-16.96` | 200 pins, all with at least 1 writer; uncapped debug counts: 243 places; 45 of the 49 PR 2 places with a writer |

Per-blog counts, per-status request counts and the method difference against PR 2's 24/49 are in `tasks/post-limit-results.md` under `## Coverage (L4)`. No post text was stored, logged or recorded; only IDs, host names, titles' presence of place names (counts), and QIDs.

### Guarantees

| What is guaranteed | Evidence | Type | Result |
| --- | --- | --- | --- |
| A real run indexes no more than 100 posts | `count(*) from writer_posts` = 100 | live | PASS |
| No blog was blocked or throttled | 0 x 403, 0 x 429, 0 robots disallow in 145 requests | live | PASS |
| No empty-search warning for a blog with rows | no `warning` line in the run log | live | PASS |
| Cap, picking and rules are documented | `docs/maintenance.md` writers' map section | docs | n/a |

### Verification and known gaps

- The sync and selection are covered by L3 and its follow-up tests; this section only records one live run. The numbers come from one run on one day.
- The live run used a scratch copy of the repo (a different port and a request counter). The shipped code was not run as is because port 8080 was in use by another server.
- Only the cached destinations of this run (Funchal, Madeira) and the override areas were search terms. The coverage of other destinations is not measured.
- `make lint` could not run: see the final run below.

## Follow-up 2 (owner decisions 9 and 10)

**Source task:** `tasks/post-limit-todo.md`, L3 follow-ups (decisions 9 and 10 in `tasks/post-limit-plan.md`). Files: `internal/writermap/sync.go`, `selection.go` and their tests.

### User journeys

- As the owner, I want a blog whose search hits name no known place to keep its rows and be retried, so that an odd response does not wipe its pins.
- As the owner, I want the index trimmed to exactly 100 when a mid-run failure leaves it over, so that the cap is a hard cap.

### Design choices

- **Decision 9:** the guard now checks `CountDestinationPosts(blog, known) == 0` (hits that pass the title/URL filter), not `len(blog.Hits) == 0`. Same handling as before: rows kept and held inside the cap, state not saved. Warning: `writers sync <host>: warning: no destination posts matched, keeping N indexed posts, retrying next run`.
- **Decision 10:** after the store loop, `trimOverflow` sums the indexed rows of all active and held blogs. If over `MaxIndexedPosts`, `Trim` ranks the picks of the blogs updated in this run (fewest recent comments, then oldest, across blogs) and the lowest are deleted until the total is 100. Failed, blocked and guard-held blogs are never trimmed. One log line: `writers sync: trimmed N posts to stay within 100`.
- Ranking across blogs compares raw comment counts; blogs with no readable comments rank by date only.

### RED

`go test ./internal/writermap/ -run 'HitsThatName|OverCapAfter|UnderTheCap'` failed for the intended reasons:
- `"map[]" should have 20 item(s), but has 0` ("the rows are kept") and the other blog did not get 80 (decision 9).
- `expected: 81 actual: 54` ("its lowest-ranked (oldest) picks went") and `does not contain "trimmed 27"` (decision 10; the failed blog's 5 candidates leave 97 picks for the other blog, so 27 are trimmed).
- `selection_test.go`: `undefined: writermap.CountDestinationPosts`, `undefined: writermap.Trim`.
- `TestSyncJob_UnderTheCapTrimsNothing` passed at once (pins today's behaviour).

### GREEN

The same command passed; `go test -race -cover ./internal/writermap/` gave `coverage: 94.5% of statements`.

| What is guaranteed | Test name | Type | Result |
| --- | --- | --- | --- |
| 20 indexed rows, hits name no place: rows kept, warning names the host and "no destination posts matched", state not advanced, other blog gets 80 | `TestSyncJob_HitsThatNameNoPlaceKeepRowsWarnAndRetry` | integration | PASS |
| A fetch error on a blog with 30 old rows, new picks for the other: total 100, failed blog's 30 untouched, lowest-ranked of the updated blog dropped, "trimmed 27" logged | `TestSyncJob_OverCapAfterAFailedBlogTrimsTheLowestRankedOfTheUpdatedBlogs` | integration | PASS |
| At or under the cap nothing is trimmed or logged | `TestSyncJob_UnderTheCapTrimsNothing` | integration | PASS |
| Distinct place-naming hits are counted once | `TestCountDestinationPosts_CountsEachPlaceNamingPostOnce` | unit | PASS |
| `Trim` drops the globally lowest-ranked picks, only among the given picks | `TestTrim_DropsTheGloballyLowestRankedPicks` | unit | PASS |
| `Trim` with 0 returns nothing; with more than there are returns all | `TestTrim_NothingToDropOrMoreThanThereIs` | unit | PASS |

### Final run

`go vet ./...` and `gofmt -l internal cmd` clean; `make test build` passed. `make lint` not run (known golangci-lint go1.26/go1.27.2 mismatch). No live network calls.

### Known gaps

- The "not covered" note in the decision 6/7 section above is now fixed: hits that name no place keep the rows.
- Cross-blog trim order compares raw comment counts, which are not comparable between blogs of different size. Accepted for a rare edge case.
