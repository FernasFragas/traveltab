# Local validation — 2026-09-19

Smoke and stress passed using k6 v2.2.0 on macOS arm64 against
`http://127.0.0.1:8081`. The k6 binary was downloaded to temporary storage from
Grafana's official GitHub release and its SHA256 matched the release checksum:
`37a028506bf13578de66c906296803775df28cc2504b1a8c5d786b2803e757c7`.

The fixture used the real TravelTab routes, templates, SQLite cache, and visit
tracking with three seeded cities, fake providers, and zero simulated provider
delay. It ran on the same machine as k6. No live application was tested.

| Measurement | Smoke | Stress |
| --- | ---: | ---: |
| Completed journeys | 3 | 7,585 |
| HTTP requests, including warmup | 13 | 22,759 |
| Passing checks | 26 / 26 | 45,518 / 45,518 |
| HTTP failures | 0 | 0 |
| Maximum concurrent users | 1 | 50 |
| Home p95 | 2.76 ms | 11.93 ms |
| Search p95 | 1.43 ms | 10.55 ms |
| Itinerary p95 | 0.77 ms | 1.63 ms |

Stress finished at **2026-09-19 21:38:43 UTC**, lasted 300.53 seconds, and averaged
75.73 requests/second across ramps and plateaus. Both k6 runs exited successfully
with every threshold passing. The separate three-minute `load` profile was not run.

The database contained exactly **15,184 expected visits** from the smoke and
stress runs, with SQLite `integrity_check` returning `ok`. Complete captured
fixture logs contained no error, locked, or failed messages. This also checked
for analytics failures that the application would otherwise hide behind HTTP 200.

Additional checks: JavaScript syntax validation passed; `go test -race ./...`
passed across all Go packages, including compilation of the fixture server.
The raw k6 summaries and stress samples remain in the ignored `loadtests/results/`
directory. See [README.md](README.md) to repeat the runs.

No saturation point was observed at the tested load. These results describe warm
cache, local fixture performance; they do not establish production capacity or
measure real providers, cold cache traffic, or browser rendering. The next
investigation is to repeat this profile with `-reporter-delay 50ms` and compare
latency and achieved throughput before testing real provider traffic.
