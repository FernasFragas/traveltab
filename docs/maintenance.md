# Maintenance notes

Setup, routes, providers, and generator commands are in the [README](../README.md).
Package boundaries are in [architecture.md](architecture.md) and the UI contracts are in
[design.md](design.md). The redesign is implemented, and the
[September 2026 results](../tasks/project-fix-results.md) record its verification and limits.
Open work is listed under [Next up](#next-up).

## Accessibility limits

- **Focus order below 900px.** The planner is shown before the map through CSS `order`, but the
  DOM keeps the map first for the desktop reading order, so keyboard focus visits the map card
  (three stops) before the form. Reordering the DOM would move the mismatch to desktop.
- **The map frame is a third-party document.** Its own focus indication and keyboard behaviour
  are unverified until the live check; the page scrolls it into view and rings the card.
- **Not announced:** the start of a destination search; the hidden spinner text stays in the
  reading order. Below the 320px WCAG reflow width (375px at 200% is 188px) the page scrolls
  horizontally.
- Re-run `tasks/redesign/checks/a11y-audit.mjs` after any colour or layout change; it fails on
  regressions (see the [reproduce steps](../tasks/artifacts/redesign/final-verification-2/README.md#reproduce)).

## Planner decisions and limits

- Wikipedia coverage alone ranks administrative areas and other non-attractions highly.
  The generated Wikidata type filter is therefore required; unknown types are excluded.
  Classification uses exact deny types because broad ancestor exclusions can remove real
  landmarks. Review changes to `internal/placetypes/rules.go` and the generated map together.
- Indoor/outdoor labels are approximations. Station buildings can classify as outdoor,
  and market squares can classify as mixed. Mixed counts as sheltered during rain scheduling.
- Wikipedia geosearch splits a capped 10 km search into seven smaller searches. A smaller
  search can still hit the 500-result cap; there is no recursive splitting.
- Walking distances are straight-line estimates, not street routing. Compact grouping targets
  three or four stops, but isolated places can remain alone and sparse cities return fewer days.
- Requested dates represent local calendar days. Forecast certainty describes the date horizon;
  it does not prove that forecast data exists. Check rain-data availability separately.
- Shared plans are rebuilt from current data. With the same place data, grouping is deterministic;
  refreshed forecasts can change day order. Links do not preserve a historical snapshot.
- Source cache refresh failures serve stale entries when available. Cold requests still depend on
  public API latency, particularly Overpass retries. Parallel Wikimedia lookups can also time out.

## Destination photo limits

- A lookup needs a Wikidata entity that matches the resolved name, country and coordinates.
  Places the search ranks below its top five names are found through a nearby-article fallback;
  a country or territory whose ISO code differs from its Wikidata country (for example Puerto Rico
  against the United States) is refused, and so is a genuinely ambiguous match. Each refusal shows
  "Photo unavailable" rather than a guess.
- During a Wikimedia outage every uncached search waits up to the four-second lookup deadline
  before showing "Photo unavailable" (in parallel with the video request); errors are not cached,
  and there is no circuit breaker yet.
- Only metadata is cached. Images load from Commons in the browser, and some are large (a
  1.7 MB PNG was selected for "Lisboa"). Caching image bytes is a separate enhancement if that
  matters.
- The photo is the entity's own Wikidata image, which is sometimes a landmark rather than a
  skyline, and the caption is generic ("View of <city>").

## City guide limits

- **Only the 20 starter cities have a guide.** Every other search gets the "no reviewed guide yet"
  card with a Wikivoyage search link. Adding a city means adding it to `internal/guides/city.go`,
  generating it (`go run ./cmd/guides -only=...`) and reviewing it by hand; see
  [the review notes](../tasks/guides-review.md#re-reviewing).
- **A guide is matched by the resolved name and country.** OpenWeather decides the name, so a
  search that resolves to a local spelling ("Lisboa", "Wien") shows the missing state, not a guess.
- **The review was done by an AI from free sources.** A person should skim the intros before a
  wider launch. Wikivoyage pages move on: five had newer revisions by the review date, and the
  intros are pinned to the reviewed revision (linked as a permalink), not to today's page.
- **Regeneration hides a guide.** A city whose Wikivoyage revision changed is regenerated as an
  unreviewed entry, replacing the reviewed text in the file (git keeps the old one) until it is
  reviewed again.
- The site does not check the guide file for factual accuracy; the tests check its shape
  (attribution fields, review date, length and that it round-trips through the generator's writer).

## Next up

In priority order. Updated 2026-09-24.

1. **Ship the redesign.** Commit the verification work (the preview checks, evidence and results)
   and merge PR #7.
2. **Destination photos for every search.** Done. Every search resolves a Wikimedia photo live
   (keyless, identity-checked, cached 30 days; "none found" one hour), and a cached city page is
   used only when its country and coordinates match, so Paris, France and Paris, Texas no longer
   share data. Verification, live samples, latency and the [remaining limits](#destination-photo-limits)
   are in the [results](../tasks/destination-photos-results.md). Remove this item when the list is
   next reordered.
3. **Close the verification gaps.** The contrast and accessibility audit and the
   screenshots for every named preview scenario are **done** (2026-09-25, offline, with
   [assertions and a requirement matrix](../tasks/artifacts/redesign/final-verification-2/README.md);
   11 defects fixed, one focus-order limitation below 900px documented). Still open from the
   [final browser evidence](../tasks/artifacts/redesign/final-verification/README.md):
   - live checks of the map embed, Wikimedia photos, YouTube playback and the Booking landing
     page in a normal browser (see the [external checks](../tasks/artifacts/redesign/final-verification/external-results.md)).
4. **Sitemap storage.** Done. `application.Storage` has `RecordSitemapSlug` and
   `ListSitemapSlugs`, backed by a `sitemap_slugs` table with one row per slug. On open, the
   SQLite store copies slugs from a legacy `__sitemap_index__` cache row into that table and
   deletes the row, so existing URLs are kept. Rolling back to an older build loses that index
   until cities are visited again. Remove this item when the list is next reordered.
5. **City guides.** Done. `guides/guides.json` was reviewed against Wikivoyage on 2026-09-24: all 20
   starter cities were rewritten to what the cited revision supports, 26 wrong or misleading draft
   statements were corrected and 56 unsupported or dated ones removed, and each entry now records
   its article title, review date and note. The destination page shows a reviewed intro with its
   attribution, or an explicit "no reviewed guide yet" card, and serving a request never calls
   Ollama. The [review](../tasks/guides-review.md), the [results](../tasks/guides-results.md) and
   the [remaining limits](#city-guide-limits) are recorded. Remove this item when the list is next
   reordered.
6. **Slug helpers.** Done. `internal/slug` now holds the `city-country` format; `guides.Slug` and
   `httpserver.Slug`/`ParseSlug` are thin wrappers. Both old helpers behaved the same for real
   inputs (two-letter codes); the only difference was whitespace inside the country, now
   hyphenated in both. Diacritics are kept, not stripped, and a test pins every
   `guides/guides.json` key. Remove this item when the list is next reordered.
7. **Export interoperability.** Import the ICS and KML exports into real calendar and map apps.
   Parser checks don't prove importer compatibility. Event times are defaults, not real visit
   schedules.
8. **Indexing after deployment.** Check the public trip URLs, canonical tags and sitemap in
   search-engine diagnostics. Local tests can't show this.

## Historical verification boundaries

The September 19–20, 2026 planner work used recorded Lisbon, Tavira, and Kyoto fixtures for
repeatable offline acceptance checks. The Kyoto stay fixture uses an older center coordinate,
so it does not establish hotel availability around every computed itinerary center.

That work also checked real generated ICS/KML files with independent parsers, but did not import
them into Google Calendar, Apple Calendar, Outlook, or Google My Maps. Historical browser checks
used fixture providers and did not verify the production map or booking landing page.

Historical Lisbon Wikimedia measurements improved from 18.4–18.6 seconds with one worker to
6.3–7.8 seconds with four; one parallel attempt timed out. These are observations from that run,
not performance guarantees. For repeatable load testing and measured scope, use the
[load-test guide](../loadtests/README.md) and its [recorded results](../loadtests/validation.md).
