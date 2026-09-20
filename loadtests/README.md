# TravelTab load tests

Start with one smoke run. The fixture server uses TravelTab's actual handlers,
templates, cache, and visit tracking with fake API reporters and a temporary SQLite
database. It never calls weather, hotel, video, or itinerary providers.

Run both commands from the repository root, in separate terminals:

```sh
go run ./cmd/loadtest-server
```

```sh
k6 run loadtests/traveltab.js
```

Prerequisites: the project's Go toolchain and [k6](https://grafana.com/docs/k6/latest/set-up/install-k6/).
Stop the server with Ctrl-C after testing; its temporary database is removed.
The server prints the temporary path so it can be removed manually after a forced kill.

Once smoke passes, run the next profile:

```sh
PROFILE=load k6 run loadtests/traveltab.js
PROFILE=stress k6 run loadtests/traveltab.js
```

| Profile | Workload | Duration excluding setup/graceful stop |
| --- | --- | --- |
| smoke | 1 user, 3 journeys | Usually about 3 seconds |
| load | Ramp to 10 users, hold, ramp down | 3 minutes |
| stress | Step through 10, 25, 50 users, then ramp down | 5 minutes |

Each journey loads the home page, searches a city using HTMX, then submits an
itinerary. Checks verify status and rendered content. Requests time out after 5
seconds. Initial targets are fewer than 1% request failures, more than 99% passing
checks, and p95 below 500ms for home/search and 1000ms for itinerary. A request
failure rate of 1% or more aborts after the first 10 seconds; latency failures
produce a nonzero exit at completion. These are starting targets to review, not
an established production SLO.

Results are written to `loadtests/results/<profile>-summary.json` (overwritten on
the next run). Keep a comparison with
`SUMMARY_PATH=loadtests/results/baseline.json`. To inspect
latency as load rises, retain samples:

```sh
PROFILE=stress k6 run --out json=loadtests/results/stress-samples.json loadtests/traveltab.js
```

The fixture has three preseeded cities and no provider latency by default. It
measures warm cache throughput, template rendering, session work, and SQLite
visit writes. It does not measure cold cache traffic, real provider limits, browser
rendering, external assets, or production capacity. Server logs matter: visit
write failures are logged by the app but do not become HTTP errors. Watch CPU,
memory, and those logs alongside k6's latency and error metrics. A simulated
reporter delay can model waiting without API traffic:

```sh
go run ./cmd/loadtest-server -reporter-delay 50ms
```

For an environment you control, override the URL explicitly:

```sh
BASE_URL=https://your-staging-host.example PROFILE=smoke k6 run loadtests/traveltab.js
```

A normal deployment uses real API providers and records test visits. Confirm its
capacity and provider budget before progressing to load or stress. Restart the
fixture between comparisons to reset visit data. Record the profile, machine,
delay, latency, failure rate, and **one next bottleneck to fix**.

See [the recorded validation](validation.md) for the completed smoke and stress
runs, measured latency, visit-persistence checks and fixture limitations.

Implementation references: [ramping VUs](https://grafana.com/docs/k6/latest/using-k6/scenarios/executors/ramping-vus/),
[thresholds](https://grafana.com/docs/k6/latest/using-k6/thresholds/), and
[custom summaries](https://grafana.com/docs/k6/latest/results-output/end-of-test/custom-summary/).
