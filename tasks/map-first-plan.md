# Map-first destination page

**Status:** implemented and verified locally; only the owner's real-phone check (M8) is open. Owner decisions made 2026-10-14 (below). Tasks: [map-first-todo.md](map-first-todo.md).

**Goal:** make the destination page look and work like the mockup [TravelTab Funchal Map Experience.png](<../TravelTab Funchal Map Experience.png>), with a **dark 3D vector map inside a light TravelTab UI**:
- the **map is the page**: full height under the header, with the place panel docked on the right (desktop) or a bottom card and sheet (phone);
- **no satellite**: the look comes from a tuned dark vector style, 3D buildings and terrain;
- **richer pins**: category icons, writer clusters, labelled callouts with photos, numbered day-plan stops;
- a **redesigned place panel**: hero photo, tabs, primary actions, writers' posts list.

**Independent of the remaining Writers' Map work.** Open items in [todo.md](todo.md) (8b, 8d, 9a–9d) are measurement, release and backend. None of them touch the files below.

## Decided (2026-10-14, from the owner's map-stack notes)
- **Stack stays:** MapLibre GL JS 5.24 + OpenMapTiles schema (served by OpenFreeMap) + OSM data + Mapterhorn terrain. **No library change.**
- **No satellite imagery.** The mockup's satellite look is replaced by the dark vector style. The photoreal city isn't the target.
- **Dark map, light app.** The map keeps its own palette; the app UI stays ivory/white/charcoal/terracotta (current `docs/design.md` tokens).
- **TravelTab data stays vibrant** on the dark map: notable **blue**, writers **terracotta**, day plan **turquoise**, writers' itineraries **purple**.
- **TravelTab owns discovery:** hide generic OSM POIs (shops, ATMs, parking, small amenities). Keep city/district labels, major roads and streets, parks, water, major landmarks.
- **No glow effects** (lit roads, glowing windows, water reflections). Target is Electroverse-like: dark background, clear roads, subtle buildings and landcover, clean labels.
- **Tiles: OpenFreeMap now.** Self-hosted PMTiles (Portugal first) stays the **escape hatch**, as its own plan. It costs storage and bandwidth, so it conflicts with "free sources only" until there's a free host.
- **Never `tile.openstreetmap.org` as the primary source.** Today it's only the Leaflet fallback's per-error backup, which is within the tile policy.

## Today vs the mockup

| Area | Today | Target |
|---|---|---|
| Page | Long scroll: weather hero → Writers' Map card (65/35) → planner → city guide → itinerary → stays → videos | Map fills the screen under the header; Overview / Itinerary / Stays / Videos are **tabs** |
| Brand | Signpost icon, Georgia headings, terracotta, cream | Same tokens and signpost icon; name becomes **TravelTabByPatronFragas** |
| Header | Search with arrow submit, text nav | Search with arrow submit and **no clear ×** (owner, 2026-10-14), tabs with underline, **Save trip** (later) |
| Base map | OpenFreeMap dark + Electroverse palette, 3D buildings (height falls back to **0**, so untagged buildings are flat), Mapterhorn terrain; generic POIs visible | Same base, tuned: **POIs hidden**, **fallback building height**, slightly warmer tint to sit with terracotta, label density by zoom |
| Layer chips | Four chips above the map | Phone: chips under the search. Desktop: in a layers menu |
| Map controls | Zoom, scale | **Locate me**, **2D/3D**, **layers**, zoom, scale; phone **locate FAB** |
| Pins | HTML markers, one per place | Notable: **category icon in a blue circle**. Writers: **terracotta, clustered** (native MapLibre). Day plan: **turquoise numbered**, dashed route. **Photo callouts** on selected place and plan stops |
| Place panel | Photo, meta, monograms, AI summary, hours, Wikipedia, "Save to plan", posts in a `<details>` | Hero photo + ×, **tabs**, **Add to plan** / **Save to visit** / **Share**, "Writers' posts (N)" list |
| Phone | Panel stacks under the map | Full-screen map, **bottom peek card**, bottom sheet, bottom nav with active tab |

## Owner decisions (2026-10-14)

