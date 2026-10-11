# PR 4 — Application package coverage

Source: [tasks/todo.md](../../tasks/todo.md), PR 4 base-place port change. The port change put `internal/application` in scope for the required 80% package coverage gate. These tests cover existing behavior; production code was not changed.

## Journeys

- As a traveler searching for a city, I want weather and wave data combined for that city, with provider failures returned to the caller.
- As a traveler selecting a namesake city, I want its chosen coordinates passed to the weather provider.
- As a traveler opening videos, I want the fetched list or a provider error returned.
- As a traveler requesting an invalid trip length, I want a validation error before any source lookup.

## Red and green

Before the tests, `go test -coverprofile=/private/tmp/traveltab-application.cover ./internal/application/...` returned `coverage: 44.2% of statements`, below the 80% gate. `go tool cover -func=/private/tmp/traveltab-application.cover` showed the functions below at `0.0%`.

| Behavior | Command run | RED output excerpt | GREEN output excerpt |
| --- | --- | --- | --- |
| Weather and wave merge; provider errors | `go test -coverprofile=/private/tmp/traveltab-application-final.cover ./internal/application/...` | `GenerateReport 0.0%` / `finishWeatherReport 0.0%` | `GenerateReport 100.0%` / `finishWeatherReport 100.0%` |
| Chosen coordinates and unsupported provider | `go test -coverprofile=/private/tmp/traveltab-application-final.cover ./internal/application/...` | `GenerateReportForPlace 0.0%` | `GenerateReportForPlace 100.0%` |
| Video list and provider error | `go test -coverprofile=/private/tmp/traveltab-application-final.cover ./internal/application/...` | `NewVideoStreamReporters 0.0%` | `NewVideoStreamReporters 100.0%` |
| Invalid trip length | `go test -coverprofile=/private/tmp/traveltab-application-final.cover ./internal/application/...` | `Plan 0.0%` | `Plan 100.0%` |

The GREEN coverage command returned `ok weatherservice/internal/application 0.244s coverage: 100.0% of statements`. `go test ./internal/application/ -run 'TestWeatherReporters_|TestVideoStreamReporters_|TestTripPlanner_' -count=1` returned `ok weatherservice/internal/application 0.309s`.

## Guarantees

| What is guaranteed | Test name | Level | Result |
| --- | --- | --- | --- |
| Weather fields, lowercase country, wave height, coordinate query and Waze URL are combined | `TestWeatherReporters_GenerateReportMergesWeatherAndWaves` | Unit | PASS |
| Weather and wave errors are returned without a partial report | `TestWeatherReporters_PropagatesSourceErrors` | Unit | PASS |
| A chosen place reaches the coordinate-aware provider and its wave lookup | `TestWeatherReporters_SelectedCoordinates` | Unit | PASS |
| Unsupported coordinate selection and selected-provider errors are returned | `TestWeatherReporters_SelectedCoordinatesErrors` | Unit | PASS |
| Fetched videos are returned and provider errors are propagated | `TestVideoStreamReporters_GenerateReport` | Unit | PASS |
| Invalid trip length is rejected by the wrapped planner | `TestTripPlanner_RejectsInvalidDays` | Unit | PASS |

`go test -cover ./internal/application/...` returned `coverage: 100.0% of statements`. `go test -race ./internal/application/...` returned `ok weatherservice/internal/application 1.562s`. No live sources were used.
