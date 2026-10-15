# Map look v2 — Tasks

Source of truth for progress. Read [the plan](map-look-v2-plan.md) first.

Execution note (2026-10-14): the owner requested implementation with agents on the existing
map-first working tree. That work was preserved and used as the baseline; no commits, pushes
or branch/history changes were made. L1–L9 are implemented; final regression evidence is being
completed in [the TDD report](../docs/tdd/map-look-v2.tdd.md).

**How to read this file**
- Order: L0 → L1 → … → L11, by **one agent** (L3–L9 share `map-panel.js`, `map-panel.css`, `writers-map.js`, `map-pins.css`, `map_card.go.tpl`).
- **Owns** = the only files the task may edit. None overlap with the open Writers' Map tasks in [todo.md](todo.md).
- Start after [map-first](map-first-todo.md) is **committed**, from an up-to-date `main` (`git fetch`, `git status -sb`).
- TDD per [AGENTS.md](../AGENTS.md) §5. Browser changes: runtime check in `cmd/design-preview` at **375 and 1440 px**, console clean. Clear the browser cache before screenshots (stylesheets aren't versioned).
- Browser checks need WebGL: start the isolated browser with `--use-angle=swiftshader --enable-unsafe-swiftshader`.
- No commits or pushes. The owner commits.

**User journeys**
- *As a traveller, I want to recognise TravelTab at a glance, so that the product feels like one brand.*
- *As a traveller, I want layer pills above the map and controls on it, so that the map stays clear.*
- *As a traveller, I want to tell places, writers' pins and my plan apart by colour, shape and icon, so that I know what I'm looking at.*
- *As a traveller, I want a place's story, writers, hours and a way to get there in one panel, so that I can decide without leaving the map.*
- *As a traveller on a phone, I want a place to open in a half-height sheet, so that I still see the map.*
- *As a traveller, I want confirmation when I save a place, a way to undo it, and a way back to my saved places.*

---

### L0. Owner decisions · owner
- [x] Done 2026-10-14, recorded in the plan (**Decided**, **Later**, **Dropped**); updated for the revised mockups

### L1. Brand and search field
- [x] Done 2026-10-14

**Owns:** `views/index.go.tpl` (brand, footer and search form only), `public/redesign/destination.css` (brand only), `public/redesign/search.css`, `internal/adapters/httpserver/*_test.go` (brand and search-form assertions only), `tasks/redesign/checks/a11y-audit.mjs` (brand text and search submit only), `tasks/redesign/checks/hero-phone.browser.mjs` (search submit only)

- Inline SVG **solid peaks** (terracotta, `aria-hidden`); "TravelTab" bold serif; "ByPatronFragas" on a second line, smaller terracotta sans.
- Brand link's accessible name stays `TravelTabByPatronFragas`; `<title>`, trip titles and ICS text unchanged.
- Search: rounded field (not a pill), **no arrow button**; Enter and suggestion picks still submit.
- **Verify:** template tests: brand `aria-label` and both wordmark parts, no signpost; no `.tt-search-submit`. Browser: Enter submits; logo fits with the bookmark at 375; no overflow.

### L2. Header bookmark and saved list
- [x] Done 2026-10-14

**Owns:** `views/index.go.tpl` (header actions only), `public/redesign/search.css` (header layout only), `public/redesign/map-saved.js` (list only), `public/redesign/map-saved.css`, `tasks/redesign/checks/map-saved.test.mjs`

- Bookmark in a **bordered rounded button** (desktop and phone), `aria-label="Saved places"`, `aria-expanded`. No avatar.
- Opens a small list of saved places; empty state "No saved places yet". Picking one focuses it on the map and opens its panel. Escape closes and returns focus.
- **Verify:** JS tests: list renders saved items; empty state; Escape returns focus. Browser at 375 and 1440.

### L3. Layout: frame, pill band, map card, panel card
- [x] Done 2026-10-14

**Owns:** `views/map_card.go.tpl` (layers and frame markup only), `public/redesign/writers-map-layout.css`, `public/redesign/map-layers.js` (labels and icons only), `public/redesign/map-layers.css` (pills only), `tasks/redesign/checks/map-layers.test.mjs`, `tasks/redesign/checks/writers-map-layout.browser.mjs`

- Desktop: soft rounded page frame; pill band above the map; map a **16 px rounded card**; panel a separate rounded card that scrolls on its own.
- Pills: white, pill radius, coloured icon (pin, quill, dotted route, star). Labels: desktop "Notable places · Writers' pins · Writers' itineraries · Your plan"; phone "**Notable · Writers · Itineraries · Your plan**" (full names stay as accessible names).
- Phone: pills in one row under the search; map rounded at the top.
- **Verify:** unit: labels, accessible names and pressed states; browser: map and panel fill the viewport below the band at 1440; phone row fits or scrolls without page overflow at 375.

### L4. Dark map controls
- [x] Done 2026-10-14

**Owns:** `public/redesign/writers-map.js` (controls only), `public/redesign/writers-map-leaflet.js` (controls only), `public/redesign/map-layers.css` (controls only), `tasks/redesign/checks/writers-map-core.test.mjs`

- Top-right column: round **compass N** (resets bearing; MapLibre only), locate, **3D**, zoom +/− group, layers. Dark translucent, white icons, 44 px targets. Scale bar bottom right.
- **Verify:** unit with a fake engine: compass resets bearing to 0; hidden on Leaflet; control order. Browser: icon contrast ≥ 3:1, focus visible, no overlap with the attribution, toast or full-map card at 375 and 1440.

### L5. Teardrop pins, palette and labels
- [x] Done 2026-10-14

**Owns:** `public/redesign/map-pins.css`, `public/redesign/map-panel.js` (pin rendering only), `public/redesign/writers-map.js` (cluster paint only), `public/redesign/map-dayplan.js` (pin look only), `public/redesign/map-itineraries.css` (colours only), `public/redesign/writers-map-layout.css` (pill icon colours only), `tasks/redesign/checks/map-pins.test.mjs`, `tasks/redesign/checks/map-dayplan.test.mjs`

- **Teardrop** pins: notable terracotta + white category icon (camera, food, museum, church…), nature kinds green; writers' pins purple + quill (`bi-feather`).
- Clusters: grey circles + count. Your plan: orange numbered teardrops + **orange dotted route**. Writers' itineraries: **purple dashed** with small stop dots.
- Labels: white text on dark translucent rounded boxes, above the zoom threshold.
- **Verify:** unit: kind → colour/icon table incl. nature and unknown; cluster paint grey; route dash patterns; contrast ≥ 3:1 for each pin colour and ≥ 4.5:1 for label text. Browser: Funchal fixture shows each kind.

### L6. Selected pin and callout
- [x] Done 2026-10-14

**Owns:** `public/redesign/map-panel.js` (selected pin only), `public/redesign/map-pins.css` (selected only), `tasks/redesign/checks/map-pins.test.mjs`

- Selected place: terracotta teardrop with a ring + white callout: **photo thumbnail** (lazy, hidden without a photo) · **name** · **area** · **›**. Clicking the callout opens the panel (phone: the sheet).
- **Verify:** unit: selection sets the class and callout content; no `<img>` without a photo; callout click opens the panel; keyboard reachable.

### L7. Place panel redesign
- [x] Done 2026-10-14

**Owns:** `views/place_panel.go.tpl`, `public/redesign/map-panel.css`, `public/redesign/map-panel.js` (panel only), `internal/adapters/httpserver/placepanel_test.go`, `tasks/redesign/checks/map-panel.test.mjs`, `tasks/redesign/checks/map-panel.browser.mjs`

- Order: photo with ×; bold serif title + **bordered bookmark button** (Save to visit toggle); "Kind · Area"; **writer card** (three states, see plan); one paragraph (AI summary, else description); **+ Add to plan**; **Save to visit** + **Share**; **link cards** Wikipedia ("View on Wikipedia") and Directions ("Open in OpenStreetMap"); **Hours card** (▾ expands); "**Writers' posts (N) ›**" with **post cards** (quill tile, title, blog name, language).
- Remove the AI summary / Details tabs.
- Directions: `https://www.openstreetmap.org/directions?route=%3B{lat}%2C{lon}`, new tab.
- Itinerary state is computed in `map-panel.js` from the itineraries in the loaded GeoJSON (stop QIDs); blog names from the existing sources data.
- **Verify:** template tests: no tabs; summary vs description; Wikipedia card absent without URL; Directions URL; Hours card absent without hours; one post card per mention. JS tests: writer card picks itinerary → mentions → none; › scrolls to posts; bookmark mirrors Save to visit. Browser at 375 and 1440.

### L8. Saved toast with Undo, planner line
- [x] Done 2026-10-14

**Owns:** `public/redesign/map-saved.js` (toast and count only), `public/redesign/map-saved.css`, `views/trip_card.go.tpl` (saved-line slot only), `tasks/redesign/checks/map-saved.test.mjs`

- Saving shows a dark toast "**Added to your saved places · Undo**" at the bottom of the map for a few seconds, in a polite live region. **Undo** removes that place.
- "N saved places will be included · Clear" moves into the Itinerary planner card.
- **Verify:** unit: save → toast text and live region; Undo removes the place and hides the toast; toast hides itself; planner line count and Clear.

### L9. Phone sheet and bottom nav
- [x] Done 2026-10-14

**Owns:** `public/redesign/map-panel.js` (sheet only), `public/redesign/map-panel.css` (phone only), `views/map_card.go.tpl` (peek card removal only), `views/index.go.tpl` (bottom nav only), `public/redesign/navigation.js` (bottom nav only), `tasks/redesign/checks/map-panel.browser.mjs`

- Selecting a place opens the sheet at about **half the screen**, handle on top, inset rounded photo with ×. Drag or tap the handle (Enter/Space too) to expand or collapse. Escape closes; focus returns to the pin. Peek card removed.
- Bottom nav: **text tabs with an underline** on the active one, no icons.
- **Verify:** browser at 375: sheet ~50% height with the map visible; handle toggles full height; Escape returns focus; active tab underlined and `aria-current`; no overflow.

### L10. "View full map" card, journeys and regressions
- [x] Done 2026-10-14

**Owns:** `views/map_card.go.tpl` (map footer only), `public/redesign/writers-map-layout.css` (map footer only), `internal/adapters/httpserver/writermap_test.go` (map footer assertions only), `tasks/redesign/checks/pr8c-journeys.browser.mjs`, `tasks/redesign/checks/browser-smoke.mjs`, `cmd/design-preview/fixture.go` (Funchal fixture data only)

- Card: city thumbnail (or map icon), "View full map", "Open {City} on a larger map", external-link icon; same external URL.
- Fixture: give Funchal a photo, hours, a writers' itinerary stop, posts and nature/museum/food kinds so every state shows.
- Journeys: search (Enter) → pin → callout → panel → Add to plan → Itinerary shows the stop; save → toast → Undo; save → header list → back to the place.
- **Verify:** all `tasks/redesign/checks/*.mjs` pass; a11y audit clean; screenshots at 375 and 1440 next to `../phone.png` and `../desktop.png`.

### L11. Evidence and docs
- [x] Done 2026-10-14

**Owns:** `docs/tdd/map-look-v2.tdd.md` (new), `docs/design.md` (brand, layout, palette and panel sections), `CHANGELOG.md`, `README.md` (screenshot only)

- Journeys, RED/GREEN per behaviour, guarantees table, coverage (touched Go packages ≥ 80%), known gaps (the plan's **Later** and **Dropped** lists).
- **Verify:** `make test lint build` passes.
