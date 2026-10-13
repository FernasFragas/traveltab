# Writer posts by city — TDD and offline measurement

Source: [plan](../../tasks/writer-posts-city-plan.md), [spike](../../tasks/writer-posts-city-results.md), and P1–P5 in [todo](../../tasks/writer-posts-city-todo.md).

Journeys:
- As a traveller opening the Louvre in Paris, I want only Paris posts, so that I do not read a Toronto guide.
- As a traveller opening a Madeira place, I want to keep the posts I get today, so that the fix does not empty the map.

## Measurement and choice

I copied `launch-check.db` to `/private/tmp/writer-posts-city.db` and imported existing `wikidata:v3:` gazetteer cache entries from `/private/tmp/writer-context-launch.db` and `/private/tmp/paris-check.db`. The source database was not modified. Paris used the cached `paris/france` gazetteer under a `paris/fr` key because the HTTP slug requires a two-letter country. An offline baseline binary used the current matcher with the new post threshold temporarily set to 1; it wrote only to a second temporary database. The threshold was restored before building the final app. This isolates the effect of N while keeping the local-name and shared-alias rules constant.

The temporary measuring command decoded the index's packed rune positions and applied title or distinct destination-mention counts of 2, 3 and 5 to each baseline post. Its N=2 and N=3 totals matched the app's stored results exactly. The sample uses seed `20261013` and 40 baseline mention positions per destination. The earlier [writer-context](writer-context.tdd.md) samples were reviewed for judgment standards; their positions came from an earlier rule and could not form a 40-mention sample of today's baseline. Verdicts here were judged from the indexed post title and cached place identity; some ambiguous article bodies were not available in the index. Precision estimates are therefore provisional, particularly for broad lists and place aliases. Each sample row below includes a reason. `111` means all three candidate thresholds keep that mention; `110` means N=2 and 3 keep it.

| Destination | Rule | Mentions | Posts | Pins with a writer |
|---|---|---:|---:|---:|
| Paris | today | 2,452 | 321 | 52 |
| Paris | N=2 | 2,250 | 209 | 49 |
| Paris | N=3 | 2,179 | 173 | 49 |
| Paris | N=5 | 2,038 | 132 | 49 |
| Lisbon, including Lisboa | today | 9,772 | 1,593 | 262 |
| Lisbon, including Lisboa | N=2 | 8,436 | 1,032 | 256 |
| Lisbon, including Lisboa | N=3 | 7,648 | 805 | 251 |
| Lisbon, including Lisboa | N=5 | 6,709 | 604 | 240 |
| Funchal (Madeira area context) | today | 1,819 | 115 | 127 |
| Funchal (Madeira area context) | N=2 | 1,555 | 72 | 120 |
| Funchal (Madeira area context) | N=3 | 1,377 | 52 | 111 |
| Funchal (Madeira area context) | N=5 | 1,042 | 31 | 105 |
| Porto | today | 4,590 | 822 | 213 |
| Porto | N=2 | 4,387 | 696 | 210 |
| Porto | N=3 | 4,205 | 606 | 205 |
| Porto | N=5 | 3,926 | 498 | 201 |

**Selected N=3, confirmed by the owner.** None meets 90% sampled precision in all cities. N=5 loses 35.5% of judged right Funchal mentions, beyond the allowed 25%. N=3 has the highest sampled precision among candidates within the recall-loss limit in all four destinations. The remaining gaps are Paris 60.6%, Lisbon 43.8%, Funchal 83.9%, and Porto 75.7% sampled precision. The rule removes the named Toronto and Ponta Delgada guides, but broad non-Paris articles can still reach the Louvre after three Paris mentions. The plan's fallback permits shipping this measured gap.

## Test cycle and guarantees

| What is guaranteed | Test name | Level | RED → GREEN |
|---|---|---|---|
| A passing one- or two-mention Paris post is excluded; title or three mentions qualify | `TestMatchDestination_RequiresPostToBeAboutParis` | unit | RED at N=2: `should have 2 item(s), but has 3`; GREEN: `ok weatherservice/internal/writermap` |
| Lisboa is a Lisbon context and title name, including when stored area casing changes | `TestMapSource_ContextNamesIncludeDestinationAliases`, `TestMatchDestination_LisboaCountsAsLisbon` | unit | RED: `unknown field DestinationAliases`; GREEN: `ok weatherservice/internal/writermap` |
| Madeira area context alone does not make a Funchal post | `TestMatchDestination_OverrideAreaDoesNotMakePostAboutCity` | unit | GREEN: `ok weatherservice/internal/writermap` |
| Stored writer-context results are ignored under the new matcher version | `TestWriterDestination_IgnoresWriterContextMatcherResults` | integration, temp SQLite | RED: `Should be false`; GREEN: `ok weatherservice/internal/adapters/sqlite` |
| The embedded overrides file parses strictly with Lisboa | `TestOverrides_DecodeWithKnownFieldsOnly` | unit | GREEN: `ok weatherservice/writerdata` |

Commands and output excerpts:

```text
go test ./internal/writermap/ -run 'TestMapSource_ContextNamesIncludeDestinationAliases|TestMatchDestination_RequiresPostToBeAboutParis|TestMatchDestination_LisboaCountsAsLisbon'
# RED before aliases: unknown field DestinationAliases
# RED after aliases, before post rule: TestMatchDestination_RequiresPostToBeAboutParis: should have 2 item(s), but has 3
# GREEN after post rule: ok weatherservice/internal/writermap

go test ./internal/adapters/sqlite/ -run TestWriterDestination_IgnoresWriterContextMatcherResults
# RED with m2 key: TestWriterDestination_IgnoresWriterContextMatcherResults: Should be false

go test ./internal/writermap/ -run TestMatchDestination_RequiresPostToBeAboutParis
# RED at N=2: should have 2 item(s), but has 3
# GREEN at N=3: ok weatherservice/internal/writermap

go test -cover ./internal/writermap/...
# ok weatherservice/internal/writermap coverage: 94.6% of statements
# ok weatherservice/internal/writermap/writermaptest coverage: 92.2% of statements
go test -cover ./internal/adapters/sqlite/...
# ok weatherservice/internal/adapters/sqlite coverage: 83.4% of statements
go test -cover ./writerdata/...
# ok weatherservice/writerdata coverage: [no statements]
```

## Runtime and CI

