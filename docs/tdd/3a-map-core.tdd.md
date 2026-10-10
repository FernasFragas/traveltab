# 3a — Writers' Map Core: TDD Evidence

**Source task:** `tasks/todo.md` 3a and `tasks/plan.md` PR 3. The browser API follows `docs/writers-map-contract.md`.

## User journeys

- As a traveler, I want a map centered on the searched destination, so that I can explore its places.
- As a keyboard user, I want a list of places with usable buttons, so that I can select a place without interacting with markers.
- As a traveler, I want loading, partial and unavailable states, so that I know when place information is incomplete.
- As a traveler changing destinations, I want the map to update without duplicate map instances or old responses replacing current places.
- As a map feature developer, I want layer registration, toggling and the agreed events, so that later layers can attach to the same map.

## RED and GREEN

The first test was written before the core file. `node --test tasks/redesign/checks/writers-map-core.test.mjs` failed with `ENOENT: no such file or directory, open '.../public/redesign/writers-map.js'`. After implementing the core, the same command reported:

```text
✔ loads pinned Leaflet assets when a map shell appears
✔ renders a keyboard list, emits selection, and shows partial data status
✔ reuses the map across swaps and drops an old response
✔ toggles registered layers and reports unavailable places
ℹ pass 4
ℹ fail 0
```

The tests also caught an implementation mistake during GREEN: the first map did not attach a registered layer when the place request failed. It was fixed before the passing run.

For the keyboard fallback when Leaflet fails, `node --test tasks/redesign/checks/writers-map-core.test.mjs` was RED with `TypeError: Cannot read properties of undefined (reading 'children')` in `keeps the place list available when Leaflet cannot load`. After loading the list even when Leaflet fails, the same command reported `✔ keeps the place list available when Leaflet cannot load`, `ℹ pass 5`, `ℹ fail 0`.

## Guarantees

| What is guaranteed | Test name | Type | Result |
| --- | --- | --- | --- |
| Pinned Leaflet JS and CSS are requested with SRI | `loads pinned Leaflet assets when a map shell appears` | unit | PASS |
| A keyboard button selects a GeoJSON feature and partial data is announced | `renders a keyboard list, emits selection, and shows partial data status` | unit | PASS |
| A destination swap reuses the map and ignores an old response | `reuses the map across swaps and drops an old response` | unit | PASS |
| Layers can toggle, unavailable data leaves the map active, plan changes emit an event | `toggles registered layers and reports unavailable places` | unit | PASS |
| The keyboard list remains available after a Leaflet load failure | `keeps the place list available when Leaflet cannot load` | unit | PASS |
| The shell fits 375 and 1440 px, list button focuses, and the console has no exceptions | `writers-map-core.browser.mjs` | browser integration | PASS |

## Browser and full checks

`node tasks/redesign/checks/writers-map-core.browser.mjs` against `cmd/design-preview`, using an isolated headless Brave session, reported:

```text
PASS 375px: map shell layout, keyboard list, status, console
PASS 1440px: map shell layout, keyboard list, status, console
```

`PATH=/private/tmp/traveltab-tools:$PATH make test lint build` passed: all Go test packages reported `ok`, `golangci-lint run ./...` reported `0 issues.`, and `go build -o bin/traveltab ./cmd/web` completed. The standard installed linter was built with Go 1.26 and could not analyze this Go 1.27.2 module, so the existing local Go 1.27-compatible binary was used.

No Go package changed, so Go package coverage is not applicable. Node's `--experimental-test-coverage` does not instrument the VM-evaluated map script; its 100% empty-file report is not used as coverage evidence.

## Known gaps

- Task 3b has not yet supplied the production shell, script inclusion or GeoJSON endpoint. The browser check injects the shell into the real design-preview page and uses a Leaflet stand-in, then verifies DOM behavior and layout. Task 3c will exercise the integrated map and local fixture tiles.
- This subtask draws the basemap and list. Place markers and the place panel are later UI subtasks.
