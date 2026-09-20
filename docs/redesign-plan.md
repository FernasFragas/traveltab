# TravelTab redesign — reference implementation plan

Status: ready for implementation. This document replaces the earlier exploratory plan with a concrete visual target based on the user-supplied image. Website implementation has not started.

## Source of truth

[Reference image: goalimg.png](../goalimg.png).

Match the central desktop website and the interface inside the phone as closely as possible. The large surrounding slogans, decorative scenery, handwritten poster copy, and phone hardware present the design; they are not additional website panels. This interpretation keeps the responsive site faithful to the actual interface shown.

The user's confirmed decisions remain:

- Balanced dashboard, with weather, map, planning, accommodation, and travel videos easy to reach.
- Travel magazine appearance with destination photography, bold headings, and warm colors.
- One scrolling page with section shortcuts, not mutually exclusive tabs.
- Expandable itinerary days, with the first day initially open and the others closed.

The image now controls proportions, visual hierarchy, component density, and desktop/mobile arrangements. Functional substitutions for unsupported mockup details are recorded below.

## Desktop composition

```text
┌───────────────────────────────────────────────────────────┐
│ TravelTab    Section links              Search destination │
├───────────────────────────────────────────────────────────┤
│ Lisbon skyline photo             │ Lisbon                 │
│ Optional small editorial caption │ Short destination line │
│                                  │ Temp / humidity / waves│
│                                  │ Editorial line/country │
├───────────────────────────────────────────────────────────┤
│           Overview    Itinerary    Stays    Videos         │
├───────────────────────────────────────────────────────────┤
│ Overview                                                  │
│ Brief subtitle                                            │
│ Destination map              │ Plan your trip             │
│                              │ Start → derived end | Days │
│                 View map ↗   │ [ Generate plan → ]         │
├───────────────────────────────────────────────────────────┤
│ Your itinerary                  Calendar export Map export│
│ Brief subtitle                                            │
│ Day 1 · date          Forecast       Walking distance   ∧  │
│ [1 Photo] [2 Photo] [3 Photo] [4 Photo] [Directions]       │
│   Name      Name      Name      Name                      │
│   Type      Type      Type      Type                      │
│ Day 2 · date          Forecast       Walking distance   +  │
│ Day 3 · date          Forecast       Walking distance   +  │
├───────────────────────────────────────────────────────────┤
│ Where to stay                                Check prices │
│ [Photo | property] [Photo | property] [Photo | property]   │
├───────────────────────────────────────────────────────────┤
│ Explore through video                      View all videos│
│ [Thumb | title] [Thumb | title] [Thumb | title] [Thumb...] │
├───────────────────────────────────────────────────────────┤
│ TravelTab                          Sources and real links │
└───────────────────────────────────────────────────────────┘
```

Use a centered cream shell with a subtle border and shadow, approximately 1120–1200px maximum width and 20–24px internal gutters. At narrower desktop widths, retain the reference's compact proportions without shrinking body text excessively. The screenshot is a scaled presentation board, not a requirement to fit the entire website into every viewport.

The hero has approximately 58% photograph / 42% information, around 220–260px height at full desktop width. The overview map and form share approximately equal columns with a 16px gap and roughly 240px height where content allows. Content controls height when names wrap or text is enlarged; do not clip to hit a measurement.

Use compact section spacing, around 16–24px, and aligned heading/action rows. Replace the previous large nested dark cards with a continuous editorial page, fine borders, and quiet cream surfaces.

## Mobile composition

1. Small TravelTab wordmark and working menu button; destination search on its own row.
2. Short skyline photograph with the city name overlaid near the lower left.
3. Three compact weather metrics on a solid surface below the photo.
4. Icon-and-label section shortcuts, with an active terracotta underline.
5. Plan your trip form, then a compact city map, matching the phone reference.
6. Your itinerary, with wrapping day summaries and compact image-left stop rows inside the open day.
7. Where to stay, with image-left accommodation rows.
8. Explore through video, with compact thumbnail-and-title rows.
9. Persistent bottom navigation with four working section destinations: Overview, Itinerary, Stays, Videos.

Keep all actual stops and accommodations accessible; the phone mockup's single sample rows are illustrative, not a truncation rule. Use 12–16px page gutters, comfortable touch controls, bottom safe-area padding, and sufficient space below the last content for the fixed bar. Collapse the overview to one column below approximately 900px; use the mobile header/bottom navigation below approximately 640px. Verify intermediate widths instead of treating breakpoints as fixed device categories.

## Visual specification

