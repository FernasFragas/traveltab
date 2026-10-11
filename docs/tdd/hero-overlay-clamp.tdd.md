# Hero overlay, phone label, no clamp

Source: owner decisions after `docs/tdd/pr7-browser-regressions.tdd.md`, `docs/tdd/hero-phone-line.tdd.md`, `docs/tdd/descriptions.tdd.md`. CSS and checks only; no Go behaviour changed.

**Journeys**
- As a desktop traveller, I want more of the destination photo visible next to the title, so the page looks like the mockup, with the text still readable.
- As a phone traveller, I want the title on a dark rounded label over the photo and the line below it on the page surface.
- As a traveller with a long description, I want the whole text, with no horizontal overflow, instead of a two-line cut.

## Changes
- `destination.css` desktop: `.destination-photo::after` is `linear-gradient(90deg, rgb(36 34 31 / 72%) 40%, rgb(36 34 31 / 0%))` (was flat to 65%). Same alpha, flat to 40% of the banner.
- `destination.css` phone (640 px): photo gradient `linear-gradient(transparent 25%, rgb(36 34 31 / 75%))`; `.destination-heading` gets `margin: 48px 16px 16px; padding: 4px 8px; border-radius: 4px; background: rgb(36 34 31 / 78%); color: #fffdfa; justify-self: start; max-width: calc(100% - 32px)` (no text-shadow). Tagline stays on the surface. Fallback text rule unchanged (dark ink at the top of the light photo area).
- Clamp removed from `.itinerary-stop-description` (`planner.css`) and `.destination-tagline` (`destination.css`).
- `a11y-audit.mjs`: `text-spacing-line-clamp` row and its clamp bookkeeping removed (file now equals HEAD). `hero-phone.browser.mjs`: asserts `line-clamp` is `none`, baseline `clamp: "none"`.
- Tests renamed and inverted: `TestItinerary_DescriptionStyleWrapsLongTextWithoutClamp`, `TestHeroLine_StyleWrapsLongTextWithoutClamp` (the hero twin lives in the same file and asserted the same clamp).

## RED / GREEN
Command: `go test ./internal/adapters/httpserver/ -run 'WrapsLongText'`

RED (current CSS):
```
--- FAIL: TestItinerary_DescriptionStyleWrapsLongTextWithoutClamp
    Error: ".itinerary-stop-description {\n    display: -webkit-box; ... -webkit-line-clamp: 2;\n    line-clamp: 2;\n    overflow: hidden; ..." should not contain "line-clamp"
--- FAIL: TestHeroLine_StyleWrapsLongTextWithoutClamp
    Error: ".destination-tagline {\n    display: -webkit-box; ..." should not contain "line-clamp"
FAIL	weatherservice/internal/adapters/httpserver
```
GREEN: `ok  	weatherservice/internal/adapters/httpserver	0.275s`

## Contrast (pixel audit, `a11y-audit.mjs`, isolated Brave, Lisbon fixture)
Worst measured ratio per element (required 3 for large, 4.5 for body):

| Width | Title | Region/country | Tagline |
|---|---|---|---|
| 375 (label) | 9.98 | 11.64 | 15.5 (surface) |
| 900 | 6.15 | 6.16 | 6.13 |
| 1024 | 6.16 | 6.14 | 5.95 |
| 1100 | 6.15 | 6.14 | 6.14 |
| 1280 | 6.15 | 6.14 | 6.16 |
| 1440 | 6.13 | 6.12 | 6.14 |

Weather card text 6.06 to 15.6. Fallback text ("Photo unavailable"): 9.93 at 375, 4.71 at 1440 (above 4.5). The 900 to 1280 rows came from a temporary copy of the audit with `WIDTHS` changed (deleted afterwards); the committed audit measures 375 and 1440 only.

## Check results
- `a11y-audit.mjs`: `23 passed, 0 failed`. Only limitation: `focus-order-matches-visual@375`. `text-spacing` passes as a real check (no horizontal scroll, no clipped text at 375 and 1440). axe not used (`AXE_PATH` unset).
- `scenarios.mjs`: `PASS: 2816 assertions across 110 screenshots, no JavaScript exceptions`.
- `hero-phone.browser.mjs`, `pr7-acceptance.browser.mjs`, `pr7-integration.browser.mjs`: exit 0, 375 and 1440 px, console clean.
- `make test build`: all packages ok, binary built.

| Guarantee | Test | Level | Result |
|---|---|---|---|
| Stop description has no clamp, wraps | `TestItinerary_DescriptionStyleWrapsLongTextWithoutClamp` | unit | pass |
| Tagline has no clamp, wraps | `TestHeroLine_StyleWrapsLongTextWithoutClamp` | unit | pass |
| Hero text contrast, 375 to 1440 | `a11y-audit.mjs` | browser | pass |
| Tagline below photo on phone, no clamp | `hero-phone.browser.mjs` | browser | pass |

## Known gaps
- Coverage not measured: CSS and JS only.
- Photo visibility is judged by eye from fixture photos; real photos may be brighter behind the text. Fallback: widen the flat part back toward 65%.
- `make lint` not run.
