# Visible pins, pin popup, no place-tag list

**Source:** owner decisions 2026-10-12 (follow-up to `tasks/map-3d-todo.md` M7; contract change approved, see `docs/writers-map-contract.md`).

## Journeys

- As a traveller, I want writer pins visible on the map as soon as it loads, so I can see where writers went.
- As a traveller, I want a small popup on a pin (name, "Mentioned by N writers", up to 3 blogs, Details), so I can peek before opening the side panel.
- As a keyboard user, I want to Tab to pins, press Enter for the popup, Esc to close it, and use Details to open the panel.
- As the owner, I want the row of place-name buttons gone.
- As a traveller on the dark map, I want routes and pins to stay readable.

## Were pins missing?

**Yes, off screen, not unrendered.** Repro (`-scenario writers-map`, `writers-map-leaflet`, and `-live-tiles`, Madeira, 375 and 1440 px): all 15 fixture pins were in the DOM (the fixture has 15 places, not 200), but at the fixed zoom-12 view centred on the island only 2-4 were in the map and **0 writer pins on MapLibre** (2 on Leaflet at 1440):

```
writers-map false 1440 {"pins":15,"visible":2,"writers":5,"writersVisible":0,...}
writers-map-leaflet false 1440 {"pins":15,"visible":4,"writers":5,"writersVisible":2,...}
writers-map true 1440 {"pins":15,"visible":2,"writers":5,"writersVisible":0,...}
```

Madeira is wider than the zoom-12 view, so the writers' pins sat outside it (the owner's screenshot shows the same on Leaflet). Fix: once places load, the core fits the map to the writer pins (all places if none has a writer), `maxZoom` 12.3, no animation. On the tilted MapLibre view `fitBounds` under-fits (perspective stretches the near edge: a pin was 18 px outside), so it uses padding 88 instead of 48.

Live dark style (OpenFreeMap dark + Mapterhorn), Funchal z16: the Funchal pin **was rendered but faded to 20 % opacity** because MapLibre dims markers it thinks are covered by terrain/buildings. Fix: markers use `opacityWhenCovered: '1'`. Verified live: pin fully opaque over the extruded buildings (scratchpad `live-dark-funchal-z16.png`). At z16 elsewhere in Madeira no pins are in view simply because no place is there.

## Behaviours

| Behaviour | RED (excerpt) | GREEN |
| --- | --- | --- |
| `draw.popup` exists on both engines; replace by id; `clear` closes it; `onClose` only for engine closes | `TypeError: draw.popup is not a function`; shape test `actual: [ 'clear', 'control', 'fit', 'flyTo', 'line', 'markers' ]` | `node --test tasks/redesign/checks/*.test.mjs` |
| A click on the bare map closes the popup; a click on a pin (which also reaches the map on both engines) does not | `AssertionError` (no click handler). In a real browser the popup opened by Enter was closed at once on both engines | same, plus `map-panel.browser.mjs` |
| Pin click opens popup (name, count, monograms, Details); Details calls `selectPlace`; Esc closes and returns focus | 6 failing tests in `map-panel.test.mjs`, e.g. `TypeError: Cannot read properties of undefined (reading 'lngLat')` | passing |
| Core frames writer pins, else all places | `TypeError: Cannot read properties of null (reading '0')` | passing |
| No list slot, no `renderList`, status text without "Browse places below" | `AssertionError: nothing renders a place-tag row` (actual 1); Go: `TestWriterMap_PageFlagStates ... should not contain "data-map-slot=\"list\""` | passing |
| Route default `#ff8a65`, itinerary line `#b794f6`, legend dot `#b794f6` | 3 failing tests (colour assertions) | passing |
| Pins opaque when covered | `opacityWhenCovered` undefined | `'1'` |

## What is guaranteed

| Guarantee | Test | Level | Result |
| --- | --- | --- | --- |
| `popup` same shape on both engines, replace by id, clear closes | `context.draw has the same calls on both engines`, `<engine>: popup shows an element...`, `<engine>: onClose runs when the engine closes...` | unit | PASS |
| Map click closes, pin click does not, handlers detached | `maplibre: a click on the map closes...`, `leaflet: a click on the map closes...` | unit | PASS |
| Pin popup content, Details, Esc + focus, engine close, layer remove | 7 tests in `map-panel.test.mjs` | unit | PASS |
| First view frames writer pins | `<engine>: frames the writer pins once the places load...` | unit | PASS |
| No list slot / row | `hides loading status, renders no place list...`, `TestWriterMap_PageFlagStates` | unit / Go | PASS |
| Colours, opaque pins | `map-itineraries.test.mjs`, `writers-map-core.test.mjs` | unit | PASS |
| Writer pins in view, Tab reaches pin, Enter opens, Esc closes and refocuses, real mouse click opens and stays, click on bare map closes, another pin replaces, Details opens panel and focuses its heading | `map-panel.browser.mjs` | browser, both engines, 375 and 1440 px | PASS |
| Popup is a named dialog, focus on Details, no unnamed controls | `a11y-audit.mjs` (`writers-map-popup-*`) | browser | PASS |

## Verification run (2026-10-12)

- `node --test tasks/redesign/checks/*.test.mjs`: 94 pass, 0 fail.
- Browser checks, both engines, 375 and 1440 px, console clean: `writers-map-core`, `map-panel`, `map-dayplan`, `map-itineraries`, `writers-map-layout`, `writers-map-preview`, `pr7-acceptance`, `pr7-integration`, `hero-phone`: all PASS.
- `a11y-audit.mjs`: 31 passed, 0 failed; `scenarios.mjs`: PASS, 2816 assertions, no JS exceptions (both with `OUTPUT_DIR` in the scratchpad). axe-core is not installed here, so axe did not run (the popup record states axe only when `AXE_PATH` is set).
- `make test build`, `go vet ./...`, `gofmt -l internal cmd`: clean.
- Screenshots (scratchpad `out/`): `map-popup-writers-map-{375,1440}.png`, `map-popup-writers-map-leaflet-{375,1440}.png`.

## Known gaps

- Popup blogs are read from the place's panel fragment (one extra GET per writer pin click), because the GeoJSON carries no blog names. Adding them to the GeoJSON would remove the request (server change, not made).
- Tab order inside each layer follows feature order; notable pins come before writer pins in the DOM. Not re-sorted.
- Clustering not added. At the framed Madeira view two pins 2-3 km apart touch (Pico Ruivo and Pico do Arieiro). With the real 200 places the notable layer may need it; judge on real data.
- The live-tile and live-style checks were manual, not part of `make test`.
- Itinerary pins keep `#7b3fb8` (white numbers need the darker purple); only the line and the legend dot are brightened.
