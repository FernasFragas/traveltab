# 1f — Funchal Contract Fields: TDD Evidence

**Source task:** [`tasks/todo.md`](../../tasks/todo.md) 1f; target output in [`tasks/plan.md`](../../tasks/plan.md). The owner approved the contract change on October 10, 2026.

## User journeys

- As a visitor, I want known opening hours and a Wikipedia link in the place panel, so that I can decide whether to visit and read more.
- As a visitor, I want missing place details omitted, so that the panel does not show empty rows.
- As a map developer, I want the fixture and fake source to expose both present and missing details, so that the panel can be built offline.
- As a map developer, I want writer itinerary days to carry only ordered stops, so that the removed walk card fields do not reappear.
- As a traveler, I want each plan stop's distance from the previous stop, so that itinerary rows can show how far apart stops are. This remains a contract definition until 7g builds the rows.

## RED and GREEN

### Optional place details

`go test ./internal/writermap/... -run 'TestPlace_EncodesOpeningHoursAndWikipediaURL|TestItinerary_DayEncodesOnlyStops|TestFakeSourcePlace_ReturnsOptionalPanelDetails|TestMadeiraFixture_ItineraryDaysOnlyCarryStops'`

- **RED, before production changes:** `unknown field OpeningHours in struct literal of type Place`; `unknown field WikipediaURL in struct literal of type Place`; `withHours.OpeningHours undefined`.
- **RED, after the type change but before updating the fixture:** `expected: "Open daily · 9:00 – 18:00"`, `actual: ""` in `TestFakeSourcePlace_ReturnsOptionalPanelDetails`.
- **GREEN**, with the fixture and tests complete, from
  `go test ./internal/writermap/... -run 'TestPlace_EncodesOpeningHoursAndWikipediaURL|TestPlace_OmitsUnknownOpeningHoursAndWikipediaURL|TestItinerary_DayEncodesOnlyStops|TestFakeSourcePlace_ReturnsOptionalPanelDetails|TestMadeiraFixture_ItineraryDaysOnlyCarryStops'`:

  ```text
  ok  	weatherservice/internal/writermap	0.469s
  ok  	weatherservice/internal/writermap/writermaptest	0.492s
  ```

### Removed itinerary-day fields

The same RED run failed `TestMadeiraFixture_ItineraryDaysOnlyCarryStops` because an itinerary day still had `distance_km` and `walk_minutes`: `should have 1 item(s), but has 3`. The GREEN run above passed after removing those fields from the type and fixture. `TestItinerary_DayEncodesOnlyStops` checks the serialized shape.

## Guarantees

| What is guaranteed | Test name | Type | Result |
| --- | --- | --- | --- |
| Known hours and Wikipedia URL encode under the contract keys | `TestPlace_EncodesOpeningHoursAndWikipediaURL` | unit | PASS |
| Unknown hours and URL are omitted | `TestPlace_OmitsUnknownOpeningHoursAndWikipediaURL` | unit | PASS |
| Fake place lookup returns present and absent details | `TestFakeSourcePlace_ReturnsOptionalPanelDetails` | unit | PASS |
| Writer itinerary days serialize only ordered QIDs | `TestItinerary_DayEncodesOnlyStops` | unit | PASS |
| Fixture days contain no retired fields | `TestMadeiraFixture_ItineraryDaysOnlyCarryStops` | unit | PASS |

## Coverage and full check

`go test -cover ./internal/writermap/...`:

```text
ok  	weatherservice/internal/writermap	0.322s	coverage: [no statements]
ok  	weatherservice/internal/writermap/writermaptest	0.494s	coverage: 98.3% of statements
```

The installed `golangci-lint` 2.4.0 was built with Go 1.26 and could not analyze this Go 1.27.2 module. A temporary compatible binary in `/private/tmp/traveltab-tools` completed the required gate. `PATH=/private/tmp/traveltab-tools:$PATH make test lint build` passed: all test packages reported `ok`, lint reported `0 issues.`, and `go build -o bin/traveltab ./cmd/web` completed.

## Known gaps

- GeoJSON properties and plan-stop `distance_km` are contract-only; their builders do not exist yet. PR 3 and 7g must test those outputs when implemented.
- The fixture's opening hours are illustrative; PR 4a will read real hours from Wikidata when present.
