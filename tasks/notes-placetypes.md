# Task 13 notes: place-type tool and the generated type list

Owner files: `cmd/placetypes/main.go`, `cmd/placetypes/rules.go`, `cmd/placetypes/rules_test.go`,
`planner/placetypes_gen.go` (generated), this file.

## How it works

`rules.go` is all of the rule logic, as plain functions over a class graph. It never touches the
network:

- `ClassGraph` maps a Wikidata type ID to the types it is a subclass of (P279).
- `DefaultRules()` holds the 15 indoor roots, the 21 outdoor roots and the 7 denied types from
  the validation. Bridge (Q12280) and library (Q7075) are deliberately **not** roots.
- `Rules.KindOf(typeID, graph)` answers `Deny` for an exact denied type; otherwise it walks up
  subclass-of, **at most `MaxDepth` (10) hops** and **never twice through the same type**, and
  the roots it reaches decide the Kind: indoor only → `Indoor`, outdoor only → `Outdoor`, both →
  `Mixed`, none → not kept.
- `Rules.Classify(counts, labels, graph)` gives every counted type its Kind, and always lists the
  roots and denied types too, so the generated map carries the full rule list even if a type
  never showed up in the sample.
- `Render(types)` writes the gofmt-ed `planner/placetypes_gen.go`, sorted by Wikidata **number**
  (Q515 before Q1007870), with `// label (count)` on each line. Types with no Kind are left out.

`main.go` only does the network and the file:

1. `PlacesNear` for each of the 20 cities, counting P31 types (a place seen in two cities counts
   once).
2. Walks P279 upward level by level with `GetEntities(ids, "claims|labels")`, which gives the
   parents and the English label in one request.
3. Classify → Render → write → print the review summary.

Everything fetched is cached in `$TMPDIR/traveltab-placetypes.json` (`-cache` to move it).
A first run takes about 9 minutes and ~800 requests; **a rerun from the cache takes 1.8 s and
produces a byte-identical file.**

### The deny list matches exact types only

`KindOf` checks `r.Deny[typeID]` before walking, and the walk never carries a deny to a subclass.
That is the point of `TestRules_DenyMatchesOnlyTheExactType`: a subclass of "ward of Japan" that
is also a park comes back `Outdoor`. Section 1 of the validation is why — a broad, subclass-based
exclusion removed Belém Tower (it reaches "military area"), Cabo Girão, Ria Formosa and Gion.

## RED evidence

Tests written first against zero-value bodies (`return "", false`, `return nil`, `return nil, nil`).
All 9 tests failed, on **19 assertion failures**, none on a compile error:

```
$ CGO_ENABLED=1 go test -race -count=1 ./cmd/placetypes/
--- FAIL: TestRules_RootTypeGetsItsKind (0.00s)
    rules_test.go:67: Error: Should be true
                      Messages: museum is an indoor root
    rules_test.go:68: Error: Not equal: expected "indoor", actual ""
    rules_test.go:71: Error: Should be true
                      Messages: Buddhist temple is an outdoor root even with no entry in the graph
    rules_test.go:72: Error: Not equal: expected "outdoor", actual ""
--- FAIL: TestRules_SubclassOfARootGetsItsKind (0.00s)
    rules_test.go:80: Error: Should be true   Messages: a subclass of museum is kept
    rules_test.go:81: Error: Not equal: expected "indoor", actual ""
--- FAIL: TestRules_TypeReachingIndoorAndOutdoorIsMixed (0.00s)
    rules_test.go:89: Error: Should be true   Messages: a castle museum is kept
    rules_test.go:90: Error: Not equal: expected "mixed", actual ""
--- FAIL: TestRules_DenyMatchesOnlyTheExactType (0.00s)
    rules_test.go:98:  Error: Should be true   Messages: ward of Japan is a known type
    rules_test.go:99:  Error: Not equal: expected "deny", actual ""
    rules_test.go:102: Error: Should be true
                       Messages: a subclass of ward of Japan that is also a park is kept
    rules_test.go:103: Error: Not equal: expected "outdoor", actual ""
--- FAIL: TestRules_BridgeAloneIsNotKept (0.00s)
    rules_test.go:118: Error: Should be true
                       Messages: but a monument is, so a bridge that is also a monument still counts
    rules_test.go:119: Error: Not equal: expected "outdoor", actual ""
--- FAIL: TestRules_SurvivesACycleInTheClassGraph (0.00s)
    rules_test.go:127: Error: Should be true
                       Messages: the walk gets past the cycle and finds the park
    rules_test.go:128: Error: Not equal: expected "outdoor", actual ""
--- FAIL: TestRules_StopsAtDepthTen (0.00s)
    rules_test.go:139: Error: Should be true
                       Messages: a root exactly MaxDepth hops up is still found
    rules_test.go:140: Error: Not equal: expected "outdoor", actual ""
--- FAIL: TestRender_WritesAGofmtedMapSortedByID (0.00s)
    rules_test.go:164: Error: Received unexpected error:
                       placetypes_gen.go:1:1: expected ';', found 'EOF' (and 2 more errors)
FAIL	weatherservice/cmd/placetypes	0.317s
```

