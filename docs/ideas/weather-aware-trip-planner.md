# Weather-aware trip planner

_Idea one-pager, written 2026-09-19 and updated the same day with the validation results. Replaces the old itinerary (Foursquare) and hotels (Google Places) features._

## TL;DR

- **What:** enter a city, a start date and a number of days. You get a **day-by-day plan that follows the forecast** (museums on the rainy day, viewpoints on the sunny one) and **the best area to stay for that plan**.
- **Cost:** $0, forever. Only open data, with no API keys that can bill.
- **Why it's worth building:** TravelTab is a weather app, and no other trip planner plans around the forecast. That's the portfolio story.
- **Status:** all 5 key assumptions were tested against the real APIs. They hold, with changes: see [What the results changed](#what-the-results-changed). **MVP implementation is complete**; local tests, Docker build and mobile browser checks pass. See [completion evidence](../../tasks/notes-completion.md). Human release review and deployment remain.
- **Next (v2):** shareable trip links, calendar and map export, and city intros written ahead of time by a local AI. See [Next Iteration (v2)](#next-iteration-v2).

## Problem Statement

How might we help someone who has already decided on a trip go from "I'm going to Lisbon" to "I know what to do each day and where to stay", using only data that is free forever?

## Recommended Direction

**Build the plan from famous places, split it into days by area, and let the weather choose which day gets what.**

1. **Places:** Wikipedia and Wikidata give the landmarks near the city. They're ranked by how many Wikipedia languages cover each place, which is a strong "this is famous" signal, and filtered by a list of place types so cities, stadiums and historical events drop out. That fixes the old problem of random results. Photos come from Wikimedia Commons.
2. **Days:** nearby places are grouped into N days, with 3–4 stops per day, and each day is ordered as a walking loop. This is plain Go with no API.
3. **Weather:** the Open-Meteo daily forecast decides the order. Days with 5 mm of rain or more get mostly indoor stops, and drier days get mostly outdoor ones. Only the next 7 days are rearranged, because rain forecasts aren't reliable further out.
4. **Stay:** find the center of the plan and show hotels, hostels and guest houses within 1 km from OpenStreetMap. Each gets a "Check prices" link to a Booking search with your dates filled in.

Hotels become part of the plan instead of a separate list. We don't show prices; we help you choose an area, then hand you off to book.

## Directions considered

| Direction | User value | Effort | What makes it different | Verdict |
|---|---|---|---|---|
| **A. Weather-aware planner** (this doc) | Solves a real planning problem | Medium: the hard parts are the place-type list and grouping places into days | High: nobody plans around the forecast | ✅ **Build this** |
| B. Curated city guide (Wikivoyage + AI intros written in advance) | Inspiration, but these users have already decided to go | Medium: parsing wiki markup is fiddly | Low: it would be a worse Wikivoyage | ❌ As the main feature. The AI intros come back as a v2 add-on |
| C. Top 10 must-sees only | Quick to read, but not a plan | Low: about a weekend | Low: every travel site has this | ➖ Becomes **step 1** of A |

## Free data we use

| What | Source | Key? | Limits | We must |
|---|---|---|---|---|
| Places + ranking | Wikipedia geosearch + Wikidata entity API (**not** the SPARQL query service: too slow) | No | 2–15 s and up to ~8 MB per big city; at most 500 articles per search | Split capped searches; send a `User-Agent`; cache per city for 30 days |
| Photos | Wikimedia Commons | No | — | Credit each photo (licenses vary per image) |
| Forecast | Open-Meteo | No | 10k calls/day, **non-commercial only**, **16 days ahead max** | Credit "Open-Meteo" (CC BY 4.0) |
| Hotels | OpenStreetMap via the Overpass API: `overpass-api.de`, then `overpass.private.coffee` as a fallback | No | ~10k requests/day; **the public servers say they aren't meant as an app backend**, and half our test requests failed before a retry | Retry, fall back, keep old cached hotels, hide the stay card as a last resort; credit "© OpenStreetMap contributors" |
| Booking hand-off | A plain Booking.com search URL | No | — | Nothing. It's just a link |

## Key Assumptions (validated 2026-09-19)

Tested against the real APIs for Lisbon, Porto, Tavira, Funchal and Kyoto. Details are in [the validation results](weather-aware-trip-planner-validation.md).

- [x] ✅ **Ranking by Wikipedia coverage gives good places, but only with a place-type filter.** Without the filter, just 2–5 of the top 15 are places to visit. With it: Lisbon 13/15, Porto 13/15, Funchal 13/15, Kyoto 15/15. Tavira has only 12 places, and 9 are good.
- [x] ✅ **We can tell indoor from outdoor.** Every kept place gets a label, and about 96% are right. ⚠️ Some cities are lopsided: Kyoto's top 15 has only 1 indoor place.
- [x] ⚠️ **The forecast is useful, but only for about 7 days.** Most people decide on activities within days of the trip. Rain forecasts are reliable for days 1–7, only fair for days 8–14, and beyond that no better than average weather for the season.
- [x] ⚠️ **The public Overpass server has plenty of capacity, but it's unreliable.** 5 of 10 test requests failed with 504 or 429 before a retry worked.
- [x] ✅ **OpenStreetMap has enough hotels near the plan center.** Every city has at least 5 with a website (Tavira has exactly 5). ⚠️ Only 6–19% have stars.

### What the results changed

- **Where places come from:** Wikipedia geosearch plus the Wikidata entity API, not Wikidata's SPARQL query service. SPARQL took 24–100 s per query, with frequent errors.
- **A hand-made list of place types** decides what counts as a place to visit and whether it's indoor or outdoor.
- **Rainy days** pick their indoor stops from the top ~60 places, not the top 20.
- **The weather only rearranges the next 7 days.**
- **Overpass requests get retries and a fallback server.** If both servers fail, the old cached hotels are shown. The stay card is hidden only when there's no cache at all.
- **Hotels aren't sorted by stars**, because few have them. Unnamed ones are skipped.

## MVP Scope

**In:**

- [x] One **"Plan my trip"** card on the main page, right under the weather, with a start date and **number of days (1–5)**. **No category checkboxes**, because they cluttered the old form.
- [x] Places within 10 km from Wikipedia geosearch, ranked by how many Wikipedia languages cover them. Keep the top ~60, each with a photo and an indoor/outdoor label.
- [x] **Big cities:** if the 10 km search returns the maximum of 500 results, run **7 smaller searches** (a 5 km circle in the middle and 6 around it, which together cover the 10 km circle) and merge them. Only Lisbon needed this in the validation.
- [x] **A hand-made list of place types** in Go, mapping each Wikidata type to indoor, outdoor or skip. Build it once, offline, from the types found in about 20 cities. **No class-tree lookups at runtime.** Rules learned from the validation:
  - Keep a bridge only if it's also a monument.
  - Skip libraries and a short list of exact types (city or ward of Japan, capital of Japan, former capital, government building).
  - Include mountains, islands and bookstores; add markets and lifts next.
  - Never skip broad categories like "administrative area". In Wikidata, that removes Belém Tower and nature parks too.
- [x] **A small hand-made boost list** in Go for popular places that fame alone ranks too low, or that the type list misses. Each entry is a Wikidata ID plus indoor/outdoor. A boosted place skips the type filter and ranks as if it had as many Wikipedia languages as the city's 10th place. Keep it to about 5 per city. Starting list:

  | Place | Wikidata ID | City | Indoor or outdoor | Why it needs a boost |
  |---|---|---|---|---|
  | Lisbon Oceanarium | `Q652806` | Lisbon | indoor | Ranks #27 |
  | Santa Justa Lift | `Q168001` | Lisbon | outdoor | "Elevator" isn't in the type list |
  | Mercado dos Lavradores | `Q2063403` | Funchal | indoor | "Market" isn't in the type list |
  | Arashiyama Bamboo Grove | `Q23579173` | Kyoto | outdoor | "Bamboo grove" isn't in the type list |
  | Nishiki Market | `Q11650434` | Kyoto | indoor | Ranks #93. It's also a covered street, so good for Kyoto's rainy days |

- [x] Grouping into days: aiming for 3–4 stops per day (2 when needed to stay compact), ordered as a walking loop, with the walking distance shown. Sunny days draw from the top ~20; **rainy days pick indoor stops from the top ~60**, because some cities (like Kyoto) have few indoor places near the top.
- [x] **Small towns:** if there aren't enough good places for the days asked (3 per day), plan fewer days and say so, for example "Tavira has enough highlights for 3 days, so here's a 3-day plan." (In the validation, Tavira had 12 places, 9 of them good.)
- [x] Weather rule: if the day's total rain (Open-Meteo `precipitation_sum`) is **≥ 5 mm**, make it an indoor day; otherwise an outdoor day. Keep the 5 mm in one constant so it's easy to adjust. **Only rearrange days within the next 7 days**, where the forecast is reliable. Days 8–16 show the forecast marked "less certain" but keep their order. Further out, show "Forecast appears closer to your trip". Treat missing (`null`) forecast values as "no data".
- [x] Stay card: up to 6 places to stay within 1 km of the plan center (the place in the plan with the shortest total distance to the others). Show the website, the stars when there are any, and a "Check prices" link with the dates filled in. Skip places with no name, and use the English name when there is one.
- [x] **Overpass fallback chain:** try `overpass-api.de`, then `overpass.private.coffee`, each with a short timeout and one retry. If both fail, **keep showing the old cached hotels** (hotels rarely change). With no cache at all, show the plan without the stay card: "Places to stay are unavailable right now". Keep the server list in an environment variable (`OVERPASS_URLS`) so it can change without a code change.
- [x] SQLite cache: places and hotels kept 30 days per city, the forecast kept 3 hours. **If a refresh fails, keep serving the old data** instead of showing nothing.
- [x] A credits line in the footer for OpenStreetMap, Wikidata/Commons and Open-Meteo.
- [x] Tests with stubbed APIs pass locally, and CI is configured to run them; the next remote run follows the human-managed commit/PR.
- [x] Remove the Google Places and Foursquare code and keys.

**Done when** (verified with recorded city data, cache tests and a separate mobile browser fixture; see completion evidence):

- [x] "Lisbon, 3 days, next week" gives a sensible plan, and the rainy day really does get the museums.
- [x] "Tavira, 5 days" gives a shorter plan with a clear message instead of padding with weak places.
- [x] "Kyoto, 3 days" with a rainy day still gets 3+ indoor stops that day.
- [x] A cached city loads in under 2 seconds.
- [x] The planner uses only keyless sources; no billable provider is wired into it. This is an implementation check, not a billing-account audit.
- [x] The README has a short "how the planner works" section with a diagram, ready for a portfolio.

### Build order

Each step can be tested before the next one starts. **Broken down into 17 tasks for parallel agents in [tasks/plan.md](../../tasks/plan.md) and [tasks/todo.md](../../tasks/todo.md).**

1. [x] **`cmd/placetypes`:** a small Go tool that fetches the places near ~20 cities, collects their Wikidata types, and writes a draft Go map of each type to indoor, outdoor or skip. You review it by hand once. It takes over from the throwaway Python used for the validation.
2. [x] **API clients with stubbed tests:** Wikipedia geosearch (including the 7-search split) and Wikidata entities, the Open-Meteo daily forecast, and Overpass hotels with retries and the fallback server. Recorded client fixtures and Lisbon, Tavira and Kyoto acceptance fixtures are checked in the working tree.
3. [x] **The planner:** rank, filter, group into days, apply the weather rules, and pick the place to stay. Plain Go with no network calls, so the unit tests run against the fixtures.
4. [x] **The UI:** the "Plan my trip" card and the stay card.
5. [x] **Clean up:** remove the Google Places and Foursquare code.

## Not Doing (and Why)

- **Hotel prices, availability or reviews.** Only booking companies have that data, and it costs money. We hand off with a link.
- **AI in the MVP.** Simple rules can build the plan. Running AI on Fly needs a bigger paid machine. City intros written ahead of time on a laptop come in v2.
- **Category checkboxes.** They made the old form cluttered. Fame and the weather decide instead.
- **Restaurants.** OpenStreetMap has too many, with no ranking, which would bring back the "bad results" problem.
- **Rearranging days more than 7 days out.** The forecast isn't reliable that far ahead. Typical weather for those dates (averages from past years) can come later.
- **Real walking routes.** Straight-line distance is good enough for now. The free public routing server isn't meant for production use.
- **Parsing Wikivoyage listings.** Coverage is uneven and the markup is a project of its own. It's only the fallback if Overpass keeps failing even with retries and caching. (The v2 intros only need the page's plain text.)
- **Accounts or saved trips.** Too much for the MVP. Shareable links and export come in v2 and cover most of this.
- **Running our own Overpass server.** Only worth it if traffic grows, or if both public servers keep failing.
- **VK Maps' Overpass server.** It's listed as public, but in testing it was slow (12–16 s for a tiny query) and failed the hotel query.
- **Wikidata's SPARQL query service.** It was too slow and unreliable in testing. The entity API gives the same data.
- **Sorting hotels by stars.** Only 6–19% have them. Sort by type and whether they have a website instead.

## Decisions (2026-09-19)

- ✅ **The old hotels card was costing money.** The Google Places key has been removed from Fly's secrets. Until the code is removed (in the MVP scope above), the card's call to Google Places just fails and the card shows nothing.
- ✅ **Maximum trip length: 5 days.**
- ✅ **The planner goes under the weather, on the same page.**
- ✅ **A rainy day is based on the amount of rain in mm, not on the chance of rain.** The starting cut-off is 5 mm in the day. (Weather services count 1 mm as a "rain day", but that's only a drizzle.)
- ✅ **Small towns get fewer days**, not a wider search area.
- ✅ **Big cities get several smaller searches** when the 10 km search is capped. In Lisbon, the single search reached only 9.0 km. The 7-search split found 15 more results in the last kilometer (none were places to visit), for 7 extra requests (~9 s), and only when capped.
- ✅ **A hand-made boost list** for popular places that rank too low (see MVP Scope).
- ✅ **Overpass falls back to a second public server.** ⚠️ In testing, `overpass.private.coffee` was fully down (even its status page timed out), so the fallback chain ends with the old cached hotels, then hiding the card.

## Next Iteration (v2)

Three add-ons once the MVP works. **Build them in this order**, because each one uses the one before:

1. **Shareable links** give every plan a stable URL.
2. **Export** turns that same URL into a calendar file or a map file.
3. **AI city intros** go at the top of the trip page.

| Add-on | User value | Effort | Main risk |
|---|---|---|---|
| Shareable links | High: send the plan to whoever you're traveling with | Low | The plan has to come out the same every time |
| Export | High: the plan ends up in your calendar or map | Low | Time zones |
| AI city intros | Medium: nice to read, but the plan works without it | Medium | The model making things up |

### 1. Shareable trip links

`/trip/lisbon-pt?days=3&from=2026-10-02` always opens the same plan.

- [ ] Add a `GET /trip/:city` route. It renders the normal page, with the weather on top and the plan below, with the planner already filled in. Submitting the planner puts this URL in the address bar (HTMX `hx-push-url`), so copying the address shares the plan.
- [ ] Make the plan deterministic: the same input always gives the same stops on the same days. No randomness when grouping; break ties with the Wikidata ID.
- [ ] Search basics: a page title and description ("3 days in Lisbon: …"), a canonical URL **without the date**, and a `sitemap.xml` that lists only cities already in the cache.

**Watch out:**

- **The weather changes, so the plan can't be fully frozen.** The stops and how they're grouped into days stay the same. Only the order of the days follows the latest forecast, with a note saying "Days reordered for the latest forecast". A friend opening your link next week gets fresher weather, which is a feature, not a bug.
- **Search engine crawlers** opening cities that aren't cached would hit Wikipedia, Wikidata and Overpass. Add a simple limit on how many uncached cities can be fetched per minute.

### 2. Export the trip

Three buttons under the plan:

- [ ] **Add to calendar (`.ics`):** one event per stop at default times (for example 10:00, 12:00, 15:00 and 17:00, 1.5 hours each), with the place's location and a link back to the trip. Only shown when the trip has a start date.
- [ ] **Google My Maps (`.kml`):** one folder per day and one pin per stop. You import it at mymaps.google.com.
- [ ] **Bonus: "Open Day 1 in Google Maps":** a plain Google Maps link with walking directions through that day's stops. It needs no key, costs nothing, and is the most useful of the three on a phone. With 3–4 stops a day, it stays within Google's mobile limit of 3 stops between the start and the end.

The files are served at `/trip/lisbon-pt.ics?…` and `/trip/lisbon-pt.kml?…`: the same URL as the page, with a different extension.

**Watch out:**

- **Time zones.** Get the city's time zone from Open-Meteo (`timezone=auto`) and write event times in UTC. Import Go's `time/tzdata` so the slim Docker image doesn't need time zone files.
- **`.ics` files have a few formatting rules** (escape commas and semicolons, wrap long lines). Test the file in Google Calendar, Apple Calendar and Outlook.

### 3. City intros, written ahead of time by a local AI

A short intro at the top of the trip page: why go, what each area is like, and how to get around. It's written on your laptop and costs nothing to serve.

- [ ] Turn the empty `cmd/console` into `cmd/guides`. For each city in the list, it fetches the Wikivoyage page text and asks a small model in Ollama (7–8B is plenty) for a ~120-word intro **using only that text**.
- [ ] **Check for made-up places:** reject and retry any intro that names a place that isn't in the Wikivoyage text.
- [ ] Save the intros to `guides/guides.json` in the repo, embed the file in the app with `go:embed`, and load it at startup.
- [ ] Store the Wikivoyage revision for each city, so a rerun only rewrites cities whose page changed.
- [ ] Show "Based on Wikivoyage (CC BY-SA)" with a link under each intro. The license requires it.

**Why a JSON file in the repo instead of SQLite:** the database lives on the Fly volume, not on your laptop, so there's no simple way to copy rows into it. A file in git ships with every deploy, and **you can read every intro the AI wrote in the diff before it goes live.**

**Watch out:**

- **Write a general city intro, not "3 days in Lisbon".** The plan can be 1–5 days long, so a fixed "3 days" text would contradict it.
- **Which 100 cities?** `/stats` only returns the top 10, and it only started counting recently. Start with a hand-picked list in `guides/cities.txt`, then add your most-searched cities over time.

### v2 assumptions to validate

- [ ] **The plan comes out the same every time.** Test: build "Lisbon, 3 days" twice, a day apart and after a cache refresh, and compare the stops.
- [ ] **The calendar file works everywhere.** Test: import it into Google Calendar, Apple Calendar and Outlook, and check that the times are right.
- [ ] **A small local model sticks to the source text.** Test: generate 10 cities and read them all. Allow at most 1 wrong fact across the 10.
- [ ] **Search engines index the trip pages.** Test: submit the sitemap in Google Search Console (free) and check back after 2–4 weeks.

### v2 not doing

- **Writing intros on demand for any city.** That needs AI on the server, which means a paid machine.
- **AI text for each trip.** Intros are per city, not per plan.
- **Accounts or saved trips.** The link is the save button.
- **Short links (`/t/abc123`).** They'd need a database row for every trip, and readable URLs are enough.
- **Search engines indexing dated URLs.** Only the version without a date is canonical, which avoids thousands of near-duplicate pages.

### v2 open questions

- [ ] City in the URL: `lisbon` or `lisbon-pt`? There's more than one Lisbon.
- [ ] Default event times: are 10:00, 12:00, 15:00 and 17:00 sensible, or should days start later?
- [ ] Which cities go in the first list of 100?
- [ ] Will you read every AI intro before it ships, or spot-check some?
