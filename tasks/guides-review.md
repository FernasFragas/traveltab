# City guide review

Reviewed 2026-09-24 by Claude (Sonnet 5), working from free sources only. This is the review behind `reviewed: true` in [guides/guides.json](../guides/guides.json); the integration is in [guides-results.md](guides-results.md).

## Key finding

**The generated drafts were not safe to publish as they were.** Seventeen of twenty contained statements that the cited Wikivoyage page does not support or contradicts. The clearest: Mexico City's "population of over 22 million" (that is the urban area; the city has 9.2 million), Kyoto's airport train fare, Berlin's "trendy nightlife of Potsdamer Platz", Marrakech's airport "well-connected by bus and train to other parts of Morocco", and Tavira's "Roman bridge" (dated to the 16th or 17th century). Every intro was rewritten to keep only statements that trace to the cited revision.

| Outcome | Cities |
| --- | --- |
| **Reviewed and published** | 20 of 20 |
| Rewritten (text differs from the generated draft) | 20 of 20 |
| Dropped or left unreviewed | 0 |
| Corrections to draft statements (wrong or misleading) | 26, in 17 cities |
| Removals (unsupported, dated or filler) | 56 |
| Intro length | mean 176 words (draft: 202), max 221 |

Nothing is left unreviewed among the starter cities, so the "no reviewed guide" state is what every other city gets. A regenerated entry loses its review flag on purpose (see [Re-reviewing](#re-reviewing)).

## Method

1. **Source text.** Each cited Wikivoyage article, fetched with `curl` from the MediaWiki API (`action=query&prop=extracts&explaintext=1`). The API returns the *current* text even when given a `revids` value, so for the five pages whose revision had moved on (Istanbul, Paris, Prague, Seville, Tokyo) I diffed the wikitext at the recorded `oldid` against today's (`action=parse&oldid=…`). The differences are a suburbs paragraph and a pass tip (Paris), a transit-app sentence (Istanbul), ticket-inspector text and a dead-link tag (Prague), a hotel URL (Seville) and a dead-link tag (Tokyo). None touches a statement kept here, so every kept statement holds at the recorded revision.
2. **Claim check.** Each sentence of each draft was compared with the article's lead, Understand, Districts, Get in and Get around sections. A statement stays only if the article says it. Time-sensitive facts (fares, prices, opening hours, app availability) were removed.
3. **Second source for numbers.** Populations, rankings, UNESCO status and other hard figures were cross-checked against the English Wikipedia article (2026-09-24). Where the two disagreed, the figure was dropped rather than picked: Barcelona (1.7 million vs "nearly two million"), Porto, Prague, Florence, Seville's rank (fifth vs fourth) and Rome's metro population. Agreement was recorded for Amsterdam's 2010 UNESCO listing, Berlin's zoo, Tavira's 37 churches, Porto's 1996 UNESCO listing, Tokyo's 1927 subway, Marrakech's rank and Mexico City's boroughs.
4. **Proper names.** Each intro was run through the generator's own `ValidateIntro` against the full article text (a temporary test, since removed). All 20 pass, so no place named in a kept intro is absent from its source. Names were also checked for spelling and diacritics (Marrakech, Djemaa El-Fna, Cercanías, Staroměstské náměstí, Ichijō-dōri).
5. **Wording.** The intros are summaries and paraphrases. They stay under CC BY-SA 4.0 and are shown as "Adapted from Wikivoyage", with the article, the revision permalink, the history page (authors) and the license linked, and a note that they were reworded.

## Results by city

| City | Wikivoyage article @ revision | Corrected | Removed |
| --- | --- | ---: | ---: |
| Amsterdam, NL | [Amsterdam](https://en.wikivoyage.org/w/index.php?title=Amsterdam&oldid=5359183) @ 5359183 | 1 | 2 |
| Barcelona, ES | [Barcelona](https://en.wikivoyage.org/w/index.php?title=Barcelona&oldid=5359270) @ 5359270 | 1 | 2 |
| Berlin, DE | [Berlin](https://en.wikivoyage.org/w/index.php?title=Berlin&oldid=5363362) @ 5363362 | 2 | 2 |
| Florence, IT | [Florence](https://en.wikivoyage.org/w/index.php?title=Florence&oldid=5350936) @ 5350936 | 1 | 3 |
| Funchal, PT | [Funchal](https://en.wikivoyage.org/w/index.php?title=Funchal&oldid=5367473) @ 5367473 | 1 | 3 |
| Istanbul, TR | [Istanbul](https://en.wikivoyage.org/w/index.php?title=Istanbul&oldid=5363386) @ 5363386 (page has since moved on; see Method 1) | 2 | 2 |
| Kyoto, JP | [Kyoto](https://en.wikivoyage.org/w/index.php?title=Kyoto&oldid=5343385) @ 5343385 | 1 | 3 |
| Lisbon, PT | [Lisbon](https://en.wikivoyage.org/w/index.php?title=Lisbon&oldid=5364237) @ 5364237 | 0 | 4 |
| London, GB | [London](https://en.wikivoyage.org/w/index.php?title=London&oldid=5353122) @ 5353122 | 1 | 2 |
| Marrakesh, MA | [Marrakech](https://en.wikivoyage.org/w/index.php?title=Marrakech&oldid=5367668) @ 5367668 | 2 | 2 |
| Mexico City, MX | [Mexico City](https://en.wikivoyage.org/w/index.php?title=Mexico_City&oldid=5353669) @ 5353669 | 4 | 3 |
| New York, US | [New York City](https://en.wikivoyage.org/w/index.php?title=New_York_City&oldid=5354261) @ 5354261 | 1 | 3 |
| Paris, FR | [Paris](https://en.wikivoyage.org/w/index.php?title=Paris&oldid=5354829) @ 5354829 (page has since moved on; see Method 1) | 2 | 4 |
| Porto, PT | [Porto](https://en.wikivoyage.org/w/index.php?title=Porto&oldid=5355253) @ 5355253 | 0 | 3 |
| Prague, CZ | [Prague](https://en.wikivoyage.org/w/index.php?title=Prague&oldid=5367676) @ 5367676 (page has since moved on; see Method 1) | 2 | 3 |
| Rome, IT | [Rome](https://en.wikivoyage.org/w/index.php?title=Rome&oldid=5355689) @ 5355689 | 1 | 4 |
| Seville, ES | [Seville](https://en.wikivoyage.org/w/index.php?title=Seville&oldid=5356315) @ 5356315 (page has since moved on; see Method 1) | 1 | 3 |
| Tavira, PT | [Tavira](https://en.wikivoyage.org/w/index.php?title=Tavira&oldid=5357108) @ 5357108 | 2 | 3 |
| Tokyo, JP | [Tokyo](https://en.wikivoyage.org/w/index.php?title=Tokyo&oldid=5357290) @ 5357290 (page has since moved on; see Method 1) | 1 | 3 |
| Vienna, AT | [Vienna](https://en.wikivoyage.org/w/index.php?title=Vienna&oldid=5357766) @ 5357766 | 0 | 2 |

## Details

### Amsterdam, NL

- Checked: Lead, Understand/Orientation and Get around (walking, bicycle) sections; population and UNESCO 2010 cross-checked on Wikipedia.
- **Corrected in the draft:**
  - Draft called the Canal District's houses '17th-century' with no support in the page (page dates the trading boom, not the houses).
- Dropped: Generic 'appeal extends beyond its historical roots' and 'unforgettable experience' filler; 'Museums ... within easy reach' (not stated).

### Barcelona, ES

- Checked: Lead, When to visit, Get in (airport), Get around (metro, integrated fares), language section; Wikipedia population.
- **Corrected in the draft:**
  - Draft said Barcelona 'offers an enjoyable experience year-round' as the sentence's point; the page's real advice is that August is busiest but shops and restaurants close then.
- Dropped: 'Family-friendly city' and 'tourist information centers readily available' (thin support); Wikivoyage's 'nearly two million' population: Wikipedia gives about 1.7 million, so no figure is shown.

### Berlin, DE

- Checked: Lead, Districts, Understand, Get around (public transport, zones, S-Bahn); zoo claim and 3.7 million cross-checked on Wikipedia.
- **Corrected in the draft:**
  - Draft described 'the trendy nightlife of Potsdamer Platz'; the page describes Potsdamer Platz as a district of 1990s-2000s glass palaces, not nightlife.
  - Draft's 'bohemian charm of Kreuzberg' is not in the page.
- Dropped: 'Taxis and ride-sharing services are also available' (not stated); 'Hub for ... politics, media and science' phrasing beyond the page's 'world city of culture, politics, media and science'.

### Florence, IT

- Checked: Lead, Understand, Get in (airport, trains), Get around (walking, bicycle); UNESCO and population on Wikipedia.
- **Corrected in the draft:**
  - Draft said taxis are 'readily available'; the page says a taxi is hard to hail and best called ahead by your hotel or restaurant.
- Dropped: 'Narrow, winding streets and grand piazzas' (not in the page); 'International airport' and 'frequent bus services to major Italian cities' (not stated); Population figure (page 367,000 for 2022; Wikipedia now 361,625).

### Funchal, PT

- Checked: Whole page except Eat/Sleep listings; population and fennel etymology on Wikipedia.
- **Corrected in the draft:**
  - Draft called the toboggan ride an 'activity' without saying where; the page places it at Monte and describes men steering the sledge.
- Dropped: 'Budget-friendly eateries to high-end Italian restaurants' (the page lists one Italian restaurant; no such range); 'One of Portugal's most enchanting cities' (page says most beautiful; opinion, dropped); 'Hidden gems' filler.

### Istanbul, TR

- Checked: Lead, Orientation, Climate, Get in (airports), Get around (İstanbulkart, ferries) at the cited revision (a later revision changes only a transit-app sentence).
- **Corrected in the draft:**
  - Draft's 'Despite its reputation, Istanbul's climate is oceanic' garbled the page's point that Istanbul is not a year-round sunny destination.
  - Draft said both airports connect 'via metro, bus, taxi and airport shuttle'; the page describes metro only for Sabiha Gökçen, so the claim was dropped.
- Dropped: 'Largest city in Europe' reworded to the page's 'most populous city in Europe'; 'Unparalleled fusion' and 'captivating' filler.

### Kyoto, JP

- Checked: Lead, Understand, Orientation, Climate, Get in (airports, trains); 794-1868 dates on Wikipedia.
- **Corrected in the draft:**
  - Draft put the Haruka one-way fare at 'around 3,000 yen'; the page gives 2,900 yen (non-reserved) or 3,430 yen (reserved). Fares change, so no price is shown.
- Dropped: Repeated 'Osaka's airports' sentences (merged); 'Multi-lingual Official Travel Guide site' (external site not verified); Prices.

### Lisbon, PT

- Checked: Lead, Understand, Climate, Orientation; no factual error found in the draft.
- No factual error found in the draft.
- Dropped: 'Noisy discos coexisting harmoniously', patisserie and rooftop-bar colour; 'Less frantic than other million-city destinations' (page: 'often perceived'); Lisboa Card and 'detailed maps' sentence; 'Excellent public transport links' to Sintra and Cascais (rephrased as a plain list).

### London, GB

- Checked: Lead, Districts, Get in (airports), Get around (Tube, National Rail, walking); population cross-checked on Wikipedia.
- **Corrected in the draft:**
  - Draft said the Tube 'covers much of the city'; the page says its 11 lines cover the central area and northern suburbs, with National Rail mainly serving the south.
- Dropped: 'Buses provide comprehensive coverage, taxis readily available' (not stated in this form); 'Whether you're a history buff' filler.

### Marrakesh, MA

- Checked: Lead, Understand, Get in (airport), Get around (bus, petit taxi, caleche); fourth-largest ranking on Wikipedia.
- **Corrected in the draft:**
  - Draft said the airport 'is well-connected by bus and train to other parts of Morocco'; the page says no such thing (the train station and bus stations are separate from the airport).
  - Draft used 'Marrakech' and 'Marrakesh' inconsistently; the page's title is Marrakech, 'also spelt Marrakesh'.
- Dropped: 'Must-visit' and 'ideal base for exploring Morocco's diverse landscapes' (opinion); Ride-hailing apps (page notes Uber only since Feb 2026; time-sensitive).

### Mexico City, MX

- Checked: Lead, Districts, Understand, Get around (Metro, Metrobús, taxis), Talk; population, altitude, 16 boroughs and UNESCO on Wikipedia.
- **Corrected in the draft:**
  - Draft said the city has 'a population of over 22 million'; 22 million is the urban area, the city proper has about 9.2 million.
  - Draft said La Villa de Guadalupe 'draws millions of pilgrims yearly'; the page says only 'a large crowd of pilgrims every day'.
  - Draft said Interlomas, Azcapotzalco, Tláhuac and Iztacalco 'showcase the city's diverse architecture and cultural heritage'; the page describes them as residential, industrial, pottery-making and a sports and racing area.
  - Wikivoyage says three UNESCO sites are here, including Barragán sites 'in Chapultepec'; the Barragán House and Studio is in Tacubaya and Xochimilco is also listed, so the count is dropped.
- Dropped: Neighborhood-by-neighborhood list; 'Melting pot ... welcomes visitors with open arms' filler; UNESCO count of three (see above).

### New York, US

- Checked: Lead, Boroughs, Understand (orientation, climate); five boroughs and population on Wikipedia.
- **Corrected in the draft:**
  - Draft said the 'outer boroughs' offer 'a more laid-back, authentic experience'; the page's authenticity advice is about ethnic neighborhoods and cheaper eating, not the outer boroughs as a whole.
- Dropped: 'Iconic yellow taxis' (not in the page); '4th largest metropolis in the world' (not repeated; ranking is unverified); 'Power, wealth and diversity' filler.

### Paris, FR

- Checked: Lead, Districts, Understand at the cited revision (the current revision differs only in a suburbs paragraph and a pass tip); population on Wikipedia.
- **Corrected in the draft:**
  - Draft said Paris 'boasts an impressive number of Michelin-starred restaurants, second only to Tokyo'; that ranking is Wikivoyage's own statement and volatile, so it is not repeated.
  - Draft's 'around 14 million visitors annually' is a dated figure, dropped.
- Dropped: 'Paris Pratique par Arrondissement' map and its price; List of fashion houses; 'City of Light / City of Love' epithets; Metropolitan-area population (13.3 million on Wikipedia; Wikivoyage's 'almost 13 million' is dated).

### Porto, PT

- Checked: Lead, Understand (history, geography), Get around (metro); UNESCO 1996 and second-largest ranking on Wikipedia.
- No factual error found in the draft.
- Dropped: Population figure (Wikivoyage 238,000 for 2024, Wikipedia 273,476); 'Vibrant and industrious' and 'enriching' filler; Taxi and ride-hailing sentence.

### Prague, CZ

- Checked: Lead, Districts, Understand, Get around at the cited revision (the current revision changes only a ticket-inspector paragraph and a dead link); UNESCO and population on Wikipedia.
- **Corrected in the draft:**
  - Draft said navigating 'is made easier by a simplified district system displayed on street signs'; the simplification is Wikivoyage's own, and the street signs show the older district number.
  - Draft said 'Praha 1 and Praha 2' hold the largest concentration of attractions; the page says Praha 1 by far, and Praha 2 'also' has important historic areas.
- Dropped: 'Getting around is straightforward' introducing the house-number explanation (non sequitur); 'Unique quirkiness that permeates every corner' filler; Population figure (page 1.2 million; Wikipedia 1.4 million).

### Rome, IT

- Checked: Lead, Districts, Understand, Get around (walking, public transport); population and UNESCO on Wikipedia.
- **Corrected in the draft:**
  - Draft called the public transport 'efficient'; the page does not, and it warns that the metro gets crowded.
- Dropped: 'Rental cars available' (the page says driving in Rome is best avoided); 'Fashion-forward' and 'nightlife' colour; Metropolitan-area population (page 4.5 million; Wikipedia 4.2 million); 'Around 4% of the city's area' figure.

### Seville, ES

- Checked: Lead, Understand, Get around at the cited revision (the current revision changes only a hotel URL); population and rank on Wikipedia.
- **Corrected in the draft:**
  - Wikivoyage ranks Seville Spain's fourth-largest city; Wikipedia now says fifth, so no ranking is shown.
- Dropped: 'A wide range of accommodations to suit various budgets' (page lists budget hotels but makes no such claim); Scooter rentals (page prices them; time-sensitive); Population figure.

### Tavira, PT

- Checked: Whole page; 37 churches confirmed on Wikipedia.
- **Corrected in the draft:**
  - Draft listed 'a Roman bridge' among the city's history; the page's 'Roman Bridge' dates from around the 16th or 17th century (rebuilt in 1992).
  - Draft called the castle 'a medieval castle'; the page says it is of Muslim origin and was rebuilt after the Reconquista, with only sections of the wall left.
- Dropped: 'Pristine beaches' (not stated); 'Thriving urban core' (page: thriving thanks to tourism); Horse-drawn carriage and boat tour details (listings, not orientation).

### Tokyo, JP

- Checked: Lead, Districts, Understand (culture, climate), Get around (Yamanote, subway, language); subway 1927 and Metropolis status on Wikipedia; the current revision differs only in a dead-link tag.
- **Corrected in the draft:**
  - Draft said costs 'make it an accessible destination for travelers'; the page only says costs are comparable to other large developed-world cities, so the inference was dropped.
- Dropped: 'Exciting fireworks festivals' and the autumn/winter season sentences (not verified); 'Five distinct seasons' (a Wikivoyage claim not repeated); 'Warm and helpful locals' kept out as opinion.

### Vienna, AT

- Checked: Lead, Districts, Understand (culture, orientation), Get around (public transportation); population and UNESCO on Wikipedia.
- No factual error found in the draft.
- Dropped: 'Dynamic, progressive city that continues to evolve' filler; 'Metro, trams and buses connect all districts' (reworded to the page's own transit statement).

## Limits

- **The reviewer is an AI.** I read every source and checked every kept claim, but a person should skim the intros before a wider launch. The  and  fields say who checked what and when.
- Wikivoyage is itself editable and sometimes wrong. Where a second source disagreed, the figure was dropped, but only hard figures got a second source; descriptive statements ("compact old centre", "efficient metro") rest on Wikivoyage alone.
- The facts are as of 2026-09-24. Wikivoyage pages have moved on for five cities; a re-review is due when the guides are regenerated.
- Search resolves a name through OpenWeather. A city is matched by its resolved name and country, so a local spelling (for example "Lisboa") gets the "no reviewed guide" state, not a guess.

## Re-reviewing

- `go run ./cmd/guides -only=<City>` writes a new entry with the new revision and title. It never sets `reviewed`, so the regenerated intro is **unpublished until someone reviews it**. A city whose revision is unchanged is skipped and keeps its review.
- To review an entry: read the article at `wikivoyage_revision` (`https://en.wikivoyage.org/w/index.php?title=<title>&oldid=<revision>`), correct or delete anything the page does not support, then set `reviewed`, `reviewed_at` (YYYY-MM-DD) and `review_note`. `go test ./internal/guides` checks the shape (title, revision, date, length, round-trip format).
- Publishing needs all of: `reviewed: true`, a non-empty intro, a positive revision, a valid article title and a `YYYY-MM-DD` review date. Anything less stays out of the page.
