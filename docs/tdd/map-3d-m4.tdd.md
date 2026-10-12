# M4 day plan drawing API evidence

Source: [tasks/map-3d-todo.md](../../tasks/map-3d-todo.md), M4. User journeys: As a traveller, I want my generated day plan to appear as green numbered pins on the 3D map; I want to choose a day, regenerate or clear the plan, and use View on map; I want the same behavior when the map falls back to Leaflet.

## RED and GREEN

| Behavior | Command | RED excerpt | GREEN excerpt |
| --- | --- | --- | --- |
| Draw focusable numbered pins and the day control through `context.draw` | `node --test tasks/redesign/checks/map-dayplan.test.mjs` | `✖ uses drawing API for green, numbered, focusable pins and a day control`; `TypeError: Cannot read properties of undefined (reading 'layerGroup')` | `✔ uses drawing API for green, numbered, focusable pins and a day control` |
| Regenerate, toggle, clear and re-add without duplicate drawings | Same command | `✖ generate, regenerate, day toggle, clear and enable leave one current set`; `✖ re-adding on another map context draws once and clears the old context`; both failed at Leaflet `layerGroup` | `✔ generate, regenerate, day toggle, clear and enable leave one current set`; `✔ re-adding on another map context draws once and clears the old context` |
| View on map activates the plan chip | Same command | Existing behavior passed in the RED run: `✔ View on map activates the plan chip and scrolls to the shell` | `✔ View on map activates the plan chip and scrolls to the shell` |

The final unit run reported `tests 6`, `pass 6`, `fail 0`. The first GREEN attempt had two test-harness failures because VM arrays have a different realm; after normalizing those assertions, the same behavior tests passed.

## Runtime

`CDP_PORT=9329 node tasks/redesign/checks/map-dayplan.browser.mjs` passed in an isolated Brave profile with software WebGL:

```text
PASS maplibre 375px: 10 green numbered pins, regenerate without duplicates, day toggle, View on map, clear, console clean
PASS maplibre 1440px: 10 green numbered pins, regenerate without duplicates, day toggle, View on map, clear, console clean
PASS leaflet 375px: 10 green numbered pins, regenerate without duplicates, day toggle, View on map, clear, console clean
PASS leaflet 1440px: 10 green numbered pins, regenerate without duplicates, day toggle, View on map, clear, console clean
```

The first browser run with GPU disabled timed out waiting for MapLibre. A second run with software WebGL found a wrong test expectation: View on map preserves the chosen day, so it correctly restored 4 pins rather than all 10. The corrected test passed on both engines.

| Guaranteed behavior | Test name | Scope | Result |
| --- | --- | --- | --- |
| One layer reports valid plan coordinates | `registers one layer named plan and reports valid stops after swaps` | Unit | Pass |
| Missing core or shell is safe | `does nothing when the core or map shell is missing` | Unit | Pass |
| Drawing API supplies green, numbered, focusable pins and control | `uses drawing API for green, numbered, focusable pins and a day control` | Unit | Pass |
| Generate, regenerate, day toggle, clear, and enable do not duplicate pins | `generate, regenerate, day toggle, clear and enable leave one current set` | Unit | Pass |
| Re-add on another context clears the old context | `re-adding on another map context draws once and clears the old context` | Unit | Pass |
| View on map activates the chip and scrolls | `View on map activates the plan chip and scrolls to the shell` | Unit | Pass |
| Both engines at 375 and 1440 px work with clean consoles and no overflow | `map-dayplan.browser.mjs` | Browser integration | Pass |

No Go package changed, so the Go package coverage threshold does not apply. Browser fixtures are local; the pinned engine libraries still load from their CDNs.

## Project gate

`PATH=/private/tmp/traveltab-tools:$PATH make test lint build` passed outside the sandbox: all Go packages passed, `golangci-lint run ./...` reported `0 issues.`, and `go build -o bin/traveltab ./cmd/web` completed. The first sandboxed attempt failed because local HTTP test servers could not bind (`listen tcp 127.0.0.1:0: bind: operation not permitted`); the same command passed with loopback access.
