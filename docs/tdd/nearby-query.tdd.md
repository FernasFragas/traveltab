# Nearby query — N1 measurement (incomplete; owner decision required)

Source: [nearby-query plan](../../tasks/nearby-query-plan.md), [N1 task](../../tasks/nearby-query-todo.md). Measurements ran on 2026-10-12 with the application's `UserAgent` (`TravelTab/1.0 (https://traveltabbypatronfragas.com; https://github.com/FernasFragas/traveltab)`). The grouped query was read from `internal/adapters/api/wikidatanearby.go` after N2's implementation and had the N1 sitelink filter substituted into its optional sitelink line. The reduced-field probe omitted area, English Wikipedia article, and opening hours as specified by the pre-approved fallback. These live probes are separate from the offline test suite.

## Journeys and success criteria

- As a traveller opening Lisbon, Kyoto, London, Paris, New York, or Tokyo, I want notable and writer pins to appear within a cold 5 seconds with less than 5 MB of Wikidata response.
- As a traveller in Funchal, I want the same base pins and writer matches as before.

N1 was to measure the full grouped query without a filter and at N = 1, 2, 3, 5 for all seven cities, choose the largest N preserving writer matches, and time the two-query fallback if needed. The plan says to stop and ask the owner if that fallback also misses the target. The Lisbon fallback timed out after 35 seconds at N = 2, so the remaining city and N probes were stopped. **Neither one query nor two queries is selected for implementation yet.**

## Sitelink threshold and preservation

The prior Funchal `wikidata:v2` cache in `weatherservice.db` has 200 base pins and 1,409 gazetteer items. Its writer-matched fixture QIDs and four 6a QIDs have these current sitelink counts. The fixture's stored counts differ from today's live-adjacent cache, so the cache counts are the relevant threshold check.

| Place | QID | Cache sitelinks | Source |
|---|---|---:|---|
| Pico Ruivo | Q473169 | 31 | Madeira fixture writer match |
| Cabo Girão | Q856866 | 18 | Madeira fixture writer match |
| Câmara de Lobos | Q34799706 | **2** | Madeira fixture writer match |
| Pico do Areeiro | Q1192102 | 23 | Madeira fixture writer match |
| Ponta de São Lourenço | Q1296587 | 15 | Madeira fixture writer match |
| São Jorge | Q1088866 | 16 | 6a case |
| Monte parish | Q1179064 | 21 | 6a case |
| Ponta do Sol municipality | Q943911 | 31 | 6a case |
| Teleférico do Funchal | Q3289157 | 8 | 6a case |

| Minimum sitelinks | Writer-matched fixture/6a places dropped | Prior Funchal base pins dropped by sitelink threshold alone |
|---:|---|---:|
| No filter | None | 0 |
| 1 | None | 0 |
| 2 | None | 37 |
| 3 | Câmara de Lobos (Q34799706) | 112 |
| 5 | Câmara de Lobos (Q34799706) | 160 |

Thus **N = 2 is the largest threshold that retains every named writer match**, but it removes 37 of the 200 cached Funchal base pins. The plan's separate “same pins” acceptance therefore limits N to **1** if interpreted as exact QID preservation. This conflict needs an owner decision together with the query performance failure. The table describes loss due to the filter only; a later re-ranking or changed Wikidata data could also change the final top 200.

The cache check used:

```text
python3 -c 'import sqlite3,gzip,json; c=sqlite3.connect("weatherservice.db"); b=c.execute("select data from source_cache where key like ?",("wikidata:v2:%funchal%",)).fetchone()[0]; d=json.loads(gzip.decompress(b)); print(d.keys()); q={"Q473169","Q856866","Q34799706","Q1192102","Q1296587","Q1088866","Q1179064","Q943911","Q3289157"}; print([(p["qid"],p["names"][0],p.get("sitelinks")) for p in d["gazetteer"] if p["qid"] in q])'
dict_keys(['pins', 'gazetteer'])
[('Q1296587', 'Ponta de São Lourenço', 15), ('Q856866', 'Cabo Girão', 18), ('Q943911', 'Ponta do Sol', 31), ('Q1088866', 'São Jorge', 16), ('Q1179064', 'Monte', 21), ('Q1192102', 'Pico do Areeiro', 23), ('Q473169', 'Pico Ruivo', 31), ('Q3289157', 'Funchal Cable Car', 8), ('Q34799706', 'Câmara de Lobos', 2)]

python3 -c 'import sqlite3,gzip,json,collections; c=sqlite3.connect("weatherservice.db"); d=json.loads(gzip.decompress(c.execute("select data from source_cache where key like ?",("wikidata:v2:%funchal%",)).fetchone()[0])); print(collections.Counter(p.get("sitelinks",0) for p in d["pins"]).most_common(6)); print([ (p["qid"],p.get("sitelinks",0)) for p in d["pins"] if p.get("sitelinks",0)<2][:5])'
[(2, 75), (1, 37), (3, 28), (4, 20), (5, 11), (6, 10)]
[('Q100268999', 1), ('Q100945335', 1), ('Q100980125', 1), ('Q100980216', 1), ('Q100980745', 1)]
```

