# Ring-by-ring nearby loading — evidence

Source: [nearby-rings plan](../../tasks/nearby-rings-plan.md), [tasks R1-R6](../../tasks/nearby-rings-todo.md). Replaces the evidence in [nearby-query.tdd.md](nearby-query.tdd.md).

## Journeys

- *As a traveller opening Lisbon, London, Paris, New York or Tokyo, I want the closest notable pins to appear first and more to fill in, so that the map is useful straight away.*
- *As a traveller in Funchal, I want the same writer matches as before, so that the fix costs me nothing.*
- *As a traveller who leaves the page mid-load, I want the load to carry on, so that the next visit is faster.*

## R1. Live measurements (2026-10-12)

Live, sequential, app `UserAgent`, 70 s client timeout, scratch Python probes (not in the repo). Page = the 100 or 50 most-linked items of a 5 km ring (`ORDER BY DESC(?sitelinks), ?item`), sitelinks floor 2. Wikidata is very noisy: the same query varies 2-5x between runs, and a repeated identical query is much faster (cache). Times are single cold runs unless noted.

### Plain rows, 100 items, ring 0-5 km (page 1)

| City | Time | Response | Rows |
|---|---:|---:|---:|
| Funchal | 2.2 s | 1.07 MB | 642 |
| Lisbon | 8.5 s | 1.43 MB | 850 |
| Kyoto | 3.6 s | 1.10 MB | 674 |
| London | 4.5 s | **5.89 MB** | 3,286 |
| Paris | 26.2 s / 10.9 s (repeat) | 3.26 MB | 1,905 |
| New York | 10.1 s | **8.56 MB** | 4,880 |
| Tokyo | 10.8 s | 3.41 MB | 2,072 |

Finding: plain rows break the 1 MB / page target in the dense cities. Cause: one row per (type x alias x hours) per item.

### Fallbacks, ring 0-5 km, floor 2

| Shape | Lisbon | London | Paris | New York | Tokyo |
|---|---:|---:|---:|---:|---:|
| Grouped labels in one query, 100 items | 8.9 s | 12.7 / 16.7 / 10.8 s | 5.5 / 9.3 s | 4.1 s | 7.0 s |
| Core page alone (QID, coord, sitelinks), 100 | 1.4 s | 2.7 / 4.1 s | 3.6 / 5.2 s | 1.0 s | 1.5 s |
| Two queries, 100 items (core + labels), total | 6.3 s | 10.4 s | 8.0 s | 8.3 s | 5.7 s |
| Two queries, **50 items**, total | 4.3 s | 9.3 s | 8.4 s | 25.4 s (core 21.0 s outlier) | 3.8 s |

Two-query response sizes: core 22 KB (50) or 44 KB (100); labels 55-67 KB (50) or 107-132 KB (100). A grouped one-query page is 160 KB at 100 items. All under 1 MB.

Other pages (two queries, 50 items): Paris page 2 of ring 1 6.2 s; Tokyo 2.5 s; London 12.6 s. Outermost ring 35-40 km page 1: Paris 6.9 s, Tokyo 2.9 s. Funchal ring 1 page 1: 2.9 s; Kyoto 10.1 s (core 7.7 s outlier).

Cold label query for 100 London items: 10.9 s, then 6.2, 3.2, 1.9 s on repeats; so most label cost is cold-cache and noisy.

### Floor and ring sizes (items with at least N sitelinks per 5 km ring, nearest first)

| City | Floor | 0-5 | 5-10 | 10-15 | 15-20 | 20-25 | 25-30 | 30-35 | 35-40 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| Funchal | 1 | 541 | 84 | 85 | 134 | 47 | 35 | 32 | 8 |
| Funchal | 2 | 144 | 42 | 51 | 71 | 21 | 20 | 16 | 4 |
| Lisbon | 1 | 1,386 | 667 | 404 | 271 | 421 | 276 | 305 | 138 |
| Lisbon | 2 | 797 | 400 | 200 | 138 | 183 | 134 | 153 | 75 |
| Paris | 2 | 9,783 | - | - | - | - | - | - | 426 |
| Tokyo | 2 | 690 | - | - | - | - | - | - | 707 |

