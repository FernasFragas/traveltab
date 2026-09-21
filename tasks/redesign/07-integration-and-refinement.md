# Task 07 — Integration and reference fidelity refinement

Status: pending. Starts after tasks 01–05 hand off and stop editing; uses the fixture preview from task 06.

## Ownership

This task may edit the production files delivered by 01–05 after the ownership barrier. It owns reconciliation of affected existing tests, optional deployment asset-copy adjustments if required, and `tasks/redesign/handoff-07.md`. Task 06 remains the owner of verification scripts/evidence and reports failures for this task to fix.

## Deliverables

- [ ] Read every handoff and reconcile the frozen contracts before broad visual changes. Confirm all CSS/JS assets are included exactly once and every partial receives the promised data.
- [ ] Verify all render paths: initial index, searched fragment, unplanned shared city, planned shared city, successful plan response, invalid plan response, and failed plan response.
- [ ] Verify correct HTMX OOB processing using the app's installed version. Form, itinerary, and stays must update together without losing overview/map state or duplicating sections. Error swaps must not preserve a stale successful plan.
- [ ] Resolve DOM/focus issues across search, section navigation, accordion changes, videos, and browser history. Keep top/mobile navigation synchronized and correctly offset.
- [ ] Compare fixture screenshots against the reference by section. Prioritize composition, photograph crop, type hierarchy, density, color, and card proportions before decorative flourishes.
- [ ] Align hero/map/form heights, section spacing, button size, hairline borders, accordion density, number badges, chips, accommodation rows, and video thumbnails. Preserve readable touch targets instead of copying the tiny type of the scaled poster.
- [ ] Check the light theme has fully replaced old gray/blue glass surfaces and no stale inline CSS or duplicated icon dependencies remain.
- [ ] Reconcile existing render tests for intentional markup changes while preserving assertions about real stops, attribution, labels, fallback states, exports, and stays. Add behavioral regressions only where the new interaction contract needs coverage.
- [ ] Verify new local images/manifests are available in the current Docker packaging; update asset inclusion only if needed. Do not deploy.
- [ ] Respond to final verification findings from task 06 and recheck only affected behavior plus required suite checks.

## Handoff

Record the final appearance and behavior, test outcomes, and reference differences that remain because real data/providers differ: map tiles, unavailable hotel photos, missing forecast temperatures, missing visit times, and absent video durations. These should be deliberate, readable substitutions, not silent omissions. Link final screenshots and the verification report. No account, saved-trip, guide, or unimplemented legal navigation should be present as a fake control.
