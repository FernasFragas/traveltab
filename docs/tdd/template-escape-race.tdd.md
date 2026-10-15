# Template escaping data race (CI)

Source: owner-pasted CI log, 2026-10-15. `make test` failed with `WARNING: DATA RACE` in
`TestDestinationPhoto_RapidConcurrentSearchesKeepEachPhotoWithItsCity` (`testing.go:1867: race detected during execution of test`).

**Journey**
- *As a traveller, I want the site to serve correct pages when many people search at once right after a deploy, so that no request reads a half-written template.*

## Cause
- `html/template` escapes a template on its **first** `Execute` and writes its escape state then (`escape.go:43-44`).
- The gofiber engine (`template/html/v2` v2.1.3, the latest release) copies the template with `*tmpl` **without a lock** (`html.go:147`).
- Two concurrent first renders of the same page race. Only the CI's scheduling hit it; 30 local `-race` runs were clean.
- It's a production race too: the first requests after a restart.

## Fix
`internal/adapters/httpserver/server.go`: `escapeTemplates` runs every loaded template once into `io.Discard` right
after `fiber.New` loads the views. Later renders find the templates already escaped and only read them.

## Behaviour
`go test ./internal/adapters/httpserver/ -run TestViews_`

RED (fix stashed): html/template still allows `Clone`, so nothing is escaped yet
```
Messages:   	index is not escaped yet
Messages:   	content_fragment is not escaped yet
Messages:   	place_panel is not escaped yet
Messages:   	plan_response is not escaped yet
Messages:   	search_suggestions is not escaped yet
FAIL	weatherservice/internal/adapters/httpserver	0.381s
```
GREEN: `ok  	weatherservice/internal/adapters/httpserver	0.332s`

## Guarantees

| What is guaranteed | Test | Level | Result |
| --- | --- | --- | --- |
| Every rendered page is escaped before the first request | `TestViews_TemplatesAreEscapedBeforeTheFirstRequest` | Integration | PASS |
| Concurrent first searches render without a race | `TestDestinationPhoto_RapidConcurrentSearchesKeepEachPhotoWithItsCity` with the above, `-race -count=50` | Integration | PASS (no `DATA RACE`) |

## Suite and coverage
- `make test lint build` (CI-pinned golangci-lint): every package `ok`, `0 issues.`
- `go test -cover ./internal/adapters/httpserver/`: 93.0%.

## Known gaps
- The race never reproduced locally, so the RED is the deterministic escape-state test, not the race itself.
- The nil-templates guard (views failed to load) isn't covered in this package; `cmd/design-preview` tests build the server where `./views` is missing and go through it.
