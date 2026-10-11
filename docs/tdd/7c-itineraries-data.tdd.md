# 7c data — Itineraries in the GeoJSON

Source: `tasks/todo.md` 7c (it was blocked because itineraries never reached the browser). The owner chose the GeoJSON option on October 11, 2026. The contract change is recorded in `docs/writers-map-contract.md` (**GeoJSON endpoint**, **Browser extension API**).

## Journeys

- As a traveler, I want writers' routes on the map, so the browser needs each itinerary with its post link and its days of stops.
- As the owner, I want internal post references kept on the server.

## RED → GREEN

| Behaviour | Command | RED | GREEN |
| --- | --- | --- | --- |
| GeoJSON carries itineraries; empty array when none; no `post_ref` | `go test ./internal/adapters/httpserver/ -run 'TestWriterMap_GeoJSON' -count=1` | `"[]" should have 2 item(s), but has 0`; `"{\"type\":\"FeatureCollection\",\"complete\":true,\"features\":[]}" does not contain "\"itineraries\":[]"` → `FAIL` | `go test ./internal/adapters/httpserver/ -count=1 -cover` → `ok ... coverage: 92.2% of statements` |

## Guarantees

| What is guaranteed | Test name | Level | Result |
| --- | --- | --- | --- |
| Each itinerary arrives with writer host, post URL, title, language and days of QIDs in order | `TestWriterMap_GeoJSONCarriesItineraries` | integration | PASS |
| The post reference is not sent to the browser | `TestWriterMap_GeoJSONCarriesItineraries` | integration | PASS |
| No itineraries → `"itineraries":[]`, never `null` | `TestWriterMap_GeoJSONHasEmptyItinerariesWhenThereAreNone` | integration | PASS |

## Known gaps

- `Itinerary` has no blog name, so 7c labels a route by post title and host. Adding the blog name would mean changing the domain type.
- The browser layer itself is 7c's work and is not built yet.
