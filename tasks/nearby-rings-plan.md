# Ring-by-ring nearby loading: pins appear as they arrive

**Status:** plan only, not started. Owner decisions recorded 2026-10-12 (see **Decisions**). Tasks: [nearby-rings-todo.md](nearby-rings-todo.md). **Replaces** [nearby-query-plan.md](nearby-query-plan.md).

**Goal:** a cold destination shows its first notable pins quickly and fills in as more arrive, in every city (Lisbon, Kyoto, London, Paris, New York, Tokyo, Funchal). No single request is big or slow, so none hits the 30 s client timeout or Wikidata's ~60 s server limit.

## Why the old plan failed

Measured 2026-10-12 (see `docs/tdd/nearby-query.tdd.md`, N1, N1b, N1c):

- **One query is too heavy.** Time follows the item count (10–25 ms per item), driven by the per-language label joins. London, Paris, New York and Tokyo return 12–24 MB even at 2 sitelinks.
- **Radius or N alone can't fix it.** The one setting that met 5 s / 5 MB dropped Funchal pins and far Madeira places.
- **Grouping was slower** than plain rows (Lisbon, N=2: 29.5 s vs 15.6 s). Drop the grouped query.

## Fix: many small queries, nearest first

Load the 40 km area as **rings of 5 km** (0–5, 5–10, … 35–40), nearest first, in **pages of 100 items**. Each page is a small query, so each finishes in seconds.

1. **Order inside a ring:** by sitelinks, highest first, so each page is the most notable places left. Pages use `ORDER BY DESC(?sitelinks), ?item` plus `LIMIT 100 OFFSET n`, so the order is stable.
2. **Rings:** a page asks for items within the outer radius and farther than the inner radius (ring 1 has no inner bound).
3. **Stop early:** end a ring when a page has fewer than 100 items or none with at least the sitelinks floor. Stop the whole load when the **200-pin cap is filled with notable items**, or when a ring comes back thin. Never run Paris out to 1,000 pages.
4. **Sitelinks floor:** 1 or 2, set in R1 from measurement. Must keep Câmara de Lobos (2 sitelinks), so **at most 2**.
5. **Progress is visible:** after each page the server stores a **partial result**. The map polls and adds pins as they arrive, with a "still loading" status until the last ring finishes.
6. **Cache only the finished result.** A half-loaded city is never stored as complete. Failed or interrupted loads resume from the last stored page on the next visit.
7. **Writer matches** run **once at the end**. While loading, pins show without writer counts.
8. **Runs in the background**, detached from the request, so closing the page doesn't abort it. Pages go **one at a time**, never in parallel, to respect Wikidata's rate limits (reuse the 429/503 retry).

**Contract update.** `docs/writers-map-contract.md` says `Gazetteer` "keeps the full nearby list" from one query. After this fix it keeps the nearby items **with at least the floor sitelinks**, loaded in rings. Update that line in R5.

**Not changing:** the 40 km radius, the 200-pin cap, the 30 s per-request client timeout (each page is small now), `addPhotoCredits`, the retry on 429/503, the cache `v3` bump (keep it).

## Decisions (owner, 2026-10-12)

| Decision | Choice |
|---|---|
| Loading shape | **Rings of 5 km, 100 items per page, nearest first, to 40 km.** |
| Page order | **Sitelinks, highest first.** |
| Stop rule | **Stop when the 200 pins are filled with notable items or a ring comes back thin**, not at the full 40 km every time. |
| Sitelinks floor | **1 or 2** (measured in R1); must keep Câmara de Lobos. |
| Progress | **Partial results stored per page; the map polls and shows pins as they arrive.** |
| Cache | **Only the finished result is cached as complete; `v3` stays.** |
| Writer matches | **Computed once, at the end.** |
| Old query-time targets | **Dropped.** The target is now per page and for the first pins. |

**If a page still takes over ~10 s** at 100 items in a dense ring (the label joins are the cost), drop the page size to 50 or move labels to a second small query for those 100 QIDs. This is pre-approved; record it in the evidence doc. Stop and ask only if neither works.

## Dense-city mode (owner, 2026-10-13)

| Decision | Choice |
|---|---|
| Dense city | **More than 1,500 items with 2+ sitelinks within 5 km** of the centre (measured: Funchal 144, Tokyo 690, Kyoto 766, Lisbon 797 stay normal; New York 2,509, London 4,752, Paris 9,783 are dense). |
| Dense loading | **Eight 1 km rings, hard end at 8 km.** Writer matches beyond 8 km are dropped (accepted). |
| Everything else | 5 km rings to the radius, as built. |
| When | **Once, after the first page**, so first pins are not delayed; the mode is stored with the partial progress. A dense city restarts its ring cursor at 0-1 km and keeps the places already loaded. |
| Count | Capped at 1,501 (`LIMIT` inside the count), so it stays cheap. |

## Default map view (owner, 2026-10-13)

Open centred on the searched destination at about a **200 m scale bar**, keep pitch 55 and bearing -20, and **do not fit the camera to the pins** (pins outside the view are reached by panning; `reset()` returns to the same view). The requested zoom 16.7 reads **30 m** on the real scale bar, so the value was measured instead: **MapLibre zoom 14.3** reads "200 m" (about 83 px) in London at 375 and 1440 px; the Leaflet fallback opens at zoom 15.

## Targets

- **First pins** visible within **5 s** of a cold request in all seven cities.
- **Each page:** under **5 s** and under **1 MB**.
- **Whole load** finishes within the stop rule; no request over 30 s.
- Funchal and Madeira keep their writer matches (6a cases São Jorge, Monte, Ponta do Sol, Teleférico do Funchal).

## Done when

- Each city meets the targets above in R6.
- The 6a writer-pin cases still pass on the finished result.
- The contract line matches the new behaviour.
- Unit tests use recorded fixtures only (no live calls in `make test`); `make test lint build` passes.
- Evidence in `docs/tdd/nearby-rings.tdd.md`, including before/after timings.
- **Then:** the 6a precision review can run on the new Madeira gazetteer.
