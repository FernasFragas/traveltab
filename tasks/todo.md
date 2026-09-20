# Tasks: Weather-aware trip planner (MVP)

> **Package migration:** application code has moved under `internal/`. The file ownership,
> paths and constructor signatures below record the original MVP implementation. For current
> paths and dependency rules, use [architecture.md](../docs/architecture.md). In particular,
> there is no root Go package or global database: `cmd/web` opens a SQLite store and injects
> it into the HTTP adapter. Source-cache constructors are methods on that store.

Read [plan.md](plan.md) first. It has **the shared contract**, **the file-ownership table** and **how every task is built with TDD**. Background: [one-pager](../docs/ideas/weather-aware-trip-planner.md), [validation](../docs/ideas/weather-aware-trip-planner-validation.md).

## How to work a task (TDD)

Every task below is built the same way:

1. **RED:** write the tests listed under **Tests first**. Add the function with its contract signature and a zero-value body (`return nil`, `return 0`, …), then run the tests. **Each one must fail on an assertion.** A compile error doesn't count, and a test that passes straight away isn't testing anything. Keep the failing output for your notes.
2. **GREEN:** write the least code that makes them pass.
3. **REFACTOR:** clean up with the tests green, re-running the focused tests after each change.
4. **Before you're done:** run the full suite once: `make test`, `make lint`, `make build`.

**Commands**

- Focused: `CGO_ENABLED=1 go test -race -count=1 -run '<Pattern>' ./<package>/`
- Full suite: `make test`
- Lint: `make lint`
- Build: `make build`

**Test sizes:**

- **Small:** no I/O, in one process. Most tests.
- **Medium:** SQLite, template files or fixture files on disk.
- **Large:** the real internet or a real browser, and always opt-in (`LIVE_API_TESTS=1`) or run by hand.

**Test style:**

- Names read like a spec: `TestRank_DenyTypeBeatsIndoorType`.
- Arrange, Act, Assert, with one concept per test.
- Check the result, not which functions were called.
- Use fakes, not gomock.
- Each test tells its whole story. Repeating setup is fine when it keeps a test readable.

**Current status (2026-09-20):** all 17 MVP implementation tasks are complete. The final
`make test` (race detector), `make lint`, `make build`, and native ARM Docker build pass.
The 375 px browser check passes. See [completion evidence](notes-completion.md) and
[mobile screenshots](notes-ui-validation.md).

**Remaining release work:** human review of the generated type list, commit/PR and remote CI,
then deployment approval. Delivery remains with the human as specified in plan.md. v2 is
explicitly deferred until after the MVP ships.

Checked test entries below mean the current test passes. Historical RED claims are kept
separate: missing original evidence cannot be recreated after implementation. The three live
API tests intentionally remain opt-in; no default test is skipped or disabled.

---

## Phase 0: Foundation

## Task 1: Planner contract and geo helpers

**Description:** Create `planner/types.go` exactly as in plan.md's contract, plus the geo helpers in `planner/geo.go`, the empty `PlaceTypes` placeholder, and `api/useragent.go`. The types and constants have no behavior, so compiling them is their check. The helpers are built test-first.

**Tests first (RED): all small, in `planner/geo_test.go`**

- [x] `TestDistanceKM_LisbonToPortoIsAbout274Km`: (38.7223, -9.1393) to (41.1496, -8.6110) is 274 km ±1%.
- [x] `TestDistanceKM_SamePlaceIsZero`
- [x] `TestWalkingLoop_StartsAtTheMostFamousStop`: the stop with the most `Sitelinks` comes first.
- [x] `TestWalkingLoop_BreaksFameTiesByLowerID`
- [x] `TestWalkingLoop_VisitsEveryStopOnce`
- [x] `TestWalkingLoop_WalkKMIncludesTheWayBack`: for 3 points on a line, the loop length is twice the span.
- [x] `TestWalkingLoop_SingleStopWalksZeroKM`
- [x] `TestMedoid_PicksThePlaceClosestToAllOthers`
- [x] `TestMedoid_BreaksTiesByLowerID`
- [x] `TestImageURL_BuildsACommonsThumbnailURL`: "Torre de Belém.jpg" at width 400 gives `…/Special:FilePath/Torre_de_Bel%C3%A9m.jpg?width=400`.
- [x] `TestImageURL_IsEmptyWithoutAnImage`
- [x] `TestImagePageURL_LinksToTheCommonsFilePage`

**Make them pass (GREEN):** haversine for distance, nearest neighbour for the loop.

**Refactor:** keep `geo.go` free of anything that isn't geometry or image URLs.

**Acceptance criteria:**
- [x] `planner/types.go` retains the shared contract (formatting aside).
- [x] Production `planner` code imports neither `weatherservice` nor `weatherservice/api`; external city tests use the real API adapters with fixtures.
- [x] `planner/placetypes_gen.go` now holds the generated map (superseding Task 1’s empty placeholder), and `api/useragent.go` holds `UserAgent`.

**Verification:**
- Historical RED: see the task notes; unrecorded original failures remain unverified (not a new implementation task).
- [x] GREEN: `CGO_ENABLED=1 go test -race -count=1 ./planner/`
- [x] Full suite: `make test`, `make lint`, `make build`

**Dependencies:** None

**Files likely touched:** `planner/types.go`, `planner/geo.go`, `planner/geo_test.go`, `planner/placetypes_gen.go`, `api/useragent.go`

**Estimated scope:** Medium

---

## Task 2: Remove the old hotels and itinerary code

**Description:** Delete the Google Places hotels feature and the Foursquare itinerary feature: API clients, reporter types, the `/generate-itinerary` route and handler, their templates, and their environment keys. `NewAppServer` then takes only the weather and video reporters. This task is mostly deletions, so it starts with **tests for what must change** (they fail now) and **guard tests for what must keep working** (they pass before and after).

**Tests first (RED): small, in-process Fiber app**

