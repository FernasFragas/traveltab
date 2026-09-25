# September 2026 repair verification

Date: 2026-09-24. Tested base commit: `a737cecd6cd80042e5237644720a2f8d25fe4e9a`, with the working-tree changes described below. This records the completed repair and redesign-verification milestone. The original redesign task specs, handoffs 01–03 and the 01–03 milestone evidence were removed after completion and remain in git history (commit `c496423`). The design spec and contracts are consolidated in [docs/design.md](../docs/design.md). The open work is in [maintenance follow-ups](../docs/maintenance.md#next-up).

## Completed changes

- HTTP cache failures and malformed JSON now return errors and zero view data, allowing destination and shared-trip routes to fetch fresh data. Focused regressions cover full-page, HTMX, and shared routes, including valid cache hits and old cache compatibility. The original code failed 14 of the new regression cases in an isolated copy.
- The deterministic design preview now uses an explicit cache miss, `pt` country codes, correctly matched video queries, scenario variants, and a visibly styled local map fixture. The HTTP smoke suite checks its real routes, OOB response roots, and ICS/KML exports.
- Browser fixes reset the planner form after HTMX history restoration and use the correct `data-video-id` dataset property. The full [interaction regression run](artifacts/redesign/final-verification/interaction-results.md) passes in Brave with HTMX 1.9.11; negative controls prove each original defect is detected.
- Documentation now points here for verification status.

## Checks and evidence

| Check | Result |
| --- | --- |
| `go test -race -count=1 ./...` | Passed on the repaired working tree |
| `make lint` and `make build` | Passed after running linter outside a sandbox that blocks package loading |
| Preview `go test -race -count=1 ./cmd/design-preview` | Passed, including route tests and all 12 named scenarios |
| Default preview HTTP smoke | Passed on localhost with sandbox escalation |
| `docker build -t traveltab .` | Passed after the CSS refinement; a disposable container confirmed web binary, templates, planner script, and Lisbon hero asset on the earlier build |
| JavaScript syntax, whitespace and Markdown links | Passed (`node --check`, `git diff --check`, local-link check) |
| Browser regression suite | Passed in Brave/HTMX 1.9.11, including original-code negative controls; see linked evidence |
| Four-width browser acceptance | Passed for default and fallback at 375×812, 768×1024, 1100×1000 and 1440×1000. Slow/network/focus, 400/500, shared reload, exports and booking dates passed in the default run; see the result matrix |

## Browser and external evidence

The [interaction regression evidence](artifacts/redesign/final-verification/interaction-results.md) records the passing and deliberately failing controls. Four-width screenshots, a requirement matrix, and a section-by-section comparison with the reference are in the [final browser evidence](artifacts/redesign/final-verification/README.md). A CSS refinement puts the desktop day date beside its number, matching the reference hierarchy. [External reachability checks](artifacts/redesign/final-verification/external-results.md) are separate from fixture evidence. Waze, a sample Wikimedia image, and the YouTube privacy embed responded with HTTP 200; that does not verify browser map rendering or playback. Booking returned a robot challenge, leaving landing-page dates and search results unverified.

Spot checks of design token color pairs give 15.63:1 for main ink on the warm surface, 6.06:1 for muted text, and 6.11:1 for terracotta. The later [full audit](artifacts/redesign/final-verification-2/README.md) measured 5,465 pixel-sampled pairs across 41 states, including image overlays, with no text below 4.5:1.

## Remaining limits

- Fixture maps, photos, videos, and booking links do not establish the behavior of external services. The separate checks linked above verify endpoint reachability only; Booking results and browser playback remain unverified.
- The full contrast/accessibility audit and browser rendering (with assertions) for every named preview scenario were completed offline on 2026-09-25; see the [final-verification-2 matrix](artifacts/redesign/final-verification-2/README.md). It fixed 11 defects and documents one focus-order limitation at phone width. Actual calendar/map application imports remain open.
- The maintenance backlog (sitemap storage, reviewed city guides, slug helper consolidation, importer interoperability and deployment indexing) is tracked in [maintenance notes](../docs/maintenance.md#next-up).

No commit, push, deployment, or production-database change is part of this repair run.
