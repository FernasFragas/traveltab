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