## Live query results

The grouped query's response was never received within the 20-second probe timeout at N = 2, so its item count and response size cannot be reported. The initial independent all-fields probe also timed out at 35 seconds for Lisbon. `N = 0, 1, 3, 5` and the other five cities were not run after the fallback failure triggered the plan's stop condition.

| City | Query | N | HTTP/status | Items | Response bytes | Elapsed | Target met? |
|---|---|---:|---:|---:|---:|---:|---|
| Funchal | Exact N2 grouped query | 2 | Client read timeout | — | — | 20.145 s | No |
| Lisbon | Exact N2 grouped query | 2 | Client read timeout | — | — | 20.131 s | No |
| Funchal | Reduced-field fallback query 1 | 2 | 200 | 353 | 449,975 | 2.835 s | Yes for query 1 |
| Lisbon | Reduced-field fallback query 1 | 2 | Client read timeout | — | — | 35.126 s | No |

The fallback row measures query 1 only. Query 2 over the 200 pin QIDs was not attempted because Lisbon query 1 already missed the 5-second target. A small Funchal N = 2 `COUNT(?item)` query returned HTTP 200, 370 matching rows, and 238 bytes in 0.308 seconds; this confirmed the endpoint responded while the grouped queries were slow. The count includes repeated coordinates for some items, whereas the grouped fallback returned 353 distinct item bindings. The HTTP client in the app still has a 30-second timeout, so the 35-second fallback probe would fail in the app as well.

Recorded command/output excerpts:

```text
python3 /private/tmp/nearby_measure.py Funchal 2
{"city": "Funchal", "n": 2, "fallback": false, "seconds": 20.145, "error": "The read operation timed out"}

python3 /private/tmp/nearby_measure.py Lisbon 2
{"city": "Lisbon", "n": 2, "fallback": false, "seconds": 20.131, "error": "The read operation timed out"}

python3 /private/tmp/nearby_measure.py Funchal 2 --fallback
{"city": "Funchal", "n": 2, "fallback": true, "status": 200, "seconds": 2.835, "bytes": 449975, "items": 353}

python3 /private/tmp/nearby_measure.py Lisbon 2 --fallback
{"city": "Lisbon", "n": 2, "fallback": true, "seconds": 35.126, "error": "The read operation timed out"}
```

## N1 result and outstanding work

The measured N = 2 single query and its pre-approved two-query fallback both miss the target. A new query design or a revised target needs the owner's decision before N3/N4 proceed. N1 did not edit API code. No RED/GREEN test or coverage result is claimed for this measurement-only task. N6 end-to-end evidence and `make test lint build` remain outstanding.

## N1b: radius x N probes (option B), 2026-10-12

Live, sequential, app `UserAgent`, 70 s client timeout, exact grouped query from `wikidatanearby.go` with `?item wikibase:sitelinks` filter `>= N` and the radius substituted. Probe script was scratch-only (not in the repo). Wikidata was noisy: the same Lisbon 15 km N=2 query gave 30 s then a 502; treat times as +-2x.