`TestRules_ClassifyAlwaysListsEveryRootAndDenyType` was written after the other eight (it covers
the "the map always holds the whole rule list" promise) and was run on the stub first as well.
It sits above `TestRender_…` in the file, so that test's line numbers moved down by 18 after the
run quoted above:

```
--- FAIL: TestRules_ClassifyAlwaysListsEveryRootAndDenyType (0.00s)
    rules_test.go:156: Error: Not equal:
        expected: main.TypeInfo{ID:"Q900001", Label:"town museum", Count:4, Kind:"indoor"}
        actual  : main.TypeInfo{ID:"", Label:"", Count:0, Kind:""}
    rules_test.go:158: Error: Not equal: expected "outdoor", actual ""
                       Messages: Buddhist temple is listed even with no count
    rules_test.go:159: Error: Not equal: expected "deny", actual ""
                       Messages: ward of Japan is listed even with no count
```

The four tests that assert something is **not** kept (bridge, a pure cycle, depth 11, a subclass
of a denied type) would pass on their own against the stub, so each of them carries a positive
control in the same test — that control is what failed at RED.

## GREEN

```
$ CGO_ENABLED=1 go test -race -count=1 -v ./cmd/placetypes/
--- PASS: TestRules_RootTypeGetsItsKind (0.00s)
--- PASS: TestRules_SubclassOfARootGetsItsKind (0.00s)
--- PASS: TestRules_TypeReachingIndoorAndOutdoorIsMixed (0.00s)
--- PASS: TestRules_DenyMatchesOnlyTheExactType (0.00s)
--- PASS: TestRules_BridgeAloneIsNotKept (0.00s)
--- PASS: TestRules_SurvivesACycleInTheClassGraph (0.00s)
--- PASS: TestRules_StopsAtDepthTen (0.00s)
--- PASS: TestRules_ClassifyAlwaysListsEveryRootAndDenyType (0.00s)
--- PASS: TestRender_WritesAGofmtedMapSortedByID (0.00s)
ok  	weatherservice/cmd/placetypes	1.413s
```

`golangci-lint run ./cmd/placetypes/...` reports 0 issues, and `gofmt -l` is empty for both
`cmd/placetypes/` and the generated file.

## The generated run (2026-09-20, live APIs)

```
5014 places carry 1621 different types
class graph: 4299 types (depth 0: 1621, 1: 1179, 2: 639, 3: 370, 4: 204,
                         5: 117, 6: 79, 7: 48, 8: 29, 9: 9, 10: 4)
1621 types seen: 412 kept, 1209 dropped (7 of them denied outright)
```

`planner/placetypes_gen.go` holds **419 entries**: 151 Indoor, 249 Outdoor, 12 Mixed, 7 Deny
(412 kept + the 7 denied). Nine of the 20 cities capped the 500-result geosearch and split into
7 smaller searches. The three hand checks asked for in the task:

