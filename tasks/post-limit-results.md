# L1 results: which popularity signal?

**Key point:** no free signal picks Portugal posts by itself. Comments work on 9 of 11 blogs, but the top-commented posts are mostly **not about Portugal** (about 12 of 81 by title). **Rank only among destination-matching posts**, and **don't use an even 9-per-blog share**: give the slots to the blogs that have the most Portugal posts.

## Recommendation
- **Signal and order:** (1) restrict candidates to destination-matching posts (see below), (2) rank them by the recent-comments tally, (3) fill the rest newest first. Sticky is not worth a request: only 1 of 11 blogs uses it (Viaje Comigo, 12 posts, mostly announcements), and 1 of its 12 is about Portugal.
- **Rank only among destination-matching posts: yes.** Unfiltered, the top 9 by comments are Portugal-relevant for 8 of 81 picks (12 after fixing the title check, see Caveats). Filtering by `search=` alone is not enough either, because `search=` matches body text: 17 of 76 picks (21 adjusted). Add a **title/URL match** on the search hits, which leaves at least 9 candidates for 9 of 11 blogs (counts in the table).
- **Even share (9 each): does not fit.** International blogs have almost no Portugal posts (Nomadic Matt 6, Crowded Planet 2, Adventurous Kate 9, Planet D 28 title matches among the search hits). Keep the plan's rule that unused share passes on, but **pass it to Portugal-heavy blogs first** (Vortex, Viaje Comigo, Portugalist, Alma de Viajante, Viagens à Solta), not in allowlist order. Or cap international blogs at 3 to 5 each.
- **Comments are unreliable as the only signal:** closed on 2 blogs (404), and the newest 1,000 comments cover very different spans (see table). Tie-break by newest.

## Per blog (picks = 9 per signal)
Hits = `X-WP-Total` for Madeira / Açores(Azores) / Lisboa(Lisbon) / Porto. PT = picks whose title mentions a Portuguese destination (simple title match; see Caveats).

| Blog | Comments reachable / seen / span | Sticky | Search hits | PT picks /9: comments · newest · popular-among-search-hits | Notes |
|---|---|---|---|---|---|
| Alma de Viajante (pt) | yes / 1000 / 2022-03 to 2026-10 | 0 | 458 / 50 / 337 / 851 | 0 · 1 · 0 | Top posts: insurance, Morocco, Sardinia, Cape Verde |
| Viaje Comigo (pt) | **no (404)** | 12 (1 PT) | 613 / 157 / 736 / 1732 | n/a · 1 · n/a | Sticky are announcements and tours |
| Vortex Magazine (pt) | yes / 885 / 2015-04 to 2026-09 | 0 | 638 / 177 / 668 / 899 | 6 · 1 · 7 | Only 904 comments exist. Content is culture/trivia about Portugal, not places to visit |
| Viagens à Solta (pt) | yes / 1000 / 2018-10 to 2026-09 | 0 | 98 / 13 / 89 / 113 | 2 (6 by hand) · 1 · 2 (6 by hand) | Best fit: Madeira, Açores, Caldas da Rainha, Tomar, Sistelo among the top |
| Portugalist (en) | **no (404)** | 0 | 47 / 42 / 120 / 85 | n/a · 8 · n/a | Newest are mostly about living in Portugal (language, visas), not places |
| Nomadic Matt (en) | yes / 969 / 2020-10 to 2021-11 | 0 | 0 / 3 / 21 / 10 | 0 · 0 · 0 (4 candidates) | Comment span looks like old data; top posts are Tulum, COVID |
| The Planet D (en) | yes / 992 / 2022-02 to 2026-06 | 0 | 5 / 11 / 39 / 58 | 0 · 0 · 3 | |
| Earth Trekkers (en) | yes / 1000 / 2026-03 to 2026-10 | 0 | 74 / 41 / 363 / 184 | 0 · 0 · 1 | |
| Salt in our Hair (en) | yes / 997 / 2022-10 to 2026-10 | 0 | 29 / 13 / 27 / 52 | 0 · 0 · 3 | |
| The Crowded Planet (en) | yes / 981 / 2015-05 to 2025-04 | 0 | 1 / 1 / 3 / 21 | 0 · 0 · 1 | Almost nothing on Portugal |
| Adventurous Kate (en) | yes / 1000 / 2020-12 to 2026-10 | 0 | 1 / 2 / 34 / 44 | 0 · 0 · 0 | Top posts are birthdays and recaps |

