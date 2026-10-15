# Design reference

The destination page follows the [desktop look-v3 reference](../desktop.png) and
[phone reference](../phone.png), using the data already available to the app.
The earlier [reference image](../goalimg.png) still informs the light TravelTab controls and
content cards. This file covers the layout, template contracts and data rules. Map-first
verification is in [the TDD evidence](tdd/map-first.tdd.md); the updated appearance and journeys
are recorded in [map-look-v2 evidence](tdd/map-look-v2.tdd.md).

## Layout

**Brand:** solid terracotta peaks sit beside a two-line wordmark: bold serif “TravelTab” above
smaller terracotta “ByPatronFragas”. The accessible name, document and trip titles, and calendar
text remain `TravelTabByPatronFragas`. Search submits with Enter or a suggestion; the rounded
field has no clear or arrow button. A bordered bookmark button opens saved places.

**Desktop:** the header orders brand, centered search, section tabs and bookmark. A 157 px
photo banner shows the city and a horizontal weather strip with sunset, forecast popover and
planning link. The left column holds full-label layer pills (all toggle; Notable places is filled terracotta while on,
overlays that are on are white with filled coloured icons, off pills muted with outline icons), a `min(505px, 60vh)` map and
three teaser cards; the roughly 510 px right column holds a place or city card and scrolls
independently. The footer is hidden on the map page. Section tabs preserve the mounted map.

**Mobile:** search has its own row, followed by a 100 px photo hero with weather chip,
scrollable full-label pills, a 205 px map, an inline place card and the day, stay and video
teasers. Selecting Details scrolls to the card and focuses its title; initial auto-selection
never steals focus. Escape and the map back button return to the map. Bottom navigation
shows icons above labels, a bed SVG for Stays and an underline below the active section.

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
| `--tt-accent-tint` | `#F8E6E0`; terracotta text/icons have 5.14:1 contrast |
| `--tt-writer` / `--tt-writer-tint` | `#6B4FA8` / `#F1EEF8`; 5.54:1 contrast |
| `--tt-serif` | Georgia, Times New Roman, serif; descriptions, metadata and teaser titles |
| `--tt-olive` | `#62674A` |
| `--tt-radius` | `8px` |
| Fonts | Georgia for headings, Inter for body text |

### Map palette and controls

The destination map uses a separate dark vector palette while the page controls and place panel
keep the light TravelTab tokens above. The default OpenFreeMap style is tuned in
`public/redesign/writers-map.js`: navy `#202A32` background, muted olive `#29382F` parks, deep
blue `#000E29` water, and subdued purple-grey buildings. Major roads, streets, district and
water labels remain legible; generic shop, parking and amenity POIs are hidden. Untagged
buildings have an 8 m extrusion fallback, and Mapterhorn terrain remains available in 3D.
Notable places use terracotta (`#cf795b`) teardrops with white category icons; nature places use
green (`#459b6b`). Writer places use purple (`#a582d4`) teardrops with a quill, and clusters use
grey circles with dark counts. Day-plan stops use orange (`#a56824`) numbered teardrops and an orange dotted route; writers' itineraries
use purple dashed routes with small stop dots. Every pin colour has at least 3:1 contrast on
the dark map. District labels use plain uppercase white text with a dark halo; water labels are italic blue.
The selected place uses a terracotta pin with a ring (its name label hidden) and a white callout with its serif name, readable kind,
photo and a keyboard-accessible action to open the panel.

Dark translucent map controls sit in a column at the top right: compass with red needle, locate, 3D and a combined zoom pill. A back button closes the selection and restores the city camera. The compass resets bearing and, like 3D, is hidden for Leaflet. Targets are at
least 44 px and the scale sits at the bottom right. MapLibre's compact attribution starts collapsed. The “View full map” card uses the existing
external map URL, with the city photo or a map icon. The Leaflet fallback uses EOX imagery
and keeps its attribution.

### Place panel and saving