| Check | Line in the file |
|---|---|
| Buddhist temple is outdoor | `"Q5393308": Outdoor, // Buddhist temple (59)` |
| Museum is indoor | `"Q33506": Indoor, // museum (187)` |
| Ward of Japan is denied | `"Q137773": Deny, // ward of Japan (13)` |

Bridge (Q12280) and library (Q7075) are **not** in the map, as intended.

### Top 30 kept types

```
    199  indoor    Q16970     church building
    187  indoor    Q33506     museum
    113  outdoor   Q174782    square
    108  indoor    Q207694    art museum
     88  outdoor   Q123705    neighborhood
     83  indoor    Q32815     mosque
     76  indoor    Q16560     palace
     74  outdoor   Q839954    archaeological site
     59  outdoor   Q5393308   Buddhist temple
     56  outdoor   Q79007     street
     55  indoor    Q24354     theatre building
     51  indoor    Q120560    minor basilica
     42  indoor    Q2651004   Palazzo
     42  outdoor   Q4989906   monument
     35  outdoor   Q22698     park
     35  outdoor   Q61089180  municipal part of the Czech Republic
     31  outdoor   Q82117     city gate
     30  outdoor   Q483453    fountain
     29  indoor    Q17431399  national museum
     29  outdoor   Q39614     cemetery
     25  outdoor   Q14752696  ancient Roman structure
     24  indoor    Q153562    opera house
     24  indoor    Q44613     monastery
     23  indoor    Q1060829   concert hall
     22  outdoor   Q261023    district of Vienna
     21  outdoor   Q12518     tower
     21  indoor    Q34627     synagogue
     21  indoor    Q3867560   Italian national museum
     20  indoor    Q108325    chapel
     20  indoor    Q124830411 Museum of the Italian Ministry of Culture
```

### Top 30 dropped types

```
    747  unknown   Q928830    metro station
    638  unknown   Q22808403  underground station
    238  unknown   Q55488     railway station
    169  unknown   Q22808404  station located on surface
    164  unknown   Q210272    cultural heritage
    120  unknown   Q11670533  elevated station
     97  unknown   Q11303     skyscraper
     85  unknown   Q1131296   freguesia of Portugal
     77  unknown   Q56046844  former freguesia of Portugal
     67  unknown   Q14562709  London Underground station
     60  unknown   Q2755753   area of London
     59  unknown   Q1147171   interchange station
     56  unknown   Q484170    commune of France
     55  unknown   Q55491     underground railway station
     46  unknown   Q1154710   association football venue
     45  unknown   Q43229     organization
     44  unknown   Q35112127  historic building
     44  unknown   Q3918      university
     41  unknown   Q19860854  destroyed building or structure
     39  unknown   Q110977120 Berlin U-Bahn station
     37  unknown   Q860861    sculpture
     36  unknown   Q20202072  terminus
     36  unknown   Q875538    public university
     33  unknown   Q124416148 underground metro station
     33  unknown   Q20871353  cadastral area in the Czech Republic
     33  unknown   Q26132862  Olympic sports discipline event
     32  unknown   Q3957      town
     31  unknown   Q163740    nonprofit organization
     31  unknown   Q41176     building
     31  unknown   Q4830453   business
```

The dropped list is exactly the junk the validation complained about: stations, football venues,
universities, organizations, events, plain buildings and administrative parishes.

## For the human review

Nothing here is a bug, but these are the judgement calls behind the numbers:

- **`Q132510` market is the market root.** Mercado dos Lavradores (Q2063403) is an instance of it.
  `Q330284` "marketplace" (the *space* a market runs in) is a separate branch and is **not** a
  subclass of market, so market halls typed that way are still missed. Worth adding as a root
  later if food markets keep coming up short. `Q28142754` food market does resolve (indoor).