Title-matching posts among the search hits (union of 4 terms, up to 100 per term fetched): Alma 44, Viaje Comigo 148, Vortex 224, Viagens à Solta 39, Portugalist 122, Nomadic Matt 6, Planet D 28, Earth Trekkers 49, Salt in our Hair 47, Crowded Planet 2, Adventurous Kate 9.

## Totals (Portugal-relevant picks by title)
| Option | Picks made | PT-relevant (as measured) | Adjusted by hand* |
|---|---|---|---|
| A. Recent comments (top 9) | 81 (2 blogs have none) | 8 | 12 |
| B. Sticky | 12 (1 blog) | 1 | 1 |
| C. Newest 9 | 99 | 12 | 12 |
| D. Popular among search hits | 76 (2 blogs have none) | 17 | 21 |
| E. Chain sticky, then comments, then newest | 99 | 17 | 21 |
| F. Title-matching posts only, then ranked (not run; the candidate counts above show it is possible for 9 of 11 blogs) | up to 100 | untested | untested |

\* My title match missed Portuguese towns not in its list (Caldas da Rainha, Tomar, Sistelo, Castelo Branco, Costa Vicentina), so Viagens à Solta is counted by hand: 6 of 9 for options A and D.

## Caveats
- Title match is a simple regex, so it undercounts less-known places and counts "Portuguese" culture or language posts as relevant. It is a floor, not a coverage figure. L4 measures real pins.
- `search=` returns body matches, so a "Porto" hit can be a post that only mentions Porto in passing.
- Option F was not run, to keep requests low. Its candidate pools come from the same searches.
- Search used `per_page=100` (not 1), to get IDs and totals in one request.

