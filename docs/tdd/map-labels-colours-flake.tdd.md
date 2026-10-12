# Map labels, road weight, brighter pins, preview flake

**Source:** owner decisions 2026-10-12, under M7 in `tasks/map-3d-todo.md`. Follows [map-electroverse-colours.tdd.md](map-electroverse-colours.tdd.md).

**Journeys**
- As a traveller, I want town and neighbourhood names on the map at z10 to z13, so that I can find my way.
- As a traveller, I want minor roads quiet and main roads clear, so that the map is not a web of lines.
- As a traveller, I want day-plan and itinerary pins I can see on the navy land, so that I do not lose them.
- As the owner, I want the preview check to be stable, so that a red run means a real bug.

## 1. Why labels looked missing
Not the style, the palette or the zoom. Place layers have no `minzoom` (`place_city` max 14, `place_town` and `place_suburb` max 15), no `text-opacity`, no `visibility: none`, glyph files answer 200, and the palette only sets their colour and halo (`#e0e0e0`, `#101820`). `queryRenderedFeatures` returned 115 place features at Lisbon z11.4.

Cause: with terrain on, MapLibre leaves symbols unplaced after a bare `map.jumpTo` at pitch 0 until the next camera change. My capture script jumped the camera itself. Probe on the live style, same state after a jump, then one candidate nudge:

| Nudge after the jump | Labels |
| --- | --- |
| none, `triggerRepaint`, waiting 15 s | missing |
| `panBy([0,0])`, `resize()`, `setPitch(same)`, `setTerrain(same)`, `setTerrain(null)` | appear |

In the product the camera moves through `reset` and `fit` at pitch 55 and the labels show (Madeira and a Madeira to Porto swap on the live style). So no product change; captures call `map.resize()` after jumping. Noted in `docs/maintenance.md`.

## 2. Behaviours (labels and road weight)
Same palette table in `writers-map.js`, default style only. It now sets layout properties (`text-size`) with `setLayoutProperty`, and paint properties (`line-width`, `line-opacity`) as before.

| Layer | Change |
| --- | --- |
| `highway_minor` | width z10 0.4, z13 1.2, z20 20 (was 1.8 at z13); opacity 0.35 at z10 to 0.9 at z14 |
| `highway_major_inner`, `highway_major_casing` | secondary and tertiary (they share the primary layers) 0.7 and 1.2 px at z10, 1.4 and 2 px at z13, opacity 0.4 at z10 to 1 at z14; primary and trunk 2 and 3 px at z10, 3.6 and 4.8 px at z13 (about the old weight) |
| `highway_major_subtle` | z8 width 1.5 primary or trunk, 0.5 other |
| motorways | unchanged |
| `place_other`, `place_suburb`, `place_village`, `place_town`, `place_city` | `text-size` 11 (was 10), colour `#e0e0e0`, halo `#101820` |
| `place_city_large` | unchanged (14) |

Colours stay in the palette. The fixture style (`mapstyle.go`) has the same minor-road width and opacity on `road-secondary`. It has no glyphs, so it has no label layers: nothing to mirror.

Unit RED then GREEN (`node --test tasks/redesign/checks/writers-map-core.test.mjs`):
```
RED   actual: undefined,
      expected: '["interpolate",["exponential",1.55],["zoom"],10,0.4,13,1.2,20,20]'
RED   TypeError: Cannot read properties of undefined (reading 'text-size')
RED   (second pass) actual: undefined, expected: '["interpolate",["exponential",1.3],["zoom"],10,["match",["get","class"],["primary","trunk"],2,0.7],13,...'
GREEN ℹ pass 102  ℹ fail 0   (all *.test.mjs)
```
Go: `go test ./cmd/design-preview/ -run ThinsMinorRoads`
```
RED   road-secondary width 2 opacity null, want ["interpolate",["exponential",1.55],...
GREEN ok  weatherservice/cmd/design-preview  coverage: 86.8% of statements
```
Side-by-side, live style, 38.73133, -9.26065, z11.4, pitch 0 (reference screenshots from the earlier run, Electroverse is the left half):
- `scratchpad/ev2/side-by-side-1440.png`, `scratchpad/ev2/side-by-side-375.png`
- ours alone: `scratchpad/ev2/ours-1440.png`, `ours-375.png`
- Lisbon centre z13 (38.7139, -9.1394) 1440: `scratchpad/ev2/ours-z13-lisbon-1440.png`

