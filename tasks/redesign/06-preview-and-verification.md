# Task 06 — Fixture preview and final verification

Status: pending. Prepare the preview in parallel with tasks 01–05; final verification follows task 07.

## Owned files

- New `cmd/design-preview/main.go` and fixture support files in that directory
- New `tasks/redesign/preview.md`
- New browser-check scripts under `tasks/redesign/checks/`
- New screenshots/results under `tasks/artifacts/redesign/`
- `tasks/redesign/handoff-06.md`

Do not modify `cmd/loadtest-server`, production templates, or other agents' tests. Read the [plan](../../docs/redesign-plan.md) and [contracts](contracts.md).

## Prepare immediately

- [ ] Create a local fixture server using the actual HTTP server, templates, and `SetTripPlanner` interface. Use temporary storage; never write the production SQLite database or depend on `.env` secrets.
- [ ] Supply deterministic Lisbon overview data, four distinct itinerary stops with/without photos, three days with mixed forecast availability, accommodation variants, and multiple videos. Add a second destination to catch stale-city content. Document how fixture dates remain valid as time passes.
- [ ] Support normal, missing-photo, missing-forecast, empty-video, unavailable-stay, and planner-failure cases. Include one-day and five-day plans, long names, and slow responses.
- [ ] Document a single preview command, fixture URLs/scenario selection, and any browser tooling prerequisites. Reuse available browser tooling; do not assume Playwright is installed.
- [ ] Keep fixture-only map placeholders unmistakable. Deterministic local fixtures must not be described as verification of the live map, external photography, or YouTube playback.

## Final checks after task 07

- [ ] Run the relevant HTTP rendering tests and `go test ./...`. Coordinate required fixes through 07; do not weaken tests just to make the redesign pass.
- [ ] Capture initial, planned, and fallback screenshots at 375×812, 768×1024, and 1440×1000. Include a roughly 1100px-wide capture to compare the central reference panel's proportions. Save full-page and initial-viewport images where useful.
- [ ] Compare the actual desktop page to the central desktop panel and actual mobile page to the phone screen, excluding poster margins and phone hardware. Record visual discrepancies by section, not a misleading full-poster pixel score.
- [ ] Verify no document-level horizontal overflow; readable type; weather/hero density; compact stays/videos; mobile order; fixed navigation clearance; long labels; 200% zoom; visible focus; contrast; and reduced motion.
- [ ] Exercise search → plan → expand another day → regenerate → search another city → shared trip URL → back/forward. Verify correct city/results, preserved controls, first-day-open behavior, unique IDs, and no duplicate handlers.
- [ ] Check 400 and 500 plan responses, search failure, cleared stale results, loading/announcement behavior, OOB swaps, calendar/KML downloads, day directions, and booking dates.
- [ ] Check real external map and video behavior separately when reachable; record network-blocked features explicitly. Ensure placeholder checks do not count as live-provider passes.
- [ ] Write a concise result matrix and reference-comparison notes, with exact commands and artifact paths, in `handoff-06.md`.

Exit criteria: evidence supports all design-plan acceptance criteria, or clearly names unresolved blockers for the integration owner. Do not mark the redesign complete with unexplained visual or interaction failures.
