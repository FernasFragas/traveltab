# M5: writer itineraries on both map engines

Source: `tasks/map-3d-todo.md` M5 and `tasks/map-3d-plan.md` drawing API. User journeys:

- As a traveller, I want a writer's route shown as an ordered line and numbered stops, so I can follow it on either map engine.
- As a traveller, I want to switch or hide routes without duplicates, so the map stays readable after redraws and style reloads.
- As a traveller, I want a stop to show the writer and a safe link to the source post, so I can read the route details.

## RED and GREEN

Command for unit behaviour: `node --test tasks/redesign/checks/map-itineraries.test.mjs`.

| Behaviour | RED excerpt | GREEN excerpt |
| --- | --- | --- |
| Ordered stops and line through `context.draw` | `TypeError: Cannot read properties of undefined (reading 'layerGroup')` at `draw` | `✔ draws numbered buttons and one styled route through context.draw` |
| Route details and safe source link | `TypeError: Cannot read properties of undefined (reading 'layerGroup')` | `✔ a pin opens route details and a safe post link; switching closes them`; `✔ a non-http URL gets no post link` |
| Route switching and clearing | `TypeError: Cannot read properties of undefined (reading 'layerGroup')` | `✔ one route has no picker; switching routes never duplicates markers, lines or the picker`; `✔ disable clears every drawing and the popup; re-enable restores one route` |

The first RED run had `pass 3`, `fail 6`; all six failed at the old Leaflet `layerGroup` call when the test supplied only the drawing API. The final GREEN run had `pass 9`, `fail 0`.

Browser command: `CDP_PORT=9355 node tasks/redesign/checks/map-itineraries.browser.mjs` with an isolated headless Brave profile. The preview used local map fixtures. The check verifies route switching, one line and picker, the source link, chip toggle, hidden empty route, clean page console, and MapLibre style reload. Final output:

```text
PASS maplibre 375px: 7/9 numbered pins, one line, picker, popup, toggle, empty route, console clean
PASS maplibre 1440px: 7/9 numbered pins, one line, picker, popup, toggle, empty route, console clean
PASS leaflet 375px: 7/9 numbered pins, one line, picker, popup, toggle, empty route, console clean
PASS leaflet 1440px: 7/9 numbered pins, one line, picker, popup, toggle, empty route, console clean
```

An earlier browser run passed MapLibre at 375 px, then timed out after a route toggle at 1440 px with `Error: Style is not done loading.` after a style reload. Waiting for `areTilesLoaded()` as well as `isStyleLoaded()` made the check wait for the actual map reload to finish. The final run above passed without page errors.

| What is guaranteed | Test name | Level | Result |
| --- | --- | --- | --- |
| One registered itinerary layer, no core means no registration | `registers one itinerary layer and does nothing without the core` | Unit | PASS |
| Ordered numbered buttons, geographic line coordinates, purple line style | `draws numbered buttons and one styled route through context.draw` | Unit | PASS |
| Safe source link and dismissal on route switch | `a pin opens route details and a safe post link; switching closes them`; `a non-http URL gets no post link` | Unit | PASS |
| Missing or invalid stops and empty or failed route data | `missing and invalid stops are skipped, and one stop has no line`; `empty or failed route data draws nothing and leaves the chip hidden` | Unit | PASS |
| Shell chip state even before the layer is shown | `the ready event marks the shell even when the layer is off` | Unit | PASS |
| No duplicate line, markers or picker after route switches and layer toggle | `one route has no picker; switching routes never duplicates markers, lines or the picker`; `disable clears every drawing and the popup; re-enable restores one route` | Unit | PASS |
| Both engines and viewport widths, including MapLibre style reload | `map-itineraries.browser.mjs` | Browser | PASS at 375 and 1440 px on both engines |

Coverage: M5 changes only browser JavaScript and CSS; the Go package coverage threshold does not apply. `git diff --check` passed for the M5-owned files. The shared `make test lint build` gate is run by the M6 integrator after M3–M5 land.
