# Look v3: match `Desktop.png` and `Phone.png`

**Status:** complete, 2026-10-15. [Evidence](../docs/tdd/look-v3.tdd.md) and [visual comparisons](artifacts/look-v3/README.md). Polish pass V14 done the same day ([evidence](../docs/tdd/look-v3-polish.tdd.md)). Tasks: [look-v3-todo.md](look-v3-todo.md). Owner decisions **made 2026-10-15** ([Decided](#decided-2026-10-15)).

**Goal:** make the destination page look as close as possible to [Desktop.png](../desktop.png) (1448 px wide) and [Phone.png](../phone.png) (≈390 px wide). Use **data we already have**. Anything that needs new data goes to an owner decision or **Later**.

**Builds on** [map-look-v2](map-look-v2-plan.md) (done 2026-10-14). Much is already in place: brand, header bookmark, banner, pills, dark map, teardrop pins, callout, "View full map" card, toast. This plan covers **only the differences**.

**Baseline:** screenshots of today's `design-preview -scenario writers-map` (Lisbon) at 1440 and 390 px were compared with the mockups on 2026-10-15.

---

## The biggest changes

1. **Desktop gets a second row under the map**: three cards (**Plan your days · Where to stay · Travel videos**). The map gets **shorter** (≈505 px, not a full screen).
2. **The place panel is restyled**: photo carousel, icon meta row, icon buttons, link cards with icon tiles, a **facts card** (Hours · Price · Best time) and **From our writers**. The writer card and post list go.
3. **The phone becomes a scrolling page**: a short hero, pills, a **small map (≈205 px)**, then the **place card inline** (not a sheet), then **Plan your day**. The bottom nav **gets icons back**.
4. **The banner is slimmer.** Desktop has one row: city text on the left, a **horizontal weather strip (+ sunset, "Forecast ›" popover) + "Plan your trip ↗"** on the right. Phone has a **compact weather chip**.

---

## Differences, area by area

Measurements are read off the mockups. Desktop is in CSS px at 1448. For the phone, image px ÷ 1.86 = CSS px.

### 1. Header

| | Today | Target |
|---|---|---|
| **Desktop order** | brand · search · tabs (left of centre) · bookmark | brand · **search (≈570 px, centred)** · **tabs pushed right** · bookmark (44×44, far right) |
| **Placeholder** | "Search destinations…" | "**Search destinations, cities, or places…**" |
| **Phone** | brand ≈ 24 px | brand **larger** (peaks ≈ 38 px wide, wordmark ≈ 26 px); placeholder in **serif**; search ≈ 32 px below the brand |
| **Tabs** | already terracotta + underline | same, underline **3 px**, ≈ 80 px wide |

### 2. Destination banner

| | Today | Target |
|---|---|---|
| **Desktop height** | ≈ 320 px | **≈ 157 px**, full width, radius 12 |
| **Text** | name, country, tagline, photo caption | name (serif ≈ 60 px), country (serif bold ≈ 24), tagline (serif ≈ 19, **one line**). **No photo caption** ("Above the rooftops…") |
| **Weather (desktop)** | tall card: temp, details grid, 4-day forecast | **One white strip ≈ 680×86, radius 8**: icon · **temp (bold 18) / condition (bold 12) / feels like (11)** · divider · **Humidity** · **Wind** · **Waves** (blue icon, bold value, grey label) · **"Plan your trip ↗"** button (terracotta, ≈ 154×46) |
| **Sunset, forecast** | shown | **Kept (D3)**: **Sunset** joins the strip after Waves; a **"Forecast ›"** button opens a popover with the 4-day forecast (Escape closes, focus returns) |
| **Phone** | photo, then a white card with tagline + weather + forecast | **≈ 100 px photo, all text on the photo** (name ≈ 32, country, 2-line tagline). **Weather chip** top right (≈ 126×38): cloud-sun icon, temp, "Feels like", **›** opens the same popover with humidity, wind, waves, sunset and the forecast |
| **Photo credit** | visible on the photo | must stay (CC BY-SA). **Tiny translucent text at the bottom right** (D4), smaller than today, links still reachable |

### 3. Layer pills

| | Today | Target |
|---|---|---|
| **Pressed state** | white with a border | **Active pill filled terracotta, white text and icon**; others white with a border |
| **Icons** | small line icons | **Coloured icon, ≈ 20 px**: pin (terracotta, white when active), **pencil-square purple** (Writers' pins), **tablet/route purple** (Itineraries), **orange map tile** (Your plan) |
| **Labels** | "Writers' itineraries"; phone short labels | "**Notable places · Writers' pins · Itineraries · Your plan**" on **both** widths (phone row scrolls) |
| **Size** | ≈ 32 px tall | ≈ **40 px** desktop, ≈ 30 px visible on phone. **Hit area stays ≥ 44 px** (a11y rule) |
| **Phone scrollbar** | thin scrollbar visible (v2 gap) | **hidden** scrollbar, row still scrolls |

### 4. Desktop layout

- Left column ≈ **865 px**: pills → **map (≈ 865×505, radius 16)** → **three teaser cards** (≈ 285 px each, radius 12, gap 16).
- Right column ≈ **510 px**: one white **place panel** card (radius 16), top aligned with the pills and as tall as the left column. Today the panel is ≈ 400 px and the map fills a screen.
- No footer on this page (as today).

### 5. Map chrome

| | Today | Target |
|---|---|---|
| **Top left** | writers' itinerary picker | **Round back button "‹"** (44 px, dark translucent) (D6). The itinerary picker moves inside the Itineraries pill (D6) |
| **Compass** | "N" text in a circle | **Red needle above a white "N"** |
| **3D** | "2D" text button | same, styled like the others |
| **Locate** | `⌖` glyph | **`bi-crosshair`** icon |
| **Zoom** | two separate buttons | **one tall pill: + / divider / −** |
| **Layers button** | shown | **not in the mockup**: the pills do the same job (D7) |
| **District labels** | text in dark boxes | **plain white uppercase text with a soft shadow** (Campo de Ourique, Alfama…); **water labels italic blue** ("Tagus River") |
| **Pin icons** | generic | category icons: **museum `bi-bank`**, **food `bi-fork-knife`**, nature green with a tree/mountain, writers purple (the mockup shows a camera, D8) |
| **Callout** | name · area · › (photo optional) | **photo thumbnail ≈ 80×56**, **serif name ≈ 20**, kind in grey ("Historic Landmark"), **›**; white, radius 8, pointer to the pin |
| **Selected pin** | ring | ring **plus soft glow** |
| **Your plan route** | orange dotted | same, **thinner dots, closer together** |
| **Scale** | grey box "300 m" | **white line with end ticks**, "0" and "1 km" labels, no box |
| **Attribution** | wide white pill | **compact "ⓘ" that expands** (MapLibre `compact: true`). It must stay reachable |
| **"View full map"** | card with thumbnail | already matches. Phone: **hidden** (not in the phone mockup, D9) |

### 6. Place panel (desktop) / place card (phone)

One template, two layouts.

**Before a place is picked (D1)** the panel keeps a **city card in the same look**: city photo, serif title, meta row (`bi-geo-alt` + country), the Wikivoyage paragraph in serif, a filled **"Plan your trip"** button and one link card ("**Read on Wikivoyage**" / "City guide, CC BY-SA"). No weather box (it's in the banner). On the phone it fills the inline card slot under the map.

**A selected place**, order and look:

1. **Photo** (desktop ≈ 490×212, radius 10; phone ≈ 125×95 on the **left**): **"1 / N" badge** top right, **round ‹ › buttons** on the desktop sides. Badge and arrows show **only when N > 1** (D10). **No × on desktop**: the map's back button closes (D6).
2. **Title** in serif bold (≈ 30 desktop, ≈ 26 phone) + **bordered bookmark button** (44×44), as today.
3. **Meta row with icons**: kind icon (`bi-bank`…) + kind · `bi-geo-alt` + "Area, City". Serif, grey.
4. **Description** in **serif ≈ 15/1.5**, ink colour (today it's sans).
5. **Actions row:** **"⊕ Add to your plan"** filled (≈ 47% wide, 46 px tall) · **"Save"** with a bookmark icon · **"Share"** with `bi-share`, both outlined. Labels change: "Save to visit" → **"Save"**, "+ Add to plan" → **"Add to your plan"**.
6. **Two link cards side by side** (≈ 235×62): **icon tile** on the left (Wikipedia "W" serif / purple `bi-geo-alt-fill`), divider, **bold title + grey subtitle**, **↗ on the right**.
   - "**View on Wikipedia**" / "History, photos and more"
   - "**Get directions**" / "**Open in Google Maps**" (today OpenStreetMap, D11)
7. **Facts card** with three columns and terracotta line icons, split by dividers:
   - **Hours** `bi-clock`: big "10:00 – 18:30", small "Last entry 18:00" (D12)
   - **Price** `bi-ticket-perforated`: "€6", "Reduced rates available" (D13)
   - **Best time to visit** `bi-sun`: a **rule by kind**, shown as a **tip** (D14), e.g. "Morning" / "Tip: museums are quieter early". Hidden for kinds with no rule.
   - Columns with no data are **hidden**; the card is hidden if all three are empty.
8. **"From our writers"** card (lavender background, purple `bi-pencil-square`, purple serif heading, **"See all ›"**): an **italic serif quote** (D15). Replaces the **writer card** and the **Writers' posts** list.

### 7. Teaser cards (desktop row under the map; phone below the place card)

Each card: tinted terracotta **icon tile** (≈ 32 px), **serif title ≈ 20**, grey intro (2–3 lines), then a preview.

| Card | Icon | Intro | Action | Preview |
|---|---|---|---|---|
| **Plan your days** | `bi-calendar-week` | "Build your ideal itinerary with notable places, writers' picks and your saved spots." | round **›** → Itinerary | **Morning / Afternoon / Evening** dots (green, purple, orange) with place names and stacked thumbnails (D16) |
| **Where to stay** | `bi-building` | "Handpicked hotels near the best of {City}." | **"View all →"** pill → Stays | first stay: photo left, **name (serif)**, "Hotel · 5-star property", ~~From €320/night~~ (D17) |
| **Travel videos** | `bi-play-circle-fill` | "Watch destination guides, neighborhood walks and local experiences." | round **›** → Videos | first video thumbnail with a **round play button**, plus a second thumbnail with its **title** ("Lisbon in a Day / Ultimate Travel Guide"). ~~Duration "8:24"~~ (D18) |

Phone: the mockup shows **"Plan your day in {City}"** with "**View all →**", one photo on the left, then **Morning · Afternoon** in a row and a round **›**. Stays and videos follow in the same style (D9).

### 8. Phone layout

- Order: header → search → **hero (≈ 100 px)** → pills → **map ≈ 205 px tall**, radius 12 all round → **place card** → **Plan your day** → (stays, videos) → bottom nav.
- **No half-height sheet** (D5). Selecting a pin scrolls the place card into view and highlights it.
- Map controls shrink to fit: compass, 2D, locate, zoom pill (≈ 40 px visual, **44 px hit area**).

### 9. Bottom nav (phone)

| Today | Target |
|---|---|
| text tabs, underline on top | **icon above label**: Overview **`bi-compass-fill` in a terracotta circle** when active, Itinerary `bi-map`, Stays **bed** (D19), Videos `bi-play-circle`. **Terracotta underline under the active tab** (≈ 100 px wide, 3 px) |

### 10. Colours and type (no new tokens unless needed)

- Keep `--tt-accent #A64027` (the mockup's ≈ `#B5432B` would fail 4.5:1 on white text at small sizes; check before changing).
- New: `--tt-writer-tint` (lavender card ≈ `#F1EEF8`), `--tt-writer #6B4FA8` (purple text, ≥ 4.5:1 on the tint), `--tt-accent-tint` (icon tiles ≈ `#F8E6E0`).
- **Serif** (Georgia) for the panel description, meta row, callout and teaser titles; Inter stays for buttons, labels and numbers.

---

## Decided (2026-10-15)

Answered with the owner one by one. **Bold** = differs from the recommendation.

| # | Question | Decision |
|---|---|---|
| D1 | Panel before a place is picked | **Keep a city panel, restyled in the place-card look** (photo, serif title, country, guide paragraph, "Plan your trip", Wikivoyage link card; no weather) |
| D2 | Desktop map height | ≈ 505 px (`min(505px, 60vh)`), teasers below |
| D3 | Forecast and sunset | **Keep them**: sunset in the strip, the 4-day forecast in a **"Forecast ›" popover**. Phone chip › opens the same popover |
| D4 | Banner photo credit | **Tiny translucent text at the bottom right** |
| D5 | Phone place view | Inline card under the map; the v2 sheet goes |
| D6 | Map back button "‹" | Closes the selection and flies to the city; hidden when nothing is selected; replaces the desktop ×. Itinerary picker moves into the Itineraries pill |
| D7 | Layers button | Removed |
| D8 | Purple pin icon | Keep the quill |
| D9 | Phone extras | Hide "View full map"; show all three teasers |
| D10 | Photo carousel | Build the UI; it shows only when N > 1. More photos are Later |
| D11 | Directions | Google Maps link (`https://www.google.com/maps/dir/?api=1&destination=lat,lon`) |
| D12 | Hours | Today's range for simple rules, else the raw text. No "Last entry" |
| D13 | Price | Column hidden. Wikidata P2555 is Later |
| D14 | Best time to visit | **A simple rule by place kind, labelled as a "Tip"**; hidden for kinds with no rule (see below) |
| D15 | Writer quote | First post title as the quote, "— Blog name", linked; "See all ›" expands the list |
| D16 | Plan your days, no plan | "Suggested day" from the top 3 notable places; Day 1 once a plan exists |
| D17 | Stay price | Dropped |
| D18 | Video duration | Dropped |
| D19 | Bed icon | One inline SVG |
| D20 | Temperature | Whole degrees |
| D21 | Layer pills (asked during V14, revised the same day) | **Notable places, while on, is filled terracotta**; on overlays are white with a filled coloured icon. **Every pill toggles**, Notable places included (owner: "make it possible to unselect"); off pills are muted with an outline icon. **Each loaded destination starts with only Notable places on**; the choice is no longer remembered across loads |

**D14 rule table** (start here; the owner reviews the wording in V9):

| Kind contains | Best time | Tip |
|---|---|---|
| museum, gallery, palace, monastery, church, cathedral, tower, castle, fort, monument (added in V14) | Morning | "Tip: quieter early in the day" |
| viewpoint, miradouro, lookout, bridge | Sunset | "Tip: best light at golden hour" |
| park, garden, beach | Morning | "Tip: cooler and calmer early" |
| market | Morning | "Tip: freshest stalls in the morning" |
| anything else | — | column hidden |

## Later (each its own plan)
- More photos per place (Commons), the **Price** column (Wikidata P2555), real **writer excerpts**, **video duration**, **stay prices**, **"Last entry"**, crowd-based **Best time** (beyond the D14 rule).

## Risks
- **Shared files.** `map_card.go.tpl`, `place_panel.go.tpl`, `map-panel.js/.css`, `writers-map.js`, `writers-map-layout.css` and `weather_card.go.tpl` are edited by several tasks. **One agent, in order.**
- **Reverts v2 decisions** (sheet, text-only bottom nav, short pill labels, writer card, Layers button). The v2 checks (`map-panel.browser.mjs`, `pr8c-journeys.browser.mjs`, `writers-map-layout.browser.mjs`, `map-look-v2*.test.mjs`) must be **rewritten, not deleted**.
- **Accessibility:** the 30 px phone pills and 40 px controls still need **44 px hit areas**; the carousel needs labelled buttons and a live "1 of N"; the auto-selected place must not steal focus on load; the a11y audit stays green.
- **Best-time tips (D14)** are heuristics: always labelled "Tip", never shown as fact, and easy to edit in one table.
- **Shorter map** must still show the selected pin and its callout without clipping at 1440 × 900.
- **Fixture data:** the preview's Lisbon has one place and no photos, so the look can't match the mockup there. **Add a richer Lisbon fixture** (≈ 12 places across categories, photos, hours, 2 writer mentions) for screenshots.
- **Headless screenshots** need WebGL: start the browser with `--use-angle=swiftshader --enable-unsafe-swiftshader`.
