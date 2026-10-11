# Wikidata descriptions: hero line fallback (7i gap) and stop description (7g gap)

Source: follow-up to `tasks/todo.md` 7g and 7i (owner-approved). English Wikidata descriptions only; no extra network calls.

**Journeys**
- As a traveller, I want a line under the city name even when there is no curated tagline, so I know what the place is.
- As a traveller, I want a short line under each stop name, so I can tell what it is without opening it.
- As a traveller, I want no empty line or blank space when there is no description.
- As a traveller, I want saved writers' places to show their description in the plan too.

## Behaviours

| Behaviour | Command |
|---|---|
| Stop detail request adds `descriptions`; `Place.Description` set (English) | `go test ./internal/adapters/api/ -run PlacesNear` |
| City entity request adds `descriptions` (still one `wbgetentities` call); photo carries it | `go test ./internal/adapters/api/ -run DestinationPhoto` |
| Places and photo caches round-trip the field; old rows without it still load | `go test ./internal/adapters/sqlite/ -run 'Description|BeforeDescriptions'` |
| Hero line: tagline wins, else description, else no element; escaped | `go test ./internal/adapters/httpserver/ -run 'HeroLine|DescriptionFills|NoTaglineAnd|CuratedTagline'` |
| Stop row: description element only when set; escaped; saved places map it | `go test ./internal/adapters/httpserver/ -run 'Itinerary|SeedsTheIncluded'` |
| CSS clamps both to 2 lines | `... -run 'DescriptionStyle|HeroLine_Style'` |
| Preview fixture: some stops described (one long), one not; Paris, Porto described, Paris TX not | `go test ./cmd/design-preview/ -run 'FixturePlaces_Most|FixturePhotos_Describe'` |

**RED** (tests first):
```
--- FAIL: TestWikimedia_PlacesNear_CarriesTheEnglishWikidataDescription
    expected: "castle in Tavira, Portugal"  actual: ""
    "claims|labels" does not contain "descriptions"
--- FAIL: TestDestinationPhoto_CarriesTheWikidataDescriptionOfTheCity   (expected: "capital and largest city of France", actual: "")
--- FAIL: TestDestinationPhoto_DescriptionComesFromTheEntityRequestWithoutExtraCalls  ("claims|sitelinks" does not contain "descriptions")
--- FAIL: TestItinerary_ShowsAShortDescriptionUnderTheStopNameWhenItHasOne   (expected 1, actual 0)
--- FAIL: TestPlanRoute_SeedsTheIncludedPlacesIntoTheRequest   (Description:"Saved description" vs "")
--- FAIL: TestItinerary_DescriptionStyleClampsLongTextToTwoLines
--- FAIL: TestFixturePlaces_MostStopsHaveADescriptionAndOneDoesNot   (0 of 7 places described)
--- FAIL: TestFixturePhotos_DescribeParisAndPortoButNotParisTexas
httpserver hero tests: build failed, presentationView.HeroLine / Description undefined (the tests reference the missing code)
```
**Cache tests passed on first run** (the field is a plain JSON field). To check they can fail, `Description` was temporarily tagged `json:"-"`: `TestSourceCache_PlaceDescriptionSurvivesTheCache` failed; tag removed.

**GREEN**: all of the above pass. `make test build` ok. `go vet ./...` and `gofmt -l internal cmd` clean (`make lint` not run: golangci-lint go1.26/go1.27.2 mismatch).

## Guarantees

| Guarantee | Test | Level | Result |
|---|---|---|---|
| Stop description from Wikidata `en`, same request, `languages=en` | `TestWikimedia_PlacesNear_CarriesTheEnglishWikidataDescription` | unit (recorded fixture) | pass |
| Place without one has none | `TestWikimedia_PlacesNear_PlaceWithoutADescriptionHasNone` | unit | pass |
| City description trimmed, on the photo | `TestDestinationPhoto_CarriesTheWikidataDescriptionOfTheCity` | unit (fake Wikimedia) | pass |
| No extra call: one `wbgetentities`, props has `descriptions` | `TestDestinationPhoto_DescriptionComesFromTheEntityRequestWithoutExtraCalls` | unit | pass |
| No description gives empty | `TestDestinationPhoto_CityWithoutADescriptionHasNone` | unit | pass |
| Places cache keeps it; old rows load with none | `TestSourceCache_PlaceDescriptionSurvivesTheCache`, `..._RowsCachedBeforeDescriptionsStillLoad` | integration (SQLite) | pass |
| Photo cache keeps it | `TestDestinationPhotoCache_DescriptionSurvivesTheCache` | integration | pass |
| Tagline > description > nothing | `TestPresentationHeroLine_TaglineWinsOverTheWikidataDescription`, `TestDestinationPhoto_CuratedTaglineBeatsTheDescription` | unit | pass |
| Page and HTMX fragment show it, escaped | `TestDestinationPhoto_DescriptionFillsTheHeroLineWhenThereIsNoTagline` | integration (real templates) | pass |
| No tagline, no description: no `destination-tagline` element | `TestDestinationPhoto_NoTaglineAndNoDescriptionRenderNoHeroLine` | integration | pass |
| Stop description rendered only when set, escaped | `TestItinerary_ShowsAShortDescription...`, `..._StopsWithoutADescriptionRenderNoDescriptionElement` | integration | pass |
| Saved places carry `Description` into the plan request | `TestPlanRoute_SeedsTheIncludedPlacesIntoTheRequest` | integration | pass |
| 2-line clamp in CSS | `TestItinerary_DescriptionStyleClampsLongTextToTwoLines`, `TestHeroLine_StyleClampsLongTextToTwoLines` | unit | pass |
| Fixture shows both, keeps one stop without | `TestFixturePlaces_...`, `TestFixturePhotos_...` | unit | pass |

## Coverage (`go test -cover`)
api 82.6%, planner 96.4%, httpserver 92.5%, application 100%, sqlite 81.7%, cmd/design-preview 81.6%.

## Runtime
`go build` preview (`-scenario default`), isolated headless Brave, 375 and 1440 px, Porto / Lisbon / Nophoto and a Lisbon plan:
- Porto shows "city in northern Portugal" (Wikidata-style fixture description), Lisbon keeps its tagline, Nophoto has no `.destination-tagline` element and no empty one.
- Plan stops: 8 of 10 with a description, 2 without (no element). The long one is cut at 2 lines with an ellipsis (checked by height and `scrollHeight > clientHeight`). No horizontal overflow, no element wider than its card. Console and page errors: none.
- Screenshots checked: Porto hero at 1440, Lisbon hero at 1440 and 375, stops at 375 and 1440.
- At 375 px the hero line is hidden by the existing rule `.destination-tagline { display: none }` (max-width 640px); I left that alone, so the Wikidata description also does not show on small phones.
- Re-run, PASS at 375 and 1440: `pr7-acceptance.browser.mjs`, `pr7-integration.browser.mjs`.

## Known gaps
- The city description rides on the photo lookup. A city with no usable photo (the lookup returns nil) shows no description either. Fixing that means returning a photo-less result, which the SQLite photo cache treats as incomplete (`Complete()`), so it was left out.
- A curated hero without a tagline skips the lookup (`startPhotoLookup`), so it gets no description. Only Lisbon is curated and it has a tagline.
- Photo cache entries written before this change have no description until they expire (30 days).
- The recorded fixture `testdata/wikimedia/entities_tavira.json` was edited by hand to add two English descriptions (Castle of Tavira, Igreja da Misericórdia), in Wikidata's response shape. It was not re-recorded from the network.
- Writers' map `Place.Description` is mapped as is; this change does not check that it is English.
