# Writers' Map Index Spike — Results

**Verdict:** keep the phrase/QID/position schema for PR 5, with the planned five-word sub-span cap — size and lookup pass easily. **Both remaining checks fail their proposed bars (re-run October 10, 2026); the owner decided how to handle them the same day, and PR 5 is unblocked** (see [Owner decisions](#owner-decisions-october-10-2026)):
- **Coverage: 24 of 49** places mentioned by ≥3 blogs survive the Wikidata nearby query + the planner's type filter (22 inside the 200 cap). The filter drops villages, parishes, municipalities, cable cars and a waterfall.
- **Precision: 31/40 = 77.5%** right place (bar ≥90%). Main causes: posts about all of Portugal, and the cable car's "Monte" alias.

Run October 6, 2026. The reproducible runner is [`writers-map-spike/run.py`](writers-map-spike/run.py). It streamed the public WordPress API responses through memory with the identified `TravelTab-research/0.1 (blog overlap check)` user agent and a 1.5-second delay between pages. It stopped on access blocks. **It did not check `robots.txt`.** The script now checks it before each host (added October 7, 2026), so a rerun respects it; the October 6 numbers came from the run without the check. No post body, title, or URL was written to the database or output. Only matched QIDs/names, post IDs, and offsets were indexed. The temporary database and downloaded candidate-name cache were removed after measurement.

## Measurements

| Measure | Result | Proposed bar | Outcome |
| --- | ---: | ---: | --- |
| Public posts processed | 4,800 total: viajecomigo 4,207; Salt in Our Hair 593 | Two full blogs | Met |
| Unique Madeira Wikidata candidates | 357 QIDs, 593 folded en/pt names | — | — |
| Indexed name occurrences | 2,892 | — | — |
| SQLite size | 679,936 bytes (0.68 MB decimal) | — | — |
| Rows per 1,000 posts | 602.5 | — | — |
| Linear projection to 20,400 posts | 2.89 MB | ≤300 MB | Pass |
| Warm indexed phrase lookup, p50 of 101 | 0.034 ms (`madeira`, up to 200 rows) | ≤200 ms | Pass |
| Precision sample (Oct 10 re-run) | 31/40 = 77.5% | ≥90% | **Fail** |
| ≥3-blog places kept by nearby query + filters (Oct 10 re-run) | 24/49 type filter; 22/49 inside the 200 cap | Record count | **Recorded; low** |

The size benchmark used a temporary SQLite table keyed by post ID, QID, folded phrase, and character position, with phrase and QID lookup indexes. It stores mention occurrences, not post content. The size projection is a linear extrapolation from these two blogs; it does not include extra rows for generated sub-spans. Keeping sub-spans capped at five words bounds that additional growth.

## Coverage and precision re-run (October 10, 2026)

Runner: [`writers-map-spike/coverage.py`](writers-map-spike/coverage.py). Names-only output: [`writers-map-spike/madeira-coverage.json`](writers-map-spike/madeira-coverage.json) (place names, QIDs, blog hosts, post IDs, hand verdicts — no post text). It repeats the October 5 deep run (OSM gazetteer + Wikidata aliases, `fetch_deep.py` post rules, `match.py` matcher) with posts kept in memory, `robots.txt` checked per host, and today's `writerdata/overrides.json` stoplist.

### Corpus

| | Oct 5 deep run | Oct 10 re-run |
| --- | ---: | ---: |
| Posts | 231 | **228** |
| Places mentioned | 102 | 97 |
| ≥3 blogs | 55 | **49** |
| ≥4 blogs | 41 | 37 |
| Single-source | 27 | 26 |

- **Per-host posts are identical** except travellemming (3 → 0).
- **travellemming.com's `robots.txt` now disallows the posts API**, so it was skipped. `writerdata/sources.json` should mark it `blocked` (not in 2a's Owns — left for the owner).
- earthtrekkers timed out again: 23 posts, same as October 5.
- The 55 → 49 drop comes from losing travellemming plus the three added stopwords (Fernandes, Infante, Achada). The October 5 names were never kept, so the 49 replace the 55 as the reference set.

### Coverage: what Wikidata nearby + filters keeps

Nearby query: `wikibase:around` at the `overrides.json` Madeira area (32.75, −16.96, 70 km) → 1,504 items → 587 after the planner's type rule (`planner.kindOf` over `PlaceTypes`) → top 200 by sitelinks.

- [x] **All 49** are within the radius.
- [ ] **24/49** pass the type filter.
- [ ] **22/49** are inside the 200 cap (Nazaré and Ponta do Rosto fall below it).

**Why 25 places are dropped** (direct P31 type not in `PlaceTypes`):

| Missing type | Places | Examples |
| --- | ---: | --- |
| freguesia of Portugal (Q1131296) | 15 | São Jorge (8 blogs), Monte (7), Caniçal, Faial, Seixal, Curral das Freiras |
| municipality of Portugal (Q13217644) | 4 | Ponta do Sol (7), Calheta, Porto Moniz, São Vicente |
| aerial lift (Q13409250) | 2 | Teleférico do Funchal (7), Teleférico do Jardim Botânico |
| locality (Q3257686) / place (Q98929991) | 3 | Ribeiro Frio, Paul do Mar, São Lourenço |
| transport service (Q1067164) | 1 | Carros de cesto do Monte |
| waterfall (Q34038) | 1 | Véu da Noiva |

**This conflicts with the plan.** The plan excludes "municipalities as such", and 4a's acceptance expects Câmara de Lobos to be included, but its Wikidata items are a municipality (Q623736) and a freguesia (Q2629775), both filtered out. With today's `PlaceTypes`, 4a can't pass that acceptance and the map loses all four places mentioned by ≥7 blogs (São Jorge's 8 is partly inflated by the Azores island of the same name; see precision).

