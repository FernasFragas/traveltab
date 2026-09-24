# Maintenance notes

Setup, routes, providers, and generator commands are in the [README](../README.md).
Package boundaries are in [architecture.md](architecture.md) and the UI contracts are in
[design.md](design.md). The redesign is implemented, and the
[September 2026 results](../tasks/project-fix-results.md) record its verification and limits.
Open work is listed under [Next up](#next-up).

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

## Next up

In priority order. Updated 2026-09-24.

1. **Ship the redesign.** Commit the verification work (the preview checks, evidence and results)
   and merge PR #7.
2. **Destination photos for every search.** Today only Lisbon has a hero photo. The
   [plan](../tasks/destination-photos-plan.md) is ready: a keyless Wikimedia lookup, an
   identity-safe cache, and a fix for namesake cities (for example Paris, France and Paris, Texas)
   sharing the city cache.
3. **Close the verification gaps.** Still open from the
   [final browser evidence](../tasks/artifacts/redesign/final-verification/README.md):
   - a full contrast and accessibility audit;
   - browser screenshots for every named preview scenario;
   - live checks of the map embed, Wikimedia photos, YouTube playback and the Booking landing
     page in a normal browser (see the [external checks](../tasks/artifacts/redesign/final-verification/external-results.md)).
4. **Sitemap storage.** Replace the `__sitemap_index__` cache entry with a typed method for
   listing cities, and keep existing URLs.
5. **City guides.** Review `guides/guides.json`, then show reviewed intros on the page with
   Wikivoyage source, revision and attribution. The proper-name validator is a heuristic, not a
   factual review. Only Lisbon, London, Seville and Vienna had a manual review. Serving a page must
   never call Ollama.
6. **Slug helpers.** The guide and HTTP slug helpers implement the same format independently. Merge
   them into a small shared package, keeping the generator independent of the HTTP adapter.
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