| Element | Reference-aligned target |
| --- | --- |
| Background | Warm cream `#F7F3EB`; optional very subtle texture, never behind small text |
| Surfaces | Warm white `#FFFDFA`, no gray glass or heavy blur |
| Text | Near-black `#24221F`, supporting gray-brown `#666158` |
| Borders | Fine warm gray `#DED8CC` |
| Main action/active accents | Burnt terracotta `#A64027`; darker hover `#87321F` |
| Chips | Pale olive/sand backgrounds, dark readable labels |
| Weather | Golden condition icon; restrained blue humidity/wave icons |
| Headings/wordmark | Georgia or comparable editorial serif; city approximately 36–42px, section titles 24–28px desktop |
| Body/controls | Existing Inter, generally 14–16px; secondary metadata approximately 12–13px where readable |
| Corners | Approximately 8px, consistent and restrained |
| Shadows | Subtle shell shadow; minimal individual-card shadows |
| Spacing | 4/8/12/16/24/32px scale |
| Icons | One consistent Bootstrap Icons family; small and paired with useful labels |

Treat these colors as starting values sampled by visual judgment, not claimed exact pixel measurements. Verify actual text contrast and refine against screenshots. Preserve the reference's editorial serif/sans contrast, cream surfaces, thin separators, and image-led content before adding decorative details.

## Component details

### Header, navigation, and footer

Build a compact wordmark instead of the current oversized logo. The desktop search is a pill-shaped field aligned right. Provide functional editorial navigation; do not create dead Destinations/Guides/About or profile links simply to copy the mockup.

The four section shortcuts are real anchors. Keep their targets present before results exist. The active underline reflects the current section; sticky navigation respects heading offsets and reduced motion. The mobile menu opens working navigation and closes with keyboard support. The bottom bar uses existing sections rather than unimplemented Saved/Profile destinations.

Define the footer once outside the search swap region. Include existing real social links and readable source credits. Do not add fake legal pages or copy the old fragment's placeholder links.

### Destination banner and current weather

Use a warm, real Lisbon skyline photograph with architectural detail and a river view as close to the reference composition as available imagery allows. A small italic caption can sit over the photo when configured. Desktop city information sits beside it; mobile moves the city name onto the photo and metrics below.

Display actual current temperature/condition, humidity, and wave height. Preserve readable value/label hierarchy and subtle vertical separators. Use correct condition icons. Optional city tagline, editorial line, and country label come from curated metadata; unknown cities retain their own identity with a designed neutral hero fallback.

### Overview and form

Retain the actual city map embed, cropped/framed cleanly with rounded corners and a working external map link. Match surrounding dimensions and colors; the provider controls the internal tile style and markers. No static mock map in production.

Plan your trip uses a serif title, short supporting text, date group and duration selector, a wide terracotta Generate plan button, and a small explanatory line. Keep the existing start-date and 1–5 day inputs. The end shown beside the start is derived and read-only: last itinerary day equals start plus days minus one. Accommodation checkout behavior remains separate.

### Itinerary

Use a single bordered accordion group rather than individually nested dark day cards. Each summary aligns a day marker, day/date, forecast status, walking icon/distance, and expand/collapse indicator. Open the first day on initial rendering and after a new successful plan. Multiple days may remain open for comparison.

On desktop, expanded stops form a row of up to four landscape photographs with numbered circular badges, short names, and indoor/outdoor chips. Put walking directions in a quiet tile at the end or on a separate row when space is tight. On phones, use image-left rows. Preserve credits and a deliberate missing-image treatment.

Export actions sit beside the section heading. Links remain separate from accordion toggles. Keep missing/uncertain forecast labels honest and readable. Successful planning brings the itinerary into view and announces completion; failures show the form error, retain entered values, and remove obsolete results.

### Accommodation

Match the reference's compact image-left cards: approximately three across desktop, stacked on mobile. Include name, actual type, known stars, and website, with Check prices aligned to the section heading. Additional returned stays wrap into further rows.

Optional verified property photography enriches the presentation. Missing photos use a quiet placeholder; unknown ratings or unavailable prices are not invented. Keep existing unavailable-service notes and dated booking handoff behavior.

### Videos

Use compact thumbnail-and-title cards, approximately four across desktop and stacked on mobile. Show the first four with a working View all videos expansion if more exist. Load a player only after activation, keep a real YouTube watch link available, and handle failed images or unavailable videos. Runtime is omitted unless supported by a later data change.

## Reference details versus available data

