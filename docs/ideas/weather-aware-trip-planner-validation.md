# Validation: weather-aware trip planner

_Run on 2026-09-19 against the real free APIs for Lisbon, Porto, Tavira, Funchal and Kyoto. Goes with [weather-aware-trip-planner.md](weather-aware-trip-planner.md)._

## TL;DR

| # | Assumption | Result |
|---|---|---|
| 1 | Ranking by Wikipedia coverage gives good places | ✅ **Only with a place-type filter.** Without it, just 2–5 of the top 15 are places to visit. With the filter, 13–15 of 15 are. |
| 2 | We can tell indoor from outdoor | ✅ **Every kept place gets a label, and about 96% are right.** But some cities are lopsided (Kyoto has 1 indoor place in its top 15). |
| 3 | The forecast is useful within its 16-day window | ⚠️ **Yes, but trust only the first 7 days.** |
| 4 | Light, cached use of the public Overpass server is fine | ⚠️ **There's plenty of capacity, but it's unreliable.** 5 of 10 attempts failed before a retry worked. |
| 5 | OpenStreetMap has 5+ hotels with a website near the plan center | ✅ **All 5 cities.** Tavira only just made it (exactly 5). |

**Biggest surprise:** Wikidata's SPARQL query service is too slow and unreliable to use while a visitor waits. Use Wikipedia's geosearch plus Wikidata's entity API instead.

## 1. Ranking by Wikipedia coverage

**Raw ranking fails.** The top 15 is mostly countries, cities, airports, stadiums, universities and historical events (Eurovision 2018, the Kyoto Animation arson attack). Only 2–5 of 15 are places to visit.

**A place-type filter fixes it.** We keep only places whose Wikidata type is a museum, church, park, square, castle, viewpoint, beach and so on. After one round of fixes:

