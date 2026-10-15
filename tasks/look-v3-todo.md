# Look v3 — Tasks

Source of truth for progress. Read [the plan](look-v3-plan.md) first.

**How to read this file**
- Original order: V0 → V1 → … → V13. On 2026-10-15 the owner requested parallel agents; work was split by exclusive file ownership and integrated centrally. V3–V11 share `map_card.go.tpl`, `place_panel.go.tpl`, `map-panel.js/.css`, `writers-map.js` and `writers-map-layout.css`.
- **Owns** = the only files the task may edit. None overlap with the open Writers' Map tasks in [todo.md](todo.md) (8b–8d, 9a–9d are backend/ops).
- Start from an up-to-date `main` (`git fetch`, `git status -sb`), after the owner has committed map-look-v2.
- TDD per [AGENTS.md](../AGENTS.md) §5. Browser changes: runtime check in `cmd/design-preview -scenario writers-map` at **375 and 1440 px**, console clean.
- **Every visual task ends with a side-by-side screenshot** (yours vs the mockup crop) saved in `tasks/artifacts/look-v3/<task>/`. Clear the browser cache first; stylesheets aren't versioned.
- Browser checks need WebGL: `--use-angle=swiftshader --enable-unsafe-swiftshader`.
- **D…** refers to the owner's decisions in the plan's **Decided** table.
- No commits or pushes. The owner commits.

**User journeys**
- *As a traveller, I want the city, its weather and a way to start planning in one slim banner, so that the map is on screen sooner.*
- *As a traveller, I want to see which layer is on at a glance, so that I know what the pins mean.*
- *As a traveller, I want a place's photos, key facts and links in one panel, so that I can decide without leaving the page.*
- *As a traveller, I want previews of my day, stays and videos under the map, so that I can jump to them.*
- *As a traveller on a phone, I want the map, the place and my day in one scroll, so that I don't juggle sheets.*
- *As a traveller on a phone, I want icons in the bottom nav, so that I find sections faster.*

---

### V0. Owner decisions · owner
- [x] Done 2026-10-15, recorded in the plan (**Decided**, D1–D20 plus the D14 rule table)

### V1. Rich Lisbon fixture for screenshots
- [x] Done 2026-10-15

**Owns:** `cmd/design-preview/writersmap.go`, `cmd/design-preview/writersmap_test.go`, `cmd/design-preview/photos.go` (fixture photos only)

- Lisbon gets ≈ 12 places across categories (museum, food, viewpoint, nature), each with a photo, kind, area and description. **Belém Tower** gets hours `Tu-Su 10:00-18:30`, a Wikipedia URL and **2 writer mentions**. Photos are served locally (no external hosts).
- A Lisbon day plan fixture: Morning Belém Tower, Afternoon Jerónimos Monastery, Evening Alfama.
- **Verify:** `go test ./cmd/design-preview/ -run TestWriterMapPreview` checks the place count, categories, Belém hours and mentions; a 1440 screenshot shows pins across the map.

### V2. Tokens and type
- [x] Done 2026-10-15

**Owns:** `public/styles.css` (tokens only), `docs/design.md` (tokens table only)

- Add `--tt-accent-tint`, `--tt-writer-tint`, `--tt-writer` and a `--tt-serif` font token. Check every new text/background pair with the a11y audit's contrast helper: **4.5:1 text, 3:1 icons**.
- **Verify:** `node tasks/redesign/checks/a11y-audit.mjs` stays green; the new pairs are listed with their ratios in the TDD report.

### V3. Header layout and placeholder
- [x] Done 2026-10-15

**Owns:** `views/index.go.tpl` (header only), `public/styles.css` (header rules), `public/redesign/search.css`, `tasks/redesign/checks/look-v3-header.test.mjs` (new)

- Desktop grid: **brand · search (max 570 px, centred) · tabs (right) · bookmark**. Tab underline 3 px.
- Placeholder "**Search destinations, cities, or places…**". Phone: larger brand (peaks ≈ 38 px), **serif placeholder**.
- **Verify:** unit: placeholder text, DOM order, accessible names unchanged (`TravelTabByPatronFragas`). Browser: at 1440 the tabs' right edge sits within 72 px of the bookmark; at 375 no horizontal overflow.