- [x] `TestServer_ItineraryRouteIsGone`: `POST /generate-itinerary` returns 404.
- [x] `TestListGeneralInfo_PageHasNoHotelsCard`: neither the full page nor the HTMX fragment contains the hotels markup.
- [x] `TestLoadEnvKey_NoLongerReadsPlacesOrFoursquareKeys`: the keys struct has no such fields. Update `Test_LoadEnvKey` to match.

**Guard tests (write them first; they pass before and after)**

- [x] `TestCheckDatabase_LoadsOldRowsThatStillHaveHotels`: a cached `city_data` row with a `Hotels` field still loads, because unknown JSON fields are ignored.
- [x] The existing `TestListGeneralInfo_*` tests for weather and videos stay green. Tests for the removed hotels behavior (`TestRetireveFreshInformation_HotelsError`, the hotels mocks) are deleted along with it.

**Make them pass (GREEN):** delete the code, templates and keys, and change `NewAppServer(weatherReporters, videoStreamReporters)`.

**Refactor:** remove helpers, mocks and imports that nothing uses any more.

**Acceptance criteria:**
- [x] No removed provider or itinerary code remains in production Go/templates; removal regression tests retain the old route names.
- [x] Only templates used solely by hotels or the itinerary are deleted. Check each with `grep` first.

**Verification:**
- Historical RED: see the task notes; unrecorded original failures remain unverified (not a new implementation task).
- [x] Full suite: `make test`, `make lint`, `make build`
- [x] Weather/video regression tests pass and the real page renders the fixture city in a browser without the old hotels card.

**Dependencies:** None

**Files likely touched:**
- `server.go`, `server_test.go`, `reporters.go`, `cmd/web/main.go`, `env.go`, `env_test.go`
- `api/foursquare.go`, `api/googlaPlaces.go`, `api/googlePhotos.go` (delete)
- `views/content_fragment.go.tpl`, `views/index.go.tpl`
- Templates to delete: `hotels_card`, `itinerary_*`, `form_dates`, `form_categories`, `category_checkboxes`, `map_initialization`, `map_markers`

