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
4. Overview: map on the left and the "Plan your trip" form on the right, in equal columns, with the city guide card full width below them.
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
`/redesign/destination.css`, `/redesign/planner.css`, `/redesign/discovery.css`, `/redesign/guide.css`.

| Token | Value |
| --- | --- |
| `--tt-bg` / `--tt-surface` | `#F7F3EB` cream / `#FFFDFA` warm white |
| `--tt-ink` / `--tt-muted` | `#24221F` / `#666158` |
| `--tt-border` | `#DED8CC` decorative dividers and card edges |
| `--tt-border-control` | `#8F887B` form-field boundaries (date, select, search pill): 3.46:1 on `--tt-surface`, 3.18:1 on `--tt-bg` |
| `--tt-accent` / `--tt-accent-hover` | `#A64027` terracotta / `#87321F` |
| `--tt-olive` | `#62674A` |
| `--tt-radius` | `8px` |
| Fonts | Georgia for headings, Inter for body text |

- Accessibility rules the audit enforces ([evidence](../tasks/artifacts/redesign/final-verification-2/README.md)): 4.5:1 text and 3:1 icons and field boundaries, focus rings of at least 3:1 that are not clipped by `overflow: hidden` (use an inset ring there, light on dark photo overlays), controls at least 44px, and the current section never shown by colour alone.
- Spacing scale: 4/8/12/16/24/32px.
- Icons: Bootstrap Icons only.
- Selectors are scoped per component: `.destination-*`, `.weather-*` and `.map-*`; `.trip-*` and
  `.itinerary-*`; `.stay-*` and `.video-*`; `.guide-*` for the city guide. Shared classes use `.tt-*`. Don't add unscoped
  element rules to component stylesheets.
- No frontend framework or build step. Pages use Go templates, HTMX 1.9.11 and the plain scripts
  `navigation.js`, `planner.js` and `discovery.js`.

## Template contracts

**Page composition.** `index` owns the document, header, `#content-area`, footer and bottom
navigation. A search replaces `#content-area` with `content_fragment`, which renders:
`weather_display` → shortcuts → `section#overview` (`map_card` + `trip_card`, then `guide_card`) →
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
  It is nil when no photograph could be shown. `.Presentation.PhotoUnavailable` is then true and
  the template renders the explicit "Photo unavailable" state.
- `.Tagline`, `.PhotoCaption`, `.CountryLabel`: optional, keyed by city and country.
- `.ConditionIcon`: a Bootstrap Icons class, with a thermometer fallback.
- `.MapURL`: an external map link. The iframe keeps `.GeneralInfo.EmbedURL`.
- `.Videos`: `{Title, VideoID, ThumbnailURL, WatchURL}`, derived from the provider data.
- `.Guide`: a `guideView` with two states, never mixed. `Available` has the reviewed `Paragraphs`, the
  Wikivoyage `SourceTitle` with `SourcePageURL`, `SourceRevisionURL`/`SourceRevision` and `SourceHistoryURL`,
  `ReviewedOn` and the CC BY-SA 4.0 `LicenseName`/`LicenseURL`. Otherwise `City` and a `SearchURL` (a Wikivoyage
  search for the city) drive the "no reviewed guide yet" state. All text is plain and escaped.
- `tripCard.EndLabel`: the last itinerary day (`start + days - 1`). This is not the stay checkout
  date.
- `tripPlanView.StayCards`: `{Name, Kind, Stars, Website, Image}`. A stay image matches on
  destination and identity, never on display name alone.

Hero images come from two places. A verified local manifest entry (`presentation_assets/manifest.json`,
currently Lisbon only) is an optional override and needs no lookup. Every other destination gets a
live Wikimedia photo, resolved for the searched destination's identity (name, country,
coordinates) by one server-side builder used by the initial page, the HTMX search fragment and
shared trip pages. Photo metadata is cached separately from the weather/video JSON, and the
lookup result is merged into the view, never into cached provider data. `imageView` stays HTTP-only.

The hero figure carries `data-photo-state`: `ready` (a photograph with its credit and license),
`unavailable` (the server found none, or the lookup failed) or `failed` (set by the image's own
`error` event). Both `unavailable` and `failed` show `.destination-photo-fallback` ("Photo
unavailable") in the same frame, and `failed` also hides the image, credit and caption so nothing
is left orphaned. The state is an attribute, so it survives HTMX history snapshots. The handlers
are inline `onload`/`onerror` attributes on the image, so a swapped-in image brings its own and
nothing is registered twice. The hero frame keeps its size in every state.

**The city guide card** (`views/guide_card.go.tpl`, `public/redesign/guide.css`) sits in `section#overview`, after
the map/planner grid, and is rendered by `content_fragment` only. It is not part of the planner response, so the
`/plan` swap contract above is unchanged, and the section IDs are too. `data-guide-state` is `reviewed` or
`missing`. The `reviewed` state shows the intro, a "Reviewed" badge and a source line: "Adapted from Wikivoyage:
<article> (revision N), by Wikivoyage contributors, licensed under CC BY-SA 4.0", then "Summarised and reworded.
Checked against that revision on <date>." Each of the four links opens in a new tab with `noopener noreferrer`. On
screens 900px and wider the source line sits beside the text. The `missing` state says "We don't have a reviewed
guide for <city> yet" and offers a Wikivoyage search link. The guide is chosen from the *resolved* city and country
by the same builder as the photo, and only entries a person has reviewed are ever shown (see
[architecture](architecture.md#presentation-and-assets)).

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
  pages. The guide card's links go to Wikivoyage, its source; there is no guide page of our own.
- Never show generated or unreviewed guide text, and never show a guide without its Wikivoyage article,
  revision and license. A city with no reviewed guide says so.

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