(`scratchpad` is `/private/tmp/claude-501/-Users-fernando-personal-projects-traveltab/3db1f481-7924-450e-abf9-93fbf9871511/scratchpad`.)

## 3. Brighter map colours
| Item | Old | New | Where |
| --- | --- | --- | --- |
| Day-plan pins | `#2e7d32` | `#43a047` | `map-dayplan.js` |
| Itinerary numbered pins | `#7b3fb8` | `#8e5bd6` | `map-itineraries.css` (the colour lives in the stylesheet) |
| Day-plan line | `#a64027` | `#ff8a65` | already the engine default (`writers-map.js`); `#a64027` was only a sample colour in `writers-map-core.test.mjs`, the day plan draws no line. Nothing to change. |

Chip dots (`writers-map-layout.css`, not `map-layers.css`): "Your day plan" `#17b978` and "Writers' itineraries" `#b794f6`. Neither chip uses the old pin colours, so they are unchanged.

RED then GREEN:
```
RED   map-dayplan.test.mjs        actual: false, expected: true     (pin background === '#43a047')
RED   map-itineraries.test.mjs    actual: '#7b3fb8', expected: '#8e5bd6'
RED   map-dayplan.browser.mjs     'rgb(67, 160, 71)' vs 'rgb(46, 125, 50)'  (run on the new product, then the check was updated)
GREEN ℹ pass 102, ℹ fail 0; map-dayplan and map-itineraries browser checks PASS on both engines at 375 and 1440
```
The unit test also asserts the white number contrast on `#8e5bd6` is at least 4.5.

### Contrast (WCAG 2.x, fill or line vs base)
| Colour | Land `#18263b` | Water `#000e29` | Park `#203838` | Road `#4c576c` | White numbers |
| --- | --- | --- | --- | --- | --- |
| Plan pin `#2e7d32` (old) | 2.97 | 3.74 | 2.43 | 1.42 | 5.13 |
| **Plan pin `#43a047`** | **4.61** | **5.81** | 3.78 | 2.20 | **3.30** |
| Itinerary pin `#7b3fb8` (old) | 2.35 | 2.96 | 1.93 | 1.12 | 6.47 |
| **Itinerary pin `#8e5bd6`** | **3.35** | **4.22** | 2.74 | 1.60 | **4.55** |
| Route line `#ff8a65` (default) | 6.58 | 8.29 | 5.39 | 3.14 | n/a |
| Day-plan sample `#a64027` (old) | 2.45 | 3.09 | 2.01 | 1.17 | n/a |

Findings:
- The itinerary pin is 4.55 against white, so the numbers pass 4.5, but on land it is still 3.35 (about 3 for graphics). It has a white 2 px ring.
- **The day-plan pin's white number drops from 5.13 to 3.30**, under 4.5 for 13 px bold text. It is the price of the brighter fill. The owner did not ask for a number colour change; dark numbers (about `#0b1f0d`, 6.6 on the fill) would fix it. Open point. The writer pin `#f35a2c` (3.33) and notable pin were already below 4.5 and are unchanged.

## 4. The flaky `writers-map-preview.browser.mjs`
Root cause: **the check, not the product.** After the Porto swap the check waited for the status slot to be `''`. The slot is already `''` when the new map shell mounts (`mount` calls `showStatus(shell, '')`) and stays `''` until the places arrive, so the wait passed at once and the pin list was read before `loadPlaces` had drawn the pins. Whether it flaked depended on whether the geo+json response beat the next poll (50 ms): 4 of 7 runs at 1440 maplibre. Coimbra was safe only because its status is not empty after loading.

