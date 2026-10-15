# Map look v2: match the new mockups

**Status:** done 2026-10-14 (L1–L11); verified in the [TDD evidence](../docs/tdd/map-look-v2.tdd.md), which lists the remaining visual differences. Decisions made with the owner the same day. Tasks: [map-look-v2-todo.md](map-look-v2-todo.md).

**Goal:** make the destination page look like [Unknown.png](../phone.png) (desktop) and [Unknown-2.png](../desktop.png) (phone), using **only data we already have**. Anything that needs new data, accounts or a new engine goes to **Later**. (Supersedes the earlier drafts based on `Desktop.png` / `Phone.png` and the first `Unknown` mockups.)

**Builds on** [map-first](map-first-plan.md). Start only after the map-first working tree is **committed** (it is still uncommitted).

## In scope

| Area | Today | Target |
|---|---|---|
| **Brand** | Signpost + "TravelTabByPatronFragas" in one word | **Solid terracotta peaks** (inline SVG) + "**TravelTab**" in the bold serif, "**ByPatronFragas**" on a **second line** in smaller terracotta sans |
| **Header** | Name, search with arrow, tabs | Same tabs; **bookmark** in a bordered rounded button (saved list). Search: rounded field, **no clear ×**, **no arrow** (Enter submits). Desktop page sits in a soft rounded frame |
| **Layer pills** | Chips above the map | A **band of white pills above the map**, each with a coloured icon: **pin** (notable), **quill** (writers' pins), **dotted route** (writers' itineraries), **star** ("Your plan"). Phone: short labels **Notable · Writers · Itineraries · Your plan** |
| **Map frame** | Full bleed under the header | **Rounded map card (16 px)** with page margins; desktop panel a **separate rounded card** on the right; phone map rounded at the top |
| **Map controls** | White buttons | **Dark, translucent** column top right: round **compass N**, locate, **3D**, zoom +/−, layers. **Scale bar** bottom right |
| **Pins** | Blue circles, terracotta writers, turquoise plan | **Teardrop pins**: notable **terracotta** with a white category icon (camera, food, museum…), **nature green**; writers' pins **purple + quill**; clusters **grey circles** with the count; your plan **orange numbered** teardrops + **orange dotted route**; writers' itineraries **purple dashed** |
| **Labels** | White boxes above a zoom | Place names in **white text on dark translucent rounded boxes** next to pins |
| **Selected place** | Purple pin + photo callout | **Terracotta pin with a ring** + white callout: **photo thumbnail · name · area · ›** |
| **"View larger map ↗"** | Small white link | **Card**: city thumbnail, "**View full map**", "**Open {City} on a larger map**", external-link icon. Same external URL |
| **Saving** | Saved-count line under the map | Dark toast "**Added to your saved places · Undo**" at the bottom of the map. The "N saved places will be included · Clear" line moves to the **Itinerary planner** |
| **Place panel** | Photo, tabs, actions, posts in a list | Photo with ×; bold serif title + **bordered bookmark button**; "Kind · Area"; **writer card**; **one paragraph**; **+ Add to plan**; **Save to visit** + **Share**; two **link cards** (Wikipedia, Directions); **Hours card** (expandable); "**Writers' posts (N) ›**" with **post cards** |
| **Phone** | Peek card → sheet; bottom nav with icons | Place opens a **half-height sheet** with a handle and an inset rounded photo. **Bottom nav: text tabs with an underline**, no icons |

## Decided (2026-10-14)
- **Brand:** peaks logo and two-line wordmark. Accessible name, `<title>`, trip titles and ICS text stay **`TravelTabByPatronFragas`**.
- **Palette:** the colours above replace map-first's. Pills, pins, routes, legend and `docs/design.md` change together. Every pin colour ≥ **3:1** on the dark map.
- **Search:** no clear × (owner) and, matching the mockup, **no arrow button**. The search keeps an accessible name and submits on Enter and on picking a suggestion.
- **Panel text:** no tabs. **AI summary** when there is one, else the **Wikidata description**.
- **Writer card** (lavender, quill, ›), first that applies:
  1. In a loaded writers' itinerary → "**Featured in N writers' itineraries**", "Including {blog names}". Computed in the browser from the itineraries already in the GeoJSON (stops are QIDs).
  2. Mentioned → "**Mentioned by N writers**" + monograms.
  3. Neither → "**No writer mentions yet**", no ›.
  - › scrolls to the writers' posts.
- **Link cards:** **Wikipedia** ("View on Wikipedia", hidden without a URL) and **Directions** ("Open in OpenStreetMap", new tab, no walking time).
- **Hours card:** clock, "Hours", the opening-hours text we store, ▾ to expand when long. Hidden when there is none. No "Open now".
- **Post cards:** title + blog name and language, linking to the post. **No thumbnail or excerpt** (we never store post content; thumbnails are a contract change). A neutral quill tile stands in for the thumbnail.
- **Toast Undo:** removes the place just saved.
- **Map look:** dark vector map stays (no satellite, no glow), as decided in map-first.

## Later (each its own plan)
- **Our own full-screen map.**
- **Official site** and **street address** (new Wikidata fields: backend and frozen-contract change).
- **Post thumbnails** (frozen-contract change).
- **Computed "Open now"** (parsing OSM opening-hours rules).
- **Photo gallery**: we have one photo per place.
- **Pill dropdowns** and **More filters**.

## Dropped
- **Profile avatar**, **rating**, **price**, **"Cultural highlight"** chip, **Add note**, **post excerpts**.

## Risks
- **Shared files.** `map-panel.js`, `map-panel.css`, `writers-map.js`, `map-pins.css` and `map_card.go.tpl` are edited by several tasks. **One agent, in order.**
- **Palette and pin shape** change the pins, day plan, itineraries and legend at once; unit tests that assert colours change with them.
- **Brand change** breaks checks that read the brand text (`a11y-audit.mjs` reaches `a.tt-brand "TravelTabByPatronFragas"`). Keep the accessible name; update the checks.
- **No search button:** checks that click `.tt-search-submit` must submit with Enter instead.
- **Checks assume today's panel and phone flow** (`map-panel.browser.mjs`, `pr8c-journeys.browser.mjs`, `writers-map-layout.browser.mjs`). Rewritten, not deleted.
- **Accessibility:** dark controls and label boxes need ≥ 3:1 / 4.5:1 and visible focus; the sheet needs a keyboard way to expand; the toast is a polite live region with a focusable Undo. The a11y audit must stay green.
- **Compass** is MapLibre only; hidden on the Leaflet fallback, like 3D.
