# Look v3 Lisbon preview fixture

Source: `tasks/look-v3-todo.md` V1 and `tasks/look-v3-plan.md` fixture-data risk.

Journeys:

- As a traveler, I want varied Lisbon pins and complete place details, so I can preview the map and place panel without remote photo requests.
- As a traveler, I want a Lisbon day from Belém in the morning to Alfama in the evening, so I can preview the day teaser.

## Rich local destination and ordered day

Command run before implementation and again after implementation:

```sh
go test ./cmd/design-preview/ -run 'TestWriterMapPreview_(RichLisbon|LisbonDayPlan)'
```

RED:

```text
--- FAIL: TestWriterMapPreview_RichLisbon (0.00s)
    writersmap_test.go:25: Lisbon places = 3, want 12
--- FAIL: TestWriterMapPreview_LisbonDayPlan (0.00s)
    writersmap_test.go:62: Lisbon day stops = 4, want 3
FAIL
FAIL	weatherservice/cmd/design-preview	0.670s
```

GREEN:

```text
ok  	weatherservice/cmd/design-preview	0.581s
```

| What is guaranteed | Test name | Unit/integration | Result |
| --- | --- | --- | --- |
| Twelve Lisbon places with kinds, areas, descriptions, and locally served fixture photos; museum, cafe, viewpoint and park categories present | `TestWriterMapPreview_RichLisbon` | Unit with HTTP recorder boundary | PASS |
| Belém Tower has the specified opening hours, Wikipedia link and two writer mentions/count | `TestWriterMapPreview_RichLisbon` | Unit | PASS |
| Lisbon's first day has Belém Tower, Jerónimos Monastery and Alfama in order, with image metadata | `TestWriterMapPreview_LisbonDayPlan` | Unit | PASS |

## Coverage and limitations

Initial package coverage command:

```sh
go test -cover ./cmd/design-preview/...
```

The first invocation could not access the Go build cache within the sandbox. The escalated invocation executed the suite and reported:

```text
coverage: 86.9% of statements
FAIL	weatherservice/cmd/design-preview	1.131s
```

That invocation encountered concurrent, incomplete view changes: `html/template:map_card:35:20: no such template "overview_teasers"`. This was not a passing suite result. After the shared teaser template was added, the same package coverage command passed:

```text
ok  	weatherservice/cmd/design-preview	1.195s	coverage: 87.4% of statements
```

The final full-suite result is recorded in the coordinated look-v3 report.

The local map photos are explicitly synthetic preview images, using four distinct category palettes. They do not claim to depict the actual landmarks. Planner image fields retain Commons filenames because that is the existing planner model; local map photo URLs are available by matching the day-stop QIDs. Day slots are represented by the ordered three stops, matching the existing planner domain without adding a fixture-only time-of-day field.

The coordinated look-v3 runtime review covers the 1440 px pin screenshot and 375 px layout after the shared UI changes are complete.

## Local photos through the actual HTTP boundary

The initial raw fixture checks missed a real integration bug: production correctly removes non-Wikimedia place photo URLs. A new journey checks the actual Lisbon GeoJSON and Belém panel responses, then fetches and decodes every returned local photo as PNG.

RED command:

```sh
go test ./cmd/design-preview/ -run TestWriterMapPreview_LisbonPhotosReachBrowser
```

RED output excerpt:

```text
--- FAIL: TestWriterMapPreview_LisbonPhotosReachBrowser (0.05s)
```

The failure at `writersmap_test.go:37` reported `/map/lisbon-pt.geojson?lat=38.7223&lon=-9.1393 has no local Lisbon photo`, followed by the returned GeoJSON with no `photo_url` properties.

The fix is confined to `cmd/design-preview`: fixture data uses four exact synthetic Wikimedia marker URLs, and a preview-only reverse proxy rewrites these markers to local `/fixture-photos/` URLs in `/map/` responses. The proxy serves those images itself. Production's validator and API remain unchanged; other Wikimedia images and other responses are not rewritten.

GREEN command:

```sh
go test ./cmd/design-preview/ -run 'TestWriterMapPreview_(LisbonPhotosReachBrowser|RichLisbon)'
```

GREEN:

```text
ok  	weatherservice/cmd/design-preview	0.896s
```

| What is guaranteed | Test name | Unit/integration | Result |
| --- | --- | --- | --- |
| Lisbon GeoJSON and Belém panel contain local image URLs rather than remote fixture markers; each URL serves a valid PNG with HTTP 200 | `TestWriterMapPreview_LisbonPhotosReachBrowser` | HTTP integration with child preview process | PASS |

Final package coverage command:

```sh
go test -cover ./cmd/design-preview/...
```

```text
ok  	weatherservice/cmd/design-preview	1.269s	coverage: 81.6% of statements
```
