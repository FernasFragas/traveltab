# Shared implementation contracts

Read this before starting a task. These contracts allow independent implementation; they are proposed implementation interfaces, not descriptions of existing code.

## File and style boundaries

Task 01 owns `public/styles.css` for reset, tokens, shared buttons/fields, page composition, header/footer/navigation. Component files are loaded after it in this order: `/redesign/destination.css`, `/redesign/planner.css`, `/redesign/discovery.css`. Task 01 includes deferred scripts `/redesign/navigation.js`, `/redesign/planner.js`, `/redesign/discovery.js`; their owners are 01, 04, and 05 respectively. Use Go templates and existing HTMX; no framework migration or mandatory frontend build pipeline.

Component selectors must be scoped: `.destination-*`, `.weather-*`, `.map-*` (03); `.trip-*`, `.itinerary-*` (04); `.stay-*`, `.video-*` (05). Shared `.tt-*` classes belong to 01. Do not add unscoped heading, button, input, iframe, or section rules to component stylesheets. Remove obsolete inline styles only in files you own.

Tokens supplied by 01: `--tt-bg: #F7F3EB`, `--tt-surface: #FFFDFA`, `--tt-ink: #24221F`, `--tt-muted: #666158`, `--tt-border: #DED8CC`, `--tt-accent: #A64027`, `--tt-accent-hover: #87321F`, `--tt-olive: #62674A`, `--tt-radius: 8px`, `--tt-font-heading: Georgia, 'Times New Roman', serif`, `--tt-font-body: Inter, system-ui, sans-serif`. Add shared spacing variables for 4, 8, 12, 16, 24, and 32px. Values may be refined together after screenshot comparison.

Use Bootstrap Icons as the single UI icon family. Task 01 removes duplicate icon/font libraries only after owned markup is migrated; integration verifies no remaining Font Awesome dependency. Preserve the recognizable TravelTab wordmark; a compact decorative travel mark may use an existing icon rather than require a new generated logo.

## Page composition and IDs

`index` owns the document, header/search, `#content-area`, shared footer, and persistent mobile bottom navigation. It calls `content_fragment` inside `#content-area` on full loads. Searches continue replacing `#content-area` with `content_fragment`.

`content_fragment` renders, in order:

1. `weather_display` with the full page data: destination photo and weather banner.
2. Section shortcuts linking to `#overview`, `#itinerary`, `#stays`, `#videos`.
3. `section#overview`, heading/subtitle, and `.tt-overview-grid` containing `map_card` with full page data and `trip_card` with `.Trip`. Desktop: map left/form right. Mobile: form first/map second. If `.Trip` is nil, show a planner-unavailable explanation instead of calling a form template with nil.
4. `section#itinerary` containing `itinerary_body` with `.Trip` (possibly nil).
5. `section#stays` containing `stay_card` with `.Trip.Plan` when available, otherwise nil.
6. `section#videos` containing `video` with full page data.

Task 01 owns these initial section wrappers. Body partials do not repeat the section ID. Each section body supplies its heading. All four anchor targets exist before a plan is generated. Footer links are defined once, outside the swapped region. Top editorial links are real section links; omit unimplemented account/guide destinations.

## Planner responses

Keep `#trip-card` as the form fragment root and `hx-target="#trip-card" hx-swap="outerHTML"`. Task 04 creates `plan_response.go.tpl` with three siblings:

```html
{{ template "trip_card" . }}
<section id="itinerary" hx-swap-oob="outerHTML">
  {{ template "itinerary_body" . }}
</section>
<section id="stays" hx-swap-oob="outerHTML">
  {{ template "stay_card" .Plan }}
</section>
```

Task 02 changes all `/plan` render branches to `plan_response`, including 400/500 responses. Full page rendering calls body partials directly and never includes OOB attributes. Existing HTMX error-swap handling must continue; after a failed regeneration, obsolete results are cleared and explanatory empty states replace them. IDs must remain unique.

Task 04 owns delegated planner event handlers, installs them once, preserves/reinstates submitted form values on validation failures, and handles post-swap focus/announcements. Task 01 owns anchor active state and mobile menu behavior. Task 05 owns video activation. All handlers work after search and OOB swaps without repeated listener installation. Browser-check the real HTMX version, not only the HTML strings.

## Presentation data supplied by 02

Do not change planner algorithms or provider schemas to decorate the UI. Add HTTP presentation helpers and optional local image metadata. Existing fields remain available.

Full page and search fragment render maps gain `.Presentation` with:

- `.Hero`: optional `*imageView`, nil for an unknown/unconfigured destination.
- `.Tagline`, `.PhotoCaption`, `.CountryLabel`: optional strings, correctly keyed by city and country. Never reuse Lisbon copy for another city.
- `.MapURL`: coordinate-based external map link; the existing `.GeneralInfo.EmbedURL` remains the iframe source.
- `.Videos`: list of `{ Title, VideoID, ThumbnailURL, WatchURL }`, derived from the existing videos. No invented duration.

`imageView` exposes `URL`, `Alt`, `Credit`, `CreditURL`, `Width`, `Height`. Curated manifest entries additionally record the source and license. A curated lookup happens locally; no new image API call on the request path. Image fields remain absent when there is no verified match.

`tripCard` gains `.EndLabel`, a human-readable inclusive final itinerary date (`start + days - 1`). Submit only existing `start` and `days` parameters. Display the end as a read-only part of the date group and update it when controls change. Do not confuse the last itinerary day with accommodation checkout.

`tripPlanView` gains `.StayCards`, a list of `{ Name, Kind, Stars, Website, Image }`, with `Image` optional `*imageView`. Keep `.Stays` and other existing fields for compatibility. Match any curated accommodation image by destination plus unambiguous accommodation identity (coordinates/name), never just the display name. `stay_card` takes `*tripPlanView` or nil and uses `.StayCards`.

Existing `tripDayView` fields remain the itinerary source: `Number`, `Date`, `Rainy`, `RainMM`, `Certain`, `HasForecast`, `WalkKM`, `Stops`, `MapsURL`. Stop numbering can use an ordered list/CSS counters without adding template arithmetic. There are no forecast temperatures or actual visit times in this contract.

## Interaction and asset constraints

- One scrolling page; navigation never hides other sections as tabs would.
- First itinerary day open initially; independent native `details` elements allow multiple days open.
- Only current weather uses the weather provider's temperature. Day rows show rainfall/status from the planner; a dry forecast is not necessarily sunny.
- Unknown stay stars, missing photos, absent durations, and absent visit times are omitted or use designed fallbacks, never fabricated.
- Reference account/profile, saved-trip controls, guide pages, hotel photos, and exact map styling are not automatically available features. Use functional section destinations for navigation and document visual substitutions.
- Keep real external map attribution. Do not use a static mock map or rasterized reference screenshot as the production interface.
- The full reference is a design artifact, not a source of cropped hotel/landmark assets. Obtain appropriate real photos and record credits; do not generate fictional landmark photography to impersonate the destination.
