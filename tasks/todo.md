# Tasks: Weather-aware trip planner (MVP)

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

**Can start now:** Tasks 1 and 2.

---

## Phase 0: Foundation

## Task 1: Planner contract and geo helpers

**Description:** Create `planner/types.go` exactly as in plan.md's contract, plus the geo helpers in `planner/geo.go`, the empty `PlaceTypes` placeholder, and `api/useragent.go`. The types and constants have no behavior, so compiling them is their check. The helpers are built test-first.

**Tests first (RED): all small, in `planner/geo_test.go`**

- [ ] `TestDistanceKM_LisbonToPortoIsAbout274Km`: (38.7223, -9.1393) to (41.1496, -8.6110) is 274 km ±1%.
- [ ] `TestDistanceKM_SamePlaceIsZero`
- [ ] `TestWalkingLoop_StartsAtTheMostFamousStop`: the stop with the most `Sitelinks` comes first.
- [ ] `TestWalkingLoop_BreaksFameTiesByLowerID`
- [ ] `TestWalkingLoop_VisitsEveryStopOnce`
- [ ] `TestWalkingLoop_WalkKMIncludesTheWayBack`: for 3 points on a line, the loop length is twice the span.
- [ ] `TestWalkingLoop_SingleStopWalksZeroKM`
- [ ] `TestMedoid_PicksThePlaceClosestToAllOthers`
- [ ] `TestMedoid_BreaksTiesByLowerID`
- [ ] `TestImageURL_BuildsACommonsThumbnailURL`: "Torre de Belém.jpg" at width 400 gives `…/Special:FilePath/Torre_de_Bel%C3%A9m.jpg?width=400`.
- [ ] `TestImageURL_IsEmptyWithoutAnImage`
- [ ] `TestImagePageURL_LinksToTheCommonsFilePage`

**Make them pass (GREEN):** haversine for distance, nearest neighbour for the loop.

**Refactor:** keep `geo.go` free of anything that isn't geometry or image URLs.

**Acceptance criteria:**
- [ ] `planner/types.go` matches the contract in plan.md word for word.
- [ ] `planner` imports nothing from `weatherservice` or `weatherservice/api`.
- [ ] `planner/placetypes_gen.go` holds `var PlaceTypes = map[string]Kind{}`, and `api/useragent.go` holds `UserAgent`.

**Verification:**
- [ ] RED seen: every listed test failed on an assertion before the code existed
- [ ] GREEN: `CGO_ENABLED=1 go test -race -count=1 ./planner/`
- [ ] Full suite: `make test`, `make lint`, `make build`

**Dependencies:** None

**Files likely touched:** `planner/types.go`, `planner/geo.go`, `planner/geo_test.go`, `planner/placetypes_gen.go`, `api/useragent.go`

**Estimated scope:** Medium

---

## Task 2: Remove the old hotels and itinerary code

**Description:** Delete the Google Places hotels feature and the Foursquare itinerary feature: API clients, reporter types, the `/generate-itinerary` route and handler, their templates, and their environment keys. `NewAppServer` then takes only the weather and video reporters. This task is mostly deletions, so it starts with **tests for what must change** (they fail now) and **guard tests for what must keep working** (they pass before and after).

**Tests first (RED): small, in-process Fiber app**

- [ ] `TestServer_ItineraryRouteIsGone`: `POST /generate-itinerary` returns 404.
- [ ] `TestListGeneralInfo_PageHasNoHotelsCard`: neither the full page nor the HTMX fragment contains the hotels markup.
- [ ] `TestLoadEnvKey_NoLongerReadsPlacesOrFoursquareKeys`: the keys struct has no such fields. Update `Test_LoadEnvKey` to match.

**Guard tests (write them first; they pass before and after)**

- [ ] `TestCheckDatabase_LoadsOldRowsThatStillHaveHotels`: a cached `city_data` row with a `Hotels` field still loads, because unknown JSON fields are ignored.
- [ ] The existing `TestListGeneralInfo_*` tests for weather and videos stay green. Tests for the removed hotels behavior (`TestRetireveFreshInformation_HotelsError`, the hotels mocks) are deleted along with it.

**Make them pass (GREEN):** delete the code, templates and keys, and change `NewAppServer(weatherReporters, videoStreamReporters)`.

**Refactor:** remove helpers, mocks and imports that nothing uses any more.

