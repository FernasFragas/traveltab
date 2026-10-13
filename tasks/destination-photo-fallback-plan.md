# Destination photo fallbacks: New York, Washington, Quebec, Toronto, Amsterdam

**Status:** done (2026-10-13). The SPARQL coordinates query timed out (8–12 s, see `docs/tdd/destination-photo-fallback.tdd.md`), so fix 1 uses Wikidata's search API with a geo filter. The 59-city live retry passed, all 55 existing photo files stayed the same, and the five requested app pages showed city photos and credits. Tasks: [destination-photo-fallback-todo.md](destination-photo-fallback-todo.md).

**Goal:** the hero photo shows for the cities that miss it today, without ever showing a namesake's photo. The 55 photos in the D1 baseline, including Amsterdam, keep their files.

**Independent of the open work.** The original [destination photos plan](destination-photos-plan.md) is done. This work owns `internal/adapters/api/destinationphoto.go`, `destinationphoto_test.go`, the alias response field in `wikimedia.go`, new fixtures under `internal/adapters/api/testdata/wikimedia/` and `docs/tdd/destination-photo-fallback.tdd.md`. It doesn't touch the nearby-rings files (`wikidatanearby.go`).

## What fails (measured 2026-10-13)

The real lookup ran on 59 popular destinations, with names and coordinates from OpenWeather as the app gets them (full list in **Baseline cities** below). **54 OK, 5 with no photo:**

| Search | OpenWeather name | Cause |
|---|---|---|
| New York | New York | Name search finds the **state** (Q1384, ~200 km away). The city (Q60) is labelled "New York City". |
| Washington | Washington | Finds Washington **State** (3,600 km away). The city (Q61) is "Washington, D.C.". |
| Quebec City | Quebec | Finds the **province** (580 km away). The city (Q2145) is "Quebec City". |
| Toronto | Old Toronto | OpenWeather's name is a former city (Q919169), with no coordinates or settlement type. |
| Amsterdam | Amsterdam | Netherlands (Q55) has its ISO code `NL` only as a **deprecated** statement. **Fixed by fix 2.** |

**Why the existing fallback misses them:** `nearbyCities` needs an English Wikipedia title that reduces to exactly the name, among the **50 nearest** articles. In dense city centres, the city's own article isn't among the 50 nearest, and its title differs anyway.

The lookup returns "no photo" in about 1.5 s, not a timeout. "No photo" is cached for one hour (`destinationNoPhotoTTL`), so a fix applies quickly.

## Fix 1: nearby populated places from Wikidata search ✅ done

**Replaces the SPARQL design**, which took 8–12 s with no result for New York, far over the 4 s lookup limit.

After the name search and `nearbyCities` both find nothing, make **one Wikidata search API request**:

```
https://www.wikidata.org/w/api.php?action=query&list=search&format=json&srlimit=5
  &srsearch=haswbstatement:P1082 nearcoord:25km,<lat>,<lon>
```

- `haswbstatement:P1082` keeps only items with a population figure; `nearcoord:25km,…` keeps items within `DestinationMatchKM`.
- It runs on the search index, not the query service. Probed 2026-10-13: **~0.35 s**, and the right city came first in every case:

| OpenWeather name | Coordinates | First result | Time |
|---|---|---|---|
| New York | 40.713, -74.006 | Q60 (New York City) | 0.33 s |
| Washington | 38.895, -77.037 | Q61 (Washington, D.C.) | 0.34 s |
| Quebec | 46.814, -71.208 | Q2145 (Quebec City) | 0.38 s |
| Old Toronto | 43.653, -79.384 | Q172 (Toronto) | 0.37 s |