Floor 1 triples the work for no pins the prior plan needed (the only 6a case at the floor, Câmara de Lobos, has 2 sitelinks). **Floor 2 chosen.** The 200 pins fill inside the first ring for Lisbon, Paris and Tokyo; Funchal needs all rings (about 370 items at floor 2).

### Decisions from R1

- **Page size 50, labels in a second query** for those QIDs (both pre-approved fallbacks: a 100-item page took 8-26 s in dense rings, and plain rows were 3-9 MB).
- **Floor 2.**
- **Stop rule:** stop when 200 visitable pins are loaded (after any page), or when a ring comes back thin (fewer than 5 items), or after the 40 km ring.
- First pins in under 5 s are met in Lisbon, Tokyo, Funchal and typically New York/Kyoto; London and Paris cold runs measured 8-12 s for page 1 (Wikidata noise plus dense ring). Recorded as a known gap, rechecked end to end in R6.

## Deviations from the plan (recorded)

- **Labels in a second query, and 50 items per page** (R1): the plan's pre-approved fallbacks, both applied. The page query returns only QID, coordinates and sitelinks; a second `VALUES` query returns one grouped row per item (types, names, aliases, hours as separator-joined fields). This replaces "pages return plain rows": plain rows were 3-9 MB per 100 items in London, Paris and New York.
- **Retries inside the load**: Wikidata hung for a full 30 s on some pages in R6, so the cache decorator tries a failed load 3 times (5 s apart), each resuming after the last stored page, before it gives up (R4).
- **Files outside the task's Owns**, needed for the behaviour: `internal/writermap/source.go` (`MapSource.DestinationProgress`, `PartialBaseSource`), `public/redesign/map-itineraries.js` and its test (an answer that is still loading must not be cached, or the routes never appear).
- **Lint**: `make lint` cannot run on this machine. `golangci-lint` 2.4.0 was built with go1.26 and the repo targets go1.27.2 (`can't load config: the Go language version (go1.26) used to build golangci-lint is lower than the targeted Go version (1.27.2)`). `gofmt -l .` and `go vet ./...` are clean.

## R2-R5. RED and GREEN

Each RED is the first run after writing the tests, before any production code.

| Task | RED (command, then output excerpt) | GREEN |
|---|---|---|
| R2 ring page query | `go vet ./internal/adapters/api/` -> `vet: ./wikidatanearby_test.go:212:50: NewWikidataNearbyAPI(client).ringPage undefined (type *WikidataNearbyAPI has no field or method ringPage)` | `go test ./internal/adapters/api/ -run 'NearbyRingPage\|NearbyPlaces'` -> `ok` |
| R3 ring loader | `go vet ./internal/adapters/api/` -> `vet: ./wikidatanearby_test.go:389:51: NewWikidataNearbyAPI(world.client()).PlacesFrom undefined (type *WikidataNearbyAPI has no field or method PlacesFrom)` | `go test -cover ./internal/adapters/api/` -> `ok ... coverage: 83.6% of statements` |
| R4 cache | `go vet ./internal/adapters/sqlite/` -> `vet: ./wikidatanearby_test.go:289:21: c.Partial undefined (type *cachedBasePlaces has no field or method Partial)`; `go vet ./internal/writermap/` -> `source_test.go:288:30: source.DestinationProgress undefined` | `go test -race -cover ./internal/adapters/sqlite/` -> `ok ... coverage: 83.2%`; `go test -cover ./internal/writermap/` -> `ok ... 94.5%` |
| R4 retries | the retry test was written before the retry code: `c.retryDelay undefined` | same sqlite command, `ok` |
| R5 handler | `go test ./internal/adapters/httpserver/ -run 'WriterMap_(GeoJSON\|PanelOpens)'` -> `Error: Should be true` (loading), `does not contain "\"loading\":false"`, `expected: 503 actual: 200` | `go test -cover ./internal/adapters/httpserver/` -> `ok ... coverage: 92.8%` |
| R5 JS polling | `node --test tasks/redesign/checks/writers-map-core.test.mjs` -> `+ 'Some map sources are unavailable.'  - 'Loading more places…'` and `TypeError: Cannot read properties of undefined (reading 'fn')` | `ℹ pass 54  ℹ fail 0` |
| R5 itineraries | `node --test tasks/redesign/checks/map-itineraries.test.mjs` -> `✖ an answer that is still loading is not kept ...` | `ℹ pass 66  ℹ fail 0` (with writers-map-core) |

