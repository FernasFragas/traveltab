# Maintenance notes

Setup, routes, providers, and generator commands are in the [README](../README.md).
Package boundaries are in [architecture.md](architecture.md). The active work is the
[redesign](../tasks/redesign/README.md): tasks 01–03 are implemented; 04–07 remain pending.

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

## Follow-ups

- Complete redesign tasks 04–07 before treating the reference design as finished. The current
  [browser evidence](../tasks/artifacts/redesign/milestone-01-03/README.md) identifies the verified
  flows and the remaining external-provider and visual checks.
- Replace the sitemap's `__sitemap_index__` cache entry with a storage method for listing cities.
- Guide generation remains separate from serving the site. Review `guides/guides.json` before
  integration, and include the Wikivoyage source/revision and attribution required for publication.
  The proper-name validator is a heuristic, not a factual review. The recorded manual review
  covered Lisbon, London, Seville, and Vienna; the other starter cities had automated validation.
- Guide and HTTP slug helpers implement the same format independently. A shared package could
  remove duplication without making the generator depend on the HTTP adapter.
- Import calendar/KML exports into actual calendar and map applications. Structural parsing
  checks do not establish compatibility with every application's importer. Calendar event times
  are assigned defaults, not real visit schedules.
- Verify search-engine indexing after deployment; local route and sitemap tests cannot establish it.

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
