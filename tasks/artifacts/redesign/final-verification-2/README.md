# Final verification 2: accessibility audit and scenario screenshots

Run 2026-09-25 against the working tree, in Brave (Chrome/154.0.8037.58) driven over CDP, on the deterministic design preview only. The browser was an isolated headless instance with its own debug port and temporary profile; each preview process ran on its own free localhost port. HTMX was the pinned 1.9.11 script (SRI checked). The earlier evidence in [final-verification](../final-verification/README.md) is untouched.

**Outcome:** the audit found 11 real defects (rows 1 to 11 below), all fixed, plus one exempt disabled-control ratio (row 12). One design-mandated focus-order mismatch at 375 px is documented as a limitation, not fixed. Final run: `a11y-audit.mjs` 25 checks, 24 pass, 1 documented limitation, 0 fail; `scenarios.mjs` 2,744 assertions, 0 mismatches, 110 screenshots.

## Reproduce

Start an isolated browser, then run the two scripts from the repo root. They build and start their own previews, so no preview needs to be running.

```sh
curl -fL https://unpkg.com/htmx.org@1.9.11 -o /tmp/traveltab-htmx-1.9.11.js
"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser" --headless=new --remote-debugging-port=9327 --user-data-dir=$(mktemp -d) about:blank
CDP_PORT=9327 HTMX_PATH=/tmp/traveltab-htmx-1.9.11.js node tasks/redesign/checks/scenarios.mjs
CDP_PORT=9327 HTMX_PATH=/tmp/traveltab-htmx-1.9.11.js AXE_PATH=/path/to/axe.min.js node tasks/redesign/checks/a11y-audit.mjs
```

`AXE_PATH` is optional. axe-core 4.10.2 was downloaded from cdn.jsdelivr.net into a scratch directory and is **not** in the repo or `package.json`. Both scripts write here by default (`OUTPUT_DIR` overrides); rerunning overwrites this evidence. Each exits non-zero on any failure.

Third-party images (stop photos, video thumbnails) were answered with a generated local gradient image and the YouTube embed was blocked, so nothing depends on a live service. Stop photos and video thumbnails in the screenshots are therefore gradient placeholders, not real photographs. Bootstrap CSS, Bootstrap Icons and the Inter font still load from their CDNs, as they do in the page itself.

## How contrast was measured

Not by trusting computed colours alone. For every line of text, icon glyph and CSS-generated stop number, the script records the computed colour, opacity and line boxes, hides all text, screenshots the page, and reads the actual background pixels under each box. The reported ratio is against the lightest **and** darkest pixel there, so photo overlays, gradients, chips and translucent bars are measured as rendered. Form-field text uses computed colours (the select chevron sits inside the same box). A second pass replaces the hero and every third-party image with pure white, the worst case behind light text on a translucent dark overlay. WCAG thresholds: 4.5:1 text, 3:1 large text, icons and UI components. Disabled controls are exempt and reported.

Coverage: 41 page states (Lisbon initial, 5-day plan with all days open, Porto with photo, Coimbra and Photoerror and Nophoto photo-unavailable heroes, broken-image hero, Paris Texas, Longguide, 500 and 400 plan errors, search error, disabled submit, fallback, forecast-uncertain, six-stay, ten-video expanded, video activated, mobile menu open, third-party images blocked, white worst-case photos) at 375 and 1440 px: 5,465 pairs.

## Defects found, and ratios before and after

Ratios are the lowest measured pair. "Before" is a full run of the same script on a copy of the tree with the fixes reverted ([before-fix results](a11y/before-fix/a11y-results.json)).