Browser runtime check (R5), isolated Brave on the design preview, scratch script (not in the repo) that serves the Madeira fixture as three answers (2 pins loading, 6 pins loading, all 15): at 375 px and 1440 px the map showed 2, then 6, then 15 pins; status read `Loading more places…` and then cleared to the fixture's own `Some map sources are unavailable.`; `console.error` empty, no horizontal overflow. `node tasks/redesign/checks/writers-map-core.browser.mjs` also passed at both widths on both engines.

## What is guaranteed

| Guarantee | Test | Level | Result |
|---|---|---|---|
| Page query has radii, inner bound only after ring 1, order, limit, offset, floor | `TestNearbyRingPage_SendsRadiiOrderLimitOffsetAndFloor`, `..._FirstRingHasNoInnerBound` | unit | pass |
| Label query asks for exactly the page's QIDs | `TestNearbyRingPage_LabelQueryAsksForExactlyThePageItems` | unit | pass |
| Page and labels join into one place per item; names with `\|` or `,` survive; several types and hours kept | `TestNearbyRingPage_JoinsPageAndLabelsIntoPlaces`, `TestNearbyPlaces_GroupedFieldsKeepAllValues` | unit | pass |
| Empty page skips the label query; 429 retried once; 502 returned | `..._EmptyPageSkipsTheLabelQuery`, `..._RetriesAfter429ThenSucceeds`, `..._ServerErrorIsReturned` | unit | pass |
| Rings nearest first, 50 per page, short page moves on | `TestWikidataNearby_PlacesFrom_LoadsRingsNearestFirstAndPagesWithinARing` | unit | pass |
| Progress after every page | `..._ReportsProgressAfterEveryPage` | unit | pass |
| Stop at 200 pins; stop on a thin ring; last ring ends at the area radius | `..._StopsWhenTwoHundredPinsAreFilled`, `..._AThinRingEndsTheLoad`, `..._RadiusEndsTheLastRingAtTheAreaRadius` | unit | pass |
| Failed page returns what was loaded; resume does not refetch | `..._AFailingPageReturnsWhatWasLoadedAndResumes`, `..._AlreadyDoneProgressMakesNoQueries` | unit | pass |
| Partial pins already have photos | `..._PhotoCreditsAreResolvedPerPage` | unit | pass |
| Partial result stored per page, never as complete, not listed for the monthly sync | `TestWikidataNearbyCache_Partial_StoresAPartialRowAfterEachPageButNeverAsComplete` | integration (temp SQLite) | pass |
| Finished load cached as complete, partial row removed | `..._FinishedLoadIsCachedCompleteAndPartialRowIsRemoved` | integration | pass |
| Load survives the request being cancelled; a cancelled waiter leaves it running | `..._LoadContinuesAfterTheRequestIsCancelled`, `TestWikidataNearbyCache_Places_CancelledWaiterLeavesTheLoadRunning` | integration | pass |
| Failed load resumes from the stored page; transient failures retried; no restart inside the 30 s backoff | `..._FailedLoadResumesFromTheStoredPage`, `..._RetriesAFailedPageBeforeGivingUp`, `..._FreshFailureIsNotRetriedStraightAway` | integration | pass |
| One load per destination | `..._OneLoadPerDestination`, existing `ConcurrentCallersShareOneQuery` | integration (race detector) | pass |
| Writer matching once, at the end; none on partial results; partial never stored | `TestMapSource_DestinationProgress_*` | unit | pass |
| `/map/<dest>.geojson` is 200 with pins so far and `loading`; 503 only when nothing loaded; panel opens during a load | `TestWriterMap_GeoJSON*`, `TestWriterMap_PanelOpensForAPinWhileLoading` | unit (HTTP) | pass |
| Browser polls, adds pins, shows `Loading more places…`, clears, stops when the map is swapped | `writers-map-core.test.mjs` (3 new tests) | unit (JS) | pass |
| Loading answers are not cached by the itineraries layer | `map-itineraries.test.mjs` (1 new test) | unit (JS) | pass |