| Reference detail | Current capability | Implementation decision |
| --- | --- | --- |
| Lisbon hero and editorial copy | No city hero field | Add optional local curated metadata and a real credited Lisbon image; neutral fallback for other cities |
| Hotel photographs | Planner stays have no photos | Optional identity-matched curated images; designed placeholder otherwise |
| Day temperature and sunny icon | Planner exposes rainfall only | Use rainfall/status in the same visual slot; never reuse today's temperature as a daily forecast |
| Visit start/end times | No actual visit-duration schedule | Omit time ranges; retain stop order/type. Calendar export's assigned times do not establish visit durations |
| Video duration | Source exposes title and ID only | Omit runtime; derive thumbnail/watch URLs from the real ID |
| Pale street map with selected pins | Existing Waze city embed | Retain functional embed and attribution; document internal style differences |
| Account, Saved, Profile, guide/legal pages | No corresponding implemented flows | Use working section navigation and real footer links |
| Date-range control | Start date plus number of days | Native start input plus derived inclusive end label and duration |

These substitutions preserve the visual hierarchy without presenting fictional travel information. Exact imagery/provider styling may differ; that is a recorded fidelity limit, not permission to replace the reference layout with a generic dashboard.

## Asset handling

Keep the original supplied reference file intact and linked from the task documentation. It is not a production background or a source of cropped interface/landmark assets.

Add local destination/stay images with a manifest recording identity, source, license, credit URL, alt text, and dimensions. Use verified real photography, not fabricated landmark/property photographs. Preserve Wikimedia credits for existing stop photos. Reserve image dimensions, lazy-load below-the-fold photography, and provide error fallbacks without a city mismatch. Avoid new request-time image API dependencies.

## Parallel implementation architecture

Use the existing Go/Fiber templates, HTMX, and lightweight CSS/JavaScript. Split component stylesheets to avoid parallel edit conflicts. Preserve existing planner algorithms, providers, cache format, shared trip URLs, exports, and booking semantics.

The page shell composes shared body partials on initial load and search. The planner response updates the form plus the itinerary and stays through explicit out-of-band HTMX replacements, keeping the map and hero in place. Both successful and error responses follow that contract; stale results must not survive a failed generation.

The [task index](../tasks/redesign/README.md) defines ownership, scheduling, and dependencies. The [shared contracts](../tasks/redesign/contracts.md) freeze template inputs, section IDs, view fields, asset conventions, and update behavior.

Implementation sequence:

1. Run independent component and data tasks in parallel against the contracts. Prepare a deterministic preview using real templates at the same time.
2. Integrate after component owners hand off. Compare desktop and mobile screenshots with the corresponding regions of the reference and adjust proportions/typography.
3. Run final behavior and visual verification, fix identified issues, and produce screenshots and a result matrix. No deployment is included.

## Acceptance criteria

- The central desktop and phone layouts visibly match the reference's hierarchy, cream/terracotta palette, serif headings, compact cards, photography emphasis, and density.
- Desktop hero/photo proportions and equal overview columns are close to the target; mobile shows form before map and compact image-left content rows.
- At 375×812, 768×1024, and 1440×1000, the document has no horizontal overflow or clipped controls; the fixed bar does not cover content. Also compare around 1100px wide to the reference desktop panel.
- Readable text and usable controls take precedence over copying tiny poster-scale text. Body contrast is at least 4.5:1; focus is visible; main touch targets aim for at least 44px height; content works at 200% zoom.
- All four section shortcuts work before and after search/plan swaps, with unique IDs and visible destination headings.
- First day opens initially; multiple days can open independently by keyboard; every stop and accommodation remains accessible.
- Current weather and daily forecast are clearly distinct; missing imagery, unknown stars, uncertain forecasts, no videos, and failed stay service have deliberate states.
- Search, regenerate, change-city, shared-link, and back/forward flows show the correct data. Error responses retain useful form values and do not leave a previous itinerary presented as current.
- Calendar/KML downloads, day directions, and dated booking links still work.
- Photo credits remain available, layout is stable while images load, and video players load only on activation.
- Relevant rendering/route checks and the existing Go suite pass. Real-browser checks cover actual HTMX replacement, focus, navigation, and accordion behavior.
- Final screenshots and a discrepancy report distinguish verified behavior from unavailable external embeds and data-driven deviations from the reference.

## Scope boundary

This is a presentation and navigation redesign with the view metadata required to support it. New trip editing, accounts, saved trips, guide pages, live hotel/photo providers, visit-duration scheduling, expanded daily forecasts, and custom route-map infrastructure are separate feature work. The design should still resemble the reference closely with its documented data substitutions.
