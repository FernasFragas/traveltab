# 3e — satellite basemap

Source: [tasks/todo.md](../../tasks/todo.md), task 3e, and the map decision in
[tasks/plan.md](../../tasks/plan.md). Journeys:

- As a traveler, I want the writers' map to show satellite imagery with visible source credit, so I can see the terrain and know where the image came from.
- As the site operator, I want to supply another tile URL and credit for fixtures or a later provider, so the map source can change without rewriting the map.
- As a traveler, I want OSM tiles if the chosen imagery fails, so the map remains usable.

## RED → GREEN

- Default EOX tiles and credit: `node --test --test-name-pattern='freely licensed satellite|keeps configured tiles' tasks/redesign/checks/writers-map-core.test.mjs` was RED with `✖ uses the freely licensed satellite tiles and their required credit by default`; the actual URL was `https://tile.openstreetmap.org/{z}/{x}/{y}.png`. The same command was GREEN with `✔ uses the freely licensed satellite tiles and their required credit by default`.
- Configured URL, credit and fallback: the same RED command reported `✖ keeps configured tiles and attribution and falls back to OSM after an imagery error`; the URL remained `/tiles/{z}/{x}/{y}.png`. The same command was GREEN with `✔ keeps configured tiles and attribution and falls back to OSM after an imagery error`.
- CC BY 4.0 link: `node --test --test-name-pattern='freely licensed satellite' tasks/redesign/checks/writers-map-core.test.mjs` was RED with `AssertionError` because the credit did not contain `https://creativecommons.org/licenses/by/4.0/`. The same command was GREEN with `✔ uses the freely licensed satellite tiles and their required credit by default`.
- Explicit OSM URL credit: `node --test --test-name-pattern='credits OSM when' tasks/redesign/checks/writers-map-core.test.mjs` was RED with `✖ credits OSM when its URL is configured without an explicit credit`; actual credit was EOX's. The same command was GREEN with `✔ credits OSM when its URL is configured without an explicit credit`.

## Verification

| What is guaranteed | Test name | Kind | Result |
| --- | --- | --- | --- |
| EOX 2016 WMTS is the default with its source and licence credit | `uses the freely licensed satellite tiles and their required credit by default` | JS unit | PASS |
| URL and credit overrides work; one tile error switches to OSM and its credit | `keeps configured tiles and attribution and falls back to OSM after an imagery error` | JS unit | PASS |
| An explicit OSM URL receives OSM's credit | `credits OSM when its URL is configured without an explicit credit` | JS unit | PASS |
| Existing map lifecycle and list behavior remain intact | `node --test tasks/redesign/checks/writers-map-core.test.mjs` | JS unit | `ℹ pass 8`, `ℹ fail 0` |
| Local preview keeps one map, fixture tiles, keyboard list, and no console errors at 375 and 1440 px | `CDP_PORT=9339 node tasks/redesign/checks/writers-map-preview.browser.mjs` | Browser | `PASS 375px` and `PASS 1440px` |
| Live EOX tiles load with visible credit, no horizontal overflow, and no console errors at 375 and 1440 px | `CDP_PORT=9339 node tasks/redesign/checks/writers-map-imagery-live.mjs` | Browser, live provider | `PASS 375px` and `PASS 1440px` |

The browser commands above were run with `GOCACHE=/tmp/traveltab-3e-go-cache` and an isolated
Brave instance on port 9339. The live check needs EOX and Leaflet access; it is intentionally
outside `make test`. A separate 2026-10-11 request confirmed Funchal WMTS tiles at zoom 12 and 14
returned 256×256 JPEGs.

Final gate: `PATH=/tmp/traveltab-3e-bin:$PATH GOCACHE=/tmp/traveltab-3e-go-cache GOLANGCI_LINT_CACHE=/tmp/traveltab-3e-lint-cache make test lint build` → all Go packages `ok`, `0 issues.`, and `go build -o bin/traveltab ./cmd/web`. The `/tmp` linter was golangci-lint v2.14.0 built with Go 1.27.2; the installed v2.4.0 was built with Go 1.26 and could not load this module.

Coverage: no Go package changed, so the Go ≥80% changed-package requirement does not apply. JS
branch coverage was not measured. The provider has no published numeric tile quota in the pages
reviewed, and live availability or imagery quality outside Funchal is not guaranteed. The owner
accepted the recorded terms and older-image tradeoff on 2026-10-11.