### V4. Slim banner and weather strip
- [x] Done 2026-10-15

**Owns:** `views/weather_display.go.tpl`, `views/weather_card.go.tpl`, `public/redesign/destination.css`, `internal/adapters/httpserver/presentation_test.go` (banner tests only), `tasks/redesign/checks/hero-phone.browser.mjs`

- Desktop: banner **≈ 157 px**, text left (name, country, one-line tagline), **no photo caption**. Weather becomes **one horizontal strip**: icon · temp / condition / feels like · Humidity · Wind · Waves · **"Plan your trip ↗"** (links to `#itinerary`).
- **D3:** **Sunset** joins the strip; a **"Forecast ›"** button opens a popover with the 4-day forecast. The phone chip **›** opens the same popover with humidity, wind, waves, sunset and forecast. Escape closes, focus returns.
- **D4:** photo credit becomes **tiny translucent text at the bottom right**; credit and license links stay keyboard reachable.
- Phone: **≈ 100 px** photo, all text on it, **weather chip** top right.
- **Verify:** Go template tests: strip has humidity, wind, waves, sunset, the forecast button and the plan link; forecast days render inside the popover; no caption; credit still rendered; one `h1`. JS test: popover opens, Escape closes and restores focus. Browser: banner height 150–165 px at 1440 and 90–110 px at 375; text on the photo passes 4.5:1 with the `white` image mode.

### V5. Layer pills
- [x] Done 2026-10-15

**Owns:** `public/redesign/map-layers.js` (labels and icons), `public/redesign/map-layers.css` (pills), `views/map_card.go.tpl` (pill markup only), `tasks/redesign/checks/map-layers.test.mjs`

- Active pill **filled terracotta, white text and icon**; others white with a border.
- Icons: pin, **purple pencil-square**, **purple tablet/route**, **orange map tile**. Labels "**Notable places · Writers' pins · Itineraries · Your plan**" on both widths.
- Phone row scrolls with a **hidden scrollbar**. Visual height 40 / 30 px, **hit area ≥ 44 px**.
- **Verify:** unit: labels, accessible names, `aria-pressed` drives the filled style (no colour-only state: the icon also changes from outline to filled). Browser: 44 px hit boxes; no visible scrollbar at 375.

### V6. Desktop grid and shorter map
- [x] Done 2026-10-15

**Owns:** `public/redesign/writers-map-layout.css`, `views/map_card.go.tpl` (layout wrappers only), `views/content_fragment.go.tpl`, `tasks/redesign/checks/writers-map-layout.browser.mjs`

- Two columns: left ≈ 865 px (pills, map **`min(505px, 60vh)`**, teaser row), right ≈ 510 px panel card, gap 16. The panel card is as tall as the left column and scrolls on its own.
- Phone order: hero → pills → **map ≈ 205 px** (radius 12) → place card → teasers.
- **Verify:** browser at 1440×900: map 480–520 px tall, panel ≈ 510 px wide, teaser row starts ≤ 24 px below the map. At 375: map 190–220 px; place card directly after it; no overflow. MapLibre and Leaflet.

### V7. Map chrome
- [x] Done 2026-10-15

**Owns:** `public/redesign/writers-map.js` (controls, labels, scale, attribution), `public/redesign/writers-map-leaflet.js` (same, fallback), `public/redesign/writers-map.css`, `public/redesign/map-pins.css`, `public/redesign/map-itineraries.js` / `.css` (picker move only), `tasks/redesign/checks/map-look-v2.test.mjs`, `tasks/redesign/checks/map-pins.test.mjs`

- Compass with a **red needle**; locate `bi-crosshair`; **zoom as one pill**; 2D/3D styled the same. **D7:** remove the Layers button.
- **D6:** round **‹ back** button top left: closes the selection and flies to the city; hidden when nothing is selected. Itinerary picker moves into a menu on the Itineraries pill.
- District labels: **plain white uppercase text with shadow**, no boxes; water labels **italic blue**.
- Category pin icons: `bi-bank`, `bi-fork-knife`, nature green; writers keep the quill **(D8)**.
- Callout: **thumbnail ≈ 80×56 · serif name · kind · ›**. Selected pin: ring + soft glow.
- Scale: **white line with ticks**, "0" / "1 km". Attribution: **compact ⓘ** (still reachable).
- **D9:** "View full map" hidden at phone widths.
- **Verify:** unit: controls present and labelled; zoom pill has two buttons; label paint has no background; callout markup. Browser: all controls ≥ 44 px; back button closes the panel and restores focus to the map; labels ≥ 4.5:1 on the dark map.

