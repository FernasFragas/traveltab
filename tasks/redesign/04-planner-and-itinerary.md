# Task 04 — Trip form, expandable itinerary, and HTMX behavior

Status: pending. Can start against frozen contracts. Task 02 owns handler/view changes; task 05 supplies the stay body used in the response.

## Owned files

- `views/trip_card.go.tpl`
- New `views/itinerary_body.go.tpl`, `views/plan_response.go.tpl`
- New `public/redesign/planner.css`, `public/redesign/planner.js`
- `tasks/redesign/handoff-04.md`

Read the [plan](../../docs/redesign-plan.md) and [contracts](contracts.md).

## Deliverables

- [ ] Make `trip_card` the cream overview form panel only. Match the reference's serif Plan your trip heading, brief helper text, compact date/duration fields, and full-width terracotta Generate plan button with arrow.
- [ ] Keep native start-date input and 1–5 day selector. Display the derived end date as read-only within the date group, not a second independent input. Use `.EndLabel` initially; update after input changes without timezone-induced date shifts.
- [ ] Keep hidden city/country/coordinate fields, spinner, errors, validation values, and current request parameters. Announce loading without shifting the card; prevent duplicate submission while a request is pending.
- [ ] Build `itinerary_body` with Your itinerary heading, brief subtitle, Calendar export and Map export actions, and a single lightly bordered accordion group. Render meaningful pre-plan/unavailable/error states for nil or absent plans.
- [ ] Use native independent `details` rows. Open the first returned day initially. Summary: small leading day marker, bold day number, date, weather/rain status, walking icon/distance, trailing expanded/collapsed indicator. Mobile summary can wrap into two lines.
- [ ] Expanded desktop row: up to four photo cards plus a narrow walking-directions tile. Number each actual stop starting at 1; include name, optional photo, credit, and indoor/outdoor chip. Retain all stops without horizontal page overflow.
- [ ] Mobile: compact image-left stop rows, all stops visible when the day is open, logical numbering, directions below. The reference's single sample row does not justify hiding other stops.
- [ ] Use rainfall/status in the weather slot; do not insert fictional day temperatures or sunny labels. Omit visit time ranges because the planner does not supply them.
- [ ] Create `plan_response` with the form root plus OOB itinerary and stays sections exactly as contracted. Keep the error response swap logic in the deferred script, installed once.
- [ ] After successful updates, reset to first-day-open and move focus/scroll to the itinerary heading without covering it with navigation. On failure preserve submitted values, show the form error, and ensure old results are no longer presented as the failed plan. Coordinate event ordering and error clearing with 02/07.

## Verification and handoff

Check native keyboard toggling, independent open states, first-day-open on initial shared pages and regenerated plans, numbering, long stop names, no photos, 1/3/5-day results, uncertainty/no forecast, 400/500 responses, OOB section replacement, and repeat requests without duplicate listeners. Task 02 adds response regression tests; task 06 exercises actual browser swaps. Report results and untested dependencies in `handoff-04.md`.