| # | Defect | Before | After | Fix |
| --- | --- | --- | --- | --- |
| 1 | Start-date field, Days select: 1px border `#DED8CC` is the only field boundary | 1.39:1 (need 3) | 3.46:1 | new token `--tt-border-control: #8F887B` (styles.css, planner.css) |
| 2 | Search pill border | 1.26:1 | 3.15:1 | same token |
| 3 | Weather sun/cloud icons, amber `#d98a08` on cream | 2.48:1 | 3.44:1 | `#b87200` (destination.css) |
| 4 | Focus ring on photo credit links: terracotta ring over the dark overlay | 2.87:1 (375), 2.08:1 (1440) | at least 5.6:1 (whole page, weakest stop) | light inset ring on `.destination-photo-credit a`, `.itinerary-stop-credit a`, `.stay-credit` |
| 5 | Focus ring on the first itinerary day and on stop credits clipped by `overflow: hidden` | 1 px and 0 px visible change | visible (at least 5.6:1) | inset ring on `.itinerary-day-summary` and the credits |
| 6 | Map iframe: keyboard focus lands on it below the viewport, with no scroll and no ring (cross-origin frame) | 0 px change, off screen at 375 | scrolled into view, ring on the map card | `navigation.js` sets `data-frame-focus` on window blur (keyboard only); `.map-card[data-frame-focus]` ring |
| 7 | Focused controls hidden behind the fixed bottom bar at 375 (Check prices, video play, stop credit) | 5 stops covered | 0 covered | `scroll-padding-bottom` on phones |
| 8 | Header nav shows the current section by colour only (ink vs terracotta is 2.56:1) | colour only | underline added | `.tt-header-navigation a[aria-current]` |
| 9 | "Ends" date `<output>` has no accessible name | name "" | "Ends" | `id` on the label, `aria-labelledby` (trip_card.go.tpl) |
| 10 | Target sizes under 44 px: brand link 30 and 34 px, footer brand 24 and 30 px, search input 42 and 39 px, search button 38x38 | 8 targets | 0 | `.tt-brand` min-height 44, search pill 46 px tall with a 44x44 button and 44 px input |
| 11 | After a failed plan the submitted form is replaced and focus falls to `<body>`; the next Tab skips to the guide card | focus on body | focus on Generate plan | `planner.js` refocuses the submit button when focus was lost |
| 12 | Disabled Generate plan (opacity 0.7): 2.42:1 | 2.42:1 | unchanged | exempt as a disabled control; left as is |

**Text contrast had no defect.** Before and after: 3,827 text and field-value pairs, 0 below threshold. Lowest ratios: Lisbon hero caption over the photo 4.94:1, terracotta 12px nav labels 5.57:1, muted text 5.6:1, placeholder 6.09:1. Every state, including both hero states, the guide card in both states, chips, error and search-error text and the worst-case white photo, passes. Worst case behind the credit and caption overlays measured 5.6:1 or better.

## Requirement matrix

Status is what the scripts measured in this run: **pass**, **fail**, **limitation** (fails, documented and deliberately not fixed) or **unverified** (not checkable offline).

