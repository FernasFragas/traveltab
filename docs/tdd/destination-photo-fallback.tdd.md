# Destination photo fallback: D1–D5 evidence

Source: [plan](../../tasks/destination-photo-fallback-plan.md), [tasks](../../tasks/destination-photo-fallback-todo.md) D1–D5.

Journeys:

- As a traveller searching Amsterdam, I want its city photo, so that the page shows the destination.
- As a traveller searching a namesake, I want another country's photo refused.
- As a traveller searching New York, Washington, Quebec or Old Toronto, I want the city photo even when a state, province or former city appears in the name search.

## D1: baseline and query budget

The updated plan now supplies the exact 59 OpenWeather identities. Before D3, `go test ./internal/adapters/api/ -run '^TestDestinationPhotoBaselineProbe$' -count=1 -v` ran the current lookup on all 59 (temporary probe test, removed afterward). The saved live run reported 55 photos and four no-photo results, with no provider errors; Amsterdam already succeeds after D2. Total run: 83.67 s. A first run without output capture also completed all 59 in 84.25 s with the same 55/4 outcome. A second capture attempt inside the sandbox produced DNS errors for all entries; it was rerun outside the sandbox and the table below comes from that successful capture. Times are per lookup, not image downloads.

| OpenWeather name | Country | Entity | Commons filename | Lookup time |
| --- | --- | --- | --- | ---: |
| New York | US | NONE | — | 1.785358083s |
| Los Angeles | US | Q65 | File:Los Angeles with Mount Baldy.jpg | 1.449335958s |
| San Francisco | US | Q62 | File:San Francisco from the Marin Headlands in March 2019.jpg | 1.742642125s |
| Washington | US | NONE | — | 1.503854417s |
| Chicago | US | Q1297 | File:2010-02-19 3000x2000 chicago skyline.jpg | 1.498060958s |
| Miami | US | Q8652 | File:Downtown Miami aerial 2008.jpg | 1.4743565s |
| Las Vegas | US | Q23768 | File:Downtown Las Vegas from Stratosphere 3.jpg | 1.336543625s |
| Mexico City | MX | Q1489 | File:Sobrevuelos CDMX HJ2A5091 (40386338731).jpg | 1.111282s |
| Rio de Janeiro | BR | Q8678 | File:Cidade Maravilhosa.jpg | 1.559839916s |
| Buenos Aires | AR | Q1486 | File:Vista aérea de la Plaza del Congreso 10.jpg | 1.497592125s |
| Old Toronto | CA | NONE | — | 742.526208ms |
| Montreal | CA | Q340 | File:Montreal-canada-parc-urban.jpg | 1.572286416s |
| London | GB | Q84 | File:London Skyline (125508655).jpeg | 1.286993458s |
| Paris | FR | Q90 | File:La Tour Eiffel vue de la Tour Saint-Jacques, Paris août 2014 (2).jpg | 1.754466625s |
| Rome | IT | Q220 | File:Rome Skyline (8012016319).jpg | 1.589682666s |
| Venice | IT | Q641 | File:Venezia aerial view.jpg | 1.853572125s |
| Florence | IT | Q2044 | File:Panoramica Firenze.jpg | 1.72147625s |
| Barcelona | ES | Q1492 | File:Aerial view of La Sagrada Familia, Barcelona, Spain (51227006134).jpg | 1.555176875s |
| Madrid | ES | Q2807 | File:Madrid (38624991251).jpg | 1.4235935s |
| Lisbon | PT | Q597 | File:Lisboa837.png | 1.444791833s |
| Porto | PT | Q36433 | File:O Porto (visto da Ponte Dom Luis I).jpg | 1.072056917s |
| Funchal | PT | Q25444 | File:Vista de Funchal desde Pico dos Barcelos, Madeira, Portugal, 2019-05-29, DD 37-44 PAN.jpg | 1.187263083s |
| Amsterdam | NL | Q727 | File:Amsterdam airphoto.jpg | 1.521193417s |
| Berlin | DE | Q64 | File:Cityscape Berlin.jpg | 1.9651575s |
| Munich | DE | Q1726 | File:Stadtbild München.jpg | 1.841245458s |
| Frankfurt | DE | Q1794 | File:Frankfurter Altstadt mit Skyline 2019 (100MP).jpg | 1.6556595s |
| Vienna | AT | Q1741 | File:Wien katedra sw Szczepana widok z wiezy 6.jpg | 1.386567167s |
| Prague | CZ | Q1085 | File:Petřín Tower View IMG 3020.JPG | 1.521625333s |
| Budapest | HU | Q1781 | File:Budapest, Hungary (explored) (14995308504).jpg | 992.874709ms |
| Athens | GR | Q1524 | File:Akropolis 2015.jpg | 1.908762417s |
| Istanbul | TR | Q406 | File:Historical peninsula and modern skyline of Istanbul.jpg | 1.839228334s |
| Dubrovnik | HR | Q1722 | File:The walls of the fortress and View of the old city. panorama.jpg | 1.1206575s |
| Brussels | BE | Q239 | File:Brussels view from Mont des Arts, Brussels, Belgium (cropped).jpg | 1.274699333s |
| Copenhagen | DK | Q1748 | File:Copenhagen - view from Christiansborg castle.jpg | 1.238808459s |
| Saint Petersburg | RU | Q656 | File:Peter & Paul fortress in SPB 03.jpg | 1.426916417s |
| City of Edinburgh | GB | Q23436 | File:Edinburgh cityscapes - aerial - 2025-04-19 20.jpg | 1.236780458s |
| Tokyo | JP | Q1490 | File:Skyscrapers of Shinjuku 2009 January (revised).jpg | 1.278894458s |
| Kyoto | JP | Q34600 | File:Pedestrian road with pavements, paper umbrellas and people in yukata, Higashiyama-ku, Kyoto, Japan.jpg | 1.375985s |
| Seoul | KR | Q8684 | File:덕수궁의 가을.jpg | 1.223802292s |
| Beijing | CN | Q956 | File:Skyline of Beijing CBD with B-5906 approaching (20211016171955).jpg | 1.548135167s |
| Singapore | SG | Q334 | File:Raffles Place.jpg | 1.538658s |
| Bangkok | TH | Q1861 | File:Temple of the Emerald of buddha or Wat Phra Kaew (cropped).jpg | 1.377018041s |
| Ho Chi Minh City | VN | Q1854 | File:Ho Chi Minh City Skyline.jpg | 1.079025583s |
| Hanoi | VN | Q1858 | File:Tháp Rùa - NKS.jpg | 1.150177542s |
| Pulau Bali | ID | Q4648 | File:Pura Taman Ayun terletak di Mengwi, Bali.jpg | 1.272926333s |
| Delhi | IN | Q1353 | File:Red Fort in Delhi 03-2016 img3.jpg | 1.261732917s |
| Mumbai | IN | Q1156 | File:Panoramic view of Taj Palace Hotel and Taj Tower with the iconic Gateway of India in the background (cropped).jpg | 1.294077959s |
| Dubai | AE | Q612 | File:Dubai Skyline mit Burj Khalifa (cropped).jpg | 1.666635875s |
| Marrakesh | MA | Q101625 | File:Djemaa el Fna, evening.JPG | 1.194162292s |
| Cairo | EG | Q85 | File:Cairo Skyline (2020).jpg | 1.159598875s |
| Cape Town | ZA | Q5465 | File:Ciudad del Cabo desde Cabeza de León, Sudáfrica, 2018-07-22, DD 34.jpg | 1.3279665s |
| Sydney | AU | Q3130 | File:00 3178 Sydney, Australia.jpg | 1.624727334s |
| Melbourne | AU | Q3141 | File:Melbourne CBD and Ceremonial Avenue (in 2024) as seen from the rooftop of Shrine of Remembrance.jpg | 1.326976208s |
| Auckland | NZ | Q37100 | File:Auckland CBD.jpg | 1.275943833s |
| Reykjavik | IS | Q1764 | File:Reykjavík 5545 (cropped).JPG | 1.409658792s |
| Kuala Lumpur | MY | Q1865 | File:Kl-skyline-at-night-2022.jpg | 1.183995917s |
| Vatican City | VA | Q237 | File:0 Basilique Saint-Pierre - Rome (1).JPG | 1.25961025s |
| Monaco | MC | Q235 | File:Panorama von Monaco-La Turbie.jpg | 1.353542917s |
| Quebec | CA | NONE | — | 1.393627916s |