### V8. Place panel / place card
- [x] Done 2026-10-15

**Owns:** `views/place_panel.go.tpl`, `public/redesign/map-panel.css`, `public/redesign/map-panel.js` (panel only), `internal/adapters/httpserver/placepanel_test.go`, `tasks/redesign/checks/map-panel.test.mjs`, `tasks/redesign/checks/map-panel.browser.mjs`

- Order: **photo + "1 / N" badge + ‹ ›** (only when N > 1, **D10**) → serif title + bookmark → **meta row with icons** → **serif description** → actions **"⊕ Add to your plan" · "Save" · "Share"** with icons → **two link cards with icon tiles** ("View on Wikipedia / History, photos and more"; "Get directions / Open in Google Maps" **D11**) → **facts card** → **From our writers**.
- Remove: the **writer card**, the **Writers' posts** list, the desktop **×** (D6).
- **City panel (D1)** before a place is picked, same look: city photo, serif title, `bi-geo-alt` + country, Wikivoyage paragraph in serif, filled "**Plan your trip**", one link card "**Read on Wikivoyage**". No weather box. Back button / closing a place returns to it.
- Phone: photo **left (≈ 125×95)**, title right; the rest full width.
- **Verify:** template tests: city panel has no weather box and links Wikivoyage; place order; badge and arrows absent with one photo; Wikipedia card absent without a URL; directions URL; labels "Save" and "Add to your plan" (accessible names still say what is saved). JS tests: carousel ‹ › and live "1 of N"; bookmark mirrors Save. Browser at 375 and 1440 against the mockup crops.

### V9. Facts card: hours and best-time tip
- [x] Done 2026-10-15

**Owns:** `public/redesign/opening-hours.js` (new), `public/redesign/best-time.js` (new), `tasks/redesign/checks/opening-hours.test.mjs` (new), `tasks/redesign/checks/best-time.test.mjs` (new), `views/index.go.tpl` (script tags only), `views/place_panel.go.tpl` (facts markup only), `public/redesign/map-panel.css` (facts card only)

- Parse simple OSM rules (`Mo-Su 10:00-18:30`, `Tu-Su 10:00-17:30; Mo off`) into **today's range** "10:00 – 18:30" or "Closed today". Anything else: the raw text. **No "Open now"**, no "Last entry".
- **Best time (D14):** map the place kind to the plan's rule table; show the value plus a "**Tip: …**" line; hidden for kinds with no rule. Owner reviews the wording.
- Price column is **hidden** (D13); the card is hidden when every column is empty.
- **Verify:** `node --test tasks/redesign/checks/opening-hours.test.mjs`: simple range, closed day, multiple ranges, unknown syntax → raw, empty → hidden. `node --test tasks/redesign/checks/best-time.test.mjs`: museum → Morning + tip, viewpoint → Sunset, unknown kind → hidden, matching is case-insensitive. Browser: Belém shows "10:00 – 18:30".

### V10. From our writers
- [x] Done 2026-10-15

**Owns:** `views/place_panel.go.tpl` (writers card markup only), `public/redesign/map-panel.css` (writers card only), `public/redesign/map-panel.js` ("See all" only), `internal/adapters/httpserver/placepanel_test.go` (writers card tests)

- Lavender card, purple quill heading "**From our writers**", **"See all ›"**. Quote: the **first post title** in italic serif, "— {Blog name}", linked. "See all" expands the full post list (title, blog, language). With no mentions: "No writer mentions yet", no "See all".
- **Verify:** template tests: card with 0, 1 and 2 mentions; links open in a new tab with `noopener`. Browser: "See all" toggles `aria-expanded`.

### V11. Teaser cards
- [x] Done 2026-10-15

