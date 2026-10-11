# 3d — Funchal writers' map card layout

Source: [tasks/todo.md](../../tasks/todo.md), subtask 3d, and the layout in [the Funchal mockup](<../ideas/Funchal Travel Guide Dashboard.png>).

## Journeys

- As a traveler on a wide screen, I want a map and equally tall place panel side by side so I can explore a place without losing the map.
- As a traveler on a phone, I want the place panel below the map without horizontal scrolling.
- As a traveler new to the map, I want to understand its four layers from an info icon and see a short message only when map data is partial or unavailable.
- As a traveler changing destinations, I want the layout to hold after the map swaps.

## RED → GREEN

| Behavior | Command | RED excerpt | GREEN excerpt |
| --- | --- | --- | --- |
| Card heading, info control, map and panel layout at 375 and 1440 px | `CDP_PORT=9341 GOCACHE=/private/tmp/traveltab-3d-go-cache TMPDIR=/private/tmp node tasks/redesign/checks/writers-map-layout.browser.mjs` | `Expected values to be strictly equal: + actual - expected + undefined - 'About map layers'` | `PASS 375px: writers' map card layout`; `PASS 1440px: writers' map card layout` |
| Info tooltip fits inside the phone viewport | same browser command | `tooltipRect":{"x":200.25,"y":563.453125,"width":280,"height":56.375}`; `overflow":375` | `PASS 375px: writers' map card layout`; `PASS 1440px: writers' map card layout` |
| Loading does not show a status message | `node --test --test-name-pattern='hides loading status' tasks/redesign/checks/writers-map-core.test.mjs` | `+ 'Loading places…'`; `- ''`; `fail 1` | `✔ hides loading status, renders a keyboard list, and shows partial data status`; `pass 1`; `fail 0` |

## Guarantees

| What is guaranteed | Test name | Level | Result |
| --- | --- | --- | --- |
| Title, four-layer help text, inline partial status, zoom and scale controls, and tooltip inside the viewport | `writers-map-layout.browser.mjs` | Browser | PASS at 375 and 1440 px |
| Loading status is hidden, then partial status appears when the result arrives | `hides loading status, renders a keyboard list, and shows partial data status` | JS unit | PASS |
| Map and panel have equal height with an approximately 65/35 split at 1440 px; panel stacks below the map at 375 px | `writers-map-layout.browser.mjs` | Browser | PASS |
| No horizontal scrolling on initial load or after a destination swap | `writers-map-layout.browser.mjs` | Browser | PASS at 375 and 1440 px |
| Existing map instance, keyboard list, local tiles, and unavailable state survive two destination swaps | `writers-map-preview.browser.mjs` | Browser | PASS at 375 and 1440 px |
| Preview and server route tests still pass | `go test ./cmd/design-preview ./internal/adapters/httpserver` | Integration | PASS |

`node --test tasks/redesign/checks/writers-map-core.test.mjs` returned `tests 8`, `pass 8`, `fail 0`.
`PATH=/private/tmp/traveltab-3e-bin:$PATH GOCACHE=/private/tmp/traveltab-3d-go-cache GOLANGCI_LINT_CACHE=/private/tmp/traveltab-3d-lint-cache make test lint build` exited 0: all Go packages passed, lint reported `0 issues.`, and the web app built. The compatible linter was used because the installed binary was built with Go 1.26 and this module targets Go 1.27.2.

Visual review of the local-tile preview at 375 and 1440 px against the mockup confirmed the card width, map/panel split, phone stacking and spacing. The fixture has no chips, pins or populated panel yet; those belong to 7f and 7a. The screenshots use local fixture tiles, while the live EOX basemap was selected and verified in 3e.

Coverage: no Go package changed, so the Go package 80% rule does not apply. Browser branch coverage was not measured.

## Merge follow-up (2026-10-11)

The merge of two 3d implementations kept the native `title` tooltip while retaining a browser
check for `aria-describedby`. The follow-up also checked the equal-height layout with a tall
panel and the live EOX scale bar beside its attribution.

| Behavior | Command | RED excerpt | GREEN excerpt |
| --- | --- | --- | --- |
| Described tooltip opens on focus or click and closes on Escape | `CDP_PORT=9349 node tasks/redesign/checks/writers-map-layout.browser.mjs` | `+ undefined` / `- 'About map layers'`; after adding the focus check, `+ 'visible'` / `- 'hidden'` on Escape | `PASS 375px: writers' map card layout`; `PASS 1440px: writers' map card layout` |
| Map and tall place panel stay equal height | same layout command | `AssertionError [ERR_ASSERTION]: {"canvas":420,"panel":634}` | `PASS 1440px: writers' map card layout` |
| Live imagery credit and scale bar do not overlap | `CDP_PORT=9349 node tasks/redesign/checks/writers-map-imagery-live.mjs` | `"controlsOverlap":true`, `"scaleRect":{"x":32,"y":842.265625,"width":93,"height":18.1875}`, `"creditRect":{"x":27,"y":815.0625,"width":321,"height":50.390625}` | `PASS 375px: live EOX imagery, required credit visible, no overflow or console errors`; `PASS 1440px: live EOX imagery, required credit visible, no overflow or console errors` |

| What is guaranteed | Test name | Level | Result |
| --- | --- | --- | --- |
| Four-layer explanation is associated with the info button, fits the phone viewport, and opens and closes with focus, click and Escape | `writers-map-layout.browser.mjs` | Browser | PASS at 375 and 1440 px |
| A desktop map stretches with a panel taller than 420 px | `writers-map-layout.browser.mjs` | Browser | PASS at 1440 px |
| Scale bar and required EOX credit are both readable without overlap | `writers-map-imagery-live.mjs` | Browser, live imagery | PASS at 375 and 1440 px |
| Destination swaps retain one map, the keyboard list and the empty panel state | `writers-map-preview.browser.mjs` | Browser | PASS at 375 and 1440 px |

`node --test tasks/redesign/checks/writers-map-core.test.mjs` returned `tests 8`, `pass 8`, `fail 0`.
`PATH=/private/tmp/traveltab-3e-bin:$PATH make test lint build` exited 0: all Go packages
passed, lint reported `0 issues.`, and the web app built. The alternate linter was built with
Go 1.27; the installed default was built with Go 1.26 and cannot lint this module.

Coverage: no Go package changed. Browser branch coverage was not measured. One live EOX check
run reported transient browser errors at 1440 px; an unchanged rerun passed at both widths.
