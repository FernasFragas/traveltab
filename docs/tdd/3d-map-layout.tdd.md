# 3d — Writers' Map card layout

Source: [tasks/todo.md](../../tasks/todo.md), PR 3 subtask 3d; the Funchal mockup in `docs/ideas/Funchal Travel Guide Dashboard.png`.

## Journeys

- As a traveler on a desktop, I want the map and place panel beside each other so I can inspect a place without losing the map.
- As a traveler on a phone, I want the panel below the map without horizontal scrolling.
- As a traveler, I want the map title, four-layer legend, and an explanation of the layers before I explore the map.
- As a developer checking the 3d layout locally, I want an opt-in satellite preview so the page matches the saved screenshot without browser overrides.

## Red and green

| Behavior | Command | RED excerpt | GREEN excerpt |
| --- | --- | --- | --- |
| Header explanation, four-layer row, desktop and phone geometry, empty panel | `CDP_PORT=9227 node tasks/redesign/checks/writers-map-preview.browser.mjs` | `AssertionError [ERR_ASSERTION]: The input did not match the regular expression /notable places.*writers.*day plan.*itineraries/i. Input: ''` | `PASS 375px: writer data, base-only and unavailable swaps, one map, keyboard list, local tiles`; `PASS 1440px: writer data, base-only and unavailable swaps, one map, keyboard list, local tiles` |
| Opt-in preview omits fixture tile overrides | `go test ./cmd/design-preview/ -run TestWriterMapPreview_LiveTilesOptIn -count=1` | `cmd/design-preview/fixture_test.go:150:81: too many arguments in call to newPreviewServer` | `ok  weatherservice/cmd/design-preview 0.663s` |

## Guarantees

| What is guaranteed | Test name | Level | Result |
| --- | --- | --- | --- |
| Info control explains the four layers; the legend names them | `writers-map-preview.browser.mjs` | Browser | PASS at 375 and 1440 px |
| Desktop card spans the overview, map uses about 65%, and panel matches map height | `writers-map-preview.browser.mjs` | Browser | PASS at 1440 px |
| Phone panel stacks below the map and page has no horizontal overflow | `writers-map-preview.browser.mjs` | Browser | PASS at 375 px |
| Destination swaps retain one map and the keyboard place list | `writers-map-preview.browser.mjs` | Browser | PASS at 375 and 1440 px |
| Live-tile opt-in uses the production tile URL while fixture search data stays local | `TestWriterMapPreview_LiveTilesOptIn` | Integration | PASS |

`go test -cover ./cmd/design-preview/...` returned `coverage: 80.7% of statements`. `PATH=/tmp/traveltab-3e-bin:$PATH make test lint build` exited 0: all Go packages passed, lint reported `0 issues.`, and the web app built. The alternate linter is needed because the installed binary was built with Go 1.26 while this module targets Go 1.27.2. The browser check also reported no console errors.

The root screenshot `3d-layout-map.png` was captured from `go run ./cmd/design-preview -scenario writers-map -live-tiles` at 1440 px. The capture reported an EOX satellite tile URL, EOX attribution, and `browserErrors: []`. The image is 1130 × 705 px. The fixture has no map pins or populated place panel yet; those are owned by tasks 7a and 7f. The legend is visual until 7f adds layer toggles.