The four missing results are New York, Washington, Old Toronto and Quebec. Amsterdam returned Q727 with `Amsterdam airphoto.jpg` in 1.321 s.

The proposed `action=query&list=search` endpoint was measured through the app's `WikimediaAPI.get` and `UserAgent` using `go test ./internal/adapters/api/ -run '^TestDestinationPhotoSearchProbe$' -count=1 -v` (temporary probe test, removed afterward). `srlimit=5`, `srsearch=haswbstatement:P1082 nearcoord:25km,<lat>,<lon>`:

| City | Time | Top five QIDs | Expected selected city |
| --- | ---: | --- | --- |
| New York | 341.878 ms | Q60, Q18419, Q11299, Q18426, Q18424 | Q60 |
| Washington | 239.855 ms | Q61, Q88, Q107126, Q584451, Q26807 | Q61 |
| Quebec | 212.467 ms | Q2145, Q139208, Q142131, Q7402342, Q3483898 | Q2145 |
| Old Toronto | 235.236 ms | Q172, Q50816, Q1025401, Q52998, Q44013 | Q172 |

The first three city entities have a matching label or alias; Old Toronto has no matching name in this result set, so `chooseCity` should select Q172 after location and country checks. These are expected picks before D3's implementation; D4 will check the actual results.

The proposed query was measured with an approximate New York centre (`40.7128,-74.0060`), not the unavailable original OpenWeather coordinates:

