# Task 02 — Presentation data, imagery, and handler wiring

Status: pending. Can start immediately; renderer verification awaits component templates. Read the [plan](../../docs/redesign-plan.md) and [contracts](contracts.md).

## Owned files

- `internal/adapters/httpserver/server.go`, `internal/adapters/httpserver/trip.go`
- New `internal/adapters/httpserver/presentation.go` and `presentation_test.go`
- New `internal/adapters/httpserver/redesign_response_test.go`
- New image manifest/resource files under `internal/adapters/httpserver/presentation_assets/`
- New local assets under `public/images/destinations/` and `public/images/stays/`
- `tasks/redesign/handoff-02.md`

Other existing HTTP test files belong to integration during final reconciliation. Coordinate requests instead of editing those tests concurrently.

## Deliverables

- [ ] Supply `.Presentation`, `.EndLabel`, and `.StayCards` exactly as contracted on initial loads, destination searches, shared trip pages, and plan fragments where applicable.
- [ ] Keep enrichment in HTTP presentation code; preserve planner/provider domain types and cached JSON compatibility. Recompute optional decoration when rendering, rather than requiring a database migration.
- [ ] Add a small local manifest keyed by city/country for destination hero imagery and editorial copy. Obtain a warm Lisbon skyline/Alfama photograph with a dome and river where possible. Record source, license, credit link, dimensions, and descriptive alt text. Use a path strategy that also works in the current Docker deployment; an embedded manifest is suitable.
- [ ] Support optional accommodation photos keyed by destination and verified accommodation identity. Source an initial photo only if a real matching property and usable image are available; otherwise keep the photo absent and document the resulting fidelity limit. Do not use generic interiors as though they represent a listed property.
- [ ] Populate video thumbnail/watch URLs from validated video IDs; preserve safe escaping. Leave duration absent because the current source does not return it.
- [ ] Compute a safe external city map link. Preserve existing iframe and day walking links.
- [ ] Compute inclusive itinerary end labels from the existing start/day parameters. Do not change booking checkout or calendar export semantics.
- [ ] Wire every `/plan` response branch to `plan_response` and preserve HTTP status, `HX-Push-Url`, validation, and shared-link behavior. Coordinate with task 04 on the actual OOB browser response.
- [ ] Ensure destination-search failures produce an intelligible error without blanking the last valid view; coordinate the header error region with task 01. Avoid unrelated cache/provider refactoring.

## Verification and handoff

Test known/unknown city image lookup, no wrong-city fallback, identity-safe stay photo lookup, date boundaries including a one-day trip/month rollover, empty videos, and consistent view enrichment across render paths. Once templates land, test the three plan response roots, status/header preservation, and absent-data rendering. Use `go test ./internal/adapters/httpserver` and report pending integration failures accurately. Record asset provenance and any intentionally missing data in `handoff-02.md`.