**Pick among the 5 results**, using code that already exists:
1. Load them with `matchCity` (one `GetEntities` call for labels, aliases, sitelinks, coordinates, country). The existing checks apply: settlement, within 25 km, right country.
2. **Name match first:** if any passing result's label or alias folds (`application.FoldName`) to the city name, choose among those (New York City has the alias "New York"; Washington, D.C. has "Washington"; Quebec City has "Quebec").
3. **Otherwise:** `chooseCity` over all passing results: populated first, most sitelinks, and **no photo** when two comparable results are more than 10 km apart. This covers "Old Toronto" → Toronto.

The search result order is only used to limit the candidates to 5. The choice comes from `chooseCity`, as for the other paths.

**Budget:** today's lookups take 1–2 s; this adds about 0.35 s for the search and one entity request. Each lookup must stay under `destinationPhotoTimeout` (4 s).

## Fix 2: deprecated-only country codes ✅ done

In `countryCode`: if a country has **no** preferred or normal `P297`, use its deprecated `P297`. Countries with a valid code are unchanged.

- Fixes Amsterdam (Q55 → `NL`). Live: Q727, `Amsterdam_airphoto.jpg`, 1.7 s.
- A country with **no code at all** (test case "unknown country") is still refused.

## Fix 3: regression guards ✅ done

- **Offline (`make test`):** fake-backed tests for New York (state found first, city through the fallback), Old Toronto (`chooseCity` rule) and an ambiguous result set (no photo), driven through `photoWorld`.
- **Live (`LIVE_API_TESTS=1`, not in `make test`):** add the 59 **Baseline cities** to `liveDestinations` in `destinationphoto_test.go`.

## Baseline cities (OpenWeather names and coordinates, 2026-10-13)

Coordinates are rounded to 3 decimals, as recorded. **No photo** marks the 5 misses; every other row had a photo. Hong Kong was also tried but OpenWeather's geocoding returned nothing for `Hong Kong,HK`, so it isn't in the list.

