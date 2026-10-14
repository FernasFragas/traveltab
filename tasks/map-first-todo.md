# Map-first destination page — Tasks

Source of truth for progress. Read [the plan](map-first-plan.md) first.

**Status:** M1–M7 and M9 done (2026-10-14). Only **M8**, the owner's real-phone check, is open.

**How to read this file**
- Order: M0 → M1 → M2 → M3 → (M4, M5, M6 in parallel) → M7 → M8 → M9.
- **Owns** = the only files the task may edit. **M2 is the integrator** for `views/index.go.tpl` and `views/content_fragment.go.tpl`; later tasks add only their own tags there.
- `writers-map.js` is shared: M1 (palette and buildings), M4 (controls), M6 (`draw.clusters`). Run them **one after another**, by one agent if possible.
- Run after [dayplan-pins](dayplan-pins-todo.md) (shares `map-dayplan.js`) and [slow-pins](slow-pins-todo.md) (scratch edits to `writers-map.js`, `map-panel.js`).
- Start from an up-to-date `main` (`git fetch`, `git status -sb`).
- TDD per [AGENTS.md](../AGENTS.md) §5. Browser changes: runtime check in `cmd/design-preview` at **375 and 1440 px**, console clean.
- No commits or pushes. The owner commits.

**User journeys**
- *As a traveller, I want the map to fill the screen, so that I explore the destination before reading about it.*
- *As a traveller, I want TravelTab's places to stand out on a calm dark map, so that I'm not distracted by shops and parking.*
- *As a traveller, I want writer pins grouped when I zoom out, so that the map stays readable.*
- *As a traveller, I want to tap a pin and see the place, what writers said, and add it to my plan, without leaving the map.*
- *As a traveller on a phone, I want a small card for the selected place and a sheet I can open, so that the map stays visible.*
- *As a traveller, I want to find myself on the map and switch between 2D and 3D.*

---

### M0. Owner decisions · owner
- [x] Done 2026-10-14, recorded in the plan (**Owner decisions**)

- [x] Tabs that switch views.
- [x] Keep the signpost; name becomes `TravelTabByPatronFragas`.
- [x] Clusters show the number of places.
- [x] Add to plan = save + regenerate; Save to visit = today's behaviour. Ship now: controls, phone card, Share, Details tab.

### M1. Tune the dark map style
- [x] Done (live Funchal and Lisbon screenshots in the [TDD evidence](../docs/tdd/map-first.tdd.md))

**Owns:** `public/redesign/writers-map.js` (palette, `addBuildings`, POI filter only), `tasks/redesign/checks/writers-map-core.test.mjs`, `cmd/design-preview/mapstyle.go` (add a `poi` layer to the fixture style), `mapstyle_test.go`

- Hide generic POI symbol layers; keep place, road and water labels and major landmarks.
- Building height `coalesce(render_height, 8)`.
- Warmer tint (navy + muted olive); TravelTab pin colours ≥ 3:1 against the map background.
- No glow, no hillshade.
- **Verify:** unit tests with a fake style: `poi` layers get `visibility: none`, place labels untouched; extrusion height expression has the 8 m fallback; a custom `data-map-style-url` is left alone (existing rule). Contrast test for the four pin colours against the background colour. Screenshots of live Funchal and Lisbon at z13 and z16.

### M2. Header, name and tab views · **integrator**
- [x] Done

**Owns:** `views/index.go.tpl`, `views/content_fragment.go.tpl`, `views/shell_section_links.go.tpl`, `public/redesign/navigation.js`, `public/redesign/search.js`, `public/redesign/search.css`, `public/redesign/destination.css`, `internal/adapters/httpserver/trip.go` (page title strings only), `internal/adapters/httpserver/export_ics.go` (`DESCRIPTION` line only), `internal/adapters/httpserver/*_test.go` (template, title and ICS assertions only), `README.md` (heading and first line), `tasks/redesign/checks/interaction-regressions.mjs`, `tasks/redesign/checks/hero-phone.browser.mjs`

- Name: `TravelTabByPatronFragas` in the header (text and `aria-label`), footer, default `<title>`, trip page titles, ICS description, README. Leave `User-Agent`, robots group, `PRODID` alone.
- Header: signpost + name, search with arrow submit and **no clear ×** (owner follow-up, 2026-10-14), tabs with underline (desktop), bottom nav with active tab (phone).
- Tabs switch views by hash; `#itinerary`, `#stays`, `#videos` open that tab; Back works. The map is **not** re-mounted.
- Weather hero leaves the page top (moves into the panel in M5).
- **Verify:** tests first (RED on the old name): trip title is "3 days in Funchal — TravelTabByPatronFragas"; ICS description has the new name; `User-Agent` test unchanged. Browser test: load `#stays` → Stays visible, Overview hidden; switch tabs twice → one map instance; Back; no horizontal scroll at 375.

### M3. Full-height map layout
- [x] Done

**Owns:** `views/map_card.go.tpl`, `public/redesign/writers-map-layout.css`, `public/redesign/writers-map.css`, `tasks/redesign/checks/writers-map-layout.browser.mjs`

