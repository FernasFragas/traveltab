# Task 3 notes: Wikipedia + Wikidata place source

Owner files: `api/wikimedia.go`, `api/wikimedia_test.go`, `api/testdata/wikimedia/**`, this file.

## Completion pass (2026-09-20)

Completed the interrupted parallel lookup implementation already in the working tree. Split
geosearches, page-property batches and Wikidata entity batches now run with at most four
workers per lookup. Results and errors are merged in input order, and all workers finish
before the call returns. Queued batches check cancellation before sending requests.

RED: after supplying a sequential helper to restore compilation, the existing
`TestWikimedia_PlacesNear_LooksUpBatchesTogetherButAtMostFourAtATime` failed on the assertion
`"1" is not greater than "1"`. GREEN: `go test -race -count=1 -run Wikimedia ./api/`
passes, including concurrency bounds, out-of-order completion and deterministic failure tests.
The live Wikimedia test remains opt-in. These tests establish concurrent behavior; the earlier
25-second live cold-start measurement has not been repeated after this change.

## Fixtures

Recorded live on 2026-09-19 with curl (User-Agent set), then trimmed with a throwaway Python script:

| File | What it is | Trimming |
|---|---|---|
| `geosearch_tavira.json` | `list=geosearch` around 37.1264,-7.6506, 10 km (24 results) | as recorded |
| `pageprops_tavira.json` | `prop=pageprops&ppprop=wikibase_item` for those 24 page IDs | as recorded |
| `sitelinks_tavira.json` | `wbgetentities&props=sitelinks` for the 24 Wikidata IDs | `badges` dropped |
| `entities_tavira.json` | `wbgetentities&props=claims\|labels&languages=en` | only P31 and P18 claims, only the `en` label, references/qualifiers/hashes dropped |

All four are well under 200 KB (largest 25 KB). The old hand-written `tavira.json` stub fixture was replaced
by these four, because one file cannot answer four different endpoints.

Tests that need many pages (the 500-result cap, the top-300 cut) build their data in Go instead
(`wikiTownPages` + `fakeWikimedia`), so no huge fixture is committed.

## RED evidence

Tests written first, against the zero-value stub bodies (`return nil, nil`). Every one fails on an
assertion, none on a compile error:

```
$ CGO_ENABLED=1 go test -race -count=1 -run Wikimedia ./api/
--- FAIL: TestWikimedia_PlacesNear_ReturnsPlacesWithWikidataDetails (0.00s)
    wikimedia_test.go:333: 
        	Error Trace:	/Users/fernando/personal-projects/traveltab/api/wikimedia_test.go:333
        	Error:      	Expected value not to be nil.
        	Test:       	TestWikimedia_PlacesNear_ReturnsPlacesWithWikidataDetails
        	Messages:   	Castle of Tavira should be among the places near Tavira
--- FAIL: TestWikimedia_PlacesNear_LeavesKindEmpty (0.00s)
    wikimedia_test.go:347: 
        	Error Trace:	/Users/fernando/personal-projects/traveltab/api/wikimedia_test.go:347
        	Error:      	Should NOT be empty, but was []
        	Test:       	TestWikimedia_PlacesNear_LeavesKindEmpty
--- FAIL: TestWikimedia_PlacesNear_UsesTheTitleWhenThereIsNoEnglishLabel (0.00s)
    wikimedia_test.go:369: 
        	Error Trace:	/Users/fernando/personal-projects/traveltab/api/wikimedia_test.go:369
        	Error:      	"[]" should have 1 item(s), but has 0
        	Test:       	TestWikimedia_PlacesNear_UsesTheTitleWhenThereIsNoEnglishLabel
--- FAIL: TestWikimedia_PlacesNear_SplitsACappedSearch (0.00s)
    wikimedia_test.go:390: 
        	Error Trace:	/Users/fernando/personal-projects/traveltab/api/wikimedia_test.go:390
        	Error:      	Expected value not to be nil.
        	Test:       	TestWikimedia_PlacesNear_SplitsACappedSearch
        	Messages:   	a place 9.5 km out is only found by an outer search
--- FAIL: TestWikimedia_PlacesNear_DropsSplitResultsBeyond10Km (0.00s)
    wikimedia_test.go:410: 
        	Error Trace:	/Users/fernando/personal-projects/traveltab/api/wikimedia_test.go:410
        	Error:      	"[]" should have 500 item(s), but has 0
        	Test:       	TestWikimedia_PlacesNear_DropsSplitResultsBeyond10Km
--- FAIL: TestWikimedia_PlacesNear_ReturnsEachPlaceOnce (0.00s)
    wikimedia_test.go:436: 
        	Error Trace:	/Users/fernando/personal-projects/traveltab/api/wikimedia_test.go:436
        	Error:      	Not equal: 
        	            	expected: 1
        	            	actual  : 0
        	Test:       	TestWikimedia_PlacesNear_ReturnsEachPlaceOnce
        	Messages:   	the centre and the northern search both return this place
--- FAIL: TestWikimedia_PlacesNear_DoesNotSplitAnUncappedSearch (0.00s)
    wikimedia_test.go:450: 
        	Error Trace:	/Users/fernando/personal-projects/traveltab/api/wikimedia_test.go:450
        	Error:      	"[]" should have 218 item(s), but has 0
        	Test:       	TestWikimedia_PlacesNear_DoesNotSplitAnUncappedSearch
    wikimedia_test.go:451: 
        	Error Trace:	/Users/fernando/personal-projects/traveltab/api/wikimedia_test.go:451
        	Error:      	Not equal: 
        	            	expected: 1
        	            	actual  : 0
        	Test:       	TestWikimedia_PlacesNear_DoesNotSplitAnUncappedSearch
        	Messages:   	218 results fit in one search
--- FAIL: TestWikimedia_PlacesNear_HasNoTypesBeyondTheTop300 (0.00s)
    wikimedia_test.go:466: 
        	Error Trace:	/Users/fernando/personal-projects/traveltab/api/wikimedia_test.go:466
        	Error:      	"[]" should have 320 item(s), but has 0
        	Test:       	TestWikimedia_PlacesNear_HasNoTypesBeyondTheTop300
--- FAIL: TestWikimedia_SendsTheUserAgent (0.00s)
    wikimedia_test.go:483: 
        	Error Trace:	/Users/fernando/personal-projects/traveltab/api/wikimedia_test.go:483
        	Error:      	Should NOT be empty, but was []
        	Test:       	TestWikimedia_SendsTheUserAgent
--- FAIL: TestWikimedia_ErrorIncludesTheStatusOnNon200 (0.00s)
    wikimedia_test.go:498: 
        	Error Trace:	/Users/fernando/personal-projects/traveltab/api/wikimedia_test.go:498
        	Error:      	An error is expected but got nil.
        	Test:       	TestWikimedia_ErrorIncludesTheStatusOnNon200
--- FAIL: TestWikimedia_StopsWhenTheContextIsCancelled (0.00s)
    wikimedia_test.go:510: 
        	Error Trace:	/Users/fernando/personal-projects/traveltab/api/wikimedia_test.go:510
        	Error:      	Target error should be in err chain:
        	            	expected: "context canceled"
        	            	in chain: 
        	Test:       	TestWikimedia_StopsWhenTheContextIsCancelled
--- FAIL: TestWikimedia_GetEntities_ReturnsTheRequestedEntities (0.00s)
    wikimedia_test.go:521: 
        	Error Trace:	/Users/fernando/personal-projects/traveltab/api/wikimedia_test.go:521
        	Error:      	Should be true
        	Test:       	TestWikimedia_GetEntities_ReturnsTheRequestedEntities
        	Messages:   	the castle entity should be returned
--- FAIL: TestWikimedia_NewWithoutAClientUsesA30SecondTimeout (0.00s)
    wikimedia_test.go:534: 
        	Error Trace:	/Users/fernando/personal-projects/traveltab/api/wikimedia_test.go:534
        	Error:      	Expected value not to be nil.
        	Test:       	TestWikimedia_NewWithoutAClientUsesA30SecondTimeout
FAIL
FAIL	weatherservice/api	0.283s
FAIL
```

13 tests, 13 assertion failures. `TestWikimediaLive_FindsBelemTowerInLisbon` skipped (no `LIVE_API_TESTS=1`).

## GREEN