**Estimated scope:** Large (mostly deletions; don't split, because every piece meets in `server.go`)

---

## Checkpoint A: Foundation (after Tasks 1–2)

- [x] `make lint`, `make test` and `make build` pass in the working tree
- [x] The site shows weather and videos, with no hotels card
- [x] Phase 1 dependencies are satisfied and its implementation is complete

---

## Phase 1: Parallel pieces

## Task 3: Wikipedia + Wikidata place source

**Description:** `api/wikimedia.go` implements `planner.PlaceSource`:

1. **Search:** an English Wikipedia geosearch within 10 km, returning up to 500 results.
2. **Split when capped:** if the search returns exactly 500, run 7 searches of 5 km instead. One is at the center, and six are 8.66 km out at 0°, 60°, …, 300°. Merge them by page ID and keep only results inside 10 km.
3. **Look up details:** each page's Wikidata ID (`prop=pageprops`, 50 per request), then each ID's sitelink count (`wbgetentities props=sitelinks`). **Only for the top 300 by sitelinks**, also fetch types (P31), image (P18) and the English label.
4. **Export `GetEntities`** for Task 13.

Tests stub HTTP with `roundTripFunc` (from `api/openweather_test.go`) and serve trimmed recorded or hand-built responses from `api/testdata/wikimedia/`.

**Tests first (RED): small, stubbed HTTP**

- [x] `TestWikimedia_PlacesNear_ReturnsPlacesWithWikidataDetails`: in the Tavira fixture, Castle of Tavira has its ID, coordinates, sitelinks, types and image.
- [x] `TestWikimedia_PlacesNear_LeavesKindEmpty`
- [x] `TestWikimedia_PlacesNear_UsesTheTitleWhenThereIsNoEnglishLabel`
- [x] `TestWikimedia_PlacesNear_SplitsACappedSearch`: when the first search returns 500, a place that only an outer search returns (9.5 km out) is in the result.
- [x] `TestWikimedia_PlacesNear_DropsSplitResultsBeyond10Km`
- [x] `TestWikimedia_PlacesNear_ReturnsEachPlaceOnce`: places that overlapping searches both return appear once.
- [x] `TestWikimedia_PlacesNear_DoesNotSplitAnUncappedSearch`: a boundary check on cost. With 218 results, only one geosearch request is made.
- [x] `TestWikimedia_PlacesNear_HasNoTypesBeyondTheTop300`: the 301st place by sitelinks has empty `Types`.
- [x] `TestWikimedia_SendsTheUserAgent`
- [x] `TestWikimedia_ErrorIncludesTheStatusOnNon200`
- [x] `TestWikimedia_StopsWhenTheContextIsCancelled`
- [x] `TestWikimedia_GetEntities_ReturnsTheRequestedEntities`
- Optional live check: `TestWikimediaLive_FindsBelemTowerInLisbon` (runs only with `LIVE_API_TESTS=1`)

**Make them pass (GREEN):** one method per step (search, split, IDs, sitelinks, details), each requesting 50 IDs at a time.

**Refactor:** share one request helper that sets the `User-Agent` and checks the status.

**Acceptance criteria:**
- [x] Every default test above passes. Historical test-first evidence is in the task notes.
- [x] `NewWikimediaAPI(nil)` uses a default client with a 30 s timeout.

**Verification:**
- Historical RED: see the task notes; unrecorded original failures remain unverified (not a new implementation task).
- [x] GREEN: `CGO_ENABLED=1 go test -race -count=1 -run Wikimedia ./api/`
- [x] Full suite: `make test`, `make lint`, `make build`
- [x] Large, by hand, once: `LIVE_API_TESTS=1 go test -run WikimediaLive ./api/`

**Dependencies:** Task 1

**Files likely touched:** `api/wikimedia.go`, `api/wikimedia_test.go`, `api/testdata/wikimedia/`

**Estimated scope:** Medium

---

## Task 4: Open-Meteo daily forecast source

**Description:** `api/openmeteo_forecast.go` implements `planner.ForecastSource` with `GET https://api.open-meteo.com/v1/forecast?latitude=…&longitude=…&daily=precipitation_sum&timezone=auto&forecast_days=16`. Keep it separate from the existing marine client (`api/openmateo.go`). Import `time/tzdata` so the slim Docker image works.

**Tests first (RED): small, stubbed HTTP, fixture `api/testdata/openmeteo/forecast_kyoto.json`**

- [x] `TestForecast_ReturnsSixteenDays`
- [x] `TestForecast_ReturnsTheCityTimeZone`: `Asia/Tokyo`.
- [x] `TestForecast_DatesAreMidnightInTheCityTimeZone`
- [x] `TestForecast_KeepsTheRainAmount`: 24.4 mm stays 24.4.
- [x] `TestForecast_NullRainBecomesNil`
- [x] `TestForecast_ErrorsOnNon200`
- [x] `TestForecast_ErrorsOnAnUnknownTimeZone`
- [x] `TestForecast_AsksForDailyRainInTheLocalTimeZone`: a boundary check that the query has `daily=precipitation_sum`, `timezone=auto` and `forecast_days=16`.
- Optional live check: `TestForecastLive_ReturnsKyoto`

**Make them pass (GREEN):** decode into a struct with `[]*float64` for the rain values.

**Refactor:** nothing special.

**Acceptance criteria:** every test above is written first and passes.

**Verification:**
- Historical RED: see the task notes; unrecorded original failures remain unverified (not a new implementation task).
- [x] GREEN: `CGO_ENABLED=1 go test -race -count=1 -run Forecast ./api/`
- [x] Full suite: `make test`, `make lint`, `make build`

**Dependencies:** Task 1

**Files likely touched:** `api/openmeteo_forecast.go`, `api/openmeteo_forecast_test.go`, `api/testdata/openmeteo/forecast_kyoto.json`

**Estimated scope:** Small

---

## Task 5: Overpass stay source with fallback servers

**Description:** `api/overpass.go` implements `planner.StaySource`:

- **Query:** `[out:json];nwr["tourism"~"^(hotel|hostel|guest_house)$"](around:1000,LAT,LON); out center tags;`, sent as the `data` form field. **The `[out:json];` prefix is required**: Overpass answers in XML without it, and no header or form field can change that.
- **Servers:** tried in order from `DefaultOverpassURLs` (`overpass-api.de`, then `overpass.private.coffee`). Each gets a 10 s timeout and **one retry** on 429, 5xx or timeout, honoring `Retry-After` capped at 5 s, before the next server is tried.
- **Sleeping is injectable** (`sleep func(time.Duration)`), so tests never wait.

**Tests first (RED): small, stubbed HTTP that answers per server, fixture `api/testdata/overpass/tavira.json`**

- [x] `TestOverpass_ReturnsStaysFromTheFirstServer`
- [x] `TestOverpass_RetriesOnceBeforeMovingOn`: the first server answers 504, then 200, and the result comes from the first server.
- [x] `TestOverpass_FallsBackAfterTwoFailures`: the first server answers 504 twice, and the result comes from the second.
- [x] `TestOverpass_ErrorNamesEveryFailedServer`
- [x] `TestOverpass_WaitsForRetryAfterUpToFiveSeconds`: a `Retry-After: 30` header leads to a 5 s wait, checked through the fake sleeper.
- [x] `TestOverpass_SkipsUnnamedPlaces`
- [x] `TestOverpass_PrefersTheEnglishName`
- [x] `TestOverpass_ReadsTheWebsiteFromContactWebsite`
- [x] `TestOverpass_ReadsStarsLike4SAsFour`
- [x] `TestOverpass_StarsAreZeroWhenMissing`
- [x] `TestOverpass_UsesTheWayCenterForCoordinates`
- [x] `TestOverpass_SendsTheQueryAndUserAgent`: a boundary check on the `data` form field and the header.
- Optional live check: `TestOverpassLive_FindsHotelsInTavira`

**Make them pass (GREEN):** one loop over the servers, with one retry inside.

**Refactor:** keep parsing (tags to `planner.Stay`) in its own function.

**Acceptance criteria:** every test above is written first and passes.

**Verification:**
- Historical RED: see the task notes; unrecorded original failures remain unverified (not a new implementation task).
- [x] GREEN: `CGO_ENABLED=1 go test -race -count=1 -run Overpass ./api/`
- [x] Full suite: `make test`, `make lint`, `make build`

**Dependencies:** Task 1

**Files likely touched:** `api/overpass.go`, `api/overpass_test.go`, `api/testdata/overpass/tavira.json`

**Estimated scope:** Medium

---

## Task 6: SQLite source cache that keeps old data on errors

**Description:** `sourcecache.go` in the root package wraps each source with a cache:

- **Storage:** a new table `source_cache(key TEXT PRIMARY KEY, fetched_at INTEGER, data BLOB)`, created in `InitDB` and stored as gzip JSON like `SaveCityData`.
- **Key:** the source name plus lat/lon rounded to 3 decimals.
- **TTLs:** places and stays 30 days, forecast 3 hours.
- **When a refresh fails,** it returns the old data. It returns an error only when nothing is cached.
- **The clock is injectable.**

The tests use a **fake source that counts its calls in the data it returns** ("call 1", "call 2", …), so every assertion checks returned data rather than calls.

**Tests first (RED): medium, SQLite through the package's existing `TestMain`**

- [x] `TestSourceCache_ServesAFreshEntryFromTheCache`: the second call still returns "call 1".
- [x] `TestSourceCache_RefreshesAnExpiredEntry`: after 31 days, it returns "call 2".
- [x] `TestSourceCache_ServesOldDataWhenTheRefreshFails`
- [x] `TestSourceCache_ReturnsTheErrorWhenNothingIsCached`
- [x] `TestSourceCache_SharesEntriesForNearbyCoordinates`: 38.72231 and 38.72234 give the same entry.
- [x] `TestSourceCache_ExpiresForecastsAfterThreeHours`
- [x] `TestSourceCache_KeepsSourcesApart`: places and stays for the same coordinates don't mix.
- [x] `TestInitDB_CreatesTheSourceCacheTable`: an existing database file opens and gets the table.

**Make them pass (GREEN):** one generic `cached[T]` helper used by the three wrappers.

**Refactor:** reuse the gzip code from `database.go` instead of copying it.

**Acceptance criteria:** every test above is written first and passes.

**Verification:**
- Historical RED: see the task notes; unrecorded original failures remain unverified (not a new implementation task).
- [x] GREEN: `CGO_ENABLED=1 go test -race -count=1 -run 'SourceCache|InitDB' .`
- [x] Full suite: `make test`, `make lint`, `make build`

**Dependencies:** Task 1

**Files likely touched:** `sourcecache.go`, `sourcecache_test.go`, `database.go`

**Estimated scope:** Medium

---

## Task 7: Rank places, with the type filter and boost list

**Description:** `Rank(candidates, types, boosts)` in `planner/rank.go` keeps the places worth visiting, sets their `Kind`, and sorts them by fame. `planner/boosts.go` holds `Boosts` with the 5 starting entries: Q652806 indoor, Q168001 outdoor, Q2063403 indoor, Q23579173 outdoor, Q11650434 indoor. The tests use a small type map written inside the test, not `PlaceTypes`.

**Tests first (RED): all small**

- [x] `TestRank_KeepsAPlaceWithAnIndoorType`
- [x] `TestRank_DropsAPlaceWithOnlyUnknownTypes`: a plain bridge.
- [x] `TestRank_DenyTypeBeatsIndoorType`
- [x] `TestRank_IndoorPlusOutdoorTypesMakeMixed`
- [x] `TestRank_SortsByFameThenByID`
- [x] `TestRank_ReturnsAtMostSixtyPlaces`
- [x] `TestRank_SetsKindOnEveryPlace`
- [x] `TestRank_BoostedPlaceSkipsTheTypeFilter`
- [x] `TestRank_BoostedPlaceGetsTheBoostKind`
- [x] `TestRank_BoostedPlaceRanksLikeTheTenthPlace`: a boosted place with 7 sitelinks lands in the top 10 of a 60-place list.
- [x] `TestRank_BoostWithFewerThanTenPlacesRanksLikeTheLastPlace`
- [x] `TestRank_DoesNotChangeTheInput`
- [x] `TestBoosts_HasTheFiveStartingPlaces`

**Make them pass (GREEN):** filter, then sort by an "effective fame" value. Never change `Place.Sitelinks`.

**Refactor:** keep "which kind is this place" in one small function.

**Acceptance criteria:** every test above is written first and passes.

**Verification:**
- Historical RED: see the task notes; unrecorded original failures remain unverified (not a new implementation task).
- [x] GREEN: `CGO_ENABLED=1 go test -race -count=1 -run 'Rank|Boosts' ./planner/`
- [x] Full suite: `make test`, `make lint`, `make build`

**Dependencies:** Task 1

**Files likely touched:** `planner/rank.go`, `planner/boosts.go`, `planner/rank_test.go`

**Estimated scope:** Small

---

## Task 8: Group places into days

**Description:** `GroupDays(ranked, days)` in `planner/days.go` takes the top `SunnyPoolSize` places and groups them by location:

- **Number of days:** at most `min(days, len(pool) / MinStopsPerDay)`, and at least 1 when there's at least one place.
- **Clustering:** a deterministic k-means, seeded farthest-point style from the most famous place, with ties broken by ID.
- **Choosing the stops (decided 2026-09-20):** when there are more places than slots (`days × MaxStopsPerDay`), drop by **fame and closeness together**, not fame alone. Always keep the most famous place, then keep the highest-ranked places that sit near the ones already kept, widening the search only when nothing qualifies. Dropping by fame alone gave Lisbon a 15.6 km walking day on a 2-day trip.
- **Balancing (updated 2026-09-20):** each group aims for 3–4 stops. A short day pulls the nearest sparable stop from a day that has more than the minimum, **but only if that stop is within `PullReachKM` (3 km)**. When nothing is close enough, the day keeps 2 stops rather than dragging someone across town. `MinStopsPerDay` is therefore a target, not a guarantee. **A day left with a single stop is rescued**: it may take from a day sitting at exactly `MinStopsPerDay`, so 3 + 1 becomes 2 + 2. The donor still ends with two, so a rescue can never strand another day.
- **Output:** each group ordered with `WalkingLoop`, and groups sorted by their most famous stop.

**Tests first (RED): all small**

- [x] `TestGroupDays_NeverMixesTwoFarApartAreas`: Belém-like points in the west, Alfama-like points in the east.
- [x] `TestGroupDays_EveryDayHasThreeToFourStopsUnlessNothingIsInReach`
- [x] `TestGroupDays_SmallTownGetsFewerDays`: 7 places with 5 days asked gives 2 days.
- [x] `TestGroupDays_OnePlaceGivesOneDay`
- [x] `TestGroupDays_NoPlacesGivesNoDays`
- [x] `TestGroupDays_UsesOnlyTheTop20`
- [x] `TestGroupDays_NoPlaceAppearsTwice`
- [x] `TestGroupDays_OrdersEachDayAsAWalkingLoop`
- [x] `TestGroupDays_SameInputGivesTheSameDays`
- [x] `TestGroupDays_LisbonLikeDaysWalkUnder15Km`: 20 points laid out like Lisbon's top 20.
- [x] `TestGroupDays_ShortTripsStayCompact`: the same 20 points with `days=2`. Every day walks under 8 km, and the far-apart areas (Belém-like and Alfama-like) never share a day.
- [x] `TestGroupDays_AlwaysKeepsTheMostFamousPlace`: whatever the number of days, the top-ranked place is in the plan.

**Make them pass (GREEN):** the number of days, then k-means, then balancing, then `WalkingLoop`.

**Refactor:** name the steps as small functions; keep every loop order deterministic (no map iteration order).

**Acceptance criteria:** every test above is written first and passes.

**Verification:**
- Historical RED: see the task notes; unrecorded original failures remain unverified (not a new implementation task).
- [x] GREEN: `CGO_ENABLED=1 go test -race -count=1 -run GroupDays ./planner/`
- [x] Full suite: `make test`, `make lint`, `make build`

**Dependencies:** Task 1

**Files likely touched:** `planner/days.go`, `planner/days_test.go`

**Estimated scope:** Medium

---

## Task 9: Schedule days around the forecast

**Description:** `Schedule(groups, pool, fc, start, today)` in `planner/weather.go` dates the days and lets the weather decide which group goes on which day:

- **Dates:** day *i* is `start + i days`, in the forecast's time zone (UTC without a forecast).
- **Certain and Rainy:** `Certain` means the date is 0–6 days after `today`. `Rainy` means Certain, a rain value is present, and it's at least 5 mm.
- **Only certain days are rearranged:** the groups with the most Indoor plus Mixed stops go to rainy dates.
- **Rainy days are topped up:** outdoor stops are swapped for the **nearest unused** Indoor or Mixed places from `pool` until there are at least 3.

**Tests first (RED): all small**

- [x] `TestSchedule_DatesFollowTheStartDate`
- [x] `TestSchedule_DatesUseTheForecastTimeZone`
- [x] `TestSchedule_RainyDayGetsTheMostIndoorGroup`
- [x] `TestSchedule_RainyDayIsToppedUpToThreeIndoorStops`: a Kyoto-like case where the groups have 1 indoor stop and the pool has more further down.
- [x] `TestSchedule_TopUpUsesTheNearestUnusedIndoorPlaces`
- [x] `TestSchedule_NoPlaceAppearsTwice`
- [x] `TestSchedule_FiveMMIsRainy`
- [x] `TestSchedule_JustUnderFiveMMIsDry`
- [x] `TestSchedule_MissingRainIsNotRainy`
- [x] `TestSchedule_DaysSevenOrMoreOutAreNotCertain`
- [x] `TestSchedule_DaysSevenOrMoreOutAreNotRearranged`
- [x] `TestSchedule_DryDaysKeepTheirStops`
- [x] `TestSchedule_NilForecastKeepsTheOrderWithNoWeather`

**Make them pass (GREEN):** date the days, mark them, swap groups, then top up.

**Refactor:** separate "which day is rainy" from "which group goes where".

**Acceptance criteria:** every test above is written first and passes.

**Verification:**
- Historical RED: see the task notes; unrecorded original failures remain unverified (not a new implementation task).
- [x] GREEN: `CGO_ENABLED=1 go test -race -count=1 -run Schedule ./planner/`
- [x] Full suite: `make test`, `make lint`, `make build`

**Dependencies:** Task 1

**Files likely touched:** `planner/weather.go`, `planner/weather_test.go`

**Estimated scope:** Medium

---

## Task 10: Pick places to stay and build the booking link

**Description:** In `planner/stays.go`:

- `PickStays(center, stays)` keeps named places within 1 km of the center. It sorts them by website first, then stars (highest first), then distance, then name, and returns at most 6.
- `BookingURL(city, country, start, days)` returns `https://www.booking.com/searchresults.html?ss=<city, country>&checkin=YYYY-MM-DD&checkout=YYYY-MM-DD`.

**Tests first (RED): all small**

- [x] `TestPickStays_DropsPlacesFartherThan1Km`
- [x] `TestPickStays_DropsUnnamedPlaces`
- [x] `TestPickStays_PutsPlacesWithAWebsiteFirst`
- [x] `TestPickStays_ThenSortsByStars`
- [x] `TestPickStays_ThenSortsByDistance`
- [x] `TestPickStays_ReturnsAtMostSix`
- [x] `TestBookingURL_CheckoutIsStartPlusDays`
- [x] `TestBookingURL_EscapesCityAndCountry`: "São Paulo, Brazil".

**Make them pass (GREEN):** filter, then `sort.SliceStable` with the four keys.

**Refactor:** nothing special.

**Acceptance criteria:** every test above is written first and passes.

**Verification:**
- Historical RED: see the task notes; unrecorded original failures remain unverified (not a new implementation task).
- [x] GREEN: `CGO_ENABLED=1 go test -race -count=1 -run 'PickStays|BookingURL' ./planner/`
- [x] Full suite: `make test`, `make lint`, `make build`
- [x] Browser verifies the generated Booking URL carries city, check-in and check-out; its external landing page was not independently checked

**Dependencies:** Task 1

**Files likely touched:** `planner/stays.go`, `planner/stays_test.go`

**Estimated scope:** Small

---

## Task 11: Planner route and "Plan my trip" card, with a fake planner

**Description:**

- **`trip.go`:** adds the `TripPlanner` interface, `SetTripPlanner`, and `GET /plan?city=&country=&lat=&lon=&start=YYYY-MM-DD&days=N`, which returns the `trip_card` HTMX fragment.
- **Clock:** the server gets `now func() time.Time` (default `time.Now`) so tests can fix the date.
- **Card:** `views/trip_card.go.tpl` holds the form (start date defaulting to tomorrow, a 1–5 days select, hidden city, country, lat and lon) and the result (the note, then each day with its date, a rain badge in mm, "less certain" when not `Certain`, and each stop's photo, name and walk distance).
- **When it shows:** the card appears under the weather **only when a planner is set**.
- **Tests use a fake planner** whose `Plan` returns a fixed 3-day plan built from the request. So the tests check the rendered HTML, not which calls were made.

**Tests first (RED): medium, because templates load from `./views`, in-process Fiber**

- [x] `TestPlanRoute_RejectsDaysOutsideOneToFive`: 400 with a friendly message in the fragment.
- [x] `TestPlanRoute_RejectsABadStartDate`
- [x] `TestPlanRoute_RejectsAStartDateInThePast`
- [x] `TestPlanRoute_RejectsMissingCoordinates`
- [x] `TestPlanRoute_RendersEveryDayOfThePlan`: 3 day headings, with stop names and walk distances.
- [x] `TestPlanRoute_ShowsTheRainBadgeOnARainyDay`
- [x] `TestPlanRoute_MarksLessCertainDays`
- [x] `TestPlanRoute_ShowsAFriendlyMessageWhenPlanningFails`
- [x] `TestMainPage_HidesThePlannerCardWithoutAPlanner`
- [x] `TestMainPage_ShowsThePlannerCardWithCityAndCoordinates`
- [x] `TestMainPage_PlannerStartsTomorrowByDefault`: uses the fixed clock.

**Make them pass (GREEN):** parse and validate the input, call the planner, render the fragment.

**Refactor:** move input parsing into a `parsePlanRequest` function with its own small tests if it grows.

**Acceptance criteria:** every test above is written first and passes.

**Verification:**
- Historical RED: see the task notes; unrecorded original failures remain unverified (not a new implementation task).
- [x] GREEN: `CGO_ENABLED=1 go test -race -count=1 -run 'PlanRoute|MainPage' .`
- [x] Full suite: `make test`, `make lint`, `make build`
- [x] Browser check with a temporary fixture harness: `/plan` returns 200, no JavaScript exceptions, and screenshots at 375 px are saved in tasks/artifacts/

**Dependencies:** Tasks 1, 2

**Files likely touched:** `trip.go`, `trip_test.go`, `server.go`, `views/trip_card.go.tpl`, `views/content_fragment.go.tpl`, `views/index.go.tpl`

**Estimated scope:** Medium (6 files, but the template changes are small)

---

## Task 12: Stay card, photo credits and data credits

**Description:**

- **Stay card:** `views/stay_card.go.tpl` goes inside the trip card. It shows each place's name, an icon for hotel, hostel or guest house, stars only when above 0, and the website link, plus a "Check prices" button using `Plan.BookingURL`. When `StaysNote` is set, it shows the note instead of the list.
- **Photo credits:** every place photo links to its Commons page with "Photo: Wikimedia Commons".
- **Footer credits:** "Places: Wikipedia & Wikidata · Photos: Wikimedia Commons · Weather: Open-Meteo (CC BY 4.0) · Map data © OpenStreetMap contributors".
- **Styling** in `public/styles.css`.

**Tests first (RED): medium render tests with a fixed `Plan`, in `trip_render_test.go`**

- [x] `TestTripCard_ListsEveryStay`: up to 6.
- [x] `TestTripCard_ShowsStarsOnlyWhenKnown`
- [x] `TestTripCard_ShowsTheStaysNoteInsteadOfAList`
- [x] `TestTripCard_HasACheckPricesLink`
- [x] `TestTripCard_EveryPhotoLinksToItsCommonsPage`
- [x] `TestFooter_ShowsAllFourDataCredits`

**Make them pass (GREEN):** templates first, then CSS.

**Refactor:** reuse card styles from the weather card where they fit.

**Acceptance criteria:** every test above is written first and passes.

**Verification:**
- Historical RED: see the task notes; unrecorded original failures remain unverified (not a new implementation task).
- [x] GREEN: `CGO_ENABLED=1 go test -race -count=1 -run 'TripCard|Footer' .`
- [x] Full suite: `make test`, `make lint`, `make build`
- [x] Browser check with fixture planner: no sideways scrolling at 375 px, no JavaScript exceptions, and before/after screenshots are saved. The expected invalid-input 400 is recorded separately

**Dependencies:** Task 11

**Files likely touched:** `views/stay_card.go.tpl`, `views/trip_card.go.tpl`, `views/content_fragment.go.tpl`, `views/index.go.tpl`, `public/styles.css`, `trip_render_test.go`

**Estimated scope:** Medium

---

## Checkpoint B: Pieces (after Tasks 3–12)

- [x] `make lint` and `make test` pass in the combined working tree
- [x] Default tests pass. Historical RED notes are preserved; missing evidence is disclosed in notes-completion.md
- [x] The contract hasn't drifted: `planner/types.go` still matches plan.md
- [x] Most new tests are small (a quick look at the test lists)

---

## Phase 2: Joining the logic

## Task 13: Place-type tool and the generated type list

**Description:** `cmd/placetypes`:

1. Fetches the places near 20 cities with `api.WikimediaAPI`: Lisbon, Porto, Tavira, Funchal, Kyoto, Paris, Rome, Barcelona, London, New York, Tokyo, Istanbul, Prague, Amsterdam, Berlin, Seville, Florence, Vienna, Marrakesh and Mexico City.
2. Collects every P31 type with a count.
3. Walks subclass-of (P279) **upward** with `GetEntities`: depth 10 or less, cached in a temp file, and **safe against cycles**, because Wikidata has them.
4. Writes `planner/placetypes_gen.go` (`// Code generated by cmd/placetypes; DO NOT EDIT.`), sorted by ID, with `// label (count)` comments.
5. Prints a review summary: the top 30 kept types and the top 30 dropped types.

Rules, from the validation:

- **Indoor roots:** Q33506 museum, Q1007870 art gallery, Q2281788 public aquarium, Q24354 theatre building, Q16970 church building, Q2977 cathedral, Q163687 basilica, Q32815 mosque, Q34627 synagogue, Q16560 palace, Q44613 monastery, Q1060829 concert hall, Q153562 opera house, Q200764 bookstore, plus **market** (look up its ID; Mercado dos Lavradores, Q2063403, is one).
- **Outdoor roots:** Q22698 park, Q1107656 garden, Q167346 botanical garden, Q174782 square, Q6017969 scenic viewpoint, Q40080 beach, Q12518 tower, Q4989906 monument, Q179700 statue, Q23413 castle, Q57821 fortification, Q839954 archaeological site, Q43501 zoo, Q39715 lighthouse, Q79007 street, Q123705 neighborhood, Q39614 cemetery, Q845945 Shinto shrine, Q5393308 Buddhist temple, Q8502 mountain, Q23442 island.
- **Deny, exact type only:** Q494721 city of Japan, Q1549591 big city, Q137773 ward of Japan, Q1025961 capital of Japan, Q27554677 former capital, Q1220959 building of public administration, Q515 city.
- **Never roots:** bridge (Q12280) and library (Q7075).

**Tests first (RED): small, with a fake class graph in the test**

- [x] `TestRules_RootTypeGetsItsKind`
- [x] `TestRules_SubclassOfARootGetsItsKind`
- [x] `TestRules_TypeReachingIndoorAndOutdoorIsMixed`
- [x] `TestRules_DenyMatchesOnlyTheExactType`: a subclass of "ward of Japan" isn't denied.
- [x] `TestRules_BridgeAloneIsNotKept`
- [x] `TestRules_SurvivesACycleInTheClassGraph`
- [x] `TestRules_StopsAtDepthTen`
- [x] `TestRender_WritesAGofmtedMapSortedByID`

**Make them pass (GREEN):** keep the rules in `rules.go` as plain functions over a class graph, with the network only in `main.go`.

**Refactor:** keep `main.go` thin (fetch, then rules, then render).

**Acceptance criteria:**
- [x] Every default test above passes. Historical test-first evidence is in the task notes.
- [x] `go run ./cmd/placetypes` regenerates the file, which compiles.
- [ ] **A human reviewed** the printed summary and the generated file before merging.

**Verification:**
- Historical RED: see the task notes; unrecorded original failures remain unverified (not a new implementation task).
- [x] GREEN: `CGO_ENABLED=1 go test -race -count=1 ./cmd/placetypes/`
- [x] Full suite: `make test`, `make lint`, `make build`
- [x] By hand (large): run the tool once, and check that the map includes Buddhist temple (outdoor), museum (indoor) and ward of Japan (deny)

**Dependencies:** Tasks 1, 3

**Files likely touched:** `cmd/placetypes/main.go`, `cmd/placetypes/rules.go`, `cmd/placetypes/rules_test.go`, `planner/placetypes_gen.go`

**Estimated scope:** Medium

---

## Task 14: `Build`, which runs the whole planning chain

**Description:** `Build(ctx, req, places, forecast, stays, now)` in `planner/build.go`:

1. Check that `Days` is 1–5.
2. Get the places. On error, return it.
3. `Rank` with `PlaceTypes` and `Boosts`, then `GroupDays`. With 0 groups, return an error. With fewer groups than asked, set a note whose article and plural agree with the number: "Tavira has enough highlights for 2 days, so here's a 2-day plan." and, for one day, "… for 1 day, so here's a 1-day plan."
4. Get the forecast. If that fails, plan without weather. If the trip starts more than 16 days away, add "Forecast appears closer to your trip".
5. `Schedule` with `today` taken from `now` in the city's time zone.
6. Take the `Medoid` of all stops, then get the stays. If that fails, set `StaysNote` to "Places to stay are unavailable right now". Otherwise `PickStays`.
7. Set `BookingURL`, using the number of days the traveller **asked for** (`req.Days`), not the number the plan shrank to.

Tests use **fake sources** (small structs with canned data or a canned error). They swap `PlaceTypes` for a small map with `t.Cleanup` to restore it, so they don't run in parallel.

**Tests first (RED): all small**

- [x] `TestBuild_PlansTheRequestedNumberOfDays`
- [x] `TestBuild_RejectsZeroOrSixDays`
- [x] `TestBuild_FailsWhenPlacesCannotBeLoaded`
- [x] `TestBuild_FailsWhenNoPlaceSurvivesRanking`
- [x] `TestBuild_SmallTownNoteNamesTheCity`
- [x] `TestBuild_PlansWithoutWeatherWhenTheForecastFails`
- [x] `TestBuild_SaysTheForecastComesLaterForFarTrips`
- [x] `TestBuild_StillPlansWhenStaysFail`: `StaysNote` is set.
- [x] `TestBuild_PicksStaysAroundThePlanCenter`
- [x] `TestBuild_SetsTheBookingLink`
- [x] `TestBuild_TodayIsTakenInTheCityTimeZone`: 23:30 UTC is already the next day in Tokyo.
- [x] `TestBuild_SameInputGivesTheSamePlan`

**Make them pass (GREEN):** call the functions in order; no new logic beyond the steps above.

**Refactor:** keep the note texts in one place.

**Acceptance criteria:** every test above is written first and passes.

**Verification:**
- Historical RED: see the task notes; unrecorded original failures remain unverified (not a new implementation task).
- [x] GREEN: `CGO_ENABLED=1 go test -race -count=1 -run Build ./planner/`
- [x] Full suite: `make test`, `make lint`, `make build`

**Dependencies:** Tasks 1, 7, 8, 9, 10

**Files likely touched:** `planner/build.go`, `planner/build_test.go`

**Estimated scope:** Small

---

## Phase 3: Integration

## Task 15: Wire the real planner into the app

**Description:**

- **`tripplanner.go` (root):** `NewTripPlanner(places, forecast, stays)` returns a `TripPlanner` that calls `planner.Build` with `time.Now()`.
- **`env.go`:** reads `OVERPASS_URLS` (comma-separated, empty when unset).
- **`cmd/web/main.go`:** builds the three API clients (using `api.DefaultOverpassURLs` when `OVERPASS_URLS` is empty), wraps each with the Task 6 cache, and calls `server.SetTripPlanner(...)`. **The root package must not import `api/`.**

**Tests first (RED)**

- [x] Small: `TestTripPlanner_PlansFromTheGivenSources`, using fake sources.
- [x] Small: `TestLoadEnvKey_ReadsOverpassURLs`
- [x] Small: `TestLoadEnvKey_OverpassURLsAreEmptyWhenUnset`
- [x] Medium: `TestPlanRoute_WithTheRealTripPlanner`. It goes through the real route, `NewTripPlanner`, `planner.Build` and the templates, with fake sources, and the fragment shows the planned days and places to stay. This is the integration test across the whole chain.

**Make them pass (GREEN):** the adapter, the environment variable, then the wiring in `main.go`. `main.go` itself has no test; it's covered by the checks below.

**Refactor:** nothing special.

**Acceptance criteria:**
- [x] Every default test above passes. Historical test-first evidence is in the task notes.
- [x] `go list -deps .` shows the root package doesn't import `weatherservice/api`.

**Verification:**
- Historical RED: see the task notes; unrecorded original failures remain unverified (not a new implementation task).
- [x] GREEN: `CGO_ENABLED=1 go test -race -count=1 -run 'TripPlanner|LoadEnvKey|PlanRoute' .`
- [x] Full suite: `make test`, `make lint`, `make build`, and `docker build -t traveltab .`
- [x] Prior real-API HTTP smoke plus current browser fixture: photos, stays and dated plans render. Cached-route tests pass; see notes-wiring.md and notes-ui-validation.md for the precise validation boundaries.
- [x] Offline integration guard: all three providers return errors; fresh and expired caches both render the identical complete plan.

**Dependencies:** Tasks 3, 4, 5, 6, 11, 14 (and 13, for good results)

**Files likely touched:** `tripplanner.go`, `tripplanner_test.go`, `env.go`, `env_test.go`, `cmd/web/main.go`

**Estimated scope:** Medium

---

## Task 16: City acceptance tests for Lisbon, Tavira and Kyoto

**Description:** These tests are the one-pager's "Done when" list in code. `planner/cities_test.go` (package `planner_test`) runs `planner.Build` with the **real API clients** over a stubbed HTTP transport that serves recorded responses from `planner/testdata/cities/<city>/`. The forecasts are hand-edited so Lisbon and Kyoto each have a rainy day 2 days after the fixed `now`. Recording is opt-in: `RECORD_FIXTURES=1` uses the live APIs and saves the responses.

**RED comes for free:** write and run these **before Task 13 merges**. With the empty `PlaceTypes` placeholder, no place survives ranking, so every test fails. They turn green when Task 13's list lands. **Merge this task after Task 13** so `main` stays green.

**Tests first (RED): medium, fixture files on disk, no network**

- [x] `TestCities_LisbonIncludesBelemTowerAndJeronimos`
- [x] `TestCities_LisbonSkipsBridgesAndTheNationalLibrary`
- [x] `TestCities_LisbonRainyDayIsMostlyIndoor`
- [x] `TestCities_LisbonHasAtMostSixNamedStays`
- [x] `TestCities_TaviraFiveDaysGetsFewerDaysAndANote`
- [x] `TestCities_KyotoRainyDayHasThreeIndoorStops`
- [x] `TestCities_KyotoHasNoWards`: Fushimi-ku and Higashiyama-ku never appear.

**Make them pass (GREEN):** nothing to write here. Task 13's type list makes them pass. If one still fails after Task 13, treat it as a bug: add a smaller test that reproduces it in the right task's file, fix it there, and keep this test as the guard.

**Acceptance criteria:**
- [x] Every city test passes with the generated types. The empty-type-list mutation fails; original pre-Task-13 RED evidence was not saved.
- [x] The fixtures are trimmed (about 2 MB in total at most) and contain no secrets.

**Verification:**
- Historical RED: see the task notes; unrecorded original failures remain unverified (not a new implementation task).
- [x] GREEN: `CGO_ENABLED=1 go test -race -count=1 -run Cities ./planner/`, with the network off
- [x] Full suite: `make test`, `make lint`, `make build`

**Dependencies:** Tasks 3, 4, 5, 14. Turns green with Task 13.

**Files likely touched:** `planner/cities_test.go`, `planner/testdata/cities/`

**Estimated scope:** Medium

---

## Checkpoint C: End to end (after Tasks 13–16)

- [x] `make test` passes, including recorded city tests and offline fake-provider checks; no live provider calls are made
- [x] Recorded-city acceptance tests plus the separate browser fixture check cover the MVP:
  - [x] "Lisbon, 3 days, next week" gives a sensible plan, and the rainy day gets the museums
  - [x] "Tavira, 5 days" gives a shorter plan with a clear message
  - [x] "Kyoto, 3 days" with a rainy day gets 3 or more indoor stops that day
  - [x] A cached city loads in under 2 seconds
  - [x] The trip planner uses only the keyless sources wired in cmd/web
- [ ] Human release review of the completed implementation and docs

---

## Phase 4: Polish

## Task 17: README "How the planner works" and CHANGELOG

**Description:** A docs-only task, so there's no TDD cycle. Update the README:

- the feature list: add the planner and the stay card, and remove "Nearby hotels … (removed for now)"
- the data sources: add Wikipedia/Wikidata, the Open-Meteo forecast and Overpass; remove Google Places and Foursquare
- the routes: add `GET /plan`, remove `/generate-itinerary`
- the environment variables: remove `PLACES_API_NEW` and `FOURSQUARE_API_KEY`, add `OVERPASS_URLS`
- a new **"How the planner works"** section with a diagram of `Rank → GroupDays → Schedule → Medoid → PickStays` and the fallbacks
- the credits and licenses
- the Tests section: all tests run offline, the city tests are acceptance tests, `RECORD_FIXTURES=1` re-records fixtures, and `LIVE_API_TESTS=1` runs the live checks

Add the changes to `CHANGELOG.md` under `[Unreleased]`.

**Acceptance criteria:**
- [x] The README has no mention of Google Places, Foursquare or the itinerary endpoint.
- [x] The "How the planner works" section fits on one screen and has a diagram.
- [x] The CHANGELOG `[Unreleased]` section lists the planner (Added) and the old hotels and itinerary code (Removed).

**Verification:**
- [ ] Release review: verify README rendering on GitHub after the PR is created
- [x] No test run is needed for a docs-only change

**Dependencies:** Task 15 (and 16 for the Tests section)

**Files likely touched:** `README.md`, `CHANGELOG.md`

**Estimated scope:** Small

---

## Checkpoint D: Complete

- [x] All automated MVP acceptance checks pass; only the three explicitly opt-in live API tests are skipped by default
- [ ] CI is green on `main`
- [ ] Human approves the deploy (`fly deploy`)
