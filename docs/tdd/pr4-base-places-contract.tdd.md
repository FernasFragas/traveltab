# PR 4 base places contract correction

Source: `tasks/todo.md` 4a–4c and the October 10 owner decision in `tasks/writers-map-results.md`.

Journey: As a writer map matcher, I want the complete nearby gazetteer alongside the capped base pins, so writer-mentioned places omitted from the base layer can still be matched.

## RED and GREEN

| Behaviour | Command | RED | GREEN |
| --- | --- | --- | --- |
| One source result keeps pins and gazetteer separate | `go test ./internal/writermap/writermaptest/ -run TestFakeBaseSource_PlacesKeepsPinsAndGazetteerSeparate` | `undefined: writermap.BasePlaces`; `result.Pins undefined`; `result.Gazetteer undefined` | `ok weatherservice/internal/writermap/writermaptest 0.537s` |

| What is guaranteed | Test name | Level | Result |
| --- | --- | --- | --- |
| The fake returns distinct pin and gazetteer lists | `TestFakeBaseSource_PlacesKeepsPinsAndGazetteerSeparate` | unit | PASS |
| Callers cannot mutate the fake's place names or types | `TestFakeBaseSource_EditingAResultLeavesTheFakeUnchanged` | unit | PASS |

Coverage command: `go test -cover ./internal/writermap/...` → `writermaptest` 92.2%; `writermap` has no statements. `go test -cover ./internal/application/...` initially returned 44.2%. Offline reporter and planner tests raised it to 100%; see [PR 4 application coverage](pr4-application-coverage.tdd.md).

No remaining contract gaps for the two-list result. The adapter and cache verification are recorded in their own PR 4 reports.