```sparql
SELECT DISTINCT ?item ?label ?alias ?sitelinks WHERE {
  SERVICE wikibase:around {
    ?item wdt:P625 ?coord .
    bd:serviceParam wikibase:center "Point(-74.0060 40.7128)"^^geo:wktLiteral .
    bd:serviceParam wikibase:radius "25" .
  }
  ?item wdt:P31 ?type .
  VALUES ?type { wd:Q515 wd:Q1549591 wd:Q3957 wd:Q532 wd:Q5119 wd:Q486972 wd:Q5084 wd:Q15284 wd:Q1637706 wd:Q702492 wd:Q13217644 wd:Q1093829 }
  ?item wikibase:sitelinks ?sitelinks .
  OPTIONAL { ?item rdfs:label ?label . FILTER(LANG(?label) = "en") }
  OPTIONAL { ?item skos:altLabel ?alias . }
}
```

Using the app's `UserAgent`, `curl --get --max-time 12 --data-urlencode query@/private/tmp/destination-photo-fallback.rq https://query.wikidata.org/sparql` returned `curl: (28) Operation timed out after 12005 milliseconds with 0 bytes received`; `status=000 time=12.005969 size=0`. A narrowed query selecting only QID and sitelinks, ordered by sitelinks with `LIMIT 50`, also returned `curl: (28) Operation timed out after 8005 milliseconds with 0 bytes received`; `status=000 first=0.000000 total=8.005546 size=0`. A trivial WDQS Q60 query returned `status=200 first=3.012517 total=3.013759`, and a Wikidata API health request returned `status=200 first=0.364115 total=0.365232`. The spatial query cannot fit the lookup's four-second budget as measured. The other four spatial probes were not run because New York alone disqualifies this design.

Read-only canonical-name `wbsearchentities` probes returned these first hits with `limit=5`, each over the Wikidata API with the app's `UserAgent`:

| Search | First QID | Time |
| --- | --- | ---: |
| New York City | Q60 | 0.306 s |
| Washington, D.C. | Q61 | 0.324 s |
| Quebec City | Q2145 | 0.335 s |
| Toronto | Q172 | 0.327 s |

At this D1 checkpoint, these probes suggested a faster path but did not establish a safe general variant rule. D3 and D4 subsequently used the updated plan's geo-filtered search and 59-city list.

## D2: deprecated-only country code

For Amsterdam's Netherlands item Q55, a deprecated `P297=NL` is usable only when no preferred or normal code exists. The existing unknown-country and no-country tests still guard against accepting an unverified country.

