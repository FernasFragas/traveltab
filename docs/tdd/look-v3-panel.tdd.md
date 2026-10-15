# Look v3 panel, facts and writers

Source: `tasks/look-v3-todo.md` V8, V9, V10 and V12 inline card behavior.

Journeys:
- As a traveller, I want a place's photos, facts and links together so I can decide without leaving the page.
- As a traveller on a phone, I want an inline place card and natural scrolling so I do not juggle sheets.
- As a traveller, I want writer post titles and attribution so I can open the original stories.

## Template behavior

Command: `go test ./internal/adapters/httpserver/ -run TestPlacePanel -count=1`.

RED excerpt, before the new template:

```text
--- FAIL: TestPlacePanel_ListsEveryWriterPost
Messages: missing "From our writers"
--- FAIL: TestPlacePanel_V3ActionsAndSinglePhoto
FAIL weatherservice/internal/adapters/httpserver 0.420s
```

GREEN after the template change:

```text
ok  weatherservice/internal/adapters/httpserver 0.561s
```

## Facts

Command: `node --test tasks/redesign/checks/opening-hours.test.mjs tasks/redesign/checks/best-time.test.mjs`.

RED: missing implementation files, intentionally referenced by new tests:

```text
Error: ENOENT: no such file or directory, open 'public/redesign/best-time.js'
Error: ENOENT: no such file or directory, open 'public/redesign/opening-hours.js'
ℹ tests 2
ℹ pass 0
ℹ fail 2
```

GREEN:

```text
✔ kind rules are labelled tips, case insensitive and hidden for unknown kinds
✔ simple range and closed weekdays
✔ multiple ranges, unknown syntax and empty input
ℹ tests 3
ℹ pass 3
ℹ fail 0
```

## Interactive regression checks

Command: `node --test tasks/redesign/checks/map-panel.test.mjs tasks/redesign/checks/map-look-v2-behavior.test.mjs tasks/redesign/checks/opening-hours.test.mjs tasks/redesign/checks/best-time.test.mjs`.

The obsolete monogram, writer count card, sheet and focus trap expectations were rewritten for the agreed inline card, writer quote and expansion behavior. Their original failures were observed; carousel interaction coverage was added after implementation and is regression coverage, not a pre-implementation RED claim.

```text
✔ back returns the panel to its city state and announces closure
✔ writer See all toggles the list and accessible expanded state
✔ carousel loops in both directions and announces the current photo
✔ phone place card stays inline and auto selection does not steal focus
✔ inline phone card leaves Tab in the natural document focus order
ℹ tests 33
ℹ pass 33
ℹ fail 0
```

| Guaranteed behavior | Test | Level | Result |
| --- | --- | --- | --- |
| Place actions, directions, no desktop close and no single-photo arrows | TestPlacePanel_V3ActionsAndSinglePhoto | Integration | PASS |
| Writers' full post list and attribution | TestPlacePanel_ListsEveryWriterPost | Integration | PASS |
| No invented hours or Wikipedia link | TestPlacePanel_PlaceWithoutHoursOrWikipediaHasNoEmptyRows | Integration | PASS |
| Valid weekly hours and closed days | simple range and closed weekdays | Unit | PASS |
| Preserve unsupported rules verbatim | multiple ranges, unknown syntax and empty input | Unit | PASS |
| Kind tips with explicit heuristic label | kind rules are labelled tips, case insensitive and hidden for unknown kinds | Unit | PASS |
| Carousel wraps and announces position | carousel loops in both directions and announces the current photo | Unit | PASS |
| Writer expansion announces state | writer See all toggles the list and accessible expanded state | Unit | PASS |
| Back restores city and signals closure | back returns the panel to its city state and announces closure | Unit | PASS |

Runtime validation, screenshots and full CI verification are recorded in the integrated `look-v3.tdd.md`. No Go production package changed in this slice; the updated template is exercised through HTTP integration tests. Only one photo currently exists per place; multi-photo behavior is covered by synthetic DOM interaction tests. No price or last-entry data is invented.

