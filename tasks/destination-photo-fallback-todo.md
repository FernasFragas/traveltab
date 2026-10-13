# Destination photo fallbacks — Tasks

Source of truth for progress. Read [the plan](destination-photo-fallback-plan.md) first.

**Status (2026-10-13):** D1–D5 done. The 59-city live retry passed; the first run had a transient Wikidata busy error for Kyoto. See the TDD report for the app check and upstream timeout observations.

**How to read this file**
- Tasks run in order D1 → D5; D2–D4 all edit `destinationphoto.go` and must not run concurrently.
- **Owns** = the only files the task may edit. None overlap with the open nearby-rings or Writers' Map tasks.
- Start from an up-to-date `main` (`git fetch` and check `git status -sb`).
- TDD per [AGENTS.md](../AGENTS.md) §5: see each test fail for the intended reason first.
- Live calls only in D1, D4 and D5, with the app's `UserAgent`, never personal details. `make test` stays offline.
- No commits or pushes. The owner commits.

**User journeys**
- *As a traveller searching New York, Washington, Quebec City, Toronto or Amsterdam, I want to see a photo of the city, so that the page looks like the destination.*
- *As a traveller searching a small town that shares a name with a famous city, I want never to see the famous city's photo, so that the page doesn't mislead me.*

---

### D1. Record the baseline and the fallback search
- [x] Done (2026-10-13): 59-city baseline, fallback search probes, and SPARQL timeout recorded.

**Owns:** `docs/tdd/destination-photo-fallback.tdd.md`

- **Already recorded:** the SPARQL query timed out after 8–12 s for New York, so it was dropped.
- Run the current lookup on the 59 **Baseline cities** in the plan (temporary test, deleted afterwards). Record the result, the Commons file name and the time for each. Amsterdam now has a photo, from D2.
- Run the D3 search (`haswbstatement:P1082 nearcoord:25km,<lat>,<lon>`, `srlimit=5`) for the 4 remaining misses. Record its time, the 5 results, and what the pick rules choose.

**Verify:** the evidence doc has the 59-city table with file names, and the expected picks Q60, Q61, Q2145 and Q172.

### D2. Accept deprecated-only country codes (fix 2)
- [x] Done (2026-10-13): `TestDestinationPhoto_DeprecatedCountryCodeOnlyWhenNoCurrentCode`, `TestDestinationPhoto_AmsterdamAcceptsDeprecatedOnlyNLCode`; live Amsterdam → Q727 in 1.7 s; `make test lint build` passed.

**Owns:** `internal/adapters/api/destinationphoto.go` (`countryCode` only), `destinationphoto_test.go`

- `countryCode`: when no preferred or normal `P297` exists, use a deprecated one.
- The existing "unknown country" and "no country" cases are still refused.

**Verify:** `go test ./internal/adapters/api/ -run 'DestinationPhoto'`

### D3. Nearby populated places fallback (fix 1)
- [x] Done (2026-10-13): fake-backed RED/GREEN tests and the Wikidata populated-place fallback.

**Owns:** `internal/adapters/api/destinationphoto.go`, `destinationphoto_test.go`, `wikimedia.go` (alias response field), `internal/adapters/api/testdata/wikimedia/` (new fixtures if needed)

- New step in `Photo`, after `nearbyCities` finds nothing: `populatedNear(ctx, id)` calls `action=query&list=search` on `wikidataAPIURL` with `srsearch=haswbstatement:P1082 nearcoord:25km,<lat>,<lon>` and `srlimit=cityCandidateLimit`.
- Pass the results to `matchCity`, so the settlement, 25 km and country checks apply.
- Pick: a passing result whose label or alias folds to the city name first, otherwise `chooseCity` over all passing results.
  - `matchCity` loads only `claims|sitelinks|descriptions` today. Load `labels|aliases` too, for this path only, so the other lookups' requests are unchanged.
- Extend the `photoWorld` fake to answer `list=search`.
- Tests (RED first):
  - New York: the name search finds only the state → the fallback picks Q60 by its alias;
  - Old Toronto: no name match → `chooseCity` picks the most-linked settlement;
  - two comparable settlements more than 10 km apart, no name match → no photo;
  - a result in another country → refused;
  - the fallback doesn't run when the name search or `nearbyCities` matched;
  - the search returns an error → error, not "no photo";
  - the request has the expected `srsearch` and `srlimit`.

**Verify:** `go test ./internal/adapters/api/ -run 'DestinationPhoto'`

### D4. Regression guards (fix 3)
- [x] Done (2026-10-13): all 69 live identities passed on retry, each under 4 s.

**Owns:** `internal/adapters/api/destinationphoto_test.go` (`liveDestinations`)

- Add the 59 **Baseline cities** from the plan to `liveDestinations`, with their OpenWeather names and coordinates.

**Verify:** `LIVE_API_TESTS=1 go test ./internal/adapters/api/ -run TestDestinationPhotoLive` passes for all 59, each under 4 s.

### D5. Check it end to end and write the evidence
- [x] Done (2026-10-13): five app pages showed city photos and credits; coverage and full gate passed.

**Owns:** `docs/tdd/destination-photo-fallback.tdd.md`

- Run the app with a temp copy of the database (`DB_PATH=…`, `make run`) and search New York, Washington, Quebec City, Toronto and Amsterdam: the hero shows a photo of the city, with its credit.
- Compare the 55 D1 baseline photo files (including Amsterdam) with D4 (same files, or differences explained).
- Finish the TDD report: journeys, RED/GREEN excerpts, guarantees table, coverage.

**Verify:** `go test -cover ./internal/adapters/api/` ≥ 80%; `make test lint build` passes.

### D6. Run the three searches at once
- [x] Done (2026-10-13). **Open:** 2 of 69 live lookups (Montreal, Vatican City) still hit the 4 s limit, on the country-code request; see the TDD report, D6.

**Owns:** `internal/adapters/api/destinationphoto.go`, `destinationphoto_test.go`, `docs/tdd/destination-photo-fallback.tdd.md`

- **Why:** New York needed 7 requests one after another (3.3–6.3 s live) and failed with `context deadline exceeded` on the Commons request.
- `Photo` runs the name search, the nearby-article search and the populated-place search concurrently, then loads every candidate in one `wbgetentities` request.
- The order of preference is unchanged: name search, then nearby articles, then populated places. A search's error is returned only when no earlier search found the city.
- Each cold lookup now always makes all three searches (two more small requests when the name search matches).

**Verify:** `go test -race ./internal/adapters/api/ -run DestinationPhoto`; New York under 4 s live; `make test lint build` passes.