```
$ CGO_ENABLED=1 go test -race -count=1 -run Wikimedia ./api/
--- PASS: TestWikimedia_PlacesNear_ReturnsPlacesWithWikidataDetails (0.01s)
--- PASS: TestWikimedia_PlacesNear_LeavesKindEmpty (0.01s)
--- PASS: TestWikimedia_PlacesNear_UsesTheTitleWhenThereIsNoEnglishLabel (0.00s)
--- PASS: TestWikimedia_PlacesNear_SplitsACappedSearch (0.05s)
--- PASS: TestWikimedia_PlacesNear_DropsSplitResultsBeyond10Km (0.05s)
--- PASS: TestWikimedia_PlacesNear_ReturnsEachPlaceOnce (0.05s)
--- PASS: TestWikimedia_PlacesNear_DoesNotSplitAnUncappedSearch (0.02s)
--- PASS: TestWikimedia_PlacesNear_HasNoTypesBeyondTheTop300 (0.47s)
--- PASS: TestWikimedia_SendsTheUserAgent (0.01s)
--- PASS: TestWikimedia_ErrorIncludesTheStatusOnNon200 (0.00s)
--- PASS: TestWikimedia_StopsWhenTheContextIsCancelled (0.00s)
--- PASS: TestWikimedia_GetEntities_ReturnsTheRequestedEntities (0.00s)
--- PASS: TestWikimedia_NewWithoutAClientUsesA30SecondTimeout (0.00s)
--- SKIP: TestWikimediaLive_FindsBelemTowerInLisbon (0.00s)
PASS
ok  	weatherservice/api	1.930s
```

`golangci-lint run ./api/` reports 0 issues, `gofmt -l api/` is empty, and `go build ./...` passes.
(`make lint` over the whole repo fails on `cmd/loadtest-server/main.go:74` errcheck, which belongs to
another lane.)

## Live check (run once, by hand)

```
$ LIVE_API_TESTS=1 go test -count=1 -v -timeout 600s -run WikimediaLive ./api/
2026/09/19 23:36:17 wikimedia: geosearch around 38.7223,-9.1393 hit the 500 result cap, splitting into 7 smaller searches
--- PASS: TestWikimediaLive_FindsBelemTowerInLisbon (16.87s)
```

The first run of this test failed, and the failure was real: the English Wikipedia page "Belém Tower"
carries Wikidata ID **Q215003**. Q193386, the example ID in the contract's `Place.ID` comment, is the
Bolivia men's national football team. The test now expects Q215003. Nothing in the contract needs
changing, but the comment in `planner/types.go` is a poor example.

Lisbon caps the first 10 km search, so the live run exercises the split path end to end: 7 searches,
then page properties, sitelinks and details in batches of 50, in about 17 seconds.

## How it works

`PlacesNear`:

1. `geosearch` 10 km, `gslimit=500`, `formatversion=2` on en.wikipedia.org.
2. Exactly 500 results means the search was capped: the results are dropped and 7 searches of 5 km
   run instead (the centre and 6 points 8.66 km out at 0°, 60°, … 300°, with `destination()` doing
   the great-circle offset). Results merge by page ID, and anything further than 10 km from the
   centre is dropped.
3. `prop=pageprops&ppprop=wikibase_item`, 50 page IDs per request. Pages with no Wikidata item are
   dropped, and two pages sharing one item are kept once.
4. `wbgetentities props=sitelinks`, 50 per request, for the sitelink count.
5. Sort by sitelinks (most first, ties by ID) and fetch `props=claims|labels&languages=en` for the
   top 300 only: P31 types, P18 image and the English label. Places past 300 keep their Wikipedia
   title and have no types or image.

`Kind` is never set here; `Rank` (Task 7) does that. Results come back most famous first.

`GetEntities(ctx, ids, props)` is the shared Wikidata helper Task 13 reuses. It batches 50 IDs per
request and always sends `languages=en`, which is harmless for `props=sitelinks`.

## Notes and deviations

- Added one test beyond the Task 3 list, `TestWikimedia_NewWithoutAClientUsesA30SecondTimeout`,
  because the acceptance criteria ask for that behavior and nothing else covered it.
- `TestWikimedia_StopsWhenTheContextIsCancelled` relies on the stub transport returning
  `req.Context().Err()`, the way a real transport does. It fails if the client builds requests
  without the context, which is the point of the test.
- `TestWikimedia_PlacesNear_ReturnsEachPlaceOnce` was checked by mutation: removing both dedup
  guards makes it fail with "expected 1, actual 2", so it is not passing by accident. Its fake data
  asserts up front that the shared place really sits inside two of the seven searches.
- Known limit of the spec'd approach: in a very dense city a 5 km split search can cap at 500 too,
  and there is no second level of splitting. Lisbon does not hit this (the live test finds Belém
  Tower, 7.5 km out).
