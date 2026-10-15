# Look v3 regression sweep

Source: `tasks/look-v3-todo.md` V13; existing `pr8c-journeys.browser.mjs` traveler journey.

As a traveler, I want search, writer pins, place details, saves and Add to plan to keep working across phone and desktop layouts, so I can build my itinerary.

## JavaScript regression suite

Command:

```sh
node --test tasks/redesign/checks/*.test.mjs
```

Initial result exposed the obsolete writer-card/sheet expectations and map-control counts:

```text
ℹ tests 153
ℹ pass 143
ℹ fail 10
```

The place-panel and map-layout owners updated their existing tests for the changed behavior. The coordinated rerun passed:

```text
ℹ tests 157
ℹ pass 157
ℹ fail 0
ℹ skipped 0
```

## Existing browser journey rewritten and preserved

The HEAD version of `pr8c-journeys.browser.mjs` was saved to `/tmp/look-v3-pr8c-before.browser.mjs`, with only its relative browser-helper import changed to the repository's absolute helper URL. It was run against the v3 implementation:

```sh
CDP_PORT=9231 OUTPUT_DIR=tasks/artifacts/look-v3/V13-before node /tmp/look-v3-pr8c-before.browser.mjs
```

RED:

```text
AssertionError [ERR_ASSERTION]: fixture shows nature, museum and food pin kinds
```

The updated journey retains search, categories, opening a writer panel, loading its photo, writer posts, hours, saving and undo, saved-list focus return, Add to plan, overflow and console checks. It replaces the obsolete sheet and writer-card expectations with inline geometry, heading focus, From our writers quote and See all, and checks all four phone navigation icons.

Command:

```sh
CDP_PORT=9231 OUTPUT_DIR=tasks/artifacts/look-v3/V13 node tasks/redesign/checks/pr8c-journeys.browser.mjs
```

GREEN:

```text
PASS Funchal 375px: search, writer pin, panel, Add to plan, itinerary, no overflow or console errors
PASS Funchal 1440px: search, writer pin, panel, Add to plan, itinerary, no overflow or console errors
```

| What is guaranteed | Test | Unit/integration | Result |
| --- | --- | --- | --- |
| Existing JavaScript behavior plus new v3 checks | `tasks/redesign/checks/*.test.mjs` | Unit | 157 PASS |
| Search → writer pin → place → save/undo → saved list → Add to plan → itinerary | `pr8c-journeys.browser.mjs` | Browser integration, 375 and 1440 px | PASS |
| Phone panel stays below map in document flow with heading focus | `pr8c-journeys.browser.mjs` | Browser integration, 375 px | PASS |
| Four phone navigation icons remain present | `pr8c-journeys.browser.mjs` | Browser integration, 375 px | PASS |
| No horizontal overflow, page errors or captured runtime exceptions | `pr8c-journeys.browser.mjs` | Browser integration, both widths | PASS |

Screenshots: `tasks/artifacts/look-v3/V13/funchal-{map,panel,itinerary}-{375,1440}.png`. Browser cache was disabled by the shared helper; Brave used SwiftShader WebGL. These journey screenshots are runtime evidence; mockup comparisons belong to the coordinated visual review. No new production package was changed in this regression task; preview-package coverage is recorded separately in `look-v3-fixture.tdd.md`.

## Remaining browser regression sweep

`node /tmp/look-v3-browser-sweep.mjs` ran each script sequentially against CDP port 9231, starting deterministic local previews. Complete output was captured in `/tmp/look-v3-browser-sweep.log`.

| Script | Runtime coverage | Initial sweep result |
| --- | --- | --- |
| `writers-map-core.browser.mjs` | Both engines at 375/1440, canvas, buildings, fixtures, console | PASS |
| `pr7-integration.browser.mjs` | Both engines at 375/1440, layers, saving and plan inclusion | PASS |
| `pr7-acceptance.browser.mjs` | Both engines at 375/1440, duplicate prevention, saving without requests, included/overflow stops, View on map and blocked storage | PASS |
| `map-dayplan.browser.mjs` | Both engines at 375/1440, numbered pins, popup/Escape, regeneration, toggle, map jump, clear | PASS |
| `browser-smoke.mjs` default | 375/768/1100/1440, planning, interactions, reduced motion and 200% zoom | PASS |
| `browser-smoke.mjs` fallback | 375/768/1100/1440, planning, interactions, reduced motion and 200% zoom | PASS |
| `map-itineraries.browser.mjs` | Existing itinerary journey | FAIL: removed layers-menu assertion; assigned to map-layout owner |
| `writers-map-preview.browser.mjs` | Existing destination preview journey | FAIL: obsolete phone-height assertion; assigned to map-layout owner |
| `look-v3-shell.browser.mjs` | New slim shell layout | FAIL: phone tagline exceeded banner; assigned to shell owner |

These failures remain explicitly recorded until the owning agents provide their repaired rerun results. `map-panel.browser.mjs`, `writers-map-layout.browser.mjs`, `hero-phone.browser.mjs` and `a11y-audit.mjs` belong to the other agents' coordinated runtime review.