RED: `go test ./internal/adapters/api/ -run TestDestinationPhoto_DeprecatedCountryCodeOnlyWhenNoCurrentCode -count=1` failed in `deprecated_only`: `expected: "NL"`, `actual: ""`.

GREEN: `go test ./internal/adapters/api/ -run 'TestDestinationPhoto_DeprecatedCountryCodeOnlyWhenNoCurrentCode|TestDestinationPhoto_RefusesEntitiesOfAnotherCountryOrPlace' -count=1` returned `ok weatherservice/internal/adapters/api 0.581s`. After splitting the ambiguity and foreign-country cases, the focused command returned `ok weatherservice/internal/adapters/api 0.576s`. `go test ./internal/adapters/api/ -run 'DestinationPhoto' -count=1` returned `ok weatherservice/internal/adapters/api 0.655s`. `go test -cover ./internal/adapters/api/ -count=1` returned `coverage: 84.2% of statements`.

A temporary live test, removed after the probe, ran `go test ./internal/adapters/api/ -run '^TestDestinationPhotoLiveAmsterdamProbe$' -count=1 -v` with Amsterdam/NL at approximate centre `52.3676,4.9041`. It returned `SourceID:Q727`, Commons file `Amsterdam_airphoto.jpg`, `err=<nil>`, `elapsed=1.74501325s`, and `PASS`. This verifies a live photo under four seconds, but the coordinates are not the unavailable OpenWeather baseline.

| What is guaranteed | Test name | Kind | Result |
| --- | --- | --- | --- |
| Deprecated-only `NL` is read, while a normal code wins over deprecated | `TestDestinationPhoto_DeprecatedCountryCodeOnlyWhenNoCurrentCode` | Unit with HTTP fake | Pass |
| Amsterdam city passes the checked country path | `TestDestinationPhoto_AmsterdamAcceptsDeprecatedOnlyNLCode` | Adapter integration with HTTP fake | Pass |
| Approximate Amsterdam live lookup returns Q727 within four seconds | Temporary `TestDestinationPhotoLiveAmsterdamProbe` | Live integration | Pass; temporary test removed |
| Unknown or absent country, wrong place or distant place remains refused | `TestDestinationPhoto_RefusesEntitiesOfAnotherCountryOrPlace` | Adapter integration with HTTP fake | Pass |

At the D2 checkpoint, the live 59-city comparison remained outstanding. Interim gate after D2: `PATH=/tmp/traveltab-lint-bin:$PATH make test lint build` passed (`go test -race -count=1 ./...` all packages `ok`; `golangci-lint run ./...` reported `0 issues.`; `go build -o bin/traveltab ./cmd/web` succeeded). The compatible lint binary was built with Go 1.27.2.

## D3: nearby populated places

When exact-name Wikidata search and nearby Wikipedia articles find no city, the adapter asks Wikidata's search index for up to five populated entities within 25 km. It runs the existing settlement, coordinate, country and ambiguity checks. A passing English label or alias matching the destination name is preferred; otherwise `chooseCity` ranks the passing places. Only this final path requests labels and aliases. A search failure remains an error.

RED: `go test ./internal/adapters/api/ -run 'TestDestinationPhoto_PopulatedNear' -count=1` failed as intended: `Expected value not to be nil` for New York, Old Toronto and the foreign-country case, and `An error is expected but got nil` for a 503 from the new search endpoint. The missing implementation never made that request.

GREEN: after adding the fallback, `go test ./internal/adapters/api/ -run 'TestDestinationPhoto_PopulatedNear' -count=1` returned `ok weatherservice/internal/adapters/api 0.545s`. The first implementation test build exposed `%g` on an integer radius; that formatting error was corrected to `%d` before this green run. After splitting the ambiguity and foreign-country cases, the focused command returned `ok weatherservice/internal/adapters/api 0.576s`. `go test ./internal/adapters/api/ -run 'DestinationPhoto' -count=1` returned `ok weatherservice/internal/adapters/api 0.345s`. `go test -cover ./internal/adapters/api/ -count=1` returned `coverage: 84.3% of statements`.