| Requirement | Status | Evidence |
| --- | --- | --- |
| Text contrast, 4.5:1 (3:1 large), all states, both widths, overlays and chips | pass | 3,827 pairs, 0 failing; [contrast-pairs.json](a11y/contrast-pairs.json) |
| Text contrast over hero photo overlays, normal and photo-unavailable heroes | pass | pixel-sampled; worst-case white photo 5.6:1; [white 375](a11y/worst-case-white-photos@375.png), [coimbra 1440](a11y/coimbra-no-photo-no-guide@1440.png), [broken image](a11y/brokenimage-failed@1440.png) |
| Guide card text, reviewed and missing states | pass | in the pairs above (Lisbon, Coimbra, Longguide) |
| Icons and UI component boundaries, 3:1 | pass | 1,602 icon and field pairs, lowest 3.15:1 (search pill). Bordered buttons that carry a text label (`.video-expand`, `.itinerary-action`, `.itinerary-directions`, day headers, chips, stop numbers) have 1.1 to 1.4:1 borders and are advisory: their labels identify them |
| Hover states of links and buttons | pass | 15 controls, lowest 5.53:1 |
| Focus rings visible, 3:1 change, not covered, at 375 and 1440 | pass | 52 stops at 375 and 51 at 1440 (date segments counted once), weakest 5.61:1; [skip link](a11y/focus-375-focus-skip-link.png) |
| Focus order follows the visual order at 1440 | pass | no backwards jump |
| Focus order follows the visual order at 375 | **limitation** | the planner is shown first by CSS `order` (design: form before map), but the DOM keeps the map first for the desktop two-column reading order. One jump: map card (3 stops) to the form. Reordering the DOM would move the mismatch to desktop |
| Skip link | pass | first Tab stop, on screen, 15.63:1; Enter focuses `#overview` |
| Landmarks | pass | banner, main, contentinfo, search and 4 uniquely named navigation landmarks, 5 named regions |
| Heading order | pass | one h1, no skipped level, 39 states |
| Accessible names: controls, iframes, images | pass | browser accessibility tree, both widths: 0 unnamed; map iframe, video player iframe titled; alt present on every image, hero alt names the city |
| axe-core 4.10.2 | pass | 0 violations at both widths; 2 incomplete (color-contrast, needs manual review, covered by the pixel method above; frame-tested: cross-origin map frame) |
| Form labels, autocomplete, described-by | pass | date "Start", select "Days", search "Search destination, including country", "Ends" output named |
| Accordion keyboard | pass | Enter and Space toggle; expanded state exposed; day 1 open, days independent |
| Video expand button keyboard | pass | Enter and Space; `aria-expanded` and label follow |
| Mobile menu keyboard | pass | Enter opens, Tab reaches links, Escape closes and returns focus |
| Live regions | pass | observed: "Destination updated. Your overview is ready.", "Planning your trip…", "Your itinerary has been updated.", plan error inserted as `role=alert`, search error in the alert region. The destination search start is **not** announced, and the visually hidden spinner text ("Finding destination…", "Planning…") stays in the reading order; both are minor and left as they are |
| Reduced motion | pass | longest transition 0.16 s and animation 0.8 s become 0.00001 s; smooth scroll becomes auto |
| 200% zoom and reflow | pass | no horizontal scroll or clipped text at 640, 720, 360 and 320 CSS px (1280 at 200%, 1440 at 200% and 400%, 1280 at 400%); [720](a11y/zoom-200-1440.png), [320](a11y/reflow-320.png) |
| 375 at 200% (188 CSS px, below the 320 px WCAG reflow width) | informational | scrollWidth 274 for 188; not required by WCAG 1.4.10, not fixed |
| Text spacing (WCAG 1.4.12) | pass | no overflow or clipping at 375 and 1440 |
| Touch targets 44 px | pass | 59 (375) and 58 (1440) targets. Exceptions: 2 photo-credit links (37 px and 12 px tall, at least 24 px or spacing-exempt) and 3 video-title link sizes that duplicate 44 px controls (WCAG 2.5.8) |
| Current section not conveyed by colour alone | pass | header nav underline; section and bottom bars use a border |
| Focus not lost after a failed plan | pass | focus on Generate plan |
| Uncaught JavaScript exceptions | pass | none |
| Negative controls: the audit fails when the UI regresses | pass | `NEGATIVE_CONTROL=muted` (19 text and 11 icon/component pairs fail), `border` (3 field boundaries fail), `focus` (49 and 48 stops fail), `targets` (12 targets fail): [logs](a11y/negative-controls) |
| Screenshot and assertion for all 12 data variants, 375 and 1440, initial and planned | pass | [variants](scenarios/variants); default also 5-day, third-party images failed, viewport shots; videos-10 expanded |
| Table: Porto (no videos), Coimbra (replacement videos, no photo), Paris France, Paris Texas, shared `/trip/paris-fr` and `paris-us` | pass | [table](scenarios/table): initial and planned each; the two Parises have different photographs and Paris, Texas never shows Paris, France's guide |
| Table: search failure, plan 500, plan 400, shared three-day plan, local map identity | pass | message, cleared plan and stays, form values, `role=alert`, map title "FIXTURE MAP — NOT LIVE WAZE" asserted |
| Table: calendar and map downloads | pass, no page screenshot | asserted by HTTP (content type, attachment, 4 events, 4 placemarks, coordinates, first date) and the linked URLs return 200; the screenshot shows the export links, since a download has no page |
| Photo fixtures: Nophoto, Photoerror, Brokenimage; Paris, Porto, Texas; Coimbra, Tokyo, Tavira | pass | [fixtures](scenarios/fixtures): "Photo unavailable", no credit, frame keeps its size, page works after provider failure, broken image reveals the fallback and hides the credit |
| Guide fixtures: reviewed vs missing, Longguide, fallback (no guide anywhere) | pass | text equals `guides/guides.json`; four attribution links; missing state with Wikivoyage link; 93-character word wraps; markup shown literally |
| Slow preview loading states | pass | busy search and spinner, disabled submit, "Planning your trip…" then "Your itinerary has been updated." ([slow](scenarios/slow)) |
| Real map, Wikimedia photographs, YouTube playback, Booking landing page, ICS and KML import into real apps | **unverified** | out of scope: live external services. Still open in [maintenance notes](../../../../docs/maintenance.md#next-up) |
| Keyboard focus and appearance *inside* the live Waze frame | **unverified** | third-party document; the wrapper ring and scroll-into-view are verified only against the local fixture map |

## Files

- `a11y/a11y-results.json`: every result, findings, the 40 lowest-margin pairs, tab stops, accessibility-tree names, axe results. `a11y/contrast-pairs.json`: all 5,465 pairs. `a11y-run.log`: the run.
- `a11y/before-fix/`: the same audit before the fixes. `a11y/negative-controls/`: logs of the four deliberate regressions.
- `scenarios/scenario-results.json` and 110 WebP screenshots (quality 80; full-page captures hide the fixed bottom bar and skip link, `*-viewport.webp` shows them in place). `scenarios-run.log`: the run.
- Scripts: `tasks/redesign/checks/a11y-audit.mjs`, `scenarios.mjs`, `lib/`.
