# Clickable day-plan pins: TDD evidence

Source: `tasks/dayplan-pins-plan.md` and `tasks/dayplan-pins-todo.md` P1–P5. This work is a prerequisite for `tasks/map-first-plan.md` M6.

## Journeys

- As a traveller, I want to open a numbered stop on the map and read its details.
- As a traveller, I want Details to open the place panel for a stop known to the map.
- As a keyboard user, I want Escape to close the popup and return focus to its pin.

## RED and GREEN

- Stop data: `go test ./internal/adapters/httpserver/ -run TestDayPlan_EmitsPopupFieldsEscaped -count=1` failed with `--- FAIL: TestDayPlan_EmitsPopupFieldsEscaped` and `does not contain "data-qid=\"\" data-kind=\"\" data-description=\"\""`. After the template edit, the same command returned `ok  weatherservice/internal/adapters/httpserver`.
- Popup behavior: `node --test tasks/redesign/checks/map-dayplan.test.mjs` failed 3 of 8 tests: the stop was missing `qid`/`kind`/`description`/`distance`, its pin had no `aria-label`, and `onClick` was absent. After the JavaScript edit, it reported `pass 8`, `fail 0`.
- Tab return: the same command failed with `actual: '#itinerary'`, `expected: '#overview'` for View on map. After the hash change it reported `pass 8`, `fail 0`.
- Coverage: `go test -cover ./internal/adapters/httpserver/... ./cmd/design-preview/...` returned `coverage: 93.2% of statements` for `internal/adapters/httpserver` and `coverage: 87.0% of statements` for `cmd/design-preview`.

| Guaranteed behavior | Test name | Level | Result |
| --- | --- | --- | --- |
| Escaped QID, kind and description in stop markup | `TestDayPlan_EmitsPopupFieldsEscaped` | Integration | Pass |
| Stop metadata reaches the map layer | `registers one layer named plan and reports valid stops after swaps` | Unit | Pass |
| Pin opens stop popup; Escape restores focus | `day pin opens a popup with details and Escape returns focus` | Unit | Pass |
| Details selects a known place; unknown stops omit it; redraw closes popup | `Details selects known place; unknown place has no Details; redraw clears popup` | Unit | Pass |
| View on map returns to Overview | `View on map activates the plan chip and scrolls to the shell` | Unit | Pass |

Browser runtime check: `CDP_PORT=9228 node tasks/redesign/checks/map-dayplan.browser.mjs`
passed at 375 and 1440 px with both MapLibre and Leaflet. It exercised numbered pins,
the stop popup and Escape, Details, View on map, and clearing a plan; the console was clean.