### Precision: 40 random first mentions, hand-reviewed

Same draw as `match.py` (`random.seed(7)`, one sample per place per blog). Snippets were reviewed in a temporary directory and deleted; only verdicts are kept.

- [ ] **31/40 = 77.5% right place** (bar ≥90%; October 5 shallow run: 37/40).

| Cause | Wrong | Examples |
| --- | ---: | --- |
| Post about all of Portugal, same name elsewhere | 5 | Bom Jesus do Monte (Braga) ×3, São Jorge (Azores), Pitos de Santa Luzia (Vila Real) |
| Cable car's "Monte" alias | 1 | "a hotel in Monte" counted as the cable car (it also turned one Braga mention into a cable-car match) |
| Shared name, different entity | 3 | Garajau beach and headland → the nature reserve; Ponta da Calheta (Porto Santo) → Calheta |

- **Cheap fixes:** drop the "Monte" alias from the cable car and "Garajau" from the reserve (`overrides.json`) → 34/40 = 85% on this sample. Still under 90%.
- **The rest needs matcher context:** only count a mention when the post (or its section) is about the destination, not just "Madeira ×3 in the body". That belongs in 6a.
- As before, many correct matches aren't recommendations: addresses, hotel names, parking, legends.

### Owner decisions (October 10, 2026)

1. **Type filter → writer-mentioned places skip it**, like planner boosts (Deny types still drop). The base layer keeps the filter and the 200 cap; the planner is unchanged. 4a returns the unfiltered nearby list as the gazetteer; 4a's Câmara de Lobos acceptance moves to the gazetteer.
2. **Precision → start PR 5 now; ≥90% becomes a 6a acceptance.** Done now: the "Monte" (cable car) and "Garajau" (reserve) aliases are removed in `writerdata/overrides.json` (34/40 on this sample). 6a adds a destination context rule and re-checks 40 samples on the real gazetteer before `WRITERS_MAP` launches.
3. **travellemming → `blocked`** in `writerdata/sources.json`, checked 2026-10-10.

**Design decision (unchanged):** keep the phrase/QID/position shape. **PR 5 is unblocked.**

## PR 8 live verification

Run on October 13, 2026, against real WordPress and Wikidata sources using a temporary SQLite database (`/private/tmp/traveltab-pr8b.db`). `WRITERS_SYNC=1`, `WRITERS_MAP=1`, `WRITERS_AI=0` and `GOMEMLIMIT=850MiB` were used locally. Fly is configured for 1 GB, but a Go heap limit on a local machine is not a Fly machine memory test. The current sync selects at most **100 posts in total**, not the original plan's 20,400-post crawl. No raw post body was saved.

The first source-indexing phase was observed from 16:08:47 to 16:12:41 local time (about 3m54s). The resulting database has 75 selected posts, 23,382 `writer_names` rows, 40 `writer_days` rows and 11 source states. It occupies 1,691,648 bytes; SQLite `dbstat` attributes 1,216,512 bytes to the `writer_*` tables and indexes. The local process had 43,344 KiB RSS after the run (`ps -o pid,rss,vsz,etime,command -p 48063`). Peak RSS and bandwidth were not captured on this first run. The machine and process are local, not Fly.

**Second live run, against a copy of that database.** One `writer_sync.last_full` was cleared in the scratch copy to make the scheduler run. The process started at 16:55:06 local time, finished selecting posts by 17:00:00 (4m54s), and logged `writers sync: refreshed 4 of 4 cached destinations` at 17:09:03: **13m57s total**. Reykjavik's first Wikidata refresh attempt failed after 278 partial places and was retried. A 1-second `ps` sampler collected 833 RSS samples during the run; its maximum was **48,160 KiB** (about 47 MiB). `nettop -P -x -L 1 -p 53297 -J bytes_in,bytes_out` reported **664,538 bytes in** and **323,195 bytes out** near completion, about 0.99 MB combined for that process. These are process counters on the local machine, including any local HTTP traffic; they are not a Fly billing measure. The final database was **3,018,752 bytes**, with 100 selected posts, 8 itinerary posts and 2,428,928 bytes in `writer_*` tables and indexes. It grew because four cached destinations were searched in the second run, whereas the first run began with fewer cached destinations and selected only 75 posts. This is **not** the unchanged-data growth check.

