# Exports carry `include` (follow-up to 7e): TDD Evidence

**Source:** owner decision "exports only", after `docs/tdd/7e.tdd.md` (known gap: .ics/.kml re-planned without `include`). Files: `internal/adapters/httpserver/trip.go` (`exportURLs`, `resolveTripPlan`), `export_urls_test.go`.

## User journeys

- As a traveler who planned with saved places, I want the .ics/.kml to match the plan on screen, so that my calendar and map hold the same stops.
- As a traveler who saved nothing, or with the writers' map off, I want exports unchanged.
- As a visitor, I want an unknown QID in an export link ignored.

## Behaviour

- `exportURLs` appends `&include=QID` for each place in `req.Include` (only QIDs that resolved on `/plan`).
- `resolveTripPlan` sets `req.Include = s.includedPlaces(...)` (writers' map off: none; unknown QIDs skipped; cap 12).
- Not changed (owner: exports only): `HX-Push-Url`, and `/trip/:slug` page re-plan. The page uses `tripCardForSlug`, not `resolveTripPlan`, so it never reads `include`; locked by a test.
- Booking link is not in the .ics/.kml output, so `travelers` is not carried.

## RED and GREEN

RED: `go test ./internal/adapters/httpserver/ -run 'ExportURLs|ExportLinks|PushURLStays|WithInclude|WithoutInclude|IgnoresInclude'` gave
`TestExportURLs_CarryOnlyTheResolvedIncludes`: expected `.../lisbon-pt.ics?days=3&from=2026-03-10&include=Q1&include=Q2`, actual `.../lisbon-pt.ics?days=3&from=2026-03-10`;
`TestPlanRoute_ExportLinksCarryTheResolvedIncludes` and `TestICSAndKML_WithIncludeContainTheSavedPlaces` also failed. The unchanged-behaviour tests passed already (regression locks).

GREEN: same command, all `--- PASS`; `go test -cover ./internal/adapters/httpserver/` ok.

## Guarantees

| What is guaranteed | Test name | Type | Result |
| --- | --- | --- | --- |
| Export URLs repeat `include=` for each resolved place | `TestExportURLs_CarryOnlyTheResolvedIncludes` | unit | PASS |
| No include: export URLs identical to before | `TestExportURLs_WithoutIncludeAreUnchanged` | unit | PASS |
| `/plan` links carry only the resolved QIDs (unknown dropped) | `TestPlanRoute_ExportLinksCarryTheResolvedIncludes` | integration | PASS |
| `HX-Push-Url` has no `include` | `TestPlanRoute_PushURLStaysWithoutInclude` | integration | PASS |
| .ics and .kml with `include` contain the saved places | `TestICSAndKML_WithIncludeContainTheSavedPlaces` | integration (fake source) | PASS |
| .ics without `include` is byte-for-byte the same as with no source wired | `TestICS_WithoutIncludeIsUnchangedByTheSavedPlaces` | integration | PASS |
| Writers' map off: export ignores `include` | `TestICS_IgnoresIncludeWhenTheWritersMapIsOff` | integration | PASS |
| `/trip/:slug` page ignores `include` | `TestTripPage_IgnoresInclude` | integration | PASS |

## Coverage

`go test -cover ./internal/adapters/httpserver/`: 92.4%.

## Known gaps

- Reloading `/trip/:slug` still re-plans without saved places (by design).
- Exports re-plan with the fake-free real planner; same-query determinism relies on the planner, not tested here.