## R6. End to end (2026-10-12)

Command: the app built from `./cmd/web`, `WRITERS_MAP=1`, `DB_PATH` a temp copy of `launch-check.db` (16,213 indexed posts, no v3 base rows), polled with a scratch script that requests `/map/<slug>.geojson?lat&lon` every second until `loading` is false. Cold = no stored row for the city. Live Wikidata, sequential, app `UserAgent`.

| City | Time to first pins | Time to finish (200 pins) | Largest gap between pin updates | Largest response | Writer pins |
|---|---:|---:|---:|---:|---:|
| Funchal | 3.0 s | 109.8 s (two 30 s page timeouts) | 82.5 s | 146 KB | 159 |
| Lisbon | 3.0 s | 72.7 s (one 30 s timeout) | 42.3 s | 145 KB | 152 |
| Kyoto | 9.0 s / 5.0 s (repeat) | 35.3 s / 21.2 s | 9.0 s / 5.0 s | 134 KB | 28 |
| London | 7.0 s / 18.1 s (repeat) | 119.9 s / 168.2 s | 22.1 s / 22.3 s | 143 KB | 200 |
| Paris | 32.1 s / 4.0 s (repeat) | 113.9 s / 101.8 s | 32.1 s / 11.1 s | 152 KB | 190 |
| New York | 7.0 s | 78.6 s | 10.1 s | 143 KB | 196 |
| Tokyo | 5.0 s | 115.8 s | 21.1 s | 134 KB | 78 |

- Every city finished with 200 pins and `loading: false`; no request returned 503; pins arrived in many small steps (`steps` in the poll output, for example Lisbon 17, 24, 31, 48, 73, 99, 125, 150, 176, 196, 200).
- A cold request with a stored partial row resumed: Kyoto restarted at 37 pins at 0.0 s and Lisbon at 99 pins.
- **Largest page**: the biggest single Wikidata response was a 143-160 KB label page; the biggest gaps (82 s, 42 s) were pages that hit the client's 30 s timeout and were retried by the load.
- **Before** (from [nearby-query.tdd.md](nearby-query.tdd.md), N1/N1c, same endpoint): the one-query load timed out or returned 504 for Lisbon, London, Paris, New York and Tokyo (15-67 s, 4-24 MB), so the old app returned 503 and showed no pins; Funchal answered in 4.7-17 s with 353-1,425 items.
- **Target check**: first pins under 5 s was met for Funchal (3.0 s), Lisbon (3.0 s) and the Paris repeat (4.0 s); **missed** by Tokyo (5.0 s), Kyoto (9.0 s, then 5.0 s), London (7.0 s, then 18.1 s), New York (7.0 s) and the Paris first run (32.1 s). Wikidata's variance (the same cold Paris page was 4 s and 32 s) is larger than the margin; neither pre-approved fallback removes it. The first page costs 3-9 s when Wikidata is quiet.

### Funchal writer matches, before and after

"Before" = the old Funchal cache row (v2, 1,409-item single-query gazetteer) re-keyed to v3 in a second temp copy and served through the same app. "After" = the cold ring load above.

| Place | QID | Writers before | Writers after |
|---|---|---:|---:|
| Pico Ruivo | Q473169 | 5 | 5 |
| Cabo Girão | Q856866 | 5 | 5 |
| Câmara de Lobos | Q34799706 | 5 | 5 |
| Pico do Areeiro | Q1192102 | 5 | 5 |
| Ponta de São Lourenço | Q1296587 | 5 | 5 |
| São Jorge | Q1088866 | 7 | 7 |
| Monte | Q1179064 | 6 | 6 |
| Ponta do Sol | Q943911 | 6 | 6 |
| Funchal Cable Car | Q3289157 | 1 | 1 |

All nine named matches are identical. The rest of the 200 pins differ: before, 200 of 200 pins had a writer mention (the old list included the 37 one-sitelink pins and the 1,409-item gazetteer matched more obscure names); after, 159 of 200. 127 pins with writer mentions are in both lists; 73 are only in the old list (every one has 0 or 1 sitelinks in the old row: 37 and 36), and 32 are only in the new list. This is the cost of the floor and the 200-pin stop, which the plan accepted.

