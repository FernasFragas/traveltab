# Design reference

The implemented interface follows the central website and phone screen in the
[reference image](../goalimg.png). The poster around them (slogans, scenery, phone hardware) is not
site content. This file covers the visual target, the template contracts, and the data rules that
changes must keep. Verification evidence is in the [results](../tasks/project-fix-results.md).

## Layout

**Desktop:** a centered cream shell, about 1120–1200px wide, with these sections in order:

1. Header: compact wordmark, section links and a pill-shaped search field.
2. Destination hero: photo on the left (about 58%) and city, tagline and three current weather
   metrics on the right. About 220–260px tall.
3. Section shortcuts: Overview, Itinerary, Stays, Videos.
4. Overview: map on the left and the "Plan your trip" form on the right, in equal columns.
5. Itinerary: a day accordion with calendar/map export actions beside the heading. Up to four
   stop photos per row.
6. Where to stay: image-left cards, about three per row, with a "Check prices" link.
7. Videos: thumbnail cards, four shown at first, with a "View all" expansion.
8. Footer, defined once outside the swapped region.

**Mobile:** the search field moves to its own row. The city name sits over the photo and the
weather metrics below it. The form comes **before** the map. Stops, stays and videos become
image-left rows. A fixed bottom navigation bar links to the four sections. Below about 900px the
overview has one column; below about 640px the page uses the mobile header and bottom bar.

## Visual tokens

Defined in `public/styles.css`. Component stylesheets load after it, in this order:
`/redesign/destination.css`, `/redesign/planner.css`, `/redesign/discovery.css`.

| Token | Value |
| --- | --- |
| `--tt-bg` / `--tt-surface` | `#F7F3EB` cream / `#FFFDFA` warm white |
| `--tt-ink` / `--tt-muted` | `#24221F` / `#666158` |
| `--tt-border` | `#DED8CC` |
| `--tt-accent` / `--tt-accent-hover` | `#A64027` terracotta / `#87321F` |
| `--tt-olive` | `#62674A` |
| `--tt-radius` | `8px` |
| Fonts | Georgia for headings, Inter for body text |

- Spacing scale: 4/8/12/16/24/32px.
- Icons: Bootstrap Icons only.
- Selectors are scoped per component: `.destination-*`, `.weather-*` and `.map-*`; `.trip-*` and
  `.itinerary-*`; `.stay-*` and `.video-*`. Shared classes use `.tt-*`. Don't add unscoped
  element rules to component stylesheets.
- No frontend framework or build step. Pages use Go templates, HTMX 1.9.11 and the plain scripts
  `navigation.js`, `planner.js` and `discovery.js`.

## Template contracts

**Page composition.** `index` owns the document, header, `#content-area`, footer and bottom
navigation. A search replaces `#content-area` with `content_fragment`, which renders:
`weather_display` → shortcuts → `section#overview` (`map_card` + `trip_card`) →
`section#itinerary` (`itinerary_body`) → `section#stays` (`stay_card`) → `section#videos`
(`video`). All four anchor targets exist before a plan is generated, and IDs stay unique.

**Planner response.** `/plan` targets `#trip-card` with `outerHTML`. Every response, including
400 and 500 errors, renders `plan_response`, which has three sibling roots:

```html
{{ template "trip_card" . }}
<section id="itinerary" hx-swap-oob="outerHTML">{{ template "itinerary_body" . }}</section>
<section id="stays" hx-swap-oob="outerHTML">{{ template "stay_card" .Plan }}</section>
```

Full-page renders never include OOB attributes. A failed regeneration clears the old itinerary and
stays and keeps the submitted form values. Event handlers are delegated and installed once, so
they keep working after swaps and history restoration.

**Presentation data** (`internal/adapters/httpserver/presentation.go`):

- `.Presentation.Hero`: optional `*imageView` with URL, alt text, credit, license and dimensions.
  It is nil when no verified image matches.
- `.Tagline`, `.PhotoCaption`, `.CountryLabel`: optional, keyed by city and country.
- `.ConditionIcon`: a Bootstrap Icons class, with a thermometer fallback.
- `.MapURL`: an external map link. The iframe keeps `.GeneralInfo.EmbedURL`.
- `.Videos`: `{Title, VideoID, ThumbnailURL, WatchURL}`, derived from the provider data.
- `tripCard.EndLabel`: the last itinerary day (`start + days - 1`). This is not the stay checkout
  date.
- `tripPlanView.StayCards`: `{Name, Kind, Stars, Website, Image}`. A stay image matches on
  destination and identity, never on display name alone.

Hero images currently come only from the local manifest
(`presentation_assets/manifest.json`, which lists Lisbon only). The
[destination photo plan](../tasks/destination-photos-plan.md) replaces this with a live lookup.

## Data honesty rules

- Only the current weather shows temperature. Day rows show rainfall and forecast certainty, and a
  dry day is not labelled sunny.
- No invented visit times, video durations, stay prices or star ratings. Missing photos, stars and
  forecasts get explicit fallback states.
- The first itinerary day is open initially and after each new plan. Days use independent native
  `details` elements.
- Keep the real map embed and provider attribution. Never ship a static mock map or crops of the
  reference image.
- Don't add links to features that don't exist, such as accounts, saved trips, guide pages or legal
  pages.

## Known differences from the reference

| Reference | Implementation |
| --- | --- |
| Styled street map with pins | Waze embed; its tile style is controlled by the provider |
| Daily temperature and sun icon | Rainfall and certainty; the planner has no daily temperatures |
| Visit time ranges, video runtimes | Omitted; the data doesn't include them |
| Hotel photos | Placeholder unless a curated, identity-matched image exists |
| Date-range control | Start date and a day count, with the end date shown read-only |
| Profile, Saved and broader site navigation | Section links only |

The [final browser evidence](../tasks/artifacts/redesign/final-verification/README.md#reference-comparison)
has a section-by-section comparison.