| What is guaranteed | Test name | Kind | Result |
| --- | --- | --- | --- |
| New York city wins by alias even when Brooklyn has more sitelinks; search uses the 25 km query and five-result limit | `TestDestinationPhoto_PopulatedNearPrefersMatchingAliasOverMoreSitelinks` | Adapter integration with HTTP fake | Pass |
| Old Toronto resolves to the established Toronto city without a name match | `TestDestinationPhoto_PopulatedNearChoosesTorontoWithoutANameMatch` | Adapter integration with HTTP fake | Pass |
| Comparable distant candidates are ambiguous | `TestDestinationPhoto_PopulatedNearRefusesAmbiguousCities` | Adapter integration with HTTP fake | Pass |
| A foreign city is refused | `TestDestinationPhoto_PopulatedNearRefusesForeignCity` | Adapter integration with HTTP fake | Pass |
| An HTTP 503 from the last search returns an error | `TestDestinationPhoto_PopulatedNearProviderErrorIsNotNoPhoto` | Adapter integration with HTTP fake | Pass |
| The new search is skipped when either earlier search found a city | `TestDestinationPhoto_FindsAnEntityTheNameSearchMissed`, `TestDestinationPhoto_SkipsTheNearbyFallbackWhenTheSearchMatched` | Adapter integration with HTTP fake | Pass |

## D4: live regression

`liveDestinations` now includes the plan's 59 exact OpenWeather identities after the original ten. The opt-in test checks each cold lookup is under four seconds, logs the selected entity and Commons file, and downloads its image.

The first `LIVE_API_TESTS=1 go test ./internal/adapters/api/ -run '^TestDestinationPhotoLive$' -count=1 -v` run reached all 69 entries in 116.08 s. Kyoto returned `wikidata search: cirrussearch-too-busy-error: Search is currently too busy. Please try again later.` after 230 ms; the other 68 returned photos. The failed run remains in `/private/tmp/destination-photo-live-2026-10-13-attempt1.log` and is not counted as a pass.

A separate retry with the same command passed all 69 in 107.41 s (`ok weatherservice/internal/adapters/api 107.928s`), with no provider errors or missing photos. The slowest lookup was Quebec at 3.220 s. Full output is in `/private/tmp/destination-photo-live-2026-10-13-attempt2.log`.

| Destination | City entity | Commons file | Lookup time |
| --- | --- | --- | ---: |
| New York | Q60 | `NYC_Downtown_Manhattan_Skyline_seen_from_Paulus_Hook_2019-12-20_IMG_7347_FRD_(cropped).jpg` | 3.017 s |
| Washington | Q61 | `Aerial_view_of_White_House_and_downtown,_Washington,_D.C_LCCN2010630891.jpg` | 2.893 s |
| Old Toronto | Q172 | `Sunset_Toronto_Skyline_Panorama_Crop_from_Snake_Island.jpg` | 2.256 s |
| Quebec | Q2145 | `Quebec_City_skyline_from_Lévis.jpg` | 3.220 s |
| Amsterdam | Q727 | `Amsterdam_airphoto.jpg` | 1.434 s |

| What is guaranteed | Test name | Kind | Result |
| --- | --- | --- | --- |
| All 59 planned OpenWeather identities and the original ten resolve to downloadable photos within four seconds in the successful live run | `TestDestinationPhotoLive` | Opt-in live integration | Pass on retry; initial Kyoto provider error recorded above |

## Interim app check: Amsterdam

`DB_PATH=/private/tmp/traveltab-photo-fallback.db WRITERS_MAP=0 WRITERS_SYNC=0 make run` started the app with a temporary copy of `weatherservice.db`. `curl -sS -L -o /private/tmp/photo-amsterdam.html -w 'status=%{http_code} time=%{time_total} bytes=%{size_download}\n' 'http://127.0.0.1:8080/?city_name=Amsterdam&country_code=NL&place_lat=52.3676&place_lon=4.9041'` returned `status=200 time=2.090521 bytes=20098`. The response contained `data-photo-state="ready"`, `alt="View of Amsterdam"`, `Amsterdam_airphoto.jpg` and an attributed Commons credit. These are approximate city-centre coordinates; the original OpenWeather coordinates remain unavailable. The server was stopped after the check.