- **Administrative city parts come in through the `neighborhood` root**: "municipal part of the
  Czech Republic" (35), "district of Vienna" (22), "municipal part of Prague" (18), boroughs of
  Berlin and Amsterdam, "neighborhood of Manhattan". That is what keeps Alfama, Baixa and Gion,
  which the validation wanted, and the path is `… > neighborhood`. Kyoto's wards are still cut,
  because "ward of Japan" is denied by exact type.
- **`Q1339195` station building is outdoor** (4 places) through `concourse > passageway > street`.
  This is how São Bento station stays in, and the validation already noted that its label is
  wrong (the famous part is the tiled hall indoors). Plain metro and railway stations are dropped.
- **`Q2281788` public aquarium comes out `Mixed`, not `Indoor`**, because "public aquarium" is a
  subclass of zoo, which is an outdoor root. `Schedule` treats `Mixed` as sheltered
  (`sheltered() = Indoor || Mixed`), so an aquarium is still a valid rainy-day stop. The Lisbon
  Oceanário is boosted to `Indoor` anyway.
- **`Q43501` zoo is `Mixed`** (park + science museum → museum), which reads right for a zoo.
- **Roman remains resolve through archaeology**, e.g. `Roman bridge > ancient Roman structure >
  Roman archaeological site > archaeological site`. A Roman bridge therefore stays while a road
  bridge does not, which matches the "keep a bridge only if it's also a monument" intent.
- **A few labels are odd but harmless**: "hometown Fuji" (a mountain), "kanji" (官寺, a
  state-sponsored temple), "vertical campus building" (reaches tower). One place each.

## Notes and deviations

- Added one test beyond the eight in the task list,
  `TestRules_ClassifyAlwaysListsEveryRootAndDenyType`, because `Classify` seeding the roots and
  denied types is what guarantees the required entries are in the map even in a small sample.
- The counts in the comments are **places, deduplicated across cities**, and only the top 300
  places per city carry types (`api.WikimediaAPI` only fetches details for those), so a count is
  "how many of the 5014 sampled places had this type", not a global Wikidata count.
- Sorting is by Wikidata *number*, not by string, so the file reads Q515, Q1007870, Q1549591…
- `CGO_ENABLED=1 go test -race -count=1 ./planner/` is green with the real map in place. That now
  includes Task 16's seven `TestCities_*` acceptance tests, which were red on missing fixtures
  while that lane was still recording them: Lisbon keeps Belém Tower and Jerónimos, skips the
  bridges and the National Library, its rainy day is mostly indoor, and Kyoto has no wards.

---

## Follow-up (2026-09-20): the "marketplace" branch, `Q330284`

The review note above ("worth adding as a root later") was picked up as follow-up work. It holds
up, and the root is now in.

### What the check found

`wbgetentities` on the five types, live:

| Type | Label / description | P279 |
|---|---|---|
| `Q132510` | market — *regular gathering of people for the purchase and sale of goods* | `Q37654` market (the economic concept), `Q15275719` recurring event |
| `Q330284` | marketplace — *space in which a market operates* | `Q39659371` retail environment, `Q13226383` facility, `Q2221906` geographic location |
| `Q28142754` | food market | `Q132510` |
| `Q2080521` | market hall — *covered space traditionally used as a marketplace* | `Q240854` hall, **`Q330284`**, `Q18760388` retail building |
| `Q13033698` | market square — *an open area where market stalls are set out* | `Q174782` square, **`Q330284`** |

So the two branches really never meet: marketplace is a *space* under retail environment, market
is an *event* under the economic concept. Nothing under marketplace can be reached from the
market root, at any depth.

`Q28142754` food market needs no root of its own — it is a direct subclass of `Q132510` and was
already `Indoor` in the generated file (line `"Q28142754": Indoor, // food market (1)`).

**The places it was costing us.** Of the 5014 sampled places, three were dropped outright and are
now kept:

| Place | City | Types | Was |
|---|---|---|---|
| **La Boqueria** (`Q1334899`) | Barcelona | `Q2080521` only | dropped |
| **Mercato Centrale** (`Q3854843`) | Florence | `Q2080521` only | dropped |
| **Loggia del Mercato Nuovo** (`Q917483`) | Florence | `Q330284`, `Q466544` loggia | dropped |