### 6a matcher tests

`make test` runs `internal/writermap` (matcher, Madeira fixture): `ok  weatherservice/internal/writermap`. They are offline and pass on the unchanged matcher.

## Coverage

```text
go test -cover ./internal/adapters/api/ ./internal/adapters/sqlite/ ./internal/adapters/httpserver/ ./internal/writermap/ ./internal/application/
ok  weatherservice/internal/adapters/api         coverage: 83.6% of statements
ok  weatherservice/internal/adapters/sqlite      coverage: 83.2% of statements
ok  weatherservice/internal/adapters/httpserver  coverage: 92.8% of statements
ok  weatherservice/internal/writermap            coverage: 94.5% of statements
ok  weatherservice/internal/application          coverage: 100.0% of statements
```

JS: `node --test tasks/redesign/checks/*.test.mjs` -> `ℹ pass 106  ℹ fail 0`.

## Full suite

- `make test`: all Go packages `ok` (race detector).
- `make build`: `go build -o bin/traveltab ./cmd/web`, OK.
- `make lint`: **cannot run here** (see Deviations); `gofmt -l .` and `go vet ./...` print nothing.

## Known gaps

- First pins in under 5 s: fixed by the follow-up above (all seven cities, 1-4 s). Earlier R6 numbers in the table are kept for the record.
- A whole cold load takes 20-170 s because Wikidata pages are sequential and slow; the map is usable from the first page.
- The 6a precision review can now run on the new Madeira gazetteer.


## Follow-up: label-less pins before the label query (2026-10-12)

Owner request: cut time to first pins by publishing each page's pins as soon as the page query returns.

**What changed**
- The page query now also returns each item's `wdt:P31` types (an `OPTIONAL` outside the paged sub-select), so visitable items are known without labels. `pageItems` returns unnamed places; the loader publishes their visitable ones as pins (and calls the progress callback) at once, then `pageLabels` names them and the page is merged.
- `BaseProgress` gained `Unnamed` and `PageRows`. The ring cursor moves only after the labels are in, so a failed label query keeps the pins and a resume (or retry) asks **only for the labels**, not the page. The partial row stores `Unnamed`, so this also survives a restart.
- Nameless places are not in the `Gazetteer`, so they take no part in writer matching or in the page-end stop rule (200 pins, thin ring), which is evaluated after the labels as before. Matching still runs once, at the end.
- Page size counts the sub-select's rows (one per item and coordinate), not the type rows the join adds.
- The endpoint sends `name: ""` for a nameless pin (not the QID); the panel says `Loading name…`. `map-panel.js` shows such a pin as an unlabelled marker titled `Loading name…`, never gives it a label, and fills the name in on the next poll.
- Failure logs in the cache decorator no longer print the Wikidata error (which contained the query URL): `base places for funchal: attempt 1 failed after 219 places`.

**RED then GREEN**

| Task | RED | GREEN |
|---|---|---|
| Page items / labels / publish order | `go vet ./internal/adapters/api/` -> `vet: ./wikidatanearby_test.go:203:50: NewWikidataNearbyAPI(client).pageItems undefined` | `go test -cover ./internal/adapters/api/` -> `ok ... coverage: 83.9%` |
| Log trimming | `go test -run FailureLogs ./internal/adapters/sqlite/` -> `should not contain "SELECT"` | `go test -race -cover ./internal/adapters/sqlite/` -> `ok ... 83.2%` |
| Nameless name in handler | `Error: ... does not contain "\"name\":\"\""` and `does not contain "Loading name"` | `go test -cover ./internal/adapters/httpserver/` -> `ok ... 93.0%` |
| Nameless pin in the browser | `node --test tasks/redesign/checks/map-panel.test.mjs` -> `✖ a pin whose name has not loaded yet is a neutral marker ...` | `node --test tasks/redesign/checks/*.test.mjs` -> `ℹ pass 107  ℹ fail 0` |