## D5: app pages and final gate

The app ran with a temporary database copy using `DB_PATH=/private/tmp/traveltab-photo-fallback-final.db WRITERS_MAP=0 WRITERS_SYNC=0 make run`. Each local page request used the plan's coordinates and the OpenWeather name that the destination page receives. For example, the Quebec request was `curl -sS -L --max-time 20 --get --data-urlencode 'city_name=Quebec' --data-urlencode 'country_code=CA' --data-urlencode 'place_lat=46.814' --data-urlencode 'place_lon=-71.208' -o /private/tmp/photo-quebec.html -w 'quebec status=%{http_code} time=%{time_total} bytes=%{size_download}\n' http://127.0.0.1:8080/`. The HTML was checked for `data-photo-state="ready"`, the city image alt text, its Commons file URL and a `Photo:` credit.

| Search | Page name | Successful response | Commons file | Credit |
| --- | --- | --- | --- | --- |
| New York | New York | `status=200 time=3.759044` | `NYC_Downtown_Manhattan_Skyline_seen_from_Paulus_Hook_2019-12-20_IMG_7347_FRD_(cropped).jpg` | Fred Hsu on en.wikipedia |
| Washington | Washington | `status=200 time=2.841081` | `Aerial_view_of_White_House_and_downtown,_Washington,_D.C_LCCN2010630891.jpg` | Carol M. Highsmith |
| Quebec City | Quebec | `status=200 time=3.695181` | `Quebec_City_skyline_from_Lévis.jpg` | Quintin Soloviev |
| Toronto | Old Toronto | `status=200 time=3.065102` | `Sunset_Toronto_Skyline_Panorama_Crop_from_Snake_Island.jpg` | Jchmrt |
| Amsterdam | Amsterdam | `status=200 time=3.114527` | `Amsterdam_airphoto.jpg` | Amsterdam Municipal Department for the Preservation and Restoration of Historic Buildings and Sites (bMA) |

Quebec initially returned HTTP 200 with `data-photo-state="unavailable"` because individual Wikidata, Wikipedia or Commons requests exceeded the existing four-second photo deadline. The logs recorded `context deadline exceeded`; these provider errors were retried, and the final page rendered the correct photo and credit. The successful live adapter run had returned Quebec in 3.220 s. This is a remaining upstream latency limit, not a deterministic selection failure.

`python3 /private/tmp/compare_destination_photo_files.py` compared the D1 table to the successful D4 log and printed `baseline photos 55 post-change photos in baseline 55`, with no `MISSING` or `CHANGED` lines. Thus all 55 D1 photo filenames remained identical; the four D1 no-photo cities gained photos.

`go test -cover ./internal/adapters/api/ -count=1` returned `coverage: 84.3% of statements`. `PATH=/tmp/traveltab-lint-bin:$PATH make test lint build` passed: the race-enabled Go tests were `ok` in every tested package, `golangci-lint run ./...` reported `0 issues.`, and `go build -o bin/traveltab ./cmd/web` succeeded.

| What is guaranteed | Test or check | Kind | Result |
| --- | --- | --- | --- |
| The five requested destinations can render the selected city photo, alt text and Commons credit | Local app page requests above | Runtime integration | Pass; Quebec required retries after upstream timeouts |
| Every D1 photo keeps its Commons filename | D1 table compared with successful D4 live log | Live regression comparison | 55/55 identical |
| Changed API package coverage is at least 80% | `go test -cover ./internal/adapters/api/ -count=1` | Offline coverage | 84.3% |
| Repository tests, lint and build succeed | `PATH=/tmp/traveltab-lint-bin:$PATH make test lint build` | Full gate | Pass |

## D6: run the three searches at once

Source: `tasks/destination-photo-fallback-todo.md` D6. Journey: *As a traveller searching New York, I want its photo on the first visit, so that a slow Wikimedia response doesn't leave the hero empty.*

**Problem.** The app logged `Error looking up a photo of New York, US: commons image information: … context deadline exceeded`. Timing each request of three live New York lookups showed 7 sequential requests (name search, entities, nearby articles, populated-place search, entities, country code, Commons image info), totalling 6.315 s, 3.291 s and 3.676 s against the 4 s limit.