| Search | Name | Country | Lat | Lon | Result |
|---|---|---|---|---|---|
| New York,US | New York | US | 40.713 | -74.006 | **No photo** |
| Los Angeles,US | Los Angeles | US | 34.054 | -118.243 | OK |
| San Francisco,US | San Francisco | US | 37.779 | -122.420 | OK |
| Washington,US | Washington | US | 38.895 | -77.037 | **No photo** |
| Chicago,US | Chicago | US | 41.876 | -87.624 | OK |
| Miami,US | Miami | US | 25.774 | -80.194 | OK |
| Las Vegas,US | Las Vegas | US | 36.167 | -115.148 | OK |
| Mexico City,MX | Mexico City | MX | 19.433 | -99.133 | OK |
| Rio de Janeiro,BR | Rio de Janeiro | BR | -22.911 | -43.209 | OK |
| Buenos Aires,AR | Buenos Aires | AR | -34.608 | -58.437 | OK |
| Toronto,CA | Old Toronto | CA | 43.653 | -79.384 | **No photo** |
| Montreal,CA | Montreal | CA | 45.503 | -73.570 | OK |
| London,GB | London | GB | 51.507 | -0.128 | OK |
| Paris,FR | Paris | FR | 48.859 | 2.320 | OK |
| Rome,IT | Rome | IT | 41.893 | 12.483 | OK |
| Venice,IT | Venice | IT | 45.437 | 12.335 | OK |
| Florence,IT | Florence | IT | 43.770 | 11.256 | OK |
| Barcelona,ES | Barcelona | ES | 41.383 | 2.177 | OK |
| Madrid,ES | Madrid | ES | 40.417 | -3.704 | OK |
| Lisbon,PT | Lisbon | PT | 38.708 | -9.137 | OK |
| Porto,PT | Porto | PT | 41.149 | -8.611 | OK |
| Funchal,PT | Funchal | PT | 32.650 | -16.909 | OK |
| Amsterdam,NL | Amsterdam | NL | 52.373 | 4.894 | **No photo** (fixed by fix 2) |
| Berlin,DE | Berlin | DE | 52.517 | 13.389 | OK |
| Munich,DE | Munich | DE | 48.137 | 11.575 | OK |
| Frankfurt,DE | Frankfurt | DE | 50.111 | 8.682 | OK |
| Vienna,AT | Vienna | AT | 48.208 | 16.373 | OK |
| Prague,CZ | Prague | CZ | 50.087 | 14.421 | OK |
| Budapest,HU | Budapest | HU | 47.498 | 19.040 | OK |
| Athens,GR | Athens | GR | 37.984 | 23.728 | OK |
| Istanbul,TR | Istanbul | TR | 41.009 | 28.966 | OK |
| Dubrovnik,HR | Dubrovnik | HR | 42.649 | 18.095 | OK |
| Brussels,BE | Brussels | BE | 50.847 | 4.352 | OK |
| Copenhagen,DK | Copenhagen | DK | 55.687 | 12.570 | OK |
| Saint Petersburg,RU | Saint Petersburg | RU | 59.939 | 30.316 | OK |
| Edinburgh,GB | City of Edinburgh | GB | 55.953 | -3.188 | OK |
| Tokyo,JP | Tokyo | JP | 35.677 | 139.764 | OK |
| Kyoto,JP | Kyoto | JP | 35.021 | 135.756 | OK |
| Seoul,KR | Seoul | KR | 37.567 | 126.978 | OK |
| Beijing,CN | Beijing | CN | 39.906 | 116.391 | OK |
| Singapore,SG | Singapore | SG | 1.290 | 103.852 | OK |
| Bangkok,TH | Bangkok | TH | 13.752 | 100.494 | OK |
| Ho Chi Minh City,VN | Ho Chi Minh City | VN | 10.776 | 106.702 | OK |
| Hanoi,VN | Hanoi | VN | 21.029 | 105.854 | OK |
| Bali,ID | Pulau Bali | ID | -8.456 | 115.270 | OK |
| Delhi,IN | Delhi | IN | 28.652 | 77.222 | OK |
| Mumbai,IN | Mumbai | IN | 19.079 | 72.878 | OK |
| Dubai,AE | Dubai | AE | 25.265 | 55.292 | OK |
| Marrakesh,MA | Marrakesh | MA | 31.626 | -7.989 | OK |
| Cairo,EG | Cairo | EG | 30.044 | 31.236 | OK |
| Cape Town,ZA | Cape Town | ZA | -33.929 | 18.417 | OK |
| Sydney,AU | Sydney | AU | -33.870 | 151.208 | OK |
| Melbourne,AU | Melbourne | AU | -37.814 | 144.963 | OK |
| Auckland,NZ | Auckland | NZ | -36.852 | 174.763 | OK |
| Reykjavik,IS | Reykjavik | IS | 64.146 | -21.942 | OK |
| Kuala Lumpur,MY | Kuala Lumpur | MY | 3.152 | 101.694 | OK |
| Vatican City,VA | Vatican City | VA | 41.903 | 12.453 | OK |
| Monaco,MC | Monaco | MC | 43.731 | 7.420 | OK |
| Quebec City,CA | Quebec | CA | 46.814 | -71.208 | **No photo** |

The initial discovery did not keep photo file names or per-city times. D1 later recorded them for the 55 photos present after fix 2 and before fix 1.

## Risks

- **Wrong city through `chooseCity`.** It only runs when no result matches by name, and the settlement, 25 km, country and ambiguity checks still apply. D4 checks this on all 59 cities.
- **Search ranking changes.** Wikidata may reorder results. Only the top 5 are used, and the final choice comes from `chooseCity`, not the order.
- **One more Wikidata request.** If the search fails, return the provider error, as the other steps do (not cached, retried on the next visit).

## Done when

- New York, Washington, Quebec, Old Toronto and Amsterdam get a photo of the right city.
- The other 54 cities get the **same** photo as in D1.
- Every existing destination-photo test still passes, including the namesake and country guards.
- Each lookup stays under 4 s.
- `make test lint build` passes; evidence in `docs/tdd/destination-photo-fallback.tdd.md`.