Two more already came in another way and only gained a type: the Grand Bazaar (`Q505954`) is also
`Q219760` bazaar, which is a subclass of market, and La Merced (`Q3377015`) is also `Q132510`.
Mercado dos Lavradores (`Q2063403`) and Nishiki Market (`Q11650434`) were never at risk — both
resolve through the market root *and* sit in `planner.Boosts`.

### Why it does not drag junk in

The walk only ever goes **up**, so a root can only pull in its own subclasses. `Q330284` has
exactly two in the whole 4299-type class graph: market hall and market square. The retail junk —
shopping centre, shopping mall, department store, shop, restaurant, café, pharmacy — hangs off
`Q39659371` retail environment, which is marketplace's **parent**, one hop the wrong way. Adding
*that* as a root would be a disaster (50 descendants in the graph, including `Q11315` shopping
centre and `Q11707` restaurant); adding marketplace is not.

### The change

`Q330284` is now an indoor root in `rules.go`, with the reasoning as a comment. Test first:
`TestRules_MarketplaceIsAnIndoorRootOfItsOwn`, with the real Wikidata parents added to
`fakeGraph()` so the test exercises the actual branch shape. RED was **5 assertion failures**, no
compile error:

```
$ CGO_ENABLED=1 go test -race -count=1 -run TestRules_Marketplace ./cmd/placetypes/
--- FAIL: TestRules_MarketplaceIsAnIndoorRootOfItsOwn (0.00s)
    rules_test.go:161: Error: Should be true
                      Messages: marketplace is a root, because it hangs off retail environment, not off market
    rules_test.go:162: Error: Not equal: expected "indoor", actual ""
    rules_test.go:165: Error: Should be true
                      Messages: a market hall reaches a root only through marketplace
    rules_test.go:166: Error: Not equal: expected "indoor", actual ""
                      Messages: a covered market is a rainy-day stop
    rules_test.go:170: Error: Not equal: expected "mixed", actual "outdoor"
                      Messages: it is both a square and a marketplace
```

The fourth assertion (food market is `Indoor` through the market root) passed at RED on purpose —
it is the control that says the second root is not needed.

### The regenerated file

The rerun took **1.3 s and zero requests**: `Q330284` and its ancestors were already in the cached
class graph, because two sampled places carried the type.

```
1621 types seen: 414 kept, 1207 dropped (7 of them denied outright)   (was 412 / 1209)
```

`planner/placetypes_gen.go`: **419 → 421 entries**. 151 → **153** Indoor, 249 → **248** Outdoor,
12 → **13** Mixed, 7 Deny unchanged. The whole diff is three lines:

```diff
+	"Q330284":    Indoor,  // marketplace (2)
+	"Q2080521":   Indoor,  // market hall (3)
-	"Q13033698":  Outdoor, // market square (1)
+	"Q13033698":  Mixed,   // market square (1)
```

### The one thing to know about

**`Q13033698` market square changed kind, Outdoor → Mixed**, because it is a subclass of both
square and marketplace. `Schedule` treats `Mixed` as sheltered, so a market square can now be
picked as a rainy-day stop. One sampled place has this type: the **Naschmarkt** in Vienna
(`Q417645`), which is roofed stalls along an open street — defensible either way, and Vienna is
not in the `TestCities_*` set. The alternative was to deny `Q13033698`, which would drop the
Naschmarkt altogether; that is worse. Flagging it rather than working around it.

### Verification

```
CGO_ENABLED=1 go test -race -count=1 ./planner/ ./cmd/placetypes/   ok, ok
  all seven TestCities_* still PASS, unchanged
golangci-lint run ./cmd/placetypes/...                              0 issues
gofmt -l cmd/placetypes/ planner/placetypes_gen.go                  empty
go build ./... && go vet ./planner/ ./cmd/placetypes/               ok
```
