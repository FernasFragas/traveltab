# Hero line on phones (7i follow-up)

Source: owner decision after `docs/tdd/7i.tdd.md` and `docs/tdd/descriptions.tdd.md`. Browser change only (CSS plus a wrapper div); no Go behaviour changed.

**Journeys**
- As a traveller on a phone, I want the one-line description below the photo, on the page surface, so I can read it and know what the place is.
- As a traveller on a phone, I want no gap when a city has no line.
- As a desktop traveller, I want the hero exactly as before.

## Change
- `views/weather_display.go.tpl`: city and region/country wrapped in `<div class="destination-heading">` (a plain block on desktop, no visual change).
- `public/redesign/destination.css` (640 px query only): banner gets a middle row; `.destination-information` spans photo + middle row as a subgrid; the heading stays over the photo (light text), the tagline sits in the middle row on the surface in ink colour; weather card moves to row 3. Caption stays hidden. Clamp unchanged.

## RED / GREEN
Check: `tasks/redesign/checks/hero-phone.browser.mjs` (isolated Brave, `CDP_PORT`). The 1440 baseline was measured before the change and is embedded in the check.

RED (before the CSS/template change):
```
AssertionError: Lisbon 375: hero line visible {"title":"Lisbon","text":"Hillside streets, ...","tagVisible":false,...,"tag":{"top":0,"bottom":0,...}}
false !== true
```
GREEN:
```
PASS 375 Lisbon: "Hillside streets, tiled facades, and life by the river." photo.bottom 328 <= tagline 328-388 <= card.top 388
PASS 375 Porto: "city in northern Portugal" photo.bottom 328 <= tagline 328-364 <= card.top 364
PASS 1440: hero geometry identical to the baseline; console clean
```
Nophoto (no line) at 375: no `.destination-tagline` element and photo bottom equals card top (gap 0).

| Guarantee | Check | Result |
|---|---|---|
| 375: tagline visible, below photo, above card, inside viewport, ink colour, 2-line clamp (Lisbon tagline, Porto description) | `hero-phone.browser.mjs` | pass |
| 375: caption hidden, no horizontal overflow | same | pass |
| 375: no line, no gap (Nophoto) | same | pass |
| 1440: photo, banner, tagline, card, title rects, colour, clamp identical to before | same | pass |
| Console and page errors empty | same | pass |

## Other runs
- `pr7-acceptance.browser.mjs`, `pr7-integration.browser.mjs`: pass at 375 and 1440.
- `make test build`: ok.
- Screenshot checked: Lisbon at 375 (title over photo, line under it on cream, card below).
- `scenarios.mjs` fails at `eq(s.weather.length, 3)` (expects 3 `.weather-metric`; 7i changed the card). `a11y-audit.mjs` fails contrast.text (4 pairs), mobile-menu-keyboard and text-spacing (`itinerary-stop-description` clipped). Neither file is touched by this change and none of the reported findings names the hero line, but I did not run them on a clean tree to confirm they failed before.

## Known gaps
- Subgrid needs Chromium 117 / Safari 16 / Firefox 71 or newer; on older browsers the line would overlay the photo area.
- `make lint` not run (golangci-lint go1.26 vs go1.27.2).