| City | Radius km | N | Time | Response | Items | < 5 s and < 5 MB? |
|---|---:|---:|---:|---:|---:|---|
| Lisbon | 20 | 2 | 24.9 s (56 s on a second filter placement) | 2.34 MB | 1,513 | No |
| Lisbon | 40 | 2 | 65 s | 504 | - | No |
| Lisbon | 15 | 2 / 3 | 502 after 30 s / 12.9 s | - / 1.47 MB | - / 916 | No |
| Lisbon | 10 | 2 / 3 | 13.7 s / 53.6 s | 1.85 / 1.26 MB | 1,191 / 784 | No |
| Lisbon | 5 | 2 / 3 / 5 | 11.6 s / 16.6 s / 2.4 s | 1.24 / 0.84 / 0.50 MB | 795 / 523 / 301 | Only N=5 |
| London | 5 | 3 | 56.1 s | 3.98 MB | 2,644 | No |
| Paris | 5 | 3 | 504 after 65 s | - | - | No |
| Kyoto | 10 | 2 | 11.9 s | 1.56 MB | 1,166 | No |
| Tokyo | 10 | 3 | 67.6 s | 3.01 MB | 2,142 | No |
| New York | 10 | 3 | 504 after 65 s | - | - | No |
| Funchal | 40 / 25 / 15 | 2 | 4.7 s / 5.2 s / 7.6 s | 0.53 / 0.47 / 0.37 MB | 353 / 308 / 241 | 40 km only |

Finding: **time tracks item count (about 10-25 ms per item) at any radius, not response size.** Size is always under 5 MB; time is not. Moving the sitelinks filter before the OPTIONALs, or into a sub-select, did not help (Lisbon 20 km N=2: 56 s and 27 s). Only Lisbon 5 km N=5 met the target, and N=5 drops Camara de Lobos (2 sitelinks) and about 160 of the 200 prior Funchal pins; a 5 km radius would also drop Ponta de Sao Lourenco and Cabo Girao from Funchal. **No single radius + N meets the targets. No code change to radius or filter was made; N3/N4 not done. Owner decision needed.**

## N1c: Q1 alone per city (option A feasibility), 2026-10-12

Live, sequential, app `UserAgent`, 70 s client timeout, 40 km. Q1 = QID, coords, types, type label, en/pt labels, description, image, sitelinks (no area, article, hours; aliases dropped). Wikidata is very noisy (the same Lisbon query returned 3.7 s and 52 s/incomplete).

| City | Q1 shape | N | Time | Response | Items | Target met? |
|---|---|---:|---:|---:|---:|---|
| Funchal | grouped, with / without aliases | 0 | 16.6 / 17.2 s | 1.67 / 1.60 MB | 1,425 | No |
| Funchal | grouped, with / without aliases | 2 | 9.4 / 10.6 s | 0.45 / 0.41 MB | 353 | No |
| Lisbon | grouped, without aliases | 0 | 504 after 65 s | - | - | No |
| Lisbon | grouped, without aliases | 2 | 29.5 s | 2.36 MB | 2,036 | No |
| Lisbon | grouped, with aliases | 2 | 504 after 65 s | - | - | No |
| Lisbon | ungrouped, OPTIONAL joins | 0 / 2 | 12.2 s / 15.6 s | 9.7 / 4.1 MB | 9,227 / 3,507 rows | No |
| Kyoto | ungrouped | 2 | 4.8 s | 4.9 MB | 4,756 rows | Borderline (size) |
| London | ungrouped | 2 | 61.3 s | 19.8 MB | 18,487 rows | No |
| Paris | ungrouped | 2 | 27.1 s | 23.5 MB | 21,937 rows | No |
| New York | ungrouped | 2 | 15.1 s | 12.8 MB | 12,008 rows | No |
| Tokyo | ungrouped | 2 | 24.7 s | 16.2 MB | 15,647 rows | No |

Single-field probes, Lisbon N=2 (ungrouped): core (coords + sitelinks) 1.0 s; + types 1.2 s; + type label 5.8 s; + en label 3.0 s; + pt label 2.0 s; + description 3.2 s; + image 1.2 s; all together 15-17 s; minimal types + en/pt labels 3.7 s (once; 52 s incomplete on repeat). The `wikibase:label` service is slower (25-29 s).

Findings:
- **GROUP BY / SAMPLE is slower than plain rows** for Q1 (Lisbon N=2: 29.5 s grouped vs 15.6 s ungrouped).
- **Cost is per-language label lookups** (labels, descriptions, type labels), roughly additive per OPTIONAL. Dropping aliases did not help (Funchal 9.4 vs 10.6 s: noise).
- **N=2 still leaves 12-22k items in London, Paris, New York, Tokyo** (the 40 km area holds 65-100k items), so those exceed 5 MB even before time.
- Q1 meets neither target for London, Paris, New York, Tokyo, Lisbon or Funchal at N<=2. Only N>=3 would shrink the big cities enough, and that drops Camara de Lobos (2 sitelinks). Per the owner's instruction, **stopped; N4 not implemented**. No RED/GREEN or N6 evidence is claimed.