An intermediate cold-map observation reported Madeira ready about 87.5 seconds after an already-started request and Lisbon ready in 65.5 seconds. Those intermediate timings have no retained command output, so they are indicative only. The four-city script was rerun against the completed cache with `python3 /private/tmp/traveltab-pr8b-latency.py`:

| Cached destination | Features | Writer pins | Writer itineraries | Five warm GeoJSON requests (ms) |
| --- | ---: | ---: | ---: | --- |
| Madeira | 200 | 124 | 4 | 6.1, 5.3, 5.0, 4.5, 4.5 |
| Lisbon | 200 | 13 | 0 | 2.0, 1.9, 1.9, 1.8, 1.7 |
| Paris | 200 | 1 | 0 | 1.6, 1.6, 1.6, 1.4, 1.7 |
| Reykjavik | 140 | 0 | 0 | 1.2, 0.9, 1.1, 0.9, 0.8 |

The same local server's cached Madeira panel (`/map/madeira-pt/Q473169?lat=32.75&lon=-16.96`) took 12.7, 6.0, 6.0, 5.6 and 5.0 ms for five sequential reads. The first cached `writermap:v3` results contain opening hours for **0/200 Madeira** and **1/200 Lisbon** map places; after the refresh, they contain **0/200 Madeira** and **0/200 Lisbon**, showing upstream result variation. The first index marked **6/75 posts** as itineraries, with 40 day rows; the second marked **8/100**, with 73 day rows. Neither provides 20 itinerary posts to review.

**Cold map check with the 100-post index.** A copy of the second database had only its `wikidata:*` and `writermap:*` cache rows removed; the writer index remained. `WRITERS_SYNC=0` kept the source selection fixed. `python3 /private/tmp/traveltab-pr8b-latency.py` ran Madeira, Lisbon, Paris and Reykjavik sequentially; `python3 /private/tmp/traveltab-pr8b-no-writers.py` then ran Longyearbyen. The first response is progressive and may contain zero pins; “ready” means its `loading` flag became false. A later localhost read confirmed Lisbon eventually finished with 200 places, but its finish time was not captured.

| Destination | First progressive response | Full result after first response | Final places | Writer pins | Five warm requests (ms) |
| --- | ---: | ---: | ---: | ---: | --- |
| Madeira | 23.1 ms | 46.31 s | 200 | 100 | 2.7, 2.4, 2.2, 2.2, 2.2 |
| Lisbon | 0.3 ms | **>180.95 s**; 74 partial places at cutoff | 200 later | 0 at cutoff | 0.9, 1.5, 1.0, 0.7, 1.1 (partial result) |
| Paris | 0.8 ms | 84.55 s | 200 | 46 | 1.8, 1.6, 1.7, 1.8, 1.8 |
| Reykjavik | 0.3 ms | 135.76 s | 153 | 28 | 1.6, 1.7, 1.8, 1.4, 1.6 |
| Longyearbyen, no writer matches | 4.8 ms | 42.21 s | 45 | **0** | 1.6, 0.9, 0.8, 0.7, 0.7 |

The four sequential requests overlap with detached background loads, so Paris and Reykjavik timings include possible contention from Lisbon's still-running load. The first response byte count was 91 for each cold city: it announces loading and is not a complete map.

The task's “monthly delta size” measure no longer corresponds to a separate job: the current capped scheduler reselects and upserts the full at-most-100-post set each due month, while `writer_sync.last_delta` is no longer written. The changed-selection second run is therefore a monthly-style run, but it cannot establish unchanged-data growth.

A deterministic SQLite check covers the unchanged-input invariant: `go test ./internal/writermap/ -run TestSyncJob_TwoUnchangedRunsKeepRowCountsAndFileSize -count=1 -v` → `--- PASS: TestSyncJob_TwoUnchangedRunsKeepRowCountsAndFileSize`. It runs three due syncs with the same fake posts and compares row counts and database file size between the latter two. This proves the storage behavior under fixed input; the second real-source run above had different selected posts and is not that control.

**Open before 8b can be marked done:** repeat a full run with the *same* cached destinations and compare database size; hand-review a fresh 40-match precision sample on the current real index; and review 20 itinerary posts or record an owner-approved smaller sample. The measured memory limit was a Go heap limit, not an actual Fly 1 GB VM, so the machine-sized build remains unverified. Lisbon's complete cold time remains bounded only below by 180.95 seconds. `tasks/todo.md` says the 6a precision review was checked by the owner, while `docs/tdd/6a.tdd.md` contains no numeric current sample, so the PR 2 result of 31/40 cannot stand in for PR 8. The owner asked to **measure first** before setting target thresholds and a launch date. No release sign-off has been recorded.
