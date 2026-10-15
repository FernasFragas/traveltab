# Look v3 polish evidence (V14)

Source: [tasks/look-v3-todo.md](../../tasks/look-v3-todo.md) V14, after the owner's status review on
2026-10-15 listed gaps 1–7 against `desktop.png` / `phone.png`. Decision D21 (pills) was asked and
recorded in the [plan](../../tasks/look-v3-plan.md).

**Journeys**
- *As a traveller, I want the layer pills to read like the mockup, so that the base layer and the overlays are clear.*
- *As a traveller, I want a place's type in plain words, so that "tower" reads as "Historic Landmark".*
- *As a traveller, I want a best-time tip for landmarks too, so that the facts card is useful at Belém.*
- *As a traveller, I want the writer's words as a quote, so that the card reads like the mockup.*
- *As a traveller on a phone, I want an uncluttered map and a compact place card, so that the small map and card stay readable.*
- *As a traveller on a phone, I want my day as one row, so that the card matches the mockup.*

## Behaviours

### 1. Pills: Notable places as the base layer (D21, later revised in 11)
`node --test tasks/redesign/checks/map-layers.test.mjs`

RED:
```
✖ upgrades the overlay chips into buttons with aria-pressed and keeps labels
    actual: [ 'BUTTON', 'BUTTON', 'BUTTON', 'BUTTON' ],
    expected: [ 'SPAN', 'BUTTON', 'BUTTON', 'BUTTON' ],
✖ notable places is the always-on base layer, not a toggle
✖ on overlays are white pills with filled icons; only the base layer is filled terracotta
ℹ pass 11
ℹ fail 3
```
GREEN: `ℹ pass 14` `ℹ fail 0`. A later test for the stray pill edge (outline drawn inside the pill)
failed (`ℹ pass 14` `ℹ fail 1`) and then passed (15 of 15).

### 2. Readable place kind
`go test ./internal/adapters/httpserver/ -run 'TestPlaceKindLabel|TestPlacePanel_MetaShowsReadableKind|TestPlacePanel_GeoJSONCarriesReadableKind|TestPlacePanel_WriterQuote'`

RED (first a build failure on the missing function, `undefined: placeKindLabel`; then with a stub that returned the raw kind):
```
expected: "Historic Landmark"
actual  : "tower"
--- FAIL: TestPlacePanel_MetaShowsReadableKind
    "<p class=\"map-panel-meta\">… tower …" does not contain "Historic Landmark"
--- FAIL: TestPlacePanel_GeoJSONCarriesReadableKind
    expected: string("Historic Landmark")
    actual  : <nil>(<nil>)
```
GREEN: `ok  weatherservice/internal/adapters/httpserver 1.142s`.

Callout and popup: `node --test tasks/redesign/checks/map-look-v2-behavior.test.mjs`. RED:
`✖ first pin callout shows the photo and kind before its action opens the panel`,
`✖ selected callout shows its readable kind beneath the name`, `ℹ pass 9` `ℹ fail 2`. GREEN after
using `kind_label`: all 164 unit tests passed at that point.

### 3. Best-time tip for landmarks
`node --test tasks/redesign/checks/best-time.test.mjs`. RED: `✖ historic landmarks share the
quieter-early tip`, `ℹ pass 1` `ℹ fail 1`. GREEN: `ℹ pass 2` `ℹ fail 0`. The wording is the
owner-approved museum tip; the plan's D14 table now lists the added kinds.

### 4. Writer quote as plain text
`TestPlacePanel_WriterQuoteIsPlainTextWithLinkedSource` RED: `missing "<p"` (the quote was a link).
GREEN with the package run above.

### 5. Phone map: collapsed credits, no duplicate label
`node --test tasks/redesign/checks/map-look-v2.test.mjs`. RED:
`✖ compact attribution starts collapsed once the map loads`,
`✖ the selected pin hides its name label because the callout shows it`, `ℹ pass 5` `ℹ fail 2`.
The first fix broke the core map fakes (`TypeError: target.querySelector is not a function`); the
fake element gained `querySelector() { return null; }`. GREEN: `ℹ tests 167` `ℹ pass 167`.

### 6–7. Phone card and day row; link titles; stays heading; callout chevron
Runtime checks added to `tasks/redesign/checks/look-v3-shell.browser.mjs` (375 and 1440 px):

| RED | Fix |
| --- | --- |
| `place title fits on one line: {"titleLines":2,"creditInsidePhoto":false,…}` | credit overlays the photo; phone title 22 px; card padding 12 px (the title column was 142 px for ~145 px of text) |
| `phone day card: one photo, Morning and Afternoon in a row` with `photos: 3, texts: 3` | phone grid: first stop's photo, two stops on a timeline (`li { display: contents }`, base dots removed) |
| `link card titles stay on one line: {"links":[2,1],…,"calloutArrow":12}` | phone link text 11/9 px, no wrap |
| `link card titles are not cut off: {"truncated":["View on Wikipedia"],…}` | tighter link-card padding and icon tile instead of an ellipsis |
| (1440) `Where to stay fits on one line` | "View all →" becomes a tinted pill on the intro line (44 px hit area) |