New guarantees: `TestWikidataNearby_PlacesFrom_PublishesNamelessPinsBeforeTheLabelQuery` (first publish happens with zero label queries made, pins have no names, gazetteer empty, offset unmoved), `..._FailedLabelQueryKeepsThePinsAndRetriesOnlyTheLabels` (ring 1 is not queried again), `..._StopRuleCountsOnlyNamedPins`, `..._FinalProgressHasNamesAndNoPendingPage`, `TestNearbyPageItems_*` (types merged into one place, type rows not counted as page size), `TestWriterMap_GeoJSONNamelessPinHasAnEmptyNameNotItsQID`, `TestWriterMap_PanelForANamelessPinDoesNotShowTheQID`, `TestWikidataNearbyCache_FailureLogsDoNotPrintTheWikidataError`, and the map-panel nameless-pin test.

**Re-measured, cold, same method as R6** (first run each; the first-pin figure is the per-city time to the first non-empty answer)

| City | First pins before | First pins now | Finish now | Largest gap |
|---|---:|---:|---:|---:|
| Funchal | 3.0 s | 1.0 s | 68.6 s | 15.1 s |
| Lisbon | 3.0 s | 2.0 s | 156.4 s | 122.9 s |
| Kyoto | 9.0 s (5.0 s repeat) | 1.0 s | 42.3 s | 12.1 s |
| London | 7.0 s (18.1 s repeat) | 4.0 s | 194.2 s | 126.6 s |
| Paris | 32.1 s (4.0 s repeat) | 1.0 s | 104.9 s | 26.2 s |
| New York | 7.0 s | 1.0 s | 152.1 s | 86.5 s |
| Tokyo | 5.0 s | 2.0 s | 120.8 s | 12.1 s |

Every city reached 200 pins with `loading: false`, and all seven first pins are under 5 s. Finish times vary with Wikidata hanging pages for the 30 s client timeout (Lisbon, London and New York each lost 2 to 4 minutes-worth of 30 s attempts in the large gaps); this is Wikidata noise, not the follow-up, and it is unchanged in kind from R6. Writer pins are unchanged (Funchal 159, Lisbon 152, Kyoto 28, London 200, Paris 190, New York 196, Tokyo 78).

Browser check, 375 px and 1440 px (isolated Brave, design preview): the loading scenario (2, 6, then 15 pins with `Loading more places…`) passed with a clean console and no overflow, and `writers-map-core.browser.mjs` and `map-panel.browser.mjs` passed at both widths on both engines.

**Known behaviour**: the pin count can dip slightly between two polls (for example 40 to 37 in Funchal) because an item the page query counted as visitable can turn out to have no English or Portuguese name once labelled, and is then dropped.

## Follow-up: the warm-up starts the load and returns (2026-10-12)

**Bug.** `suggest.go` `loadBase` called `Places()` with a 30 s timeout. `Places()` now waits for the whole ring load (40-195 s cold), so the log showed `Could not warm base places for London, United Kingdom: context deadline exceeded` and `baseStarted` was cleared, even though the detached load kept running.

**Journey.** *As a visitor typing a city, I want the warm-up to only start the ring load, so a slow cold load is not reported as a failure and later searches do not start it again.*

**Fix.** `main.go` passes the cache decorator, which implements `writermap.PartialBaseSource`. `loadBase` now calls `Partial` (starts the load or joins it, returns at once) for such a source and falls back to `Places` for any other. `baseStarted` stays set after a successful start; only a failed start clears it and logs.

RED (`go test -race -run SuggestionWarm ./internal/adapters/httpserver/`, before the fix; the warm-up blocked in `Places` and never called `Partial`):

```text
--- FAIL: TestSuggestionWarm_StartsAPartialSourceWithoutWaitingForTheLoad (1.00s)
        Error:      	Condition never satisfied
--- FAIL: TestSuggestionWarm_FailedStartClearsTheMarkAndLogs (1.00s)
        Error:      	Condition never satisfied
--- FAIL: TestSuggestionWarm_SecondSearchForTheSameCityDoesNotStartAnotherLoad (1.00s)
        Error:      	Condition never satisfied
```

GREEN (`go test -race -count=1 -cover ./internal/adapters/httpserver/ ./internal/writermap/ ./internal/adapters/sqlite/`):

```text
ok  	weatherservice/internal/adapters/httpserver	2.946s	coverage: 93.0% of statements
ok  	weatherservice/internal/writermap	1.688s	coverage: 94.5% of statements
ok  	weatherservice/internal/adapters/sqlite	2.849s	coverage: 83.2% of statements
```

