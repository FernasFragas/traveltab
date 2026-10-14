# Map look v2 implementation evidence (partial)
Journeys: tasks/map-look-v2-todo.md traveller recognition, layer identification, panel story, phone sheet, save confirmation/undo.

Initial structural behavior RED command: `node --test tasks/redesign/checks/map-look-v2.test.mjs`
Output /tmp/map-look-v2-red.log:
```
✖ brand has accessible two-line wordmark and search submits without a button
✖ saved places have a header list and polite undo toast
✖ panel presents story and cards without tabs or peek
✖ map uses card layout, full-map link and updated palette
ℹ tests 4
ℹ pass 0
ℹ fail 4
```
Panel RED command: `go test ./internal/adapters/httpserver/ -run 'TestPlacePanel_(StoryAndDirectionsCards|ShowsPartsInMockupOrder)'`
Actual output /tmp/map-look-v2-panel-red.log: `FAIL weatherservice/internal/adapters/httpserver 0.583s`; StoryAndDirectionsCards reports missing map-panel-bookmark, map-panel-writer-card, directions and post tile; old tablist still present.

Saved runtime RED command: `node --test tasks/redesign/checks/map-saved.test.mjs`
```
✖ saving shows a polite toast and Undo removes just that place
✖ header saved list has an empty state, names and Escape restores focus
✖ toast expires and a replacement cancels the previous timer
ℹ tests 12
ℹ pass 9
ℹ fail 3
```
Saved GREEN same command, /tmp/map-look-v2-saved-green.log:
```
✔ saving shows a polite toast and Undo removes just that place
✔ header saved list has an empty state, names and Escape restores focus
✔ toast expires and a replacement cancels the previous timer
ℹ tests 12
ℹ pass 12
ℹ fail 0
```

## Final focused evidence

Writer-source names RED: `go test ./internal/adapters/httpserver/ -run TestPlacePanel_HasExistingWriterSourceNamesForItineraryCard`.
Actual log `/tmp/map-look-v2-source-names-red.log`: missing `data-writer-source-host="www.earthtrekkers.com" data-writer-source-name="Earth Trekkers"`, `FAIL weatherservice/internal/adapters/httpserver 0.607s`.
GREEN after embedding existing allowlist in template data (no GeoJSON contract change): `go test ./internal/adapters/httpserver/ -run 'TestPlacePanel|TestIndex_BrandAndSearchField'`.
Actual `/tmp/map-look-v2-panel-green.log`: `ok weatherservice/internal/adapters/httpserver 0.374s`.

Selected-ring regression chronological RED (review agent observed directly before fix; no log file): `node --test --test-name-pattern='second pin' tasks/redesign/checks/map-look-v2-behavior.test.mjs` reports one failure in `opening a second pin callout leaves only the latest pin selected`, `true !== false` (first pin remains selected). GREEN same command actual `/tmp/map-look-v2-selection-green.log`: one test, one pass, zero failures.

Compass action/order and contrast: `node --test tasks/redesign/checks/writers-map-core.test.mjs tasks/redesign/checks/map-pins.test.mjs` actual `/tmp/map-look-v2-controls-contrast.log`: 65 tests,65 pass,0 fail. Tests guarantee compass resets bearing to0 and precedes locate/3D/zoom at top right. Numeric contrast test covers pin fill vs dark map ≥3:1, white category icons ≥3:1, plan numbers ≥4.5:1, grey cluster dark counts ≥4.5:1, white labels ≥4.5:1. These additional outcome tests were added after initial structural RED; do not describe them as independent chronological RED.
Final palette: notable #cf795b; nature #459b6b; writers and itineraries #a582d4; plan #a56824; clusters #9197a1 with #202a32 text; labels white on #202a32ed.

Saved runtime expanded checks: `node --test tasks/redesign/checks/map-saved.test.mjs` actual `/tmp/map-look-v2-saved-final.log`:14 tests,14 pass,0 fail. Guarantees localStorage QID compatibility, save toggling, planner request includes only savedQIDs, Add to plan request and navigation, polite Undo toast removing only last saved place, timeout replacement/expiry, empty/header list and Escape focus, cross-city destination reload waiting for target QID across partial responses, explicit unavailable legacy entry state. Cross-city tests were added after implementation; report as verification, not chronological RED.

Final all-unit command: `node --test tasks/redesign/checks/*.test.mjs` actual `/tmp/map-look-v2-units-final.log`:141 tests,141 pass,0 fail.

Additional behavior file `map-look-v2-behavior.test.mjs` verifies real itinerary `days[].qids` precedence, deduplicated blog names, mentions fallback, none fallback, first click photo/area/arrow callout before panel, missing-photo callout, sheet drag and keyboard state, and single selected ring. Reviewer replayed exact pre-task production baseline retrospectively for initial itinerary/callout failures; this is negative-control replay, not chronological test-first evidence. Root has reviewer excerpts.

Scope exceptions: route styling required map-itineraries.js; full preview place data lives in cmd/design-preview/writersmap.go rather than fixture.go; existing allowlist names require a template-only addition in internal/adapters/httpserver/writermap.go. Shared frozen GeoJSON shape unchanged. All production files edited by the implementation owner; browser agent owns browser checks and runtime validation.

Browser chronological RED→GREEN fixes: real Enter/Space on phone handle initially failed; explicit keydown now toggles. Header-list Escape initially allowed document sheet handler to steal focus; saved handler prevents default and sheet handler respects it. Browser agent captured tests and screenshots at375/1440 for final evidence.

## Final keyboard edge correction
Browser review found Hours summary excluded from the phone focus cycle when no posts follow Directions. Chronological RED command: `node --test --test-name-pattern='Hours summary' tasks/redesign/checks/map-look-v2-behavior.test.mjs`; `/tmp/map-look-v2-hours-keyboard-red.log`: one test,zero pass,one fail, `Tab from Directions must reach the following Hours summary`, `true !== false`.
After adding native `summary` to the sheet's focusable selector, same command GREEN `/tmp/map-look-v2-hours-keyboard-green.log`: one test,one pass,zero fail. Guarantees Tab from Directions reaches Hours and Tab from last Hours summary wraps to handle.
Final full JS unit rerun `node --test tasks/redesign/checks/*.test.mjs`: `/tmp/map-look-v2-units-final.log` now142 tests,142 pass,zero fail.
Final PR7 integration cluster browser check derives cluster count from rendered native clusters and bounds it by current writer-feature count, then verifies closer camera zoom and smaller clusters/individual pins, accommodating enhanced Funchal fixture.