**Change.** `Photo` starts the three searches concurrently (`searchAll`), loads all their candidates in one `wbgetentities` request with labels and aliases (`loadCandidates`), then checks the searches in the old order. A search's error is returned only when no earlier search matched.

| Behaviour | Command | RED | GREEN |
| --- | --- | --- | --- |
| The three searches are in flight together | `go test ./internal/adapters/api/ -run 'TestDestinationPhoto_RunsTheThreeSearchesAtOnce$' -count=1` | `--- FAIL … (1.00s)` `Received unexpected error` (each search waits for the other two; one after another, the first gives up after 1 s) | `ok` |
| One entity request for all candidates | `go test ./internal/adapters/api/ -run 'TestDestinationPhoto_LoadsEveryCandidateInOneEntityRequest$' -count=1` | `expected: 1` `actual  : 2` | `ok` |

Guard tests, which passed before and after because they protect the unchanged order: `TestDestinationPhoto_PrefersTheNameSearchOverTheFallbacks` (replaces `SkipsTheNearbyFallbackWhenTheSearchMatched`, whose "never sends the other searches" assertion no longer holds by design), `TestDestinationPhoto_FindsAnEntityTheNameSearchMissed` (now also offers a better-linked populated place, and the nearby article still wins), `TestDestinationPhoto_IgnoresFallbackFailuresWhenTheNameSearchMatched`.

`go test -race ./internal/adapters/api/ -run 'DestinationPhoto|HTMLText|SafeURL' -count=1` → `ok weatherservice/internal/adapters/api 1.956s`.

**Live timing after the change** (temporary timing test, removed): New York 2.981 s, 2.491 s, 2.549 s; Quebec 2.624 s; Washington 2.861 s; Paris 2.036 s. The three searches start at 0 ms; the remaining sequence is entities (0.5–1.1 s with download), country code (0.3–0.8 s), Commons image info (0.2–0.45 s).

**Live suite.** `LIVE_API_TESTS=1 go test ./internal/adapters/api/ -run TestDestinationPhotoLive -count=1 -v` → 69 identities, median 2.21 s, p90 2.71 s, 5 over 3 s, **2 failures**:
- `Montreal, CA: error after 4.001s: wikidata country Q16: … context deadline exceeded`
- `Vatican City, VA: error after 4.001s: wikidata country Q237: … context deadline exceeded`

Both ran out of time on the country-code request (`wbgetclaims … property=P297`), which each fresh `DestinationPhotoAPI` makes once per country.

| What is guaranteed | Test name | Kind | Result |
| --- | --- | --- | --- |
| The three searches run concurrently | `TestDestinationPhoto_RunsTheThreeSearchesAtOnce` | Adapter integration with HTTP fake | Pass |
| All candidates load in one entity request | `TestDestinationPhoto_LoadsEveryCandidateInOneEntityRequest` | Adapter integration with HTTP fake | Pass |
| Name search beats nearby articles and populated places | `TestDestinationPhoto_PrefersTheNameSearchOverTheFallbacks` | Adapter integration with HTTP fake | Pass |
| Nearby article beats populated places | `TestDestinationPhoto_FindsAnEntityTheNameSearchMissed` | Adapter integration with HTTP fake | Pass |
| Fallback failures don't hide a name-search match | `TestDestinationPhoto_IgnoresFallbackFailuresWhenTheNameSearchMatched` | Adapter integration with HTTP fake | Pass |
| Live lookups under 4 s | `TestDestinationPhotoLive` | Live | 67/69; Montreal and Vatican City timed out |

Coverage: `go test -cover ./internal/adapters/api/ -count=1` → `coverage: 84.6% of statements`. Full gate: `PATH=/tmp/traveltab-lint-bin:$PATH make test lint build` → tests `ok`, `golangci-lint run ./...` `0 issues.`, build succeeded.

**Known gap.** The country-code request is still a separate step after the entities load. A built-in table of country QIDs to ISO codes (about 200 entries) would remove it, and with it the cause of both live failures.
