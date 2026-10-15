# Look v3 evidence

Source: [plan](../../tasks/look-v3-plan.md), [tasks V1–V13](../../tasks/look-v3-todo.md).

Journeys: travellers see a slim city/weather banner and clear layer states; inspect photos,
facts, writer stories and directions together; preview their day, stays and videos; and use
one scrolling phone page with icon navigation. Decisions D1–D20 govern implementation.

Detailed RED/GREEN evidence is split by file ownership:

- [Preview fixture and local photo endpoints](look-v3-fixture.tdd.md)
- [Map layers, grid, chrome and both map engines](look-v3-map.tdd.md)
- [Place panel, hours, tips, writers and phone selection](look-v3-panel.tdd.md)
- [Preserved regression journeys](look-v3-regressions.tdd.md)

## Header, banner and teasers

`node --test tasks/redesign/checks/look-v3-header.test.mjs`:

RED before changing the header:
```
✖ header follows brand search tabs bookmark and retains accessible brand
✖ phone navigation has four icons including a bed SVG
ℹ tests 2
ℹ pass 0
ℹ fail 2
```
GREEN:
```
✔ header follows brand search tabs bookmark and retains accessible brand
✔ phone navigation has four icons including a bed SVG
ℹ tests 2
ℹ pass 2
ℹ fail 0
```

`go test ./internal/adapters/httpserver/ -run TestWeatherDisplay_SlimStripAndForecastPopover`
failed before production edits with `does not contain "popover"`; its additional assertions
cover forecast button, planning link, all metrics, rounded temperatures and removed caption.
The no-forecast regression now checks absence of the forecast *list*, allowing a details chip.

`go test ./internal/adapters/httpserver/ -run TestOverviewTeasers` initially failed with
`render: template overview_teasers does not exist`. The tests cover planned, unplanned and
empty states, destination labels, section links, first hotel and video absence.

GREEN after implementing both templates:
```
$ go test ./internal/adapters/httpserver/ -run 'TestOverviewTeasers|TestWeatherDisplay'
ok  weatherservice/internal/adapters/httpserver 0.569s
```

`node --test tasks/redesign/checks/teasers.test.mjs` initially failed with
`ENOENT: no such file or directory, open 'public/redesign/teasers.js'`.
A subsequent generated-hotel regression failed with `TypeError: listeners[name] is not a function`
before adding the swap handler. GREEN:
```
✔ suggested day uses three highest ranked notable places
✔ generated day replaces suggestion and later place loads preserve the plan
✔ generated plan updates the hotel preview from the rendered stays
ℹ tests 3
ℹ pass 3
ℹ fail 0
```

`node tasks/redesign/checks/look-v3-shell.browser.mjs` caught the phone tagline extending to
278px below a hero ending at 244px. Resetting the inherited sheet-era banner grid fixed it.
A further run caught desktop tabs ending at 962px while the bookmark began at 1340px;
removing legacy flex ordering from the desktop grid corrected the header.

GREEN:
```
PASS 375: header, hero text bounds, forecast Escape/focus, Belém facts, screenshots
PASS 1440: header, hero text bounds, forecast Escape/focus, Belém facts, screenshots
```
The native HTML popover provides Escape dismissal and returns focus to the trigger.

## Guarantees

| What is guaranteed | Test | Level | Result |
| --- | --- | --- | --- |
| Search placeholder, DOM order, brand name and phone icons | `look-v3-header.test.mjs` | Unit | PASS |
| Weather metrics, rounded temperatures, popover and no caption | `TestWeatherDisplay_*` | Template integration | PASS |
| Planned/unplanned/empty teasers and section targets | `TestOverviewTeasers_*` | Template integration | PASS |
| Suggested top three, generated day and hotel updates | `teasers.test.mjs` | Unit | PASS |
| Hero dimensions/text bounds and forecast focus | `look-v3-shell.browser.mjs` | Browser, 375/1440 | PASS |
| Hash views, mounted map and search navigation | `hero-phone.browser.mjs` | Browser, 375/1440 | PASS |
| Inline card, share and return to city | `map-panel.browser.mjs` | Browser, 375/1440 | PASS |
| Short map, panel width, teasers and 44px contained controls | `writers-map-layout.browser.mjs` | Browser, both engines, 375/1440 | PASS |

## Tokens

Measured with the audit's `ratio` helper in `tasks/redesign/checks/lib/browser.mjs`:

| Foreground / background | Ratio |
| --- | --- |
| `#6B4FA8` writer / `#F1EEF8` writer tint | 5.54:1 |
| `#A64027` accent / `#F8E6E0` accent tint | 5.14:1 |
| White / `#A64027` accent | 6.21:1 |

All new token text pairs meet 4.5:1 and icon pairs meet 3:1.

## Integration and visual evidence

