# Map pins and place panel on the drawing API (M3)

Source: [map-3d-todo.md](../../tasks/map-3d-todo.md), M3, and the approved drawing API in [writers-map-contract.md](../writers-map-contract.md).

## Journeys

- As a traveller, I want writer pins above notable places, with counts and labels for prominent places, so I can find places mentioned by writers.
- As a traveller using a keyboard, I want named, focusable pins and the place list, so I can open a place panel without a pointer.
- As a traveller on either map engine, I want a pin click to load the place panel and its close button to reset it.

## RED and GREEN

| Behaviour | Command | RED excerpt | GREEN excerpt |
| --- | --- | --- | --- |
| Pins use `context.draw.markers` and `clear`, with no Leaflet calls; retain priority, labels, selection and keyboard access | `node --test tasks/redesign/checks/map-panel.test.mjs` | `ℹ pass 5`, `ℹ fail 5`; `Error: pin layer must not access Leaflet` at `layerGroup` | `ℹ tests 10`, `ℹ pass 10`, `ℹ fail 0` |
| Pins and panel work on MapLibre and Leaflet at mobile and desktop widths | `CDP_PORT=9233 node tasks/redesign/checks/map-panel.browser.mjs` | First browser run failed on `TypeError: Cannot read properties of undefined (reading 'layerGroup')` while other layers still used Leaflet. After those layers were ported, an engine-neutral check initially read the Leaflet child button's z-index (`0`) instead of its marker wrapper's z-index. | `ok writers-map 375px`, `ok writers-map 1440px`, `ok writers-map-leaflet 375px`, `ok writers-map-leaflet 1440px`; clean page consoles |

## Guarantees

| What is guaranteed | Test name | Level | Result |
| --- | --- | --- | --- |
| Notable and writer layers register; writer count and priority are preserved without `L.*` | `registers the notable and writers pin layers with the map core`, `draws writer pins orange with their count, above blue notable pins` | unit | PASS |
| Five largest places get labels; a pin click selects its place | `labels only the biggest places`, `selects a place when its pin is clicked` | unit | PASS |
| Buttons have accessible titles, keyboard focus and visible focus style | `pins have accessible titles, keyboard focus and a visible focus style` | unit | PASS |
| Removing and re-adding a layer leaves one set of pins | `removing a layer takes its pins off the map and adding it again leaves one copy` | unit | PASS |
| Panel load, monograms and close behaviour remain | `loads the selected place into the panel slot`, `draws one monogram per writer with a colour from the host, and +N past three`, `close returns the panel to its empty state` | unit | PASS |
| Both engines render pins and labels, pin click loads the panel, focus outline is visible, console is clean and the layout fits 375 and 1440 px | `map-panel.browser.mjs` | browser | PASS |

M3 changes JavaScript and CSS only, so there is no changed Go package for `go test -cover`. The browser test uses local map fixtures but fetches the pinned engine libraries from their CDNs. Screenshots from `CDP_PORT=9233 SHOT_DIR=/private/tmp node tasks/redesign/checks/map-panel.browser.mjs` were inspected at 375 px for both engines.

`PATH=/private/tmp:$PATH make test lint build` passed: all Go packages passed, `golangci-lint run ./...` reported `0 issues.`, and `go build -o bin/traveltab ./cmd/web` completed. An initial sandbox run failed because `httptest` could not bind a local port; the same command passed with local socket access.
