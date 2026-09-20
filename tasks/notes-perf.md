# Cold-start performance notes

The agent that wrote the concurrency was cut off by a session limit before it measured or filed
notes. The code below is its work; the measurements and the caveat were added afterwards by the
coordinating session.

## What changed

- **`planner/build.go`** — the forecast lookup now runs in a goroutine while places are fetched on
  the main path. The channel is buffered, so the goroutine never blocks even when `Build` returns
  early, and the derived context is cancelled on the way out. The stay lookup still runs last,
  because it needs the plan's centre. The error rules are unchanged: places failing is fatal, a
  failing forecast plans without weather, failing stays set `StaysNote`.
- **`api/wikimedia.go`** — `inParallel` runs the batch lookups over at most `wikimediaConcurrency`
  (4) workers. Each worker writes its own slots in the results and errors slices, so order stays
  deterministic no matter which HTTP response arrives first.
- **`planner/build.go`** — a failing forecast now also sets a note: "Weather is unavailable right
  now, so the days are not ordered by the forecast." Covered by
  `TestBuild_SaysWeatherIsUnavailableWhenTheForecastFails`.

## Measured against the live APIs (Lisbon, 515 places)

Same code, only `wikimediaConcurrency` changed, two runs each:

| Workers | Run 1 | Run 2 |
|---|---|---|
| 1 (as before) | 18.4 s | 18.6 s |
| 4 (now) | 7.8 s | 6.3 s |

**About 2.6× faster** for the dominant part of a cold plan.

## Caveat worth knowing

The first live attempt **failed**: one Wikidata `wbgetentities` request passed the client's 30 s
timeout with 4 requests in flight (`context deadline exceeded`). Repeat runs were fine, so it
looks transient rather than throttling — but it is the failure mode to watch if cold plans start
erroring. Two straightforward answers if it recurs: retry a timed-out batch once, or drop
`wikimediaConcurrency` to 2.

## What still dominates a cold plan

A cold plan measured end to end earlier took ~57 s, and roughly 60 s of that run was **Overpass
timing out twice** (`overpass-api.de` returned 504, then `overpass.private.coffee` timed out)
before the stay card gave up. Parallelism does not touch that: the fix would be a shorter
per-server timeout, since 10 s twice over is a long time to make someone wait for an optional
card.