GREEN:
```
PASS 375: header, hero text bounds, forecast Escape/focus, Belém facts, readable kind, tip, quote, collapsed credits, pills, screenshots
PASS 1440: header, hero text bounds, forecast Escape/focus, Belém facts, readable kind, tip, quote, collapsed credits, pills, screenshots
```

### 8. Bottom nav in screenshots
Not a bug. In a 375×900 viewport scrolled to the end: `{"innerHeight":900,"navTop":835,"navBottom":900,"lastCardBottom":744.39,"position":"fixed"}`.
Full-page captures draw the fixed nav mid-page.

### 9. Search clear cross (owner request; reverses the v2 "no clear ×" decision)
`node --test tasks/redesign/checks/look-v3-header.test.mjs` RED: `✖ search has a labelled clear button that starts hidden`,
`ℹ pass 2` `ℹ fail 1`; GREEN with the full unit run (`ℹ tests 169` `ℹ pass 169`). `hero-phone.browser.mjs` now checks:
no cross when empty, a 44 px cross inside the filled box, the native cancel still hidden, a click empties the field and
the place fields, closes suggestions and returns focus. It passed in the fresh run, but it was written after the
implementation, so it was never seen failing first.

### 10. Static assets are revalidated
The owner still saw no cross on `go run ./cmd/web`, although the server sent the new markup and script. Static files had
only `Last-Modified`, so browsers chose their own cache lifetime and kept the old `search.js`.
`go test ./internal/adapters/httpserver/ -run TestStatic_` RED: `expected: "no-cache"` `actual  : ""`. GREEN:
`ok  weatherservice/internal/adapters/httpserver 1.216s`; `make test lint build`: `0 issues.`

### 11. Notable places toggles again (D21 revised)
The owner asked to be able to unselect Notable places. `node --test tasks/redesign/checks/map-layers.test.mjs` RED:
`✖ upgrades every chip into a button with aria-pressed and keeps labels`, `✖ notable places can be switched off and on again`,
`✖ a remembered notable-off state is applied`, `✖ notable is filled terracotta only while on; on overlays are white`
(`ℹ pass 10` `ℹ fail 6`). GREEN: all unit tests `ℹ tests 170` `ℹ pass 170`. Browser: `look-v3-shell.browser.mjs`
`PASS 375` / `PASS 1440`; `pr7-integration.browser.mjs` `PASS` MapLibre and Leaflet at 375 and 1440 px (notable chip off
hides its pins; all chips off empties the map).

### 12. Map credits open from the keyboard
The final sweep caught `writers-map-preview.browser.mjs` failing after Enter on the ⓘ:
`{"focused":true,"width":24,"credit":"© Mapterhorn | Local fixture map data","detailsVisible":true}`. MapLibre 5's compact
toggle expects `<details open>` to stay set and only `maplibregl-compact-show` to change (its own minimise does the same);
`collapseAttribution` had removed `open` too. `map-look-v2.test.mjs` RED: `✖ compact attribution starts collapsed once the
map loads` (`ℹ pass 6` `ℹ fail 1`) after asserting class-only collapse. GREEN: 170 of 170 unit tests;
`writers-map-preview.browser.mjs` `PASS` MapLibre and Leaflet at 375 and 1440 px; `look-v3-shell.browser.mjs` `PASS 375` / `PASS 1440`.

### 13. Only Notable places is on when a destination loads (D21 addendum)
The owner asked for only the notable pins to be selected when a location loads, with every pill still toggling.
The chips are rendered as spans with each destination, so upgrading a fresh span now resets the choice; the
`sessionStorage` memory is gone. `node --test tasks/redesign/checks/map-layers.test.mjs` RED: `✖ a loaded destination
starts with only notable places on`, `✖ loading another destination resets the chips to notable only`, `✖ other swaps on
the same destination keep the chosen chips`, and five existing tests updated for the new defaults (`ℹ pass 8` `ℹ fail 8`).
GREEN: `ℹ pass 16` `ℹ fail 0`.

Browser checks that expected writer, itinerary or plan pins at load now switch the layer on the way a user does, through a
new `showLayer` helper in `lib/browser.mjs` (look-v3-shell, map-itineraries, map-panel, pr7-acceptance, pr7-integration,
pr8c-journeys). `pr7-integration` asserts the load state: `['true', 'false', 'false', 'false']`.

### 14. View on map scrolls to the map on a phone
With the plan layer now off until pressed, `pr7-acceptance` 7g failed 2 of 6 runs (`Timed out: window.scrollY < 581`); the
original code passed 4 of 4. A probe at 375 px showed the scroll never reached the map in any run:
`{"y":939,"h":1839,"hash":"#itinerary"} 632/1532/252 …` (scrollY / page height / map top). `scrollIntoView` ran while the
map's section was still hidden; the page only moved because it got shorter. The scroll now waits for the hash change.
`node --test tasks/redesign/checks/map-dayplan.test.mjs` RED: `✖ View on map scrolls only once the overview section is
shown` (`1 !== 0`, `ℹ pass 11` `ℹ fail 1`). GREEN: `ℹ tests 172` `ℹ pass 172`. Probe after: `632 → 209 → 176`, the map in
view; `pr7-acceptance` then passed 4 of 4 (`PASS 7g maplibre 375px: … scrolls 905 -> 176`).