The place card shows a photo carousel only when multiple photos exist, a serif title and
bookmark, icon metadata with a readable kind (`kind_label`, e.g. tower → Historic Landmark), a serif description and Add to your plan, Save and Share actions.
Wikipedia and Google Maps direction cards follow. Facts show today's hours for simple OSM
rules or the raw rule for complex syntax, plus a kind-based best-time suggestion explicitly
labelled “Tip”. Missing columns are hidden; price is deferred. The lavender From our writers
card quotes the first existing post title as plain italic text; the blog credit links to the post. See all expands the post list.
Without mentions it says “No writer mentions yet”. The city card uses the same appearance,
with Wikivoyage text, credit, link card and planning action. Weather stays in the banner.

Saving shows “Added to your saved places · Undo” in a polite live region on the map. Undo
removes that place. The header bookmark opens the saved list; choosing an item opens its place
on the map, and Escape closes the list and restores focus. The saved count and Clear action
live inside the Itinerary planner.

- Accessibility rules the audit enforces ([evidence](../tasks/artifacts/redesign/final-verification-2/README.md)): 4.5:1 text and 3:1 icons and field boundaries, focus rings of at least 3:1 that are not clipped by `overflow: hidden` (use an inset ring there, light on dark photo overlays), controls at least 44px, and the current section never shown by colour alone.
- Spacing scale: 4/8/12/16/24/32px.
- Icons: Bootstrap Icons, plus one inline bed SVG.
- Selectors are scoped per component: `.destination-*`, `.weather-*` and `.map-*`; `.trip-*` and
  `.itinerary-*`; `.stay-*` and `.video-*`; `.guide-*` for the city guide. Shared classes use `.tt-*`. Don't add unscoped
  element rules to component stylesheets.
- No frontend framework or build step. Pages use Go templates, HTMX 1.9.11 and the plain scripts
  `navigation.js`, `planner.js` and `discovery.js`.

## Template contracts

**Page composition.** `index` owns the document, header, `#content-area`, footer and bottom
navigation. A search replaces `#content-area` with `content_fragment`, which renders:
`section#overview` (`map_card` with the map and place panel) → `section#itinerary` (`trip_card`
and `itinerary_body`) → `section#stays` (`stay_card`) → `section#videos` (`video`). The guide renders in the default panel when a writer map is available, and `weather_display`
renders inside `section#overview` before the map with an `h2` title (the page keeps one `h1`).
Without a writer map, they retain their page fallback. All four anchor targets exist before a plan is generated, and IDs stay unique.

**Planner response.** `/plan` targets `#trip-card` with `outerHTML`. Every response, including
400 and 500 errors, renders `plan_response`, which has three sibling roots:

```html
{{ template "trip_card" . }}
<div id="itinerary-results" hx-swap-oob="outerHTML">{{ template "itinerary_body" . }}</div>
<section id="stays" data-tt-view hidden hx-swap-oob="outerHTML">{{ template "stay_card" .Plan }}</section>
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
- Keep live map sources and provider attribution. The Waze iframe remains only as the fallback
  when the writer map is unavailable. Never ship a static mock map or crops of the reference image.
- Don't add links to features that don't exist, such as accounts, saved trips, guide pages or legal
  pages. The guide card's links go to Wikivoyage, its source; there is no guide page of our own.
- Never show generated or unreviewed guide text, and never show a guide without its Wikivoyage article,
  revision and license. A city with no reviewed guide says so.

## Known differences from the reference

| Reference | Implementation |
| --- | --- |
| Photoreal map with dense pins | Dark vector writer map; the fallback still uses the Waze embed |
| Daily temperature and sun icon | Rainfall and certainty; the planner has no daily temperatures |
| Visit time ranges, video runtimes | Omitted; the data doesn't include them |
| Hotel photos | Placeholder unless a curated, identity-matched image exists |
| Date-range control | Start date and a day count, with the end date shown read-only |
| Profile and broader site navigation | Section tabs and a saved-places bookmark |

The [final browser evidence](../tasks/artifacts/redesign/final-verification/README.md#reference-comparison)
has a section-by-section comparison.