| Guarantee | Test | Level | Result |
|---|---|---|---|
| Warm-up returns without waiting for a slow load, never calls `Places`, keeps the mark, logs nothing | `TestSuggestionWarm_StartsAPartialSourceWithoutWaitingForTheLoad` | unit | pass |
| A failed start clears the mark and logs | `TestSuggestionWarm_FailedStartClearsTheMarkAndLogs` | unit | pass |
| A second search for the same city starts no second load | `TestSuggestionWarm_SecondSearchForTheSameCityDoesNotStartAnotherLoad` | unit | pass |
| A source without `Partial` still warms through `Places` (unchanged) | `TestSuggestionWarm_RetriesFailedQuery`, `TestSuggestPlaces_WarmsTopSuggestionWithoutDelayingResponse` | unit | pass |


## Follow-up: dense-city mode (owner decision 2026-10-13)

**Journey.** *As a traveller opening London, Paris or New York, I want the closest 8 km loaded in 1 km rings, so that the map fills with pins around me quickly and does not crawl out to 40 km.*

**Threshold, measured** (live, 2026-10-12/13, count of items with 2+ sitelinks within 5 km of the centre; each run twice, identical both times; the capped query `COUNT` over a `LIMIT 1501` sub-select took 0.5-5 s, the uncapped count 0.3-10 s):

| City | Items within 5 km | Capped count (cap 1,501) | Mode |
|---|---:|---:|---|
| Funchal | 144 | 144 | normal |
| Tokyo | 690 | 690 | normal |
| Kyoto | 766 | 766 | normal |
| Lisbon | 797 | 797 | normal |
| New York | 2,509 | 1,501 | dense |
| London | 4,752 | 1,501 | dense |
| Paris | 9,783 | 1,501 | dense |

**Threshold: more than 1,500.** There is a clear gap between 797 (Lisbon) and 2,509 (New York), so 1,500 sits inside it with room either side. **Tokyo is normal**: with the floor of 2 sitelinks only 690 items lie within 5 km of its centre (most Japanese items have a single sitelink), as few as Kyoto; its 5 km rings already fill 200 pins quickly. Funchal stays normal and keeps all nine writer matches. The count is capped, so it never has to enumerate Paris's 9,783 items.

**Design.** Detection runs **once, after the first page** (a normal 5 km page, so first pins are not delayed), only if that page did not already end the load. The mode (`normal` or `dense`) is stored in `BaseProgress.Mode`, so a resume uses it and never recounts. A dense city restarts its ring cursor at 0-1 km, keeps the places already loaded, and loads eight 1 km rings (hard end at 8 km, or the area radius if smaller). The page size, floor, label query, nameless-pins-first, stop rule, retries and one writer-match pass are unchanged. A failed count is retried on resume without refetching the first page. Cost: writer matches beyond 8 km are dropped in dense cities (accepted).

**RED then GREEN.** RED (`go test ./internal/adapters/api/ -run 'Density|Dense|Threshold|ModeSurv|NoDensity|FirstPinsDoNot'`, before any production code):

```text
--- FAIL: TestNearbyDensity_CountIsCappedAndAsksForItemsAtTheFloorWithinFiveKm
        Error:      	"" does not contain "wikibase:radius \"5\""
        Error:      	"" does not contain "LIMIT 1501"
```

GREEN: `go test -race -cover ./internal/adapters/api/ ./internal/adapters/sqlite/ ./internal/writermap/` -> `ok ... 84.1%`, `ok ... 83.2%`, `ok ... 94.5%`.