## Save icon regression

Command: `node --test tasks/redesign/checks/map-saved.test.mjs`.

New label-preservation test RED:

```text
✖ saving preserves the icon and updates only its label
'Save' !== 'Saved'
ℹ tests 15
ℹ pass 14
ℹ fail 1
```

GREEN after updating only the marked text span:

```text
✔ saving preserves the icon and updates only its label
ℹ tests 15
ℹ pass 15
ℹ fail 0
```

Package coverage command: `go test -cover ./internal/adapters/httpserver/...` reached `coverage: 93.0% of statements`; this first integrated run failed obsolete fixture/banner expectations during concurrent work. It is not recorded as a passing full suite.

## Phone pin selection

Command: `node --test tasks/redesign/checks/map-panel.test.mjs`.

RED for direct phone selection:

```text
✖ phone pin selection opens the inline card directly without a popup step
+ []
- ['Q2']
ℹ tests 20
ℹ pass 19
ℹ fail 1
```

GREEN after selecting immediately on phone pin clicks:

```text
✔ phone pin selection opens the inline card directly without a popup step
ℹ tests 20
ℹ pass 20
ℹ fail 0
```

Before that final phone change, `node tasks/redesign/checks/map-panel.browser.mjs` passed:

```text
PASS 375px: inline place card, share and return to city
PASS 1440px: inline place card, share and return to city
```

Runtime Lisbon screenshots: `tasks/artifacts/look-v3/V8/375-panel.png` and `1440-panel.png`. Belém's range rendered `10:00 – 18:30`; document widths were 375 and 1425 respectively. Final direct-pin runtime verification is recorded with the integrated checks.

## Place location metadata

Command: `go test ./internal/adapters/httpserver/ -run TestPlacePanel_MetaIncludesAreaAndCity -count=1`.

RED before exposing the requested destination city to the template:

```text
--- FAIL: TestPlacePanel_MetaIncludesAreaAndCity
"Santana</span>" does not contain "Santana, Funchal"
FAIL weatherservice/internal/adapters/httpserver 0.437s
```

The slug parser supplies lowercase city names, so the display value uses Unicode title casing. GREEN:

```text
ok  weatherservice/internal/adapters/httpserver 0.483s
```

The accessibility audit's map checks were updated to validate direct inline phone selection and Escape restoring map focus; desktop still validates popup focus and dismissal. Contrast diagnostics now log each failed pair and optionally accept `AUDIT_STATES` alongside `AUDIT_ONLY=contrast-quick` for focused investigation. The diagnostic runs were interrupted when the root agent identified and took ownership of the fallback hero contrast fixes; no diagnostic pass is claimed.

## Final integrated accessibility verification

Targeted fallback verification:

`CDP_PORT=9231 AUDIT_ONLY=contrast-quick AUDIT_STATES=brokenimage-failed,photoerror OUTPUT_DIR=/tmp/look-v3-a11y-targeted node tasks/redesign/checks/a11y-audit.mjs`

```text
4 passed, 0 failed
```

Final full run after production files were frozen:

`CDP_PORT=9231 OUTPUT_DIR="$PWD/tasks/artifacts/look-v3/a11y" node tasks/redesign/checks/a11y-audit.mjs > /tmp/look-v3-a11y-verified.log 2>&1`

```text
PASS writers-map-inline-leaflet@375: pin opens inline card, focus on title, no modal or overflow; Escape restores city and map focus
PASS writers-map-popup-leaflet@1440: named dialog popup, focus on Details, no unnamed controls, Escape closes and returns focus to the pin
PASS js-exceptions: no uncaught JavaScript exceptions during the audit

31 passed, 0 failed
```

MapLibre checks passed at both widths too. Complete results, contrast pairs and screenshots are in `tasks/artifacts/look-v3/a11y/`. Existing labelled itinerary button fill/border contrast remains an advisory under the audit's established policy; its identifying text passes. No failures were suppressed.