| City | Good places in the top 15 | The misses |
|---|---|---|
| Lisbon | **13 / 15** ✅ | Belém Palace (the president's residence), Vasco da Gama Tower |
| Porto | **13 / 15** ✅ | Ponte D. Maria Pia (a disused bridge), Portus Cale (an ancient settlement with nothing to see) |
| Funchal | **13 / 15** ✅ | Madeira Natural Park (two-thirds of the island), Church of St Martin |
| Kyoto | **15 / 15** ✅ | — |
| Tavira | **9 of only 12 places** | Torre D'Ares and two forts that are ruins or private |

"Good" was judged by hand, so give the lists below a quick look yourself.

**Fixes that got it there:**

- **Keep a bridge only if it's also a monument.** Road bridges rank very high but aren't stops. Luiz I Bridge stays, because it's also a monument.
- **Drop libraries.** The National Library of Portugal ranked 11th in Lisbon.
- **Add mountains, islands and bookstores.** These bring in Pico do Areeiro, Mount Hiei, Tavira Island and Livraria Lello.
- **Exclude a short list of exact types**: city of Japan, ward of Japan, capital of Japan, former capital, and government building. These removed Kyoto's wards and São Bento Palace (the parliament).
- **Don't exclude broad categories** like "administrative area". Wikidata's class tree is messy: a nature park reaches "administrative area" through "national park", and Belém Tower does through "military area". A broad exclusion removed Belém Tower, Cabo Girão, Ria Formosa and Gion.

**Still missing** (their Wikidata types aren't in the list yet): the Santa Justa Lift, Mercado dos Lavradores and the Arashiyama bamboo grove. **Markets and lifts** are the most useful types to add next.

**Ranked too low:** the Oceanário is #27 in Lisbon and Nishiki Market is #93 in Kyoto. Both are popular but are covered in few Wikipedia languages. Ranking by fame alone underrates some popular spots, especially food markets.

<details>
<summary>Final top 15 per city</summary>

- **Lisbon:** Belém Tower, Jerónimos Monastery, Alfama, Padrão dos Descobrimentos, Lisbon Cathedral, Praça do Comércio, Castle of Saint George, Baixa, National Museum of Ancient Art, Gulbenkian Museum, Santa Engrácia, ~~Belém Palace~~, ~~Vasco da Gama Tower~~, Palace of Ajuda, Bairro Alto
- **Porto:** Luiz I Bridge, Casa da Música, Porto Cathedral, ~~Ponte D. Maria Pia~~, Palácio da Bolsa, São Bento station, Livraria Lello, São Francisco, Santo Ildefonso, Clérigos Church, Santa Clara, ~~Portus Cale~~, Soares dos Reis Museum, Serra do Pilar, Kadoorie Synagogue
- **Tavira:** Ria Formosa, Tavira Island, Santa Maria church, Fortress of Cacela, Misericórdia church, Castle of Tavira, ~~Torre D'Ares~~, Santiago church, Praia do Barril, ~~Fort of Santo António~~, ~~Fort of São João da Barra~~, Carmo church
- **Funchal:** Pico do Areeiro, Cabo Girão, Cathedral, Botanical Garden, CR7 Museum, ~~Madeira Natural Park~~, Pico das Torres, Cristo Rei, Monte church, São Tiago Fort, Sacred Art Museum, Colégio church, ~~Church of St Martin~~, Santa Clara convent, Natural History Museum
- **Kyoto:** Kiyomizu-dera, Kinkaku-ji, Fushimi Inari, Mount Hiei, Arashiyama, Imperial Palace, Ginkaku-ji, Gion, Ryōan-ji, Enryaku-ji, Tō-ji, Nijō Castle, Heian Jingū, Kamigamo Shrine, Fushimi Castle

</details>

## 2. Indoor or outdoor

- **Every kept place gets a label**, because the type filter and the indoor/outdoor labels come from the same list.
- **About 96% of labels are right** (3 wrong out of 72 checked). São Bento station is labeled outdoor, but its famous tiled hall is indoors. Kyoto Imperial Palace is labeled indoor, but you mostly walk its grounds. Nijō Castle is really both.
- ⚠️ **Some cities are lopsided.** Kyoto's top 15 has **1 indoor place**, and Porto's has **4 outdoor places**. A rainy day in Kyoto needs indoor stops from further down the list (the Kyoto National Museum is #17).

**Change to the plan:** rainy days pick their indoor stops from the top ~60 places, not the top 20.

## 3. Is the forecast useful?

- **People decide late.** In 2025, 60% of travelers booked tours and attraction tickets at least 3 days ahead, but only 17–20% booked a month or more ahead. 35% book most or all of their activities after they arrive. Museum tickets and tours are booked closest to the day. ([Arival](https://arival.travel/article/why-travelers-book-some-experiences-last-minute-and-others-in-advance/), [PhocusWire](https://www.phocuswire.com/Arival-summer-tours-and-activities))
- **Rain forecasts are good for about a week.** They're reliable for days 1–7, only fair for days 8–14, and beyond two weeks no better than average weather for the season. ([coloradoriverscience.org](https://coloradoriverscience.org/Weather_and_climate_forecasts), [IRI](https://iridl.ldeo.columbia.edu/maproom/Global/ForecastsS2S/s2sskill.html))
- **Open-Meteo works as expected.** It returned 16 days of rain in **mm** plus the city's time zone (`Europe/Lisbon`, `Atlantic/Madeira`, `Asia/Tokyo`) in 0.3 s. One day came back as `null`, so the code has to handle missing values.

**Change to the plan:** only rearrange days within the next **7 days**. Show days 8–16 as "less certain".

## 4. The public Overpass server

- **Capacity: fine.** One query per city returns 4–102 KB. With 30-day caching, 10,000 requests a day is far more than TravelTab needs.
- **Reliability: poor.** Across 5 queries we got **four "504 Gateway Timeout" errors and one "429 Too Many Requests"** before each one succeeded. Successful queries took 0.5–10 s.
- `/stats` can't show "new cities per day": it needs the token and only returns the top 10 cities. The capacity numbers above make that check unnecessary.

**Change to the plan:** retry with a backoff, cache for 30 days, and **if Overpass fails, show the plan without the stay card** ("Places to stay are unavailable right now"). Later extended with a second server and the old cached hotels: see [A second Overpass server](#a-second-overpass-server).

## 5. Hotels near the plan center

| City | Places to stay within 1 km | With a website | With stars |
|---|---|---|---|
| Lisbon | 206 | 125 | 35 |
| Porto | 283 | 94 | 37 |
| Tavira | 12 | **5** | 1 |
| Funchal | 54 | 19 | 10 |
| Kyoto | 200 | 72 | 13 |

- ✅ At least 5 with a website everywhere.
- ⚠️ **Only 6–19% have stars**, so stars can't be the main way to sort them.
- Some have **no name**, or a name only in Japanese. Skip unnamed places and prefer `name:en` when it exists.

The "plan center" here is the top-15 place with the shortest total distance to all the others. It always lands on a real place instead of, say, the middle of a river.

## Which data sources are fast enough

| Source | Result |
|---|---|
| Wikidata SPARQL query service | ❌ Cold queries took **24–100 s**, with 429, 502 and 504 errors. The subclass query **timed out after 120 s** repeatedly. Repeating the same query was fast (2–5 s) only because the server had cached it. |
| Wikipedia geosearch + Wikidata entity API | ✅ **2–15 s per city with zero errors.** It downloads up to ~8 MB for a big city, but only once every 30 days. Caveat: it returns at most 500 English Wikipedia articles, and Lisbon hit that cap. |
| Walking the place-type tree with the entity API | ✅ 471 types in 40 requests and 52 s. Do this once, offline, and save the result as a list in the code. |
| Open-Meteo | ✅ 5 of 5 answered in 0.3 s. |
| Overpass | ⚠️ See section 4. |

## Follow-up checks

These were run later the same day, to back the three decisions that answered the open questions.

### Several smaller searches for big cities

One 10 km search, compared with 7 searches of 5 km (one in the middle and 6 around it, which together cover the 10 km circle). Only results inside the original 10 km circle are counted.

| City | One search | How far it reached | 7 smaller searches | New results | New places to visit |
|---|---|---|---|---|---|
| Lisbon | **500 (capped)** | **9.0 km** | 515 | 15 | **None** (a road bridge, two tennis tournaments, parishes) |
| Kyoto | 428 | 10 km | 428 | 0 | — |
| Porto | 218 | 9.8 km | 218 | 0 | — |

The cap is real but costs little today. Splitting only when a search is capped is cheap insurance: 7 extra requests, about 9 s, once per city every 30 days.

### Boost list

These Wikidata IDs start the boost list. Checked against the cached data:

| Place | ID | Wikipedia languages | Wikidata types |
|---|---|---|---|
| Santa Justa Lift | `Q168001` | 26 | elevator, cultural heritage |
| Lisbon Oceanarium | `Q652806` | 17 | public aquarium, cultural heritage |
| Arashiyama Bamboo Grove | `Q23579173` | 21 | bamboo grove |
| Mercado dos Lavradores | `Q2063403` | 7 | market, built structure |
| Nishiki Market | `Q11650434` | 7 | food market, shopping arcade in Japan |

### A second Overpass server

Public servers with no key, from the [OpenStreetMap wiki list](https://wiki.openstreetmap.org/wiki/Overpass_API):

| Server | Status page | Tiny query | Tavira hotel query |
|---|---|---|---|
| `overpass-api.de` (main) | ✅ 0.5 s | ✅ 1.2 s | ✅ 2.7 s |
| `overpass.private.coffee` | ❌ timed out (30 s) | ❌ timed out | ❌ timed out (60 s), and all 5 cities failed too |
| VK Maps (`maps.mail.ru`) | ⚠️ 12.3 s | ⚠️ 16.2 s | ❌ 504 after 45 s |

The other servers on the list need a paid key or cover only one region.

**Takeaway:** a second public server helps only when it's up, and on this day it wasn't. The plan's fallback chain is: `overpass-api.de` → `overpass.private.coffee` → old cached hotels → hide the stay card.

## How to repeat this

The throwaway Python scripts used for this aren't in the repo. To repeat it: for each city, fetch the places within 10 km, rank them by sitelinks, apply the type list, then check hotels within 1 km of the plan center and the 16-day forecast.