1. **Tabs switch views.** Overview = full-height map + panel. Itinerary = planner form + itinerary. Stays and Videos as today. Hash URLs (`#itinerary`) keep working. Weather and the city guide live in the **panel's default state**: compact weather, "About Funchal" excerpt, "Plan your trip" → Itinerary tab.
2. **Keep the signpost icon; rename the product to exactly `TravelTabByPatronFragas`** (one word) everywhere a user sees the name:
   - header wordmark and `aria-label`, footer brand, default `<title>` (`views/index.go.tpl`);
   - trip page titles "… — TravelTabByPatronFragas" (`internal/adapters/httpserver/trip.go:327-329`);
   - ICS event description "Part of your TravelTabByPatronFragas trip" (`export_ics.go:82`);
   - README heading and first line.
   - **Unchanged** (not user-facing, or would break things): HTTP `User-Agent` and the robots.txt group name (`api/useragent.go`, `api/wordpress.go`; writers' robots rules match on it), ICS `PRODID`, Go identifiers, CSS classes.
3. **Clusters show the number of places**, in a larger ring than a single pin (which shows its writer count).
4. **Add to plan** = save the place and regenerate the itinerary now. **Save to visit** = today's "Save to plan" (included next time a plan is generated).
5. **Ship now:** locate me, 2D/3D, layers menu, phone peek card and bottom sheet, Share (Web Share + copy-link fallback), Details tab.
   **Later, own plans:** Photos tab, Nearby tab, Save trip, writer avatars and post thumbnails (frozen-contract change), PMTiles hosting.
6. **Follow-up (2026-10-14):** no grey clear cross in the search box (neither ours nor the browser's); rounder corners like the mockup (12 px photos and search, 10 px buttons and map controls, pill layer chips); the panel's default state opens with the **city photo** and its credit.

## Design

**Map style** (in `writers-map.js`'s palette step; no new style file unless tuning outgrows it)
- Hide OpenFreeMap `poi` symbol layers below a "major landmark" rank; keep `place_*`, road names and water names.
- `fill-extrusion-height`: `coalesce(render_height, 8)` so untagged buildings get a low, uniform height. `render_min_height` stays 0.
- Tint: nudge background/landcover from blue-grey towards navy + muted olive. Must keep road and label contrast; the pin colours must stay readable (≥ 3:1 against the background).
- Terrain stays on in 3D; no hillshade (unchanged decision).
- The **Leaflet fallback** keeps EOX imagery. It won't match the dark style; accepted, it's for no-WebGL and outages only.

**Layout**
- Desktop: header 72 px; map `flex: 1` and a ~400 px panel, both `100dvh − header`. Panel scrolls on its own. Attribution stays on the map.
- Phone: map fills between the header block and the bottom nav. Peek card above the nav; tapping opens the panel as a **bottom sheet** (Escape and × close, focus returns to the pin).
- Tabs reuse `navigation.js` and `shell_section_links`; inactive views get `hidden`. The map **stays mounted** across tabs.

**Map controls** (one `context.draw.control` each, both engines)
- **Locate me:** one-shot `navigator.geolocation`, fly there, show a dot. Denied → short status, no retry.
- **2D/3D:** pitch 0 + terrain off ↔ pitch 55° + terrain on. Hidden on Leaflet.
- **Layers:** the four overlay toggles from `map-layers.js`. Phone keeps the chips too.
- Zoom and scale bottom-right.

**Pins**
- **Notable:** HTML marker, Bootstrap icon by category inside a blue circle. Category from `Place.Kind` via a fixed table (viewpoint, museum, food, park/garden, trail, church, other).
- **Writers:** a **GeoJSON source with `cluster: true`** plus circle + count layers, so MapLibre clusters natively. Click a cluster → zoom to its expansion; click a single pin → today's popup → panel. This adds `draw.clusters(id, features, style, onClick)` to the drawing API. **Leaflet fallback:** same call draws unclustered markers (no new library).
- **Day plan:** turquoise numbered pins; route line turquoise and dashed (`line-dasharray`). Route stays straight stop-to-stop lines; real walking geometry needs a routing engine and is out of scope.
- **Writers' itineraries:** purple line, unchanged behaviour.
- **Callouts:** name labels on top places above a zoom threshold; photo thumbnail on the selected place and plan stops with a photo, lazy-loaded.

**Place panel:** template and CSS rewrite in the mockup's order. Same data as today; no new endpoint.

## Risks
- **Clusters change the drawing API.** It's internal JS (not the frozen Go contract), but both engines and the `map-panel.js` popup flow must keep working. Keyboard access: single pins stay focusable; clusters get a button equivalent in the keyboard list.
- **Building fallback height** makes some areas look uniform. Accepted (OSM heights are incomplete).
- **Performance:** photo callouts add images; clustering cuts DOM markers. `slow-pins` measurements (if done first) set the budget. Real-phone check from `map-3d` M7 is still pending.
- **Accessibility:** tabs need `aria-current` or `tablist` semantics; the sheet needs focus handling. The a11y audit must stay green.
- **Existing browser checks assume the old layout** (`writers-map-layout.browser.mjs`, `pr8c-journeys.browser.mjs`, `hero-phone.browser.mjs`). Rewritten, not deleted.
- **Open plans on the same files:** `dayplan-pins` owns `map-dayplan.js`; `slow-pins` makes scratch edits to `writers-map.js` and `map-panel.js`. Run those **first**.