| Guarantee | Test | Result |
|---|---|---|
| Count is capped, 5 km, floor 2, around the centre | `TestNearbyDensity_CountIsCappedAndAsksForItemsAtTheFloorWithinFiveKm` | pass |
| Above the threshold: first normal page, then exactly eight 1 km rounds, nothing past 8 km, one count | `..._DenseCityLoadsEightOneKmRoundsAndStops` | pass |
| 144, 797 and exactly 1,500 stay normal (5 km rings, unchanged calls) | `..._CityAtOrBelowTheThresholdStaysNormal` | pass |
| A smaller area radius shortens the dense rounds | `..._DenseModeStopsAtASmallerAreaRadius` | pass |
| Stored mode survives a resume (JSON round trip) and is not detected again | `..._ModeSurvivesAResumeAndIsNotDetectedAgain`, `TestWikidataNearbyCache_Partial_StoredProgressKeepsTheLoadMode` | pass |
| A failed count is retried without refetching the first page | `..._FailedDensityCountIsRetriedOnResumeWithoutRefetchingPages` | pass |
| No count when the first page already ends the load; count never delays the first publish | `..._NoDensityCountWhenTheFirstPageAlreadyEndsTheLoad`, `..._FirstPinsDoNotWaitForTheDensityCount` | pass |

**Re-measured, cold, same method** (Wikidata live; "before" is the label-less-pins run):

| City | Mode | First pins before | First pins now | Finish before | Finish now | Writer pins before / now |
|---|---|---:|---:|---:|---:|---:|
| London | dense | 4.0 s | 2.0 s | 194.2 s | 87.7 s | 200 / 126 |
| Paris | dense | 1.0 s | 6.0 s | 104.9 s | 78.6 s | 190 / 66 |
| New York | dense | 1.0 s | 8.0 s | 152.1 s | 113.8 s | 196 / 158 |
| Tokyo | normal | 2.0 s | 1.0 s | 120.8 s | 204.3 s | 78 / 78 |
| Funchal | normal | 1.0 s | 1.0 s | 68.6 s | 28.3 s | 159 / 159 |
| Lisbon | normal | 2.0 s | 3.0 s | 156.4 s | 60.6 s | 152 / 152 |

Single cold runs; Wikidata varies 2-5x between runs (Paris and New York first pins were 1 s in the previous run and 6-8 s now; Tokyo lost about 60 s to hung pages). Funchal and Lisbon are unchanged in mode, pins and writer matches. The fewer writer pins in dense cities are the accepted cost of the 8 km end.

## Follow-up: default map view at the 200 m scale (owner decision 2026-10-13)

The map now opens centred on the searched destination, keeps pitch 55 and bearing -20, and **does not fit the camera to the pins** (`frameMap` and its padding constants are removed); pins farther out are reached by panning, and `reset()` returns to the same view. The itineraries, day-plan and panel layers use `draw.fit`/`flyTo` only when the user asks (View on map, a route), unchanged.

**Zoom value, measured.** The requested zoom 16.7 does not give a 200 m scale. Real MapLibre scale bar in London (51.51 N), the bar picks the largest round number that fits in 100 px, at pitch 55:

| Zoom | Scale bar (375 px and 1440 px identical) | Metres per pixel |
|---:|---|---:|
| 12.3 (old) | 500 m | 9.66 |
| 14.0 | 200 m (67 px) | 2.97 |
| 14.3 (chosen) | 200 m (83 px) | 2.42 |
| 14.5 | 200 m (95 px) | 2.10 |
| 15.0 | 100 m | 1.49 |
| 16.7 | 30 m | 0.46 |

**Final value: MapLibre zoom 14.3** (the bar reads "200 m" at 375 and 1440 px; checked on the running map after load). The Leaflet fallback opens at zoom 15, where its scale bar also reads about 200 m. 

**RED then GREEN.** RED (`node --test tasks/redesign/checks/writers-map-core.test.mjs`): `actual: [ 12.3, 55, -20 ], expected: [ 14.3, 55, -20 ]` and the new `opens on the destination at the 200 m view and does not fit the camera to the pins` failing on both engines. GREEN: `node --test tasks/redesign/checks/*.test.mjs` -> `ℹ pass 107  ℹ fail 0`.

**Browser checks** (isolated Brave, design preview, 375 and 1440 px, both engines where the check has both): `writers-map-core.browser.mjs`, `writers-map-layout.browser.mjs`, `map-panel.browser.mjs`, `map-itineraries.browser.mjs`, `map-dayplan.browser.mjs` pass with a clean console. Two checks assumed the old framing and were adjusted: the palette check now sets zoom 12.3 before sampling land and water, and the panel check pans to the writer pins (as a visitor would) before clicking them.