With `DB_PATH=/private/tmp/writer-posts-city.db WRITERS_MAP=1 WRITERS_SYNC=0 ./bin/traveltab`, all four `/map/<destination>.geojson` requests returned HTTP 200 and stored `writermap:v3:m3:` results. The N=3 counts in the table match those stored results. `GET /map/paris-fr/Q19675?lat=48.85341&lon=2.3488` returned HTTP 200 and the Louvre Museum panel reported `View writers' posts (41)`; neither “Toronto” nor “Ponta Delgada” appeared. Existing `wikidata:v3:` keys were reused; `basePlaceCacheVersion` was not changed.

`make test lint build` inside the sandbox failed when `httptest` could not bind loopback ports. Outside the sandbox, all test packages passed; lint then stopped because the default `golangci-lint` was built with Go 1.26 while the project targets Go 1.27. The available `/private/tmp/traveltab-linter-bin/golangci-lint` is v2.14.0 built with Go 1.27.2. The final gate command was:

```text
PATH="/private/tmp/traveltab-linter-bin:$PATH" make test lint build
# all test packages: ok
# golangci-lint run ./...
# 0 issues.
# go build -o bin/traveltab ./cmd/web
```

## Judged sample sheets

| Destination | Rule | Right / kept | Precision | Right dropped / 40-sample right | Recall loss |
|---|---|---:|---:|---:|---:|
| Paris | today | 20 / 40 | 50.0% | 0 / 20 | 0.0% |
| Paris | N=2 | 20 / 35 | 57.1% | 0 / 20 | 0.0% |
| Paris | N=3 | 20 / 33 | 60.6% | 0 / 20 | 0.0% |
| Paris | N=5 | 20 / 31 | 64.5% | 0 / 20 | 0.0% |
| Lisbon | today | 14 / 40 | 35.0% | 0 / 14 | 0.0% |
| Lisbon | N=2 | 14 / 35 | 40.0% | 0 / 14 | 0.0% |
| Lisbon | N=3 | 14 / 32 | 43.8% | 0 / 14 | 0.0% |
| Lisbon | N=5 | 14 / 23 | 60.9% | 0 / 14 | 0.0% |
| Funchal | today | 31 / 40 | 77.5% | 0 / 31 | 0.0% |
| Funchal | N=2 | 29 / 35 | 82.9% | 2 / 31 | 6.5% |
| Funchal | N=3 | 26 / 31 | 83.9% | 5 / 31 | 16.1% |
| Funchal | N=5 | 20 / 25 | 80.0% | 11 / 31 | 35.5% |
| Porto | today | 28 / 40 | 70.0% | 0 / 28 | 0.0% |
| Porto | N=2 | 28 / 37 | 75.7% | 0 / 28 | 0.0% |
| Porto | N=3 | 28 / 37 | 75.7% | 0 / 28 | 0.0% |
| Porto | N=5 | 28 / 33 | 84.8% | 0 / 28 | 0.0% |

### Paris sample (seed 20261013)