**Acceptance criteria:**
- [ ] `grep -riE "foursquare|googleplaces|googlephotos|itinerary|itenary|hotels_card" --include='*.go' --include='*.tpl' .` finds nothing. Docs are left for Task 17.
- [ ] Only templates used solely by hotels or the itinerary are deleted. Check each with `grep` first.

**Verification:**
- [ ] RED seen: the 3 new tests failed before the deletions, and the guard test passed before and after
- [ ] Full suite: `make test`, `make lint`, `make build`
- [ ] Browser check (large, by hand): `make run`, search "Porto, Portugal". Weather and videos show, with no hotels card and no console errors.

**Dependencies:** None

**Files likely touched:**
- `server.go`, `server_test.go`, `reporters.go`, `cmd/web/main.go`, `env.go`, `env_test.go`
- `api/foursquare.go`, `api/googlaPlaces.go`, `api/googlePhotos.go` (delete)
- `views/content_fragment.go.tpl`, `views/index.go.tpl`
- Templates to delete: `hotels_card`, `itinerary_*`, `form_dates`, `form_categories`, `category_checkboxes`, `map_initialization`, `map_markers`

**Estimated scope:** Large (mostly deletions; don't split, because every piece meets in `server.go`)

---

## Checkpoint A: Foundation (after Tasks 1–2)

- [ ] `make lint`, `make test` and `make build` pass on `main`
- [ ] The site shows weather and videos, with no hotels card
- [ ] Phase 1 can start: Tasks 11 and 12 need Task 2; the rest need only Task 1

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

- [ ] `TestWikimedia_PlacesNear_ReturnsPlacesWithWikidataDetails`: in the Tavira fixture, Castle of Tavira has its ID, coordinates, sitelinks, types and image.
- [ ] `TestWikimedia_PlacesNear_LeavesKindEmpty`
- [ ] `TestWikimedia_PlacesNear_UsesTheTitleWhenThereIsNoEnglishLabel`
- [ ] `TestWikimedia_PlacesNear_SplitsACappedSearch`: when the first search returns 500, a place that only an outer search returns (9.5 km out) is in the result.
- [ ] `TestWikimedia_PlacesNear_DropsSplitResultsBeyond10Km`
- [ ] `TestWikimedia_PlacesNear_ReturnsEachPlaceOnce`: places that overlapping searches both return appear once.
- [ ] `TestWikimedia_PlacesNear_DoesNotSplitAnUncappedSearch`: a boundary check on cost. With 218 results, only one geosearch request is made.
- [ ] `TestWikimedia_PlacesNear_HasNoTypesBeyondTheTop300`: the 301st place by sitelinks has empty `Types`.
- [ ] `TestWikimedia_SendsTheUserAgent`
- [ ] `TestWikimedia_ErrorIncludesTheStatusOnNon200`
- [ ] `TestWikimedia_StopsWhenTheContextIsCancelled`
- [ ] `TestWikimedia_GetEntities_ReturnsTheRequestedEntities`
- [ ] Large, opt-in: `TestWikimediaLive_FindsBelemTowerInLisbon` (runs only with `LIVE_API_TESTS=1`)

**Make them pass (GREEN):** one method per step (search, split, IDs, sitelinks, details), each requesting 50 IDs at a time.

**Refactor:** share one request helper that sets the `User-Agent` and checks the status.

**Acceptance criteria:**
- [ ] Every test above is written first and passes.
- [ ] `NewWikimediaAPI(nil)` uses a default client with a 30 s timeout.

**Verification:**
- [ ] RED seen for every small test
- [ ] GREEN: `CGO_ENABLED=1 go test -race -count=1 -run Wikimedia ./api/`
- [ ] Full suite: `make test`, `make lint`, `make build`
- [ ] Large, by hand, once: `LIVE_API_TESTS=1 go test -run WikimediaLive ./api/`

**Dependencies:** Task 1

**Files likely touched:** `api/wikimedia.go`, `api/wikimedia_test.go`, `api/testdata/wikimedia/`

**Estimated scope:** Medium

---

## Task 4: Open-Meteo daily forecast source

**Description:** `api/openmeteo_forecast.go` implements `planner.ForecastSource` with `GET https://api.open-meteo.com/v1/forecast?latitude=…&longitude=…&daily=precipitation_sum&timezone=auto&forecast_days=16`. Keep it separate from the existing marine client (`api/openmateo.go`). Import `time/tzdata` so the slim Docker image works.

**Tests first (RED): small, stubbed HTTP, fixture `api/testdata/openmeteo/forecast_kyoto.json`**

- [ ] `TestForecast_ReturnsSixteenDays`
- [ ] `TestForecast_ReturnsTheCityTimeZone`: `Asia/Tokyo`.
- [ ] `TestForecast_DatesAreMidnightInTheCityTimeZone`
- [ ] `TestForecast_KeepsTheRainAmount`: 24.4 mm stays 24.4.
- [ ] `TestForecast_NullRainBecomesNil`
- [ ] `TestForecast_ErrorsOnNon200`
- [ ] `TestForecast_ErrorsOnAnUnknownTimeZone`
- [ ] `TestForecast_AsksForDailyRainInTheLocalTimeZone`: a boundary check that the query has `daily=precipitation_sum`, `timezone=auto` and `forecast_days=16`.
- [ ] Large, opt-in: `TestForecastLive_ReturnsKyoto`

**Make them pass (GREEN):** decode into a struct with `[]*float64` for the rain values.

**Refactor:** nothing special.

**Acceptance criteria:** every test above is written first and passes.

**Verification:**
- [ ] RED seen
- [ ] GREEN: `CGO_ENABLED=1 go test -race -count=1 -run Forecast ./api/`
- [ ] Full suite: `make test`, `make lint`, `make build`

**Dependencies:** Task 1

**Files likely touched:** `api/openmeteo_forecast.go`, `api/openmeteo_forecast_test.go`, `api/testdata/openmeteo/forecast_kyoto.json`

**Estimated scope:** Small

---

## Task 5: Overpass stay source with fallback servers

**Description:** `api/overpass.go` implements `planner.StaySource`:

- **Query:** `nwr["tourism"~"^(hotel|hostel|guest_house)$"](around:1000,LAT,LON); out center tags;`, sent as the `data` form field.
- **Servers:** tried in order from `DefaultOverpassURLs` (`overpass-api.de`, then `overpass.private.coffee`). Each gets a 10 s timeout and **one retry** on 429, 5xx or timeout, honoring `Retry-After` capped at 5 s, before the next server is tried.
- **Sleeping is injectable** (`sleep func(time.Duration)`), so tests never wait.

**Tests first (RED): small, stubbed HTTP that answers per server, fixture `api/testdata/overpass/tavira.json`**

- [ ] `TestOverpass_ReturnsStaysFromTheFirstServer`
- [ ] `TestOverpass_RetriesOnceBeforeMovingOn`: the first server answers 504, then 200, and the result comes from the first server.
- [ ] `TestOverpass_FallsBackAfterTwoFailures`: the first server answers 504 twice, and the result comes from the second.
- [ ] `TestOverpass_ErrorNamesEveryFailedServer`
- [ ] `TestOverpass_WaitsForRetryAfterUpToFiveSeconds`: a `Retry-After: 30` header leads to a 5 s wait, checked through the fake sleeper.
- [ ] `TestOverpass_SkipsUnnamedPlaces`
- [ ] `TestOverpass_PrefersTheEnglishName`
- [ ] `TestOverpass_ReadsTheWebsiteFromContactWebsite`
- [ ] `TestOverpass_ReadsStarsLike4SAsFour`
- [ ] `TestOverpass_StarsAreZeroWhenMissing`
- [ ] `TestOverpass_UsesTheWayCenterForCoordinates`
- [ ] `TestOverpass_SendsTheQueryAndUserAgent`: a boundary check on the `data` form field and the header.
- [ ] Large, opt-in: `TestOverpassLive_FindsHotelsInTavira`

**Make them pass (GREEN):** one loop over the servers, with one retry inside.

**Refactor:** keep parsing (tags to `planner.Stay`) in its own function.

**Acceptance criteria:** every test above is written first and passes.

**Verification:**
- [ ] RED seen
- [ ] GREEN: `CGO_ENABLED=1 go test -race -count=1 -run Overpass ./api/`
- [ ] Full suite: `make test`, `make lint`, `make build`

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

- [ ] `TestSourceCache_ServesAFreshEntryFromTheCache`: the second call still returns "call 1".
- [ ] `TestSourceCache_RefreshesAnExpiredEntry`: after 31 days, it returns "call 2".
- [ ] `TestSourceCache_ServesOldDataWhenTheRefreshFails`
- [ ] `TestSourceCache_ReturnsTheErrorWhenNothingIsCached`
- [ ] `TestSourceCache_SharesEntriesForNearbyCoordinates`: 38.72231 and 38.72234 give the same entry.
- [ ] `TestSourceCache_ExpiresForecastsAfterThreeHours`
- [ ] `TestSourceCache_KeepsSourcesApart`: places and stays for the same coordinates don't mix.
- [ ] `TestInitDB_CreatesTheSourceCacheTable`: an existing database file opens and gets the table.

**Make them pass (GREEN):** one generic `cached[T]` helper used by the three wrappers.

**Refactor:** reuse the gzip code from `database.go` instead of copying it.

**Acceptance criteria:** every test above is written first and passes.

**Verification:**
- [ ] RED seen
- [ ] GREEN: `CGO_ENABLED=1 go test -race -count=1 -run 'SourceCache|InitDB' .`
- [ ] Full suite: `make test`, `make lint`, `make build`

**Dependencies:** Task 1

**Files likely touched:** `sourcecache.go`, `sourcecache_test.go`, `database.go`

**Estimated scope:** Medium

---

## Task 7: Rank places, with the type filter and boost list

**Description:** `Rank(candidates, types, boosts)` in `planner/rank.go` keeps the places worth visiting, sets their `Kind`, and sorts them by fame. `planner/boosts.go` holds `Boosts` with the 5 starting entries: Q652806 indoor, Q168001 outdoor, Q2063403 indoor, Q23579173 outdoor, Q11650434 indoor. The tests use a small type map written inside the test, not `PlaceTypes`.

**Tests first (RED): all small**

- [ ] `TestRank_KeepsAPlaceWithAnIndoorType`
- [ ] `TestRank_DropsAPlaceWithOnlyUnknownTypes`: a plain bridge.
- [ ] `TestRank_DenyTypeBeatsIndoorType`
- [ ] `TestRank_IndoorPlusOutdoorTypesMakeMixed`
- [ ] `TestRank_SortsByFameThenByID`
- [ ] `TestRank_ReturnsAtMostSixtyPlaces`
- [ ] `TestRank_SetsKindOnEveryPlace`
- [ ] `TestRank_BoostedPlaceSkipsTheTypeFilter`
- [ ] `TestRank_BoostedPlaceGetsTheBoostKind`
- [ ] `TestRank_BoostedPlaceRanksLikeTheTenthPlace`: a boosted place with 7 sitelinks lands in the top 10 of a 60-place list.
- [ ] `TestRank_BoostWithFewerThanTenPlacesRanksLikeTheLastPlace`
- [ ] `TestRank_DoesNotChangeTheInput`
- [ ] `TestBoosts_HasTheFiveStartingPlaces`

**Make them pass (GREEN):** filter, then sort by an "effective fame" value. Never change `Place.Sitelinks`.

**Refactor:** keep "which kind is this place" in one small function.

**Acceptance criteria:** every test above is written first and passes.

**Verification:**
- [ ] RED seen
- [ ] GREEN: `CGO_ENABLED=1 go test -race -count=1 -run 'Rank|Boosts' ./planner/`
- [ ] Full suite: `make test`, `make lint`, `make build`

**Dependencies:** Task 1

**Files likely touched:** `planner/rank.go`, `planner/boosts.go`, `planner/rank_test.go`

**Estimated scope:** Small

---

## Task 8: Group places into days

**Description:** `GroupDays(ranked, days)` in `planner/days.go` takes the top `SunnyPoolSize` places and groups them by location:

- **Number of days:** at most `min(days, len(pool) / MinStopsPerDay)`, and at least 1 when there's at least one place.
- **Clustering:** a deterministic k-means, seeded farthest-point style from the most famous place, with ties broken by ID.
- **Balancing:** each group gets 3–4 stops. Overfull groups pass their farthest stops to nearby groups with room, or drop their lowest-ranked stops.
- **Output:** each group ordered with `WalkingLoop`, and groups sorted by their most famous stop.

**Tests first (RED): all small**

- [ ] `TestGroupDays_NeverMixesTwoFarApartAreas`: Belém-like points in the west, Alfama-like points in the east.
- [ ] `TestGroupDays_EveryDayHasThreeToFourStops`
- [ ] `TestGroupDays_SmallTownGetsFewerDays`: 7 places with 5 days asked gives 2 days.
- [ ] `TestGroupDays_OnePlaceGivesOneDay`
- [ ] `TestGroupDays_NoPlacesGivesNoDays`
- [ ] `TestGroupDays_UsesOnlyTheTop20`
- [ ] `TestGroupDays_NoPlaceAppearsTwice`
- [ ] `TestGroupDays_OrdersEachDayAsAWalkingLoop`
- [ ] `TestGroupDays_SameInputGivesTheSameDays`
- [ ] `TestGroupDays_LisbonLikeDaysWalkUnder15Km`: 20 points laid out like Lisbon's top 20.

**Make them pass (GREEN):** the number of days, then k-means, then balancing, then `WalkingLoop`.

**Refactor:** name the steps as small functions; keep every loop order deterministic (no map iteration order).

**Acceptance criteria:** every test above is written first and passes.

**Verification:**
- [ ] RED seen
- [ ] GREEN: `CGO_ENABLED=1 go test -race -count=1 -run GroupDays ./planner/`
- [ ] Full suite: `make test`, `make lint`, `make build`

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

- [ ] `TestSchedule_DatesFollowTheStartDate`
- [ ] `TestSchedule_DatesUseTheForecastTimeZone`
- [ ] `TestSchedule_RainyDayGetsTheMostIndoorGroup`
- [ ] `TestSchedule_RainyDayIsToppedUpToThreeIndoorStops`: a Kyoto-like case where the groups have 1 indoor stop and the pool has more further down.
- [ ] `TestSchedule_TopUpUsesTheNearestUnusedIndoorPlaces`
- [ ] `TestSchedule_NoPlaceAppearsTwice`
- [ ] `TestSchedule_FiveMMIsRainy`
- [ ] `TestSchedule_JustUnderFiveMMIsDry`
- [ ] `TestSchedule_MissingRainIsNotRainy`
- [ ] `TestSchedule_DaysSevenOrMoreOutAreNotCertain`
- [ ] `TestSchedule_DaysSevenOrMoreOutAreNotRearranged`
- [ ] `TestSchedule_DryDaysKeepTheirStops`
- [ ] `TestSchedule_NilForecastKeepsTheOrderWithNoWeather`

**Make them pass (GREEN):** date the days, mark them, swap groups, then top up.

**Refactor:** separate "which day is rainy" from "which group goes where".

**Acceptance criteria:** every test above is written first and passes.

**Verification:**
- [ ] RED seen
- [ ] GREEN: `CGO_ENABLED=1 go test -race -count=1 -run Schedule ./planner/`
- [ ] Full suite: `make test`, `make lint`, `make build`

**Dependencies:** Task 1

**Files likely touched:** `planner/weather.go`, `planner/weather_test.go`

**Estimated scope:** Medium

---

## Task 10: Pick places to stay and build the booking link

**Description:** In `planner/stays.go`:

- `PickStays(center, stays)` keeps named places within 1 km of the center. It sorts them by website first, then stars (highest first), then distance, then name, and returns at most 6.
- `BookingURL(city, country, start, days)` returns `https://www.booking.com/searchresults.html?ss=<city, country>&checkin=YYYY-MM-DD&checkout=YYYY-MM-DD`.

**Tests first (RED): all small**

- [ ] `TestPickStays_DropsPlacesFartherThan1Km`
- [ ] `TestPickStays_DropsUnnamedPlaces`
- [ ] `TestPickStays_PutsPlacesWithAWebsiteFirst`
- [ ] `TestPickStays_ThenSortsByStars`
- [ ] `TestPickStays_ThenSortsByDistance`
- [ ] `TestPickStays_ReturnsAtMostSix`
- [ ] `TestBookingURL_CheckoutIsStartPlusDays`
- [ ] `TestBookingURL_EscapesCityAndCountry`: "São Paulo, Brazil".

**Make them pass (GREEN):** filter, then `sort.SliceStable` with the four keys.

**Refactor:** nothing special.

**Acceptance criteria:** every test above is written first and passes.

**Verification:**
- [ ] RED seen
- [ ] GREEN: `CGO_ENABLED=1 go test -race -count=1 -run 'PickStays|BookingURL' ./planner/`
- [ ] Full suite: `make test`, `make lint`, `make build`
- [ ] By hand: open one generated link and check that Booking shows the right city and dates

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

- [ ] `TestPlanRoute_RejectsDaysOutsideOneToFive`: 400 with a friendly message in the fragment.
- [ ] `TestPlanRoute_RejectsABadStartDate`
- [ ] `TestPlanRoute_RejectsAStartDateInThePast`
- [ ] `TestPlanRoute_RejectsMissingCoordinates`
- [ ] `TestPlanRoute_RendersEveryDayOfThePlan`: 3 day headings, with stop names and walk distances.
- [ ] `TestPlanRoute_ShowsTheRainBadgeOnARainyDay`
- [ ] `TestPlanRoute_MarksLessCertainDays`
- [ ] `TestPlanRoute_ShowsAFriendlyMessageWhenPlanningFails`
- [ ] `TestMainPage_HidesThePlannerCardWithoutAPlanner`
- [ ] `TestMainPage_ShowsThePlannerCardWithCityAndCoordinates`
- [ ] `TestMainPage_PlannerStartsTomorrowByDefault`: uses the fixed clock.

**Make them pass (GREEN):** parse and validate the input, call the planner, render the fragment.

**Refactor:** move input parsing into a `parsePlanRequest` function with its own small tests if it grows.

**Acceptance criteria:** every test above is written first and passes.

**Verification:**
- [ ] RED seen
- [ ] GREEN: `CGO_ENABLED=1 go test -race -count=1 -run 'PlanRoute|MainPage' .`
- [ ] Full suite: `make test`, `make lint`, `make build`
- [ ] Browser check (large, by hand), with the fake planner wired in `main.go` on your machine only (don't commit it): the console has no errors, `/plan` returns 200, and a screenshot at 375 px wide looks right

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

- [ ] `TestTripCard_ListsEveryStay`: up to 6.
- [ ] `TestTripCard_ShowsStarsOnlyWhenKnown`
- [ ] `TestTripCard_ShowsTheStaysNoteInsteadOfAList`
- [ ] `TestTripCard_HasACheckPricesLink`
- [ ] `TestTripCard_EveryPhotoLinksToItsCommonsPage`
- [ ] `TestFooter_ShowsAllFourDataCredits`

**Make them pass (GREEN):** templates first, then CSS.

**Refactor:** reuse card styles from the weather card where they fit.

**Acceptance criteria:** every test above is written first and passes.

**Verification:**
- [ ] RED seen
- [ ] GREEN: `CGO_ENABLED=1 go test -race -count=1 -run 'TripCard|Footer' .`
- [ ] Full suite: `make test`, `make lint`, `make build`
- [ ] Browser check (large, by hand), with the local fake planner: no sideways scrolling at 375 px, the console is clean, and there are before and after screenshots

**Dependencies:** Task 11

**Files likely touched:** `views/stay_card.go.tpl`, `views/trip_card.go.tpl`, `views/content_fragment.go.tpl`, `views/index.go.tpl`, `public/styles.css`, `trip_render_test.go`

**Estimated scope:** Medium

---

## Checkpoint B: Pieces (after Tasks 3–12)

- [ ] `make lint` and `make test` pass on `main` with all Phase 1 work merged
- [ ] Every task noted its RED output, and no test was skipped or disabled
- [ ] The contract hasn't drifted: `planner/types.go` still matches plan.md
- [ ] Most new tests are small (a quick look at the test lists)

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

- [ ] `TestRules_RootTypeGetsItsKind`
- [ ] `TestRules_SubclassOfARootGetsItsKind`
- [ ] `TestRules_TypeReachingIndoorAndOutdoorIsMixed`
- [ ] `TestRules_DenyMatchesOnlyTheExactType`: a subclass of "ward of Japan" isn't denied.
- [ ] `TestRules_BridgeAloneIsNotKept`
- [ ] `TestRules_SurvivesACycleInTheClassGraph`
- [ ] `TestRules_StopsAtDepthTen`
- [ ] `TestRender_WritesAGofmtedMapSortedByID`

**Make them pass (GREEN):** keep the rules in `rules.go` as plain functions over a class graph, with the network only in `main.go`.

**Refactor:** keep `main.go` thin (fetch, then rules, then render).

**Acceptance criteria:**
- [ ] Every test above is written first and passes.
- [ ] `go run ./cmd/placetypes` regenerates the file, which compiles.
- [ ] **A human reviewed** the printed summary and the generated file before merging.

**Verification:**
- [ ] RED seen
- [ ] GREEN: `CGO_ENABLED=1 go test -race -count=1 ./cmd/placetypes/`
- [ ] Full suite: `make test`, `make lint`, `make build`
- [ ] By hand (large): run the tool once, and check that the map includes Buddhist temple (outdoor), museum (indoor) and ward of Japan (deny)

**Dependencies:** Tasks 1, 3

**Files likely touched:** `cmd/placetypes/main.go`, `cmd/placetypes/rules.go`, `cmd/placetypes/rules_test.go`, `planner/placetypes_gen.go`

**Estimated scope:** Medium

---

## Task 14: `Build`, which runs the whole planning chain

**Description:** `Build(ctx, req, places, forecast, stays, now)` in `planner/build.go`:

1. Check that `Days` is 1–5.
2. Get the places. On error, return it.
3. `Rank` with `PlaceTypes` and `Boosts`, then `GroupDays`. With 0 groups, return an error. With fewer groups than asked, set the note "<City> has enough highlights for N days, so here's an N-day plan."
4. Get the forecast. If that fails, plan without weather. If the trip starts more than 16 days away, add "Forecast appears closer to your trip".
5. `Schedule` with `today` taken from `now` in the city's time zone.
6. Take the `Medoid` of all stops, then get the stays. If that fails, set `StaysNote` to "Places to stay are unavailable right now". Otherwise `PickStays`.
7. Set `BookingURL`.

Tests use **fake sources** (small structs with canned data or a canned error). They swap `PlaceTypes` for a small map with `t.Cleanup` to restore it, so they don't run in parallel.

**Tests first (RED): all small**

- [ ] `TestBuild_PlansTheRequestedNumberOfDays`
- [ ] `TestBuild_RejectsZeroOrSixDays`
- [ ] `TestBuild_FailsWhenPlacesCannotBeLoaded`
- [ ] `TestBuild_FailsWhenNoPlaceSurvivesRanking`
- [ ] `TestBuild_SmallTownNoteNamesTheCity`
- [ ] `TestBuild_PlansWithoutWeatherWhenTheForecastFails`
- [ ] `TestBuild_SaysTheForecastComesLaterForFarTrips`
- [ ] `TestBuild_StillPlansWhenStaysFail`: `StaysNote` is set.
- [ ] `TestBuild_PicksStaysAroundThePlanCenter`
- [ ] `TestBuild_SetsTheBookingLink`
- [ ] `TestBuild_TodayIsTakenInTheCityTimeZone`: 23:30 UTC is already the next day in Tokyo.
- [ ] `TestBuild_SameInputGivesTheSamePlan`

**Make them pass (GREEN):** call the functions in order; no new logic beyond the steps above.

**Refactor:** keep the note texts in one place.

**Acceptance criteria:** every test above is written first and passes.

**Verification:**
- [ ] RED seen
- [ ] GREEN: `CGO_ENABLED=1 go test -race -count=1 -run Build ./planner/`
- [ ] Full suite: `make test`, `make lint`, `make build`

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

- [ ] Small: `TestTripPlanner_PlansFromTheGivenSources`, using fake sources.
- [ ] Small: `TestLoadEnvKey_ReadsOverpassURLs`
- [ ] Small: `TestLoadEnvKey_OverpassURLsAreEmptyWhenUnset`
- [ ] Medium: `TestPlanRoute_WithTheRealTripPlanner`. It goes through the real route, `NewTripPlanner`, `planner.Build` and the templates, with fake sources, and the fragment shows the planned days and places to stay. This is the integration test across the whole chain.

**Make them pass (GREEN):** the adapter, the environment variable, then the wiring in `main.go`. `main.go` itself has no test; it's covered by the checks below.

**Refactor:** nothing special.

**Acceptance criteria:**
- [ ] Every test above is written first and passes.
- [ ] `go list -deps .` shows the root package doesn't import `weatherservice/api`.

**Verification:**
- [ ] RED seen
- [ ] GREEN: `CGO_ENABLED=1 go test -race -count=1 -run 'TripPlanner|LoadEnvKey|PlanRoute' .`
- [ ] Full suite: `make test`, `make lint`, `make build`, and `docker build -t traveltab .`
- [ ] Browser check (large, by hand): `make run`, search "Lisbon, Portugal", and plan 3 days from tomorrow. A real plan with photos and places to stay appears, the console is clean, and a second identical request comes back in under 2 s.
- [ ] Offline check (large, by hand): after one cached run, turn the network off. The same plan still loads.

**Dependencies:** Tasks 3, 4, 5, 6, 11, 14 (and 13, for good results)

**Files likely touched:** `tripplanner.go`, `tripplanner_test.go`, `env.go`, `env_test.go`, `cmd/web/main.go`

**Estimated scope:** Medium

---

## Task 16: City acceptance tests for Lisbon, Tavira and Kyoto

**Description:** These tests are the one-pager's "Done when" list in code. `planner/cities_test.go` (package `planner_test`) runs `planner.Build` with the **real API clients** over a stubbed HTTP transport that serves recorded responses from `planner/testdata/cities/<city>/`. The forecasts are hand-edited so Lisbon and Kyoto each have a rainy day 2 days after the fixed `now`. Recording is opt-in: `RECORD_FIXTURES=1` uses the live APIs and saves the responses.

**RED comes for free:** write and run these **before Task 13 merges**. With the empty `PlaceTypes` placeholder, no place survives ranking, so every test fails. They turn green when Task 13's list lands. **Merge this task after Task 13** so `main` stays green.

**Tests first (RED): medium, fixture files on disk, no network**

- [ ] `TestCities_LisbonIncludesBelemTowerAndJeronimos`
- [ ] `TestCities_LisbonSkipsBridgesAndTheNationalLibrary`
- [ ] `TestCities_LisbonRainyDayIsMostlyIndoor`
- [ ] `TestCities_LisbonHasAtMostSixNamedStays`
- [ ] `TestCities_TaviraFiveDaysGetsFewerDaysAndANote`
- [ ] `TestCities_KyotoRainyDayHasThreeIndoorStops`
- [ ] `TestCities_KyotoHasNoWards`: Fushimi-ku and Higashiyama-ku never appear.

**Make them pass (GREEN):** nothing to write here. Task 13's type list makes them pass. If one still fails after Task 13, treat it as a bug: add a smaller test that reproduces it in the right task's file, fix it there, and keep this test as the guard.

**Acceptance criteria:**
- [ ] Every test above failed before Task 13 and passes after it.
- [ ] The fixtures are trimmed (about 2 MB in total at most) and contain no secrets.

**Verification:**
- [ ] RED seen, run against the empty type list
- [ ] GREEN: `CGO_ENABLED=1 go test -race -count=1 -run Cities ./planner/`, with the network off
- [ ] Full suite: `make test`, `make lint`, `make build`

**Dependencies:** Tasks 3, 4, 5, 14. Turns green with Task 13.

**Files likely touched:** `planner/cities_test.go`, `planner/testdata/cities/`

**Estimated scope:** Medium

---

## Checkpoint C: End to end (after Tasks 13–16)

- [ ] `make test` passes, including the city acceptance tests, with the network off
- [ ] Browser check (large, by hand) of every "Done when" item from the one-pager:
  - [ ] "Lisbon, 3 days, next week" gives a sensible plan, and the rainy day gets the museums
  - [ ] "Tavira, 5 days" gives a shorter plan with a clear message
  - [ ] "Kyoto, 3 days" with a rainy day gets 3 or more indoor stops that day
  - [ ] A cached city loads in under 2 seconds
  - [ ] No API keys that can bill are used
- [ ] **Human review** before the docs task

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
- [ ] The README has no mention of Google Places, Foursquare or the itinerary endpoint.
- [ ] The "How the planner works" section fits on one screen and has a diagram.
- [ ] The CHANGELOG `[Unreleased]` section lists the planner (Added) and the old hotels and itinerary code (Removed).

**Verification:**
- [ ] By hand: the README renders correctly on GitHub, including the diagram
- [ ] No test run is needed for a docs-only change

**Dependencies:** Task 15 (and 16 for the Tests section)

**Files likely touched:** `README.md`, `CHANGELOG.md`

**Estimated scope:** Small

---

## Checkpoint D: Complete

- [ ] All acceptance criteria above are met, and no test is skipped or disabled
- [ ] CI is green on `main`
- [ ] Human approves the deploy (`fly deploy`)