Product side checked: `loadPlaces` and `frameMap` are guarded by `requestNumber`, `beforeSwap` clears layers and bumps it, and the old pins are gone before the new response arrives. No race found, so no product test.

Prove-It: the check now delays every geo+json fetch by 400 ms, so the late-places case happens on every run.
```
RED   (old wait + 400 ms delay, every run)
      actual: [], expected: [ 'Clérigos Tower' ]   at writers-map-preview.browser.mjs:82
GREEN (wait for the pins when the swap shows no status message)
      PASS maplibre 375px, PASS maplibre 1440px, PASS leaflet 375px, PASS leaflet 1440px
```
Fix: the swap table has a per-case "loaded" condition. Porto waits for `.map-pin`, Coimbra for its status text. No sleeps. The delay stays in the check.

8 consecutive runs of `writers-map-preview.browser.mjs` (both engines, 375 and 1440 px), all with the delay active:
```
run 1 exit 0 PASS=4   run 2 exit 0 PASS=4   run 3 exit 0 PASS=4   run 4 exit 0 PASS=4
run 5 exit 0 PASS=4   run 6 exit 0 PASS=4   run 7 exit 0 PASS=4   run 8 exit 0 PASS=4
```

## What is guaranteed
| Guarantee | Test | Level | Result |
| --- | --- | --- | --- |
| Minor, secondary and tertiary roads thinner and dimmer, primaries and motorways untouched | `thins and dims minor roads, and secondary and tertiary roads, ...` | unit | PASS |
| Place label size set as a layout property, large cities unchanged | `sets the place label size through layout properties...` | unit | PASS |
| Custom style URL gets no layout or paint changes | `applies no layout overrides to a custom style URL`, `applies no palette overrides...` | unit | PASS |
| Fixture minor road weight | `TestPreview_MapStyleThinsMinorRoads` | unit | PASS |
| Day-plan pins `#43a047` | `uses drawing API for green, numbered, focusable pins...` | unit | PASS |
| Itinerary pins `#8e5bd6`, numbers at least 4.5:1 | `numbered pins use the brighter purple...` | unit | PASS |
| Pin colours in a real browser | `map-dayplan.browser.mjs`, `map-itineraries.browser.mjs` | integration | PASS |
| Swap waits for places | `writers-map-preview.browser.mjs` (delayed places) | integration | PASS 8 of 8 |
| Live style loads with the new palette | `writers-map-imagery-live.mjs` | live | PASS |

## Verification run (Brave headless, SwiftShader, own CDP port 9271, temp profile)
- `node --test tasks/redesign/checks/*.test.mjs`: 102 pass.
- Browser checks PASS on both engines at 375 and 1440 px, console clean: writers-map-core, map-panel, map-dayplan, map-itineraries, writers-map-layout, writers-map-preview, pr7-acceptance, pr7-integration, hero-phone (hero-phone is one engine, no map).
- `a11y-audit.mjs`: 31 passed, 0 failed. `scenarios.mjs`: 2816 assertions, 110 screenshots, no JavaScript exceptions.
- `writers-map-imagery-live.mjs` (live, owner-approved): PASS, once.
- `make test build lint` (with `/private/tmp` on PATH): pass, 0 lint issues. `cmd/design-preview` coverage 86.8%.

## Known gaps
- Minor roads at z10 to z11 are very faint (0.4 px at 35 percent opacity), by design; they appear properly at z13. Retune the two constants in `writers-map.js` if that is too quiet.
- Labels stay uppercase, as in the OpenFreeMap style; the Electroverse labels are mixed case. Not changed (visual weight only).
- Neighbourhood labels (`place_other`, `place_suburb`) have no `minzoom` in the style, so they already show from far out; no zoom range was changed.
- The label-after-jump behaviour is a MapLibre terrain quirk; no guard added because the product's own camera moves show labels. Revisit if a user report shows labels missing after a swap.

## Owner decision (2026-10-12)

Day-plan pins keep **white numbers** on `#43a047` (3.30:1, below 4.5:1 for 13 px text). Accepted by the owner; dark numbers (`#0b1f0d`, about 6.6:1) remain the option if it needs revisiting.