**Owns:** `views/overview_teasers.go.tpl` (new), `public/redesign/teasers.css` (new), `views/content_fragment.go.tpl` (include only), `views/index.go.tpl` (stylesheet link only), `internal/adapters/httpserver/presentation.go` (teaser view data only), `internal/adapters/httpserver/teasers_test.go` (new)

- **Plan your days**: Day 1 of the plan; **D16:** without a plan, the top 3 notable places labelled "Suggested day". › goes to Itinerary.
- **Where to stay**: first stay (photo, serif name, "Kind · N-star property"), "View all →" to Stays; no price **(D17)**. Without a plan: "Generate a plan to see places to stay" + link.
- **Travel videos**: two thumbnails, round play button, title; › to Videos; no duration **(D18)**. Hidden when there are no videos (Porto).
- Phone: "**Plan your day in {City}**" compact card with "View all →"; stays and videos follow **(D9)**.
- **Verify:** Go tests: each card's three states (plan / no plan / no data); links target the right section. Browser: three cards in one row at 1440, stacked at 375; section links switch the view.

### V12. Phone: inline card, no sheet; bottom nav icons
- [x] Done 2026-10-15

**Owns:** `public/redesign/map-panel.js` (phone only), `public/redesign/map-panel.css` (phone only), `views/index.go.tpl` (bottom nav only), `public/redesign/navigation.js` (bottom nav only), `public/styles.css` (bottom nav rules), `tasks/redesign/checks/pr8c-journeys.browser.mjs`

- **D5:** remove the half-height sheet; picking a pin fills the inline place card and scrolls it into view (focus moves to its title).
- Bottom nav: **icon above label**, Overview `bi-compass-fill` in a terracotta circle when active, `bi-map`, **bed** (inline SVG, **D19**), `bi-play-circle`; terracotta underline below the active tab.
- **Verify:** browser at 375: pin → card visible, focus on its title; Escape/back returns to the map; active tab has `aria-current` and the underline; nav doesn't cover the last card; no overflow.

### V13. Journeys, regressions, evidence and docs
- [x] Done 2026-10-15

**Owns:** `tasks/redesign/checks/*` (rewrites only), `docs/tdd/look-v3.tdd.md` (new), `docs/design.md`, `tasks/look-v3-plan.md` (status), `tasks/artifacts/look-v3/`

- Rewrite the v2 checks that assumed the sheet, short labels, writer card and Layers button; delete none.
- Run `make test lint build`, the a11y audit and every browser check at 375 and 1440 (MapLibre and Leaflet).
- Final **side-by-side screenshots** with the mockups; list remaining differences in the TDD report.
- Update `docs/design.md`: layout, panel, phone flow, tokens. Fix its reference links (it calls `phone.png` the desktop reference).
- **Verify:** all green; ≥ 80% coverage on changed Go packages; the TDD report follows AGENTS.md §5.

### V14. Polish: close the remaining visual gaps
- [x] Done 2026-10-15

**Owns:** `public/redesign/map-layers.js` / `.css`, `internal/adapters/httpserver/placekind.go` (new) / `placekind_test.go` (new), `internal/adapters/httpserver/writermap.go` (`kind_label`), `views/place_panel.go.tpl`, `public/redesign/map-panel.js` / `.css`, `public/redesign/best-time.js`, `public/redesign/writers-map.js` (attribution), `public/redesign/map-pins.css`, `public/redesign/teasers.css`, matching checks, `docs/tdd/look-v3-polish.tdd.md` (new), `docs/design.md`

- Pills follow **D21**: Notable places is filled while on and toggles like the others; on overlays are white. Each loaded destination starts with only Notable places on.
- Readable place types ("tower" → "**Historic Landmark**") in the panel, popup and callout.
- Towers, castles, forts and monuments get the Morning tip.
- Writer quote is plain italic text; the blog name links to the post.
- Phone map: credits start collapsed; no duplicate name label on the selected pin.
- Phone card: title on one line, photo credit overlays the photo, link titles not cut off.
- Phone "Plan your day": one photo, Morning and Afternoon in a row on a timeline.
- Desktop "Where to stay" on one line with a tinted "View all →" pill; callout › enlarged.
- **Verify:** unit tests, `look-v3-shell.browser.mjs` at 375 and 1440, full browser sweep and a11y audit.