Validation is recorded below after final integration. No commits or pushes were made.

Known visual limits: the deterministic preview uses synthetic category photos and a blank
local map style, so streets, district/water labels and real place photography cannot match
the mockups in offline screenshots. Multiple real photos, price, writer excerpts, stay
prices and video duration remain Later by decision. The scale uses its real measured distance
rather than hardcoding 1km. The phone map is 220px to contain five 44px controls vertically,
within V6's accepted 190–220px range. Tips retain the exact approved D14 wording.

## Final verification

All V1–V13 tasks are complete. Three agents worked with exclusive file ownership and the
root agent integrated the shared page. Existing unrelated working-tree changes were preserved.

The installed golangci-lint 2.4.0 was built with Go 1.26 and could not load this Go 1.27
project. The CI-pinned 2.14.0 was built into `/tmp/traveltab-look-v3-tools`; no repository
dependency or system installation was changed.

`PATH=/tmp/traveltab-look-v3-tools:$PATH make test lint build` passed. Final output:
```text
ok  	weatherservice/internal/writermap	1.878s
ok  	weatherservice/internal/writermap/writermaptest	1.382s
ok  	weatherservice/writerdata	1.332s
golangci-lint run ./...
0 issues.
go build -o bin/traveltab ./cmd/web
```

`go test -cover ./internal/adapters/httpserver/... ./cmd/design-preview/...`:
```text
ok  	weatherservice/internal/adapters/httpserver	1.020s	coverage: 93.0% of statements
ok  	weatherservice/cmd/design-preview	1.064s	coverage: 81.6% of statements
```

`node --test tasks/redesign/checks/*.test.mjs`:
```text
ℹ tests 162
ℹ suites 0
ℹ pass 162
ℹ fail 0
ℹ cancelled 0
ℹ skipped 0
ℹ todo 0
ℹ duration_ms 131.780083
```

The last route-style regression proved the old 3px route failed the intended 2px requirement:
`node --test tasks/redesign/checks/map-dayplan.test.mjs` initially had `9 pass, 1 fail`
with `3 !== 2`, then passed all 10 tests after the dotted route changed to width 2 and dash
`[1, 1.5]`. The existing pin test's matching style expectation was updated. Final runtime:
```text
PASS maplibre 375px: 10 orange numbered pins, popup and Escape, regenerate without duplicates, day toggle, View on map, clear, console clean
PASS maplibre 1440px: 10 orange numbered pins, popup and Escape, regenerate without duplicates, day toggle, View on map, clear, console clean
PASS leaflet 375px: 10 orange numbered pins, popup and Escape, regenerate without duplicates, day toggle, View on map, clear, console clean
PASS leaflet 1440px: 10 orange numbered pins, popup and Escape, regenerate without duplicates, day toggle, View on map, clear, console clean
```

The empty hotel-preview regression additionally failed with `''` instead of
`'No matching places to stay were returned.'`; all four teaser tests passed after the swap
handler began rendering the existing no-stays note. The hero browser journey now verifies
that generated day and hotel previews reflect the actual plan.

`CDP_PORT=9231 OUTPUT_DIR="$PWD/tasks/artifacts/look-v3/a11y" node tasks/redesign/checks/a11y-audit.mjs`:
```text
PASS writers-map-popup-leaflet@1440: named dialog popup, focus on Details, no unnamed controls, Escape closes and returns focus to the pin
PASS js-exceptions: no uncaught JavaScript exceptions during the audit

31 passed, 0 failed
```

The complete audit measures text/icon contrast, names, headings, keyboard order, focus,
announcements, reduced motion, reflow, target sizes and both map engines at 375 and 1440px.
Axe was not installed; the audit's built-in CDP/pixel checks ran. Labelled itinerary borders
are reported as advisory; all required contrast checks pass. Earlier audit failures from
fallback/heading overlap and the obsolete phone-popup journey were fixed. A concurrent-edit
run also captured a JavaScript syntax error; the final frozen-source run had no exceptions.

Every `*.browser.mjs` check and `browser-smoke.mjs` ran, including both engines where supported.
The first sweep's stale assumptions are preserved in `validation/browser-sweep.log`; their
rewritten passing journeys are recorded in the map/panel/regression reports. Final checks
also verify generated teasers and the selected pin/photo callout fit within the phone map.

Final [screenshots and task comparisons](../../tasks/artifacts/look-v3/README.md) were captured
with browser cache disabled and real local place-photo responses. The screenshot harness
restores browser foreground after full-page capture because headless capture otherwise
stalled camera animation; a bounds assertion also caught and fixed the callout's vertical offset.

Changed areas: destination header/banner/weather, map layers/grid/chrome, place/facts/writer
cards, teaser previews, inline phone navigation, local preview fixtures, regression checks,
this evidence and design/task documentation. All changes remain uncommitted.