| # | Indexed post | Place | N=2/3/5 | Verdict and reason |
|---:|---|---|---|---|
| 1 | [10 locais preferidos para a lua de mel](https://www.viajecomigo.com/2015/08/24/locais-lua-de-mel/) | Eiffel Tower · Q243 | 111 | wrong: a multi-city or other-city article, not a Paris guide |
| 2 | [French Atlantic Coast: Best Places to Visit on a Road Trip](https://www.saltinourhair.com/france/atlantic-coast-france/) | French · Q150 | 110 | wrong: French is a language, not a Paris stop |
| 3 | [Sabia que existe um Arco do Triunfo em Bucareste?](https://www.viajecomigo.com/2018/06/06/arco-do-triunfo-bucareste/) | Arc de Triomphe · Q64436 | 000 | wrong: a multi-city or other-city article, not a Paris guide |
| 4 | [Top 5 Architectural Wonders of IM Pei](https://theplanetd.com/top-5-architectural-wonders-of-i-m-pei/) | Louvre Museum · Q19675 | 111 | wrong: a multi-city or other-city article, not a Paris guide |
| 5 | [Top 11 das cidades mais procuradas para viagens no Outono de 2018](https://www.viajecomigo.com/2018/08/28/cidades-procuradas-viagens-outono-2018/) | Eiffel Tower · Q243 | 111 | wrong: a multi-city or other-city article, not a Paris guide |
| 6 | [11 Interesting and Fun Facts About France](https://theplanetd.com/fun-facts-about-france/) | French · Q150 | 111 | wrong: French is a language, not a Paris stop |
| 7 | [30 Free Things to Do in Paris, France (Updated 2025)](https://theplanetd.com/free-things-to-do-in-paris/) | Eiffel Tower · Q243 | 111 | right: covers Eiffel Tower in the destination area |
| 8 | [12 Things To Do in Prague in 3 Days](https://www.saltinourhair.com/czech-republic/prague/) | UNESCO · Q7809 | 000 | wrong: UNESCO is an organization, not a Paris stop |
| 9 | [3 Days in Paris: The Best Paris Itinerary for Your First Visit](https://theplanetd.com/paris-itinerary-3-days/) | Louvre - Rivoli · Q1461794 | 111 | wrong: the article does not describe Louvre–Rivoli station |
| 10 | [How to Visit Paris on a Budget – 20 Tips to Save Money](https://theplanetd.com/visit-paris-on-a-budget/) | Montmartre · Q186115 | 111 | right: covers Montmartre in the destination area |
| 11 | [7 Unforgettable Reasons to Visit Tuscany](https://theplanetd.com/visit-tuscany/) | UNESCO · Q7809 | 000 | wrong: UNESCO is an organization, not a Paris stop |
| 12 | [My Best 51 Travel Tips to Make You the World’s Savviest Traveler](https://www.nomadicmatt.com/travel-blogs/61-travel-tips/) | Louvre - Rivoli · Q1461794 | 000 | wrong: the article does not describe Louvre–Rivoli station |
| 13 | [How to Spend a Layover in Paris](https://www.adventurouskate.com/how-to-spend-a-layover-in-paris/) | Seine · Q1142326 | 111 | right: covers Seine in the destination area |
| 14 | [Paris is Wonderful in Winter](https://www.adventurouskate.com/paris-is-wonderful-in-winter/) | Centre Georges Pompidou · Q178065 | 111 | right: covers Centre Georges Pompidou in the destination area |
| 15 | [Viagem pela Roménia: programa em 13 dias](https://www.viajecomigo.com/2023/10/10/viagem-romenia-roteiro/) | Arc de Triomphe · Q64436 | 100 | wrong: a multi-city or other-city article, not a Paris guide |
| 16 | [Ultimate One Day in Paris Itinerary 2024 – An Epic Day In Paris](https://theplanetd.com/one-day-in-paris-itinerary/) | Arc de Triomphe · Q64436 | 111 | right: covers Arc de Triomphe in the destination area |
| 17 | [The Best French Food: 29 Traditional Dishes You Absolutely Cannot Miss](https://theplanetd.com/traditional-french-food/) | French · Q150 | 111 | wrong: French is a language, not a Paris stop |
| 18 | [100 Travel Tips for Paris](https://www.adventurouskate.com/100-travel-tips-for-paris/) | Montmartre · Q186115 | 111 | right: covers Montmartre in the destination area |
| 19 | [The 29 Best Things to do in Hanoi, Vietnam](https://theplanetd.com/things-to-do-in-hanoi/) | French · Q150 | 100 | wrong: French is a language, not a Paris stop |
| 20 | [Aprender francês – para principiantes](https://www.viajecomigo.com/2016/08/30/aprender-frances/) | French · Q150 | 110 | wrong: French is a language, not a Paris stop |
| 21 | [12 Best Things To Do in Paris: The Ultimate 3-day Trip](https://www.saltinourhair.com/france/paris-travel-guide/) | Eiffel Tower · Q243 | 111 | right: covers Eiffel Tower in the destination area |
| 22 | [Is Paris Expensive to Visit? Trip Costs And Budgets](https://theplanetd.com/is-paris-expensive-to-visit/) | Notre-Dame de Paris · Q2981 | 111 | right: covers Notre-Dame de Paris in the destination area |
| 23 | [33 Gorgeous, Famous Streets of Paris](https://www.adventurouskate.com/famous-streets-of-paris/) | French · Q150 | 111 | wrong: French is a language, not a Paris stop |
| 24 | [20 Best Museums in Paris To Visit in 2026](https://theplanetd.com/best-museums-in-paris/) | Louvre - Rivoli · Q1461794 | 111 | wrong: the article does not describe Louvre–Rivoli station |
| 25 | [100 Travel Tips for Paris](https://www.adventurouskate.com/100-travel-tips-for-paris/) | French · Q150 | 111 | wrong: French is a language, not a Paris stop |
| 26 | [Visitar Paris: o que fazer num roteiro de 3, 4 e 5 dias](https://www.almadeviajante.com/visitar-paris-o-que-fazer/) | Montmartre · Q186115 | 111 | right: covers Montmartre in the destination area |
| 27 | [20 Ways to Save Money in Paris](https://www.nomadicmatt.com/travel-blogs/ways-save-money-paris/) | French · Q150 | 111 | wrong: French is a language, not a Paris stop |
| 28 | [48 Hours in Lisbon, Itinerary](https://theplanetd.com/48-hours-in-lisbon-itinerary/) | UNESCO · Q7809 | 000 | wrong: UNESCO is an organization, not a Paris stop |
| 29 | [7 atrações em Paris que não pode (mesmo) perder](https://www.almadeviajante.com/atracoes-em-paris/) | Louvre Museum · Q19675 | 111 | right: covers Louvre Museum in the destination area |
| 30 | [Paris in October: Weather, Fall Tips, And What to Expect](https://theplanetd.com/paris-in-october/) | Eiffel Tower · Q243 | 111 | right: covers Eiffel Tower in the destination area |
| 31 | [How to Visit Paris on a Budget – 20 Tips to Save Money](https://theplanetd.com/visit-paris-on-a-budget/) | Eiffel Tower · Q243 | 111 | right: covers Eiffel Tower in the destination area |
| 32 | [What to do in Paris at Night](https://theplanetd.com/paris-at-night-a-photo-story/) | Louvre Museum · Q19675 | 111 | right: covers Louvre Museum in the destination area |
| 33 | [Visitar Paris – guia para a capital de França](https://www.viajecomigo.com/2014/02/22/viajar-paris/) | Eiffel Tower · Q243 | 111 | right: covers Eiffel Tower in the destination area |
| 34 | [AK Monthly Recap: March 2023](https://www.adventurouskate.com/ak-monthly-recap-march-2023/) | UNESCO · Q7809 | 111 | wrong: UNESCO is an organization, not a Paris stop |
| 35 | [How to Visit Paris on a Budget – 20 Tips to Save Money](https://theplanetd.com/visit-paris-on-a-budget/) | Arc de Triomphe · Q64436 | 111 | right: covers Arc de Triomphe in the destination area |
| 36 | [Torre Eiffel, Paris, França](https://www.viajecomigo.com/2014/02/21/torre-eiffel-paris-franca/) | Eiffel Tower · Q243 | 111 | right: covers Eiffel Tower in the destination area |
| 37 | [Torre Eiffel, Paris, França](https://www.viajecomigo.com/2014/02/21/torre-eiffel-paris-franca/) | Eiffel Tower · Q243 | 111 | right: covers Eiffel Tower in the destination area |
| 38 | [How to Plan a Day Trip to Paris From London by Train](https://www.adventurouskate.com/how-to-plan-a-day-trip-from-london-to-paris/) | Eiffel Tower · Q243 | 111 | right: covers Eiffel Tower in the destination area |
| 39 | [17 coisas grátis para fazer em Paris](https://www.viajecomigo.com/2015/10/26/coisas-gratis-em-paris/) | Eiffel Tower · Q243 | 111 | right: covers Eiffel Tower in the destination area |
| 40 | [Paris in October: Weather, Fall Tips, And What to Expect](https://theplanetd.com/paris-in-october/) | Eiffel Tower · Q243 | 111 | right: covers Eiffel Tower in the destination area |

### Lisbon sample (seed 20261013)

| # | Indexed post | Place | N=2/3/5 | Verdict and reason |
|---:|---|---|---|---|
| 1 | [16 dos palácios mais bonitos de Portugal](https://www.vortexmag.net/16-dos-palacios-mais-bonitos-de-portugal/) | Portugal · Q45 | 111 | wrong: Portugal is a country, not a Lisbon place |
| 2 | [Parques em Lisboa: 5 locais quase secretos para passear sem multidões](https://www.vortexmag.net/parques-em-lisboa/) | Alvalade · Q448231 | 111 | right: covers Alvalade in the destination area |
| 3 | [Tininha de Alfama, Lisboa](https://www.viajecomigo.com/2017/02/27/tininha-de-alfama-lisboa/) | Alfama · Q985517 | 111 | right: covers Alfama in the destination area |
| 4 | [10 Great Things To Do in Porto, Portugal](https://www.saltinourhair.com/portugal/porto-things-to-do/) | Portugal · Q45 | 110 | wrong: Portugal is a country, not a Lisbon place |
| 5 | [20 dos monumentos mais importantes de Portugal](https://www.vortexmag.net/20-dos-monumentos-mais-importantes-de-portugal/) | Portugal · Q45 | 110 | wrong: Portugal is a country, not a Lisbon place |
| 6 | [Feira das Viagens 2015, em Lisboa e no Porto](https://www.viajecomigo.com/2015/03/09/feira-viagens-2015/) | Campo Pequeno station · Q1787790 | 111 | right: covers Campo Pequeno station in the destination area |
| 7 | [Lisboa, restaurante e bar Soul Garden: sabores da Ásia e América Latina](https://www.viajecomigo.com/2023/12/24/lisboa-restaurante-e-bar-soul-garden/) | Portugal · Q45 | 111 | wrong: Portugal is a country, not a Lisbon place |
| 8 | [Fósseis na pedra de Lisboa: os rudistas com mais de 90 milhões de anos q…](https://www.vortexmag.net/fosseis-rudistas-pedra-lioz-lisboa/) | Praça do Comércio · Q999002 | 111 | right: covers Praça do Comércio in the destination area |
| 9 | [Hospital de Bonecas, Lisboa](https://www.viajecomigo.com/2016/06/11/hospital-de-bonecas-lisboa/) | Praça da Figueira · Q1305091 | 111 | right: covers Praça da Figueira in the destination area |
| 10 | [17 Best Things to Do in Guimarães, Portugal in 2025](https://theplanetd.com/best-things-to-do-in-guimaraes-portugal/) | Portugal · Q45 | 110 | wrong: Portugal is a country, not a Lisbon place |
| 11 | [Os elevadores de Lisboa: história, origem e curiosidades](https://www.vortexmag.net/os-elevadores-de-lisboa-historia-origem-e-curiosidades-2/) | Ascensor da Glória · Q99636 | 111 | right: covers Ascensor da Glória in the destination area |
| 12 | [Passagem de Ano 2015/2016 em Portugal](https://www.viajecomigo.com/2015/12/04/passagem-de-ano-2015-2016-portugal/) | Rossio Square · Q1549270 | 100 | wrong: a national article or incidental Lisbon place |
| 13 | [Os melhores locais grátis para passear em Lisboa](https://www.vortexmag.net/passear-em-lisboa-gratis/) | Campolide · Q1031527 | 111 | right: covers Campolide in the destination area |
| 14 | [The Best Portuguese Food: 28 Traditional Dishes You Have to Eat](https://theplanetd.com/traditional-portuguese-food/) | Portugal · Q45 | 111 | wrong: Portugal is a country, not a Lisbon place |
| 15 | [Onde dormir em Lisboa? Na Lata de Sardinha](https://www.viajecomigo.com/2016/11/22/onde-dormir-em-lisboa-na-lata-de-sardinha/) | Alfama · Q985517 | 111 | right: covers Alfama in the destination area |
| 16 | [O que visitar no Parque Eduardo VII, em Lisboa](https://www.vortexmag.net/o-que-visitar-no-parque-eduardo-vii-em-lisboa/) | Estufa Fria de Lisboa · Q8843624 | 111 | right: covers Estufa Fria de Lisboa in the destination area |
| 17 | [Roteiro cultural em São Paulo (em busca da poesia de suas esquinas)](https://www.almadeviajante.com/roteiro-sao-paulo/) | Freguesia de São Paulo · Q1817090 | 000 | wrong: São Paulo here is in Brazil |
| 18 | [6 Famous Explorers Who Made Modern Travel Possible](https://theplanetd.com/famous-explorers/) | Portugal · Q45 | 111 | wrong: Portugal is a country, not a Lisbon place |
| 19 | [10 das igrejas mais bonitas de Lisboa](https://www.vortexmag.net/10-das-igrejas-mais-bonitas-de-lisboa/) | National Azulejo Museum · Q2313815 | 111 | right: covers National Azulejo Museum in the destination area |
| 20 | [O que fazer em Porto Santo \| Roteiro de 1 dia a partir do Funchal (Ilha …](https://www.almadeviajante.com/o-que-fazer-em-porto-santo-madeira/) | Portugal · Q45 | 000 | wrong: Portugal is a country, not a Lisbon place |
| 21 | [D. Dinis: o Rei poeta que colocou os portugueses a escrever e a falar po…](https://www.vortexmag.net/d-dinis/) | Portugal · Q45 | 100 | wrong: Portugal is a country, not a Lisbon place |
| 22 | [15 hotéis onde já dormi em Lisboa (e recomendo)](https://www.almadeviajante.com/hoteis-onde-ja-dormi-em-lisboa/) | Chiado · Q601430 | 111 | right: covers Chiado in the destination area |
| 23 | [Guia / Roteiro para visitar Lisboa, Portugal](https://www.viajecomigo.com/2015/01/05/visitar-lisboa-portugal/) | Almada · Q328015 | 111 | right: covers Almada in the destination area |
| 24 | [Fotos para o inspirar a visitar o Palácio Fronteira](https://viagensasolta.com/fotos-para-o-inspirar-a-visitar-o-palacio-dos-marqueses-de-fronteira/) | Palace of the Marquises of Fronteira · Q1161724 | 111 | right: covers Palace of the Marquises of Fronteira in the destination area |
| 25 | [10 Enoturismos a visitar em 2020, Portugal](https://www.viajecomigo.com/2019/12/26/enoturismo-visitar-2020-portugal/) | Lisbon Region · Q27689 | 110 | wrong: a national article or incidental Lisbon place |
| 26 | [22 Best Day Trips From Lisbon in 2024](https://theplanetd.com/best-day-trips-from-lisbon/) | Portugal · Q45 | 111 | wrong: Portugal is a country, not a Lisbon place |
| 27 | [Restaurantes e bares novos para conhecer no Porto e em Lisboa](https://www.viajecomigo.com/2016/03/14/restaurantes-e-bares-novos-para-conhecer-no-porto-e-em-lisboa/) | Avenida station · Q430125 | 111 | wrong: a national article or incidental Lisbon place |
| 28 | [Ascensor da Bica, Lisboa](https://www.viajecomigo.com/2013/06/25/ascensor-bica-lisboa/) | Ascensor da Bica · Q99630 | 111 | right: covers Ascensor da Bica in the destination area |
| 29 | [Where to Buy Furniture in Portugal (2026 Prices Compared)](https://www.portugalist.com/buying-furniture-portugal/) | Portugal · Q45 | 111 | wrong: Portugal is a country, not a Lisbon place |
| 30 | [Viaje Comigo: prémios, distinções e presença nos Media](https://www.viajecomigo.com/2024/04/27/viaje-comigo-premios-distincoes-e-presenca-nos-media/) | Portugal · Q45 | 110 | wrong: Portugal is a country, not a Lisbon place |
| 31 | [Abriu oficialmente o Holiday Inn Express Porto-Boavista – Porto/Portugal](https://www.viajecomigo.com/2024/10/01/abriu-oficialmente-o-holiday-inn-express-porto-boavista-porto-portugal/) | Portugal · Q45 | 000 | wrong: Portugal is a country, not a Lisbon place |
| 32 | [EasyJet inaugura nova rota entre o Porto e São Vicente](https://www.viajecomigo.com/2026/03/30/easyjet-inaugura-nova-rota-entre-o-porto-e-sao-vicente/) | Portugal · Q45 | 000 | wrong: Portugal is a country, not a Lisbon place |
| 33 | [15 dos melhores locais para visitar em Sintra](https://www.vortexmag.net/15-dos-melhores-locais-para-visitar-em-sintra/) | Portugal · Q45 | 110 | wrong: Portugal is a country, not a Lisbon place |
| 34 | [Passagem de Ano 2015/2016 em Portugal](https://www.viajecomigo.com/2015/12/04/passagem-de-ano-2015-2016-portugal/) | Praça do Comércio · Q999002 | 100 | wrong: a national article or incidental Lisbon place |
| 35 | [10 das bibliotecas mais bonitas de Portugal](https://www.vortexmag.net/10-das-bibliotecas-mais-bonitas-de-portugal/) | Campo Grande · Q963573 | 110 | wrong: a national article or incidental Lisbon place |
| 36 | [7 segredos para descobrir no Alentejo](https://www.vortexmag.net/7-segredos-para-descobrir-no-alentejo/) | Portugal · Q45 | 000 | wrong: Portugal is a country, not a Lisbon place |
| 37 | [12 lugares em Lisboa que parecem ser no estrangeiro](https://www.vortexmag.net/12-lugares-em-lisboa-que-parecem-ser-no-estrangeiro/) | Portugal · Q45 | 111 | wrong: Portugal is a country, not a Lisbon place |
| 38 | [Portugal está preparado para um tsunami? O risco é elevado e nem tudo fu…](https://www.vortexmag.net/portugal-esta-preparado-para-um-tsunami-o-risco-e-elevado-e-nem-tudo-funciona-como-deveria/) | Portugal · Q45 | 110 | wrong: Portugal is a country, not a Lisbon place |
| 39 | [10 mulheres que marcaram a História de Portugal](https://www.vortexmag.net/10-mulheres-que-marcaram-a-historia-de-portugal/) | Portugal · Q45 | 110 | wrong: Portugal is a country, not a Lisbon place |
| 40 | [Aeroporto de Lisboa reabre o Terminal 2 a 01 de julho de 2021](https://www.viajecomigo.com/2021/06/28/aeroporto-de-lisboa-reabre-o-terminal-2-a-01-de-julho-de-2021/) | Portugal · Q45 | 111 | wrong: Portugal is a country, not a Lisbon place |

### Funchal sample (seed 20261013)

| # | Indexed post | Place | N=2/3/5 | Verdict and reason |
|---:|---|---|---|---|
| 1 | [6 hotéis na Madeira para passar umas férias inesquecíveis](https://viagensasolta.com/6-hoteis-na-madeira-para-passar-umas-ferias-inesqueciveis/) | Praça de Colombo · Q56349459 | 111 | right: covers Praça de Colombo in the destination area |
| 2 | [Chef do Desarma leva o oceano Atlântico até o Palatial, Braga](https://www.viajecomigo.com/2026/07/15/chef-do-desarma-leva-o-oceano-atlantico-ate-o-palatial-braga/) | Madeira Island · Q30188 | 000 | wrong: Braga event only mentions Madeira cuisine |
| 3 | [Madeira: o que visitar e roteiro de 3, 5 e 7 dias](https://viagensasolta.com/roteiro-do-que-visitar-na-madeira-3-5-ou-7-dias/) | Ribeiro Frio · Q1248968 | 111 | right: covers Ribeiro Frio in the destination area |
| 4 | [Ver a Madeira de cima: só para quem não tem vertigens](https://www.viajecomigo.com/2021/12/13/miradouros-e-levadas-madeira/) | Pico Ruivo · Q473169 | 110 | right: covers Pico Ruivo in the destination area |
| 5 | [Portuguese Island Life: Living in Madeira vs the Azores](https://www.portugalist.com/azores-vs-madeira/) | Faial · Q995608 | 111 | wrong: Faial is the Azores island in this comparison |
| 6 | [Onde ficar na Madeira (hotéis no Funchal e Turismo Rural na ilha)](https://www.almadeviajante.com/onde-ficar-na-madeira-hoteis/) | Machico · Q693243 | 111 | right: covers Machico in the destination area |
| 7 | [Onde ficar na Madeira (hotéis no Funchal e Turismo Rural na ilha)](https://www.almadeviajante.com/onde-ficar-na-madeira-hoteis/) | Porto Moniz · Q221412 | 111 | right: covers Porto Moniz in the destination area |
| 8 | [Ver a Madeira de cima: só para quem não tem vertigens](https://www.viajecomigo.com/2021/12/13/miradouros-e-levadas-madeira/) | Madeira Island · Q30188 | 110 | right: covers Madeira Island in the destination area |
| 9 | [Visitar Funchal: o que ver e o que fazer na capital da Ilha da Madeira](https://www.vortexmag.net/os-12-melhores-locais-para-visitar-no-funchal/) | Monte · Q1179064 | 111 | right: covers Monte in the destination area |
| 10 | [10 ilhas desertas para descobrir em Portugal](https://www.vortexmag.net/10-ilhas-desertas-para-descobrir-em-portugal/) | Desertas Islands · Q27923 | 000 | wrong: Desertas is outside Madeira island |
| 11 | [Visitar a Madeira: o que ver e o que fazer](https://www.vortexmag.net/os-10-melhores-locais-para-visitar-na-madeira/) | Curral das Freiras · Q602368 | 111 | right: covers Curral das Freiras in the destination area |
| 12 | [Ver a Madeira de cima: só para quem não tem vertigens](https://www.viajecomigo.com/2021/12/13/miradouros-e-levadas-madeira/) | Curral das Freiras · Q602368 | 110 | right: covers Curral das Freiras in the destination area |
| 13 | [Atividades na Madeira – o que fazer?](https://www.viajecomigo.com/2014/08/22/atividades-madeira/) | Machico · Q693243 | 111 | right: covers Machico in the destination area |
| 14 | [Fajã dos Padres, um pequeno tesouro na Ilha da Madeira](https://www.almadeviajante.com/faja-dos-padres-madeira/) | Fajã dos Padres · Q52063401 | 100 | right: covers Fajã dos Padres in the destination area |
| 15 | [O que fazer em Porto Santo \| Roteiro de 1 dia a partir do Funchal (Ilha …](https://www.almadeviajante.com/o-que-fazer-em-porto-santo-madeira/) | São Jorge · Q1088866 | 111 | wrong: the article is about Porto Santo |
| 16 | [Visitar a Madeira: o que ver e o que fazer](https://www.vortexmag.net/os-10-melhores-locais-para-visitar-na-madeira/) | Madeira Botanical Garden · Q894636 | 111 | right: covers Madeira Botanical Garden in the destination area |
| 17 | [Diário da nossa viagem à Madeira (e alguns desabafos)](https://viagensasolta.com/diario-da-nossa-viagem-a-madeira/) | Santana · Q752123 | 111 | right: covers Santana in the destination area |
| 18 | [Levada do Caldeirão Verde: How to Hike PR9 Madeira](https://www.saltinourhair.com/madeira/levada-do-caldeiro-verde/) | Faial · Q995608 | 110 | right: covers Faial in the destination area |
| 19 | [Madeira: o que visitar e roteiro de 3, 5 e 7 dias](https://viagensasolta.com/roteiro-do-que-visitar-na-madeira-3-5-ou-7-dias/) | Encumeada · Q19279137 | 111 | right: covers Encumeada in the destination area |
| 20 | [Diário da nossa viagem à Madeira (e alguns desabafos)](https://viagensasolta.com/diario-da-nossa-viagem-a-madeira/) | Ponta do Sol · Q943911 | 111 | right: covers Ponta do Sol in the destination area |
| 21 | [Fajã dos Padres, um pequeno tesouro na Ilha da Madeira](https://www.almadeviajante.com/faja-dos-padres-madeira/) | Fajã dos Padres · Q52063401 | 100 | right: covers Fajã dos Padres in the destination area |
| 22 | [Complete 10-Day Madeira Itinerary](https://www.saltinourhair.com/madeira/madeira-itinerary-10-days/) | Monte · Q1179064 | 111 | right: covers Monte in the destination area |
| 23 | [Casa das Mudas, um Museu de Arte Contemporânea num edifício de sonho](https://www.almadeviajante.com/casa-das-mudas-museu-de-arte-contemporanea-madeira/) | Museu de Arte Contemporânea do Funchal · Q10333759 | 111 | wrong: Casa das Mudas is not this Funchal museum |
| 24 | [Portuguese Island Life: Living in Madeira vs the Azores](https://www.portugalist.com/azores-vs-madeira/) | University of Madeira · Q1434847 | 111 | wrong: a national list or an incidental place name |
| 25 | [Diário da nossa viagem à Madeira (e alguns desabafos)](https://viagensasolta.com/diario-da-nossa-viagem-a-madeira/) | Santana · Q752123 | 111 | right: covers Santana in the destination area |
| 26 | [Madeira: o que visitar e roteiro de 3, 5 e 7 dias](https://viagensasolta.com/roteiro-do-que-visitar-na-madeira-3-5-ou-7-dias/) | Santana · Q752123 | 111 | right: covers Santana in the destination area |
| 27 | [Restaurantes na ilha da Madeira](https://www.viajecomigo.com/2014/08/14/restaurantes-madeira/) | Pico Ruivo · Q473169 | 111 | wrong: a national list or an incidental place name |
| 28 | [As aldeias e vilas mais isoladas de Portugal](https://www.vortexmag.net/as-aldeias-e-vilas-mais-isoladas-de-portugal/) | Caniçal · Q857987 | 000 | wrong: a national list or an incidental place name |
| 29 | [4 caminhadas que adorámos na Madeira (dificuldade média)](https://viagensasolta.com/4-trilhos-que-adoramos-fazer-na-madeira-dificuldade-media/) | Pico Ruivo · Q473169 | 000 | right: covers Pico Ruivo in the destination area |
| 30 | [12 dos jardins mais bonitos de Portugal](https://www.vortexmag.net/jardins-mais-bonitos-de-portugal/) | Monte · Q1179064 | 100 | wrong: a national list or an incidental place name |
| 31 | [Levada dos Balcões, a levada mais fácil da Madeira](https://www.almadeviajante.com/levada-dos-balcoes-birdwatching-madeira/) | Balcões · Q19296457 | 100 | right: covers Balcões in the destination area |
| 32 | [Atividades na Madeira – o que fazer?](https://www.viajecomigo.com/2014/08/22/atividades-madeira/) | Fajã dos Padres · Q52063401 | 111 | right: covers Fajã dos Padres in the destination area |
| 33 | [4 caminhadas que adorámos na Madeira (dificuldade média)](https://viagensasolta.com/4-trilhos-que-adoramos-fazer-na-madeira-dificuldade-media/) | Pico do Areeiro · Q1192102 | 000 | right: covers Pico do Areeiro in the destination area |
| 34 | [Praia da Prainha, uma das melhores praias da Madeira](https://www.almadeviajante.com/praia-da-prainha-melhores-praias-da-madeira/) | Ponta de São Lourenço · Q1296587 | 110 | right: covers Ponta de São Lourenço in the destination area |
| 35 | [Guia rápido para visitar o Funchal](https://viagensasolta.com/guia-rapido-para-visitar-o-funchal-madeira/) | Madeira Island · Q30188 | 111 | right: covers Madeira Island in the destination area |
| 36 | [Visitar a Madeira: o que ver e o que fazer](https://www.vortexmag.net/os-10-melhores-locais-para-visitar-na-madeira/) | Madeira Island · Q30188 | 111 | right: covers Madeira Island in the destination area |
| 37 | [18 Things to do in Madeira, Portugal](https://www.saltinourhair.com/madeira/things-to-do-madeira/) | Santana · Q752123 | 111 | right: covers Santana in the destination area |
| 38 | [10 Best Hikes in Madeira: A Must-do Trail List](https://www.saltinourhair.com/madeira/best-hikes-madeira/) | Pico do Areeiro · Q1192102 | 111 | right: covers Pico do Areeiro in the destination area |
| 39 | [Madeira: o que visitar e roteiro de 3, 5 e 7 dias](https://viagensasolta.com/roteiro-do-que-visitar-na-madeira-3-5-ou-7-dias/) | Prainha · Q10353185 | 111 | right: covers Prainha in the destination area |
| 40 | [Pico do Arieiro Hike on Madeira: How to visit PR1](https://www.saltinourhair.com/madeira/pico-do-arieiro-hike/) | Pico do Areeiro · Q1192102 | 110 | right: covers Pico do Areeiro in the destination area |

### Porto sample (seed 20261013)

| # | Indexed post | Place | N=2/3/5 | Verdict and reason |
|---:|---|---|---|---|
| 1 | [Livraria Lello, Porto: Tips For Visiting The Most Beautiful Bookstore in…](https://theplanetd.com/livraria-lello-porto/) | Livraria Lello · Q3408689 | 111 | right: covers Livraria Lello in the destination area |
| 2 | [As 10 melhores praias fluviais do Rio Douro](https://www.vortexmag.net/as-9-melhores-praias-fluviais-do-rio-douro/) | Douro River · Q14299 | 111 | wrong: a national list, other-city article, or incidental place |
| 3 | [Abriu o 17•56 Museu & Enoteca Real Companhia Velha, Vila Nova de Gaia](https://www.viajecomigo.com/2018/08/26/1756-museu-enoteca-real-companhia-velha-vila-nova-de-gaia/) | Vila Nova de Gaia · Q191135 | 111 | right: covers Vila Nova de Gaia in the destination area |
| 4 | [School Trip Mini: programa para crianças no Porto/Post/Doc](https://www.viajecomigo.com/2016/11/24/school-trip-mini-2016/) | Rivoli Theatre · Q3826131 | 111 | right: covers Rivoli Theatre in the destination area |
| 5 | [Porto: NOS em D’Bandada 2016 – 17 de setembro](https://www.viajecomigo.com/2016/09/02/porto-nos-em-dbandada-2016-17-de-setembro/) | Praça de Gomes Teixeira · Q10353459 | 111 | right: covers Praça de Gomes Teixeira in the destination area |
| 6 | [Passeio de barco com a Riversoul, no Douro – Porto](https://www.viajecomigo.com/2017/07/28/passeio-de-barco-com-a-riversoul-no-douro-porto/) | Freguesia de São Pedro da Afurada · Q2256308 | 111 | right: covers Freguesia de São Pedro da Afurada in the destination area |
| 7 | [6 teleféricos para apreciar as melhores paisagens de Portugal](https://www.vortexmag.net/melhores-telefericos-de-portugal/) | Jardim do Morro · Q10306339 | 000 | wrong: a national list, other-city article, or incidental place |
| 8 | [Crowne Plaza Porto Hotel, Portugal](https://www.viajecomigo.com/2016/01/19/hotel-crowne-porto/) | Boavista boulevard · Q16495570 | 111 | right: covers Boavista boulevard in the destination area |
| 9 | [Já visitaram o Mercado Temporário do Bolhão, Porto?](https://www.viajecomigo.com/2018/08/20/mercado-temporario-bolhao/) | Bolhão Market · Q10328805 | 111 | right: covers Bolhão Market in the destination area |
| 10 | [O que fazer no Porto (os segredos da Sara Riobom)](https://www.almadeviajante.com/o-que-fazer-no-porto-dicas-sara-riobom/) | Fontainhas · Q10283281 | 111 | right: covers Fontainhas in the destination area |
| 11 | [Viver a noite de São João no Porto como um tripeiro (um roteiro)](https://www.almadeviajante.com/noite-de-sao-joao-no-porto-roteiro/) | Avenida dos Aliados · Q16488534 | 111 | right: covers Avenida dos Aliados in the destination area |
| 12 | [As dicas do Viaje Comigo no jornal Hey Porto](https://www.viajecomigo.com/2018/02/07/as-dicas-do-viaje-comigo-no-jornal-hey-porto/) | Carvalhido · Q9697962 | 111 | wrong: a national list, other-city article, or incidental place |
| 13 | [Chefe Cozinheiro do Ano 2017 é Luís Gaspar, da Sala de Corte, Lisboa](https://www.viajecomigo.com/2017/06/11/chefe-cozinheiro-do-ano-2017-e-luis-gaspar-da-sala-de-corte-lisboa/) | Rua das Flores · Q10364989 | 000 | wrong: a national list, other-city article, or incidental place |
| 14 | [As dicas do Viaje Comigo no jornal Hey Porto](https://www.viajecomigo.com/2018/02/07/as-dicas-do-viaje-comigo-no-jornal-hey-porto/) | Porto City Centre · Q16496268 | 111 | wrong: a national list, other-city article, or incidental place |
| 15 | [Jardins e Casa da Quinta da Prelada, Porto](https://www.viajecomigo.com/2016/08/26/jardins-e-casa-quinta-prelada-porto/) | Casa da Prelada · Q1046464 | 111 | right: covers Casa da Prelada in the destination area |
| 16 | [The Perfect One Day in Porto Itinerary for 2024](https://theplanetd.com/one-day-in-porto-itinerary/) | Porto Cathedral · Q1441255 | 111 | right: covers Porto Cathedral in the destination area |
| 17 | [15 fantásticos locais grátis para visitar no Porto](https://www.vortexmag.net/locais-gratis-para-visitar-no-porto/) | Serralves Foundation · Q3393650 | 111 | right: covers Serralves Foundation in the destination area |
| 18 | [Roteiro pelo centro histórico do Porto (Património Mundial)](https://www.almadeviajante.com/patrimonio-mundial-centro-historico-do-porto-portugal/) | São Bento railway station · Q579250 | 111 | right: covers São Bento railway station in the destination area |
| 19 | [Porto: Festas de São João 2022](https://www.viajecomigo.com/2022/06/18/porto-festas-de-sao-joao-2022/) | Luiz I Bridge · Q1322447 | 111 | right: covers Luiz I Bridge in the destination area |
| 20 | [Viver a noite de São João no Porto como um tripeiro (um roteiro)](https://www.almadeviajante.com/noite-de-sao-joao-no-porto-roteiro/) | Jardim de Joao Chagas · Q10306312 | 111 | right: covers Jardim de Joao Chagas in the destination area |
| 21 | [10 dos monumentos religiosos mais importantes de Portugal](https://www.vortexmag.net/10-dos-monumentos-religiosos-mais-importantes-de-portugal/) | Kadoorie Synagogue · Q1140204 | 111 | wrong: a national list, other-city article, or incidental place |
| 22 | [Mercado de Natal – Essência do Gourmet, Porto – 2014](https://www.viajecomigo.com/2014/11/27/mercado-natal-essencia-gourmet-porto-2014/) | Avenida dos Aliados · Q16488534 | 111 | right: covers Avenida dos Aliados in the destination area |
| 23 | [Where to Stay in Porto: A Complete Guide For Your First Visit](https://theplanetd.com/where-to-stay-in-porto/) | Douro River · Q14299 | 111 | right: covers Douro River in the destination area |
| 24 | [Evento Viaje Comigo nas notícias](https://www.viajecomigo.com/2013/12/06/evento-viaje-comigo-nas-noticias/) | Vila Nova de Gaia · Q191135 | 110 | wrong: a national list, other-city article, or incidental place |
| 25 | [Casa de Mateus, Vila Real, Portugal](https://www.viajecomigo.com/2017/03/17/casa-de-mateus-vila-real-portugal/) | Clerigos tower · Q20986654 | 110 | wrong: a national list, other-city article, or incidental place |
| 26 | [Os melhores Terraços / Rooftops no Porto](https://www.viajecomigo.com/2022/06/20/os-melhores-terracos-rooftops-no-porto/) | Museu da Misericórdia do Porto · Q25419284 | 111 | right: covers Museu da Misericórdia do Porto in the destination area |
| 27 | [Cafés (lindos) a visitar no Porto no outono/inverno (e não só!)](https://www.viajecomigo.com/2020/11/20/cafes-a-visitar-no-porto/) | Vila Nova de Gaia · Q191135 | 111 | right: covers Vila Nova de Gaia in the destination area |
| 28 | [5 razões para visitar Caminha até ao final do ano](https://www.viajecomigo.com/2016/12/19/razoes-para-visitar-caminha-ate-ao-final-do-ano/) | Igreja de Santa Clara · Q3819477 | 000 | wrong: a national list, other-city article, or incidental place |
| 29 | [7 dos melhores restaurantes para comer bacalhau em Portugal](https://www.vortexmag.net/melhores-restaurantes-para-comer-bacalhau-em-portugal/) | Historic Centre of Porto · Q3848157 | 110 | wrong: a national list, other-city article, or incidental place |
| 30 | [5 cidades portuguesas que são Património Mundial da UNESCO](https://www.vortexmag.net/5-cidades-portuguesas-que-sao-patrimonio-mundial-da-unesco/) | Historic Centre of Porto · Q3848157 | 110 | wrong: a national list, other-city article, or incidental place |
| 31 | [Mercado de Natal – Essência do Gourmet, Porto – 2014](https://www.viajecomigo.com/2014/11/27/mercado-natal-essencia-gourmet-porto-2014/) | Metro do Porto · Q29757 | 111 | right: covers Metro do Porto in the destination area |
| 32 | [Livraria Lello, Porto: Tips For Visiting The Most Beautiful Bookstore in…](https://theplanetd.com/livraria-lello-porto/) | Livraria Lello · Q3408689 | 111 | right: covers Livraria Lello in the destination area |
| 33 | [Os segredos e curiosidades da cidade do Porto](https://www.viajecomigo.com/2021/04/06/segredos-cidade-do-porto/) | Clerigos tower · Q20986654 | 111 | right: covers Clerigos tower in the destination area |
| 34 | [10 dos melhores locais para correr no Porto e arredores](https://www.vortexmag.net/melhores-locais-para-correr-no-porto/) | Boavista boulevard · Q16495570 | 111 | right: covers Boavista boulevard in the destination area |
| 35 | [O que visitar em Vila Nova de Gaia](https://www.viajecomigo.com/2015/11/04/visitar-vila-nova-de-gaia/) | Vila Nova de Gaia · Q191135 | 111 | right: covers Vila Nova de Gaia in the destination area |
| 36 | [The Perfect One Day in Porto Itinerary for 2024](https://theplanetd.com/one-day-in-porto-itinerary/) | São Bento metro station (Porto) · Q10276156 | 111 | right: covers São Bento metro station (Porto) in the destination area |
| 37 | [30 Best Things to Do in Porto, Portugal In  2025](https://theplanetd.com/best-things-to-do-in-porto/) | Luiz I Bridge · Q1322447 | 111 | right: covers Luiz I Bridge in the destination area |
| 38 | [Guia de Restaurantes com Vinho do Porto](https://www.viajecomigo.com/2014/02/01/guia-de-restaurantes-com-vinho-do-porto/) | Douro and Port Wine Institute · Q2644063 | 111 | wrong: a national list, other-city article, or incidental place |
| 39 | [Sair à noite: bares no Porto](https://www.viajecomigo.com/2015/03/17/bares-porto/) | Boavista boulevard · Q16495570 | 111 | right: covers Boavista boulevard in the destination area |
| 40 | [Os 5 melhores hotéis de luxo do Porto](https://www.almadeviajante.com/melhores-hoteis-de-luxo-porto/) | Rua das Flores · Q10364989 | 111 | right: covers Rua das Flores in the destination area |
