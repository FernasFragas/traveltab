# PR 4 API package coverage

Source: PR 4 package coverage gate in `AGENTS.md`. This is a test-only change for existing API clients; no production behavior changed, so the RED/GREEN implementation cycle does not apply.

## Journeys

- As a traveler requesting hotels, I want Amadeus and Markcorps responses mapped to hotel names and IDs or URLs, with lookup failures reported.
- As a traveler requesting sea conditions, I want the latest StormGlass wave height and station location, with provider failures reported.

## Evidence

Before adding tests, `go test -coverprofile=/tmp/traveltab-api-coverage.out ./internal/adapters/api/...` reported:

```text
ok  	weatherservice/internal/adapters/api	1.265s	coverage: 68.4% of statements
```

The first post-test run reached 80.8% but failed because the new Amadeus stub consumed the request body before checking a form field. That test assertion was corrected; production code was unchanged. The passing run of the same command reported:

```text
ok  	weatherservice/internal/adapters/api	1.312s	coverage: 80.8% of statements
```

`go test -race ./internal/adapters/api/...` passed:

```text
ok  	weatherservice/internal/adapters/api	3.121s
```

| What is guaranteed | Test name | Level | Result |
| --- | --- | --- | --- |
| Amadeus sends client credentials, applies its token, maps hotels, and caps the result at ten | `TestAmadeus_FetchesTokenAndMapsTheFirstTenHotels` | offline HTTP transport | PASS |
| Amadeus reports invalid coordinates and provider failures | `TestAmadeus_ReportsInvalidCoordinatesAndTransportErrors` | offline HTTP transport | PASS |
| StormGlass returns the latest wave height and a station's location | `TestStormGlass_ReturnsLatestWaveAndStation` | offline HTTP transport | PASS |
| StormGlass handles empty waves and reports provider failures | `TestStormGlass_EmptyWaveListAndProviderErrors` | offline HTTP transport | PASS |
| Markcorps resolves the requested city and maps hotel fields | `TestMarkcorps_ResolvesCityAndMapsHotels` | offline HTTP transport | PASS |
| Markcorps reports lookup and hotel provider failures | `TestMarkcorps_ReportsLocationAndHotelFailures` | offline HTTP transport | PASS |

The package still has older untested clients, including Geoapify, OpenMateo, and YouTube. The package coverage requirement is met; these clients were outside the minimum scope of this gate.