## Method
- **Date:** 2026-10-11. **Sources:** the 11 `active` entries in `writerdata/sources.json`. Blocked ones skipped.
- **User agent:** `TravelTab/1.0 (https://traveltabbypatronfragas.com; https://github.com/FernasFragas/traveltab)` (the app's `UserAgent`).
- **Politeness:** robots.txt read first for every host (group `traveltab`, else `*`) and every path checked; none of the paths used was disallowed. Sequential, 1.2 s between requests per host, stop on 403/429.
- **Per blog:** robots.txt, up to 10 pages of `comments?per_page=100&_fields=post,date`, `posts?include=…` for the top 9 titles, `sticky=true`, newest 9, 4 destination searches.
- **Total requests: 178** (18 per blog; 8 for Viaje Comigo and Portugalist, whose comments endpoint returned 404). **No 403, no 429, no robots disallow, no errors** beyond those two 404s.
- **Stored:** post IDs, titles, counts and dates only, in a scratch folder outside the repo. No post text.

## Owner decision needed
1. **Signal:** comments tally ranked among destination-matching posts, newest as fill (recommended), or plain newest?
2. **Candidate filter:** search hits plus title match (recommended), search hits alone, or no filter?
3. **Share rule:** keep 9 each, give leftover to Portugal-heavy blogs first (recommended), or a per-blog cap on international blogs (say 3 to 5)?
4. **Blogs with no comments** (Viaje Comigo, Portugalist): use newest among destination-matching posts (recommended)?
5. **Accept lower coverage?** Even the best option leaves many picks that are not about Portugal. L4 measures the Madeira places against PR 2's 24/49.

## Coverage (L4)

**Key point:** one live sync indexed **exactly 100 posts** from 8 of the 11 active blogs, in **145 blog requests and 3 min 39 s**, with no 403, no 429 and no robots skip. For Madeira, **45 of the 49** reference places have a writer pin (PR 2: 24/49), but the methods differ (see **Method difference**). Three international blogs got **0 posts**.

### Run
- **Date:** 2026-10-11, fresh SQLite database, `WRITERS_MAP=1 WRITERS_SYNC=1`, real requests through the app's limiter, robots check and user agent.
- **Startup order:** the sync starts at once and reads the cached destinations first, so on an empty cache only the two override areas (Madeira, São Miguel) are search terms. To test Madeira, a first process (sync off) was asked for `/trip/funchal-pt`, `/map/funchal-pt.geojson` and `/map/madeira-pt.geojson`. That cached two destinations (`funchal`, `madeira`). The key was seen after the map requests; it was not checked after the trip page alone, so the map's base-place fetch is the likely source. The second process then ran the sync. Search terms: **Madeira, São Miguel, funchal** (3 per blog).
- **Port:** 8080 was held by another server (`launch-check.db`), so the run used a scratch copy of the repo on port 18080, with a request-counting transport added to the copy only. The repo code is unchanged. The scratch copy also had a debug flag to read the uncapped writer counts.
- **Time:** the blog requests took 3 min 39 s (20:53:53 to 20:57:32). The cached-destination refresh (Wikidata, not blogs) finished at 20:57:58.

### Requests (145 to blogs, all identified and rate-limited)
| Status | Count |
|---|---|
| 200 | 141 |
| 404 | 3 (comments on Portugalist; Viaje Comigo's comments after a 302; The Crowded Planet `robots.txt`, treated as "no robots") |
| 302 | 1 (Viaje Comigo comments) |
| 403, 429, robots disallow | 0 |

Per blog: 14 to 15 each (robots.txt, 3 searches, up to 10 comment pages, 1 fetch of the chosen posts). Viaje Comigo 7 and Portugalist 6 (no readable comments). The 4 allowlist-blocked sources (travellemming.com, www.vagamundos.pt, viajarentreviagens.pt, lisbonlisboaportugal.com) were skipped with no request.

### Indexed posts (100 in total; the cap held)
| Blog | Posts |
|---|---|
| www.vortexmag.net | 22 |
| www.viajecomigo.com | 21 |
| almadeviajante.com | 19 |
| www.saltinourhair.com | 19 |
| viagensasolta.com | 9 |
| www.earthtrekkers.com | 8 |
| theplanetd.com | 1 |
| www.portugalist.com | 1 |
| www.nomadicmatt.com, www.thecrowdedplanet.com, www.adventurouskate.com | 0 |

- **0 posts** for those three blogs: no hit with a place in its title or slug for the 3 terms. Their state is marked finished. No empty-search warning appeared (none had posts before).
- Titles that name Madeira or Funchal: 59 of the 100 posts (by blog: Viaje Comigo 15, Salt in our Hair 11, Vortex 11, Alma de Viajante 9, Earth Trekkers 7, Viagens à Solta 5, Portugalist 1). The other 41 name another place (not checked further).
- Stored: 26,742 name rows, 52 day rows; SQLite file 1.8 MB.

### Madeira coverage (`/map/madeira-pt.geojson`, area Madeira, 70 km)
| Measure | Result |
|---|---|
| Pins on the map (cap 200) | 200, all with at least 1 writer (before the sync: 200 pins, 0 writers) |
| Places with a writer, uncapped (debug counts) | **243** (94 with 3 or more blogs) |
| The 49 reference places (PR 2, mentioned by 3 or more blogs) with at least 1 writer | **45 of 49** |
| ... with 3 or more writers | 35 of 49 |
| ... among the 200 pins on the map | 44 of 49 |
| Reference places with no writer pin | 25 Fontes, Véu da Noiva, Jardim do Mar, São Gonçalo |
| `/map/funchal-pt.geojson` | 200 pins, 241 places with a writer (91 with 3 or more) |

**Method difference.** PR 2's 24/49 counted how many of the 49 places survive the Wikidata nearby query and the planner's type filter, ignoring the writer index. This run counts how many of the same 49 (matched by QID, from `tasks/writers-map-spike/madeira-coverage.json`) have at least one writer in the real index. The new map also adds any gazetteer place that a post mentions, so the type filter no longer drops freguesias, municipalities or cable cars. The two numbers measure different things; 45/49 is not a like-for-like improvement. The match is not checked for precision (PR 2 found 77.5% right), and the 243 places include loose matches, which this run did not review. Writer counts are blogs per place, from only 100 posts of 8 blogs.

### Open points
- Three of the 11 blogs add nothing for Madeira or the Azores. Their share is empty and the slots went to the Portugal-heavy blogs, as the plan intended.
- The "sudden empty search" check looks at search hits, not destination posts: a blog whose hits exist but none names a place still loses all its rows.
