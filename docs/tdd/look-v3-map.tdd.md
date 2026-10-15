# Look v3 map, layers and layout evidence

Source: `tasks/look-v3-todo.md` V5–V7 and the city card part of V8 (D1).

Journeys:
- As a traveller, I want full layer labels and a filled selected pill, so I know which pins are visible.
- As a traveller, I want a shorter map beside place details and previews, so I can explore the page in one scroll.
- As a traveller, I want a map back action, so I can return to the city after selecting a place.

## RED

Command:
```
node --test tasks/redesign/checks/map-layers.test.mjs tasks/redesign/checks/map-look-v2.test.mjs
```
Actual excerpts before production changes:
```
✖ upgrades the static chips into buttons with aria-pressed and keeps labels
✖ phone keeps full labels and outline/filled state
✖ v3 chrome groups zoom, exposes back and uses compact attribution
ℹ tests 17
ℹ pass 14
ℹ fail 3
```
The first failure returned `Writers' itineraries` instead of `Itineraries`; the phone returned shortened labels; the new back control was absent.

The picker test was changed to require a menu, then run before the menu implementation:
```
node --test tasks/redesign/checks/map-itineraries.test.mjs
```
```
✖ one route has no picker; switching routes never duplicates markers, lines or the picker
TypeError: Cannot read properties of undefined (reading 'children')
```
The existing select was directly in the layer slot; the expected menu was missing.

## GREEN

Command:
```
node --test tasks/redesign/checks/writers-map-core.test.mjs tasks/redesign/checks/map-layers.test.mjs tasks/redesign/checks/map-itineraries.test.mjs tasks/redesign/checks/map-look-v2.test.mjs tasks/redesign/checks/map-pins.test.mjs
```
Actual final output:
```
ℹ tests 97
ℹ suites 0
ℹ pass 97
ℹ fail 0
ℹ cancelled 0
ℹ skipped 0
ℹ todo 0
```

| What is guaranteed | Test | Level | Result |
| --- | --- | --- | --- |
| Full labels on phone and desktop, pressed state, persistence and unavailable layers | map-layers.test.mjs | unit | 12 passed |
| Picker menu, route switching and cleanup without duplicates | map-itineraries.test.mjs | unit | 12 passed |
| Back visibility, return camera and map focus; grouped zoom and compass | writers-map-core.test.mjs | unit | 65 passed |
| Category icons and photo callout; route and pin contrast | map-pins.test.mjs | unit | 3 passed |
| Back, grouped zoom, crosshair, compact attribution; preserved saved controls | map-look-v2.test.mjs | unit | 5 passed |

## Coverage and limitations

This change touches no Go package. JavaScript uses the repository's Node test runner; no numeric JS coverage threshold is defined by AGENTS.md.

`writers-map-layout.browser.mjs` was rewritten to verify a 480–520 px desktop map, approximately 510 px panel, teaser placement, a 190–220 px phone map (220 px allows five 44 px controls to stack) and an inline card, across both engines. Runtime command:
```
node tasks/redesign/checks/writers-map-layout.browser.mjs
```
First runtime failure with long details reproduced the panel expanding to 1800 px instead of scrolling:
```
AssertionError [ERR_ASSERTION]: {"canvas":505,"panel":1800,"overflow":"auto","scrollHeight":1800,"before":505}
```
After adding desktop `contain: size` to the panel, actual output:
```
PASS maplibre 375px: short map with inline place card and teasers
PASS maplibre 1440px: short map with inline place card and teasers
PASS leaflet 375px: short map with inline place card and teasers
PASS leaflet 1440px: short map with inline place card and teasers
```
The check also asserts no JavaScript console errors. This command required sandbox escalation for localhost access.

Side-by-side screenshot artifacts: `tasks/artifacts/look-v3/V6/375-comparison.png` and `1440-comparison.png` (implementation left, reference right). Basemap is the deterministic fixture, so streets and geographic labels do not appear. Phone hero styling was still being integrated at capture time and is addressed by V4. Final integrated captures are recorded by the integration owner.

The deterministic fixture camera opens at zoom 12.3 to show its dispersed pins; the production camera stays unchanged. Scale labels remain geographically accurate at the current zoom, with a white line, ticks and zero marker; the distance is not hardcoded to 1 km. Leaflet attribution expands on pointer hover or link focus; MapLibre uses its built-in compact attribution control. New water label styling and uppercase district labels apply to the default vector style; custom styles are preserved.

## Preserved preview and itinerary journeys

Updated the existing browser checks for full labels, the route chooser menu, 220 px phone map, inline city card and taller desktop panel. Retained their pin keyboard navigation, route switching, popup safety, style reload, layer disable/enable, empty routes, HTMX canvas reuse and source error coverage.

Commands:
```
CDP_PORT=9231 node tasks/redesign/checks/map-itineraries.browser.mjs
CDP_PORT=9231 node tasks/redesign/checks/writers-map-preview.browser.mjs
```
Both passed all four MapLibre/Leaflet × 375/1440 combinations with clean consoles. Actual output excerpts:
```
PASS maplibre 375px: 7/9 accessible stop dots, one line, picker, popup, toggle, empty route, console clean
PASS leaflet 1440px: 7/9 accessible stop dots, one line, picker, popup, toggle, empty route, console clean
PASS maplibre 375px: writer data, base-only and unavailable swaps, one map, focusable pins, no place list, local fixtures
PASS leaflet 1440px: writer data, base-only and unavailable swaps, one map, focusable pins, no place list, local fixtures
```
The preview journey now also focuses the compact attribution control/link and asserts that its credit text expands visibly. The fixture's MapLibre credit is plain text; production source credits can contain links. Both engine paths were verified at both widths.

## Selected callout clipping regression

Source: V7, selected pin and callout must remain inside the shorter map. The integrated shell sequence first captured the city, then selected Belém Tower. Full-page CDP capture had paused animation frames; explicitly foregrounding the tab after capture restores the same running animation frames a user has while viewing the page. The original selected-pin bounds assertion remains intact.

After restoring the test browser foreground, a separate real clipping bug was visible: at 375 px the photo callout started at y=293 while the map started at y=304. Added an assertion for the complete callout bounds and ran:
```
CDP_PORT=9231 node tasks/redesign/checks/look-v3-shell.browser.mjs
```
RED actual output:
```
AssertionError [ERR_ASSERTION]: selected photo callout fits inside the map
```
Changed only the callout's vertical offset from -85 px to -65 px. GREEN actual output:
```
PASS 375: header, hero text bounds, forecast Escape/focus, Belém facts, screenshots
PASS 1440: header, hero text bounds, forecast Escape/focus, Belém facts, screenshots
```
The final shell screenshots were refreshed in `tasks/artifacts/look-v3/final/` for city and selected place at both widths. Both the selected regular marker and the complete photo callout are required to fit inside the map by this browser check.