- Desktop: map + ~400 px panel, both `100dvh − header`, panel scrolls on its own.
- Phone: map fills between the header block and the bottom nav; chips under the search.
- **Verify:** rewrite the layout check (RED against today's 65/35 card): map height = viewport − header ± 2 px at 1440; panel 380–420 px wide; phone map fills the gap above the nav; no overflow.

### M4. Map controls · *after M1*
- [x] Done

**Owns:** `public/redesign/writers-map.js` (controls only), `public/redesign/writers-map-leaflet.js` (controls only), `public/redesign/map-layers.js`, `public/redesign/map-layers.css`, `tasks/redesign/checks/writers-map-core.test.mjs`, `tasks/redesign/checks/map-layers.test.mjs`

- Locate me (one-shot; denied → status text), 2D/3D (hidden on Leaflet), layers menu (four overlays), zoom and scale bottom-right, phone locate FAB.
- **Verify:** unit tests with a fake engine: 2D sets pitch 0 and terrain off, 3D restores both; the menu toggles the same flags as the chips; geolocation denied → message, no throw. Browser: controls don't overlap the attribution at 375 and 1440.

### M5. Place panel redesign
- [x] Done

**Owns:** `views/place_panel.go.tpl`, `public/redesign/map-panel.css`, `public/redesign/map-panel.js` (panel, tabs, sheet, default state), `internal/adapters/httpserver/placepanel_test.go`, `tasks/redesign/checks/map-panel.test.mjs`, `tasks/redesign/checks/map-panel.browser.mjs`, `public/redesign/map-saved.js` (Add to plan / Save to visit only)

- Order: hero photo + ×, title, kind · area, "Mentioned by N writers" + monograms, tabs (**AI summary · Details**), hours, Wikipedia, **Add to plan** / **Save to visit** / **Share**, "Writers' posts (N)" list.
- **Default state** (nothing selected): city photo with credit (when one exists), compact weather, "About {city}" with the Wikivoyage credit, "Plan your trip" → Itinerary tab.
- **Phone:** peek card (name, kind · distance from map centre, ›); › opens a bottom sheet; Escape/× close it and focus returns to the pin.
- **Add to plan:** save, then regenerate the itinerary with the saved places, and show the Itinerary tab's updated day. **Save to visit:** today's save.
- Share: Web Share API, else copy the link and say "Link copied".
- **Verify:** template tests per block present/absent (no AI summary → tab hidden; no posts → no list); JS tests for tab keyboard nav, sheet focus return, share fallback; browser at 375 and 1440.

### M6. Pins, clusters and callouts · *after M4, M5, dayplan-pins*
- [x] Implemented and browser verified; fixture does not reproduce every live zoom/cluster state

**Owns:** `public/redesign/writers-map.js` (`draw.clusters` only), `public/redesign/writers-map-leaflet.js` (`draw.clusters` only, unclustered), `public/redesign/map-panel.js` (pin rendering only), `public/redesign/map-dayplan.js` (pin and route look only), `public/redesign/map-itineraries.css` (colour only), `public/redesign/map-pins.css` (new), `views/index.go.tpl` (the `map-pins` stylesheet tag only), `tasks/redesign/checks/map-pins.test.mjs` (new), `tasks/redesign/checks/writers-map-core.test.mjs`

- Notable: category icon in a blue circle from `Place.Kind` (fixed table; unknown → generic).
- Writers: terracotta, `draw.clusters` with native MapLibre clustering; cluster shows the **number of places** in a larger ring; cluster click → expansion zoom; single pin → today's popup → panel. Leaflet: unclustered markers.
- Day plan: turquoise numbered pins, dashed turquoise route. Itineraries: purple.
- Callouts: name labels above a zoom threshold; photo thumbnail on the selected place and plan stops, lazy-loaded.
- **Verify:** unit tests: kind → icon table incl. unknown; `draw.clusters` adds one clustered source and its layers on MapLibre, N markers on Leaflet; cluster click calls expansion zoom; single-pin click opens the popup; clusters reachable from the keyboard list. Browser: Funchal fixture shows icons, a cluster at low zoom that splits on zoom-in, numbered stops and one photo callout; console clean.

### M7. Journeys and regressions
- [x] Done 2026-10-14: every browser check, the a11y audit and scenarios pass (see the [TDD evidence](../docs/tdd/map-first.tdd.md#m7-full-check-run))

**Owns:** `tasks/redesign/checks/pr8c-journeys.browser.mjs`, `tasks/redesign/checks/browser-smoke.mjs`, `cmd/design-preview/fixture.go` (Funchal fixture data only)

- Journeys on the tab layout: search → map → pin → panel → Add to plan → Itinerary tab shows the stop.
- **Verify:** all `tasks/redesign/checks/*.mjs` pass; a11y audit clean; screenshots at 375 and 1440 next to the mockup.

### M8. Owner real-phone check · owner
- [ ] Done

**Owns:** `docs/tdd/map-first.tdd.md` (device section)

- Funchal on a mid-range phone, 3D on: first render time, smooth panning with clusters and callouts. Folds in the pending `map-3d` M7 check.

### M9. Evidence and docs
- [x] Done 2026-10-14; `make test lint build` passes

**Owns:** `docs/tdd/map-first.tdd.md` (new), `docs/design.md` (map palette section), `CHANGELOG.md`, `README.md` (screenshot only, after M2)

- Journeys, RED/GREEN excerpts per behaviour, guarantees table, coverage (touched Go packages ≥ 80%), known gaps (no satellite or photorealism; straight-line routes; Leaflet fallback doesn't match the dark style; Photos/Nearby tabs, Save trip, post thumbnails and PMTiles deferred).
- **Verify:** `make test lint build` passes.