## Guarantees

| What is guaranteed | Test | Level | Result |
| --- | --- | --- | --- |
| Every pill toggles, Notable places included | `map-layers.test.mjs`, `pr7-integration.browser.mjs` | Unit + browser | PASS |
| A loaded destination starts with only Notable places on; a new destination resets, other swaps keep the choice | `map-layers.test.mjs`, `pr7-integration.browser.mjs` | Unit + browser | PASS |
| View on map scrolls once the overview section is shown | `map-dayplan.test.mjs`, `pr7-acceptance.browser.mjs` 7g | Unit + browser | PASS |
| Notable is filled only while on; on overlays white; outline inside the pill | `map-layers.test.mjs` | Unit (CSS) | PASS |
| Kind labels: grouped landmarks/religious sites, title case otherwise | `TestPlaceKindLabel_GroupsAndTitleCases` | Unit | PASS |
| Panel meta and GeoJSON carry the readable kind; raw kind still drives icons | `TestPlacePanel_MetaShowsReadableKind`, `TestPlacePanel_GeoJSONCarriesReadableKind` | Integration | PASS |
| Popup and callout show the readable kind | `map-look-v2-behavior.test.mjs` | Unit | PASS |
| Landmarks get the Morning tip; unknown kinds stay hidden | `best-time.test.mjs` | Unit | PASS |
| Quote is plain text, blog credit links to the post | `TestPlacePanel_WriterQuoteIsPlainTextWithLinkedSource` | Integration | PASS |
| Credits collapsed, selected label hidden | `map-look-v2.test.mjs`, `look-v3-shell.browser.mjs` | Unit + browser | PASS |
| Phone title one line, credit on photo, link titles whole, day row, stays heading, chevron size | `look-v3-shell.browser.mjs` | Browser, 375/1440 | PASS |
| Search clear cross: labelled, hidden when empty, clears and refocuses | `look-v3-header.test.mjs`, `hero-phone.browser.mjs` | Unit + browser | PASS |
| Static assets carry `Cache-Control: no-cache` | `TestStatic_AssetsAreRevalidatedOnEveryLoad` | Integration | PASS |

## Suite and coverage
Final code, fresh headless Brave with SwiftShader, 2026-10-15:

| Check | Result |
| --- | --- |
| `make test lint build` (CI-pinned golangci-lint 2.14.0) | `0 issues.`, build OK |
| `go test -cover` httpserver / design-preview | 93.1% / 81.6% |
| `node --test tasks/redesign/checks/*.test.mjs` | `ℹ tests 170` `ℹ pass 170` `ℹ fail 0` |
| `hero-phone`, `look-v3-shell`, `map-dayplan`, `map-itineraries`, `map-panel`, `pr7-acceptance`, `pr7-integration`, `pr8c-journeys`, `writers-map-core`, `writers-map-layout` | EXIT 0 |
| `writers-map-preview` | failed, fixed (see 12), then `PASS` all four engine × width runs |
| `browser-smoke.mjs` (preview on 127.0.0.1:8087, output to a scratch dir) | EXIT 0 |
| `a11y-audit.mjs` (output `tasks/artifacts/look-v3/a11y`) | `31 passed, 0 failed` |

After fix 12 (it touched `writers-map.js`) these were rerun: `writers-map-core`, `hero-phone` EXIT 0; the audit again
`31 passed, 0 failed`; `writers-map-layout` failed once (its output was discarded, so the cause is unknown) and then passed
three runs in a row. Its 12 `FAIL PAIR` lines are the audit's advisory `component-labelled` itinerary borders
(`a11y-audit.mjs:300-302`), outside this change.

`TestWriterMapPreview_LisbonPhotosReachBrowser` failed once inside `make test` right after the audit (2.05 s), then passed
6 runs alone and the next full `make test`. It starts a real preview process, so it is timing-sensitive under load.

After 13–14 (final code): `node --test` `ℹ tests 172` `ℹ pass 172`; all 11 browser checks EXIT 0; audit `31 passed, 0
failed`; `make test lint build` `0 issues.`, exit 0.

## Known gaps
- A newly generated plan stays hidden on the map until the Your plan pill or View on map is pressed, because only Notable places starts on.
- `writers-map-layout.browser.mjs` and `TestWriterMapPreview_LisbonPhotosReachBrowser` each failed once and then passed repeatedly; treat both as possibly flaky.
- The offline preview map is blank and fixture photos are synthetic, so streets, labels and real photography can't match the mockups.
- On a phone the meta row ("Historic Landmark · Belém, Lisbon") wraps to two lines in the 152 px column.
- The phone day card has no separate round › button; "View all ›" in the heading covers it, which avoids a duplicate link.
- `TravelMapLayers.createMenu` in `map-layers.js` is unused since the Layers button went (D7). Left in place, not deleted.
