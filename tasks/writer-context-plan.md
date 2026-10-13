# Writer post context: stop passing mentions from counting

**Status:** done (2026-10-13), targets not met. W = 1,000 shipped as the best W; São Vicente (Lisbon) is fixed, but Lisbon precision stays ~24% and recall loss 64% on the sample (see `docs/tdd/writer-context.tdd.md`, **Outcome**). Owner decisions recorded 2026-10-13 (see **Decisions**). Tasks: [writer-context-todo.md](writer-context-todo.md).

**Goal:** a place's "View writers' posts" lists only posts that are about that place's destination. A Madeira guide that says "fly from Lisbon" no longer counts as a Lisbon post for São Vicente (Lisbon).

**Owns:** `internal/writermap/match.go`, `match_test.go`, `internal/adapters/sqlite/writerdestination.go` (the result key only), `writerdestination_test.go`, `docs/tdd/writer-context.tdd.md`. It doesn't touch the nearby-rings files or `destinationphoto.go`.

## The problem (measured 2026-10-13)

- **Today's rule** (`inDestinationContext` in `match.go`): a mention counts if its post contains the destination's name **anywhere**. In a post with day sections, it counts if the name is in the same day section **or anywhere outside the days**, such as the introduction.
- **Example:** "18 Things to do in Madeira, Portugal" (Salt in our Hair) mentions "Lisbon" (for flights) and "São Vicente" (the Madeira town). Under today's rule it counts for the São Vicente parish in Lisbon. Its 4 posts and 2 writers are all from Madeira.
- **Scale** (`launch-check.db`):

| Posts that count as "about Lisbon" | **2,056** |
|---|---|
| …with Lisbon or Lisboa in the title | 428 |
| …with Madeira in the title | 19 |
| …with Porto, Algarve or Sintra in the title | 175 |

- **Why it wasn't caught:** 6a measured precision on Madeira only. Writers rarely mention Madeira in passing, but they mention Lisbon (a capital and airport hub) all the time.

## Fix: option 3, subject **or** nearby

A mention counts when **any** of these holds:

1. **Subject:** a destination name is in the post's **title**.
   - The title is the first line of the indexed text (`ExtractPost`), so this is a phrase position check: a context phrase with `Start < len(title)`. No re-index needed.
2. **Nearby:** a destination name is within **W characters** (runes) of the mention, before or after.
   - The index has no paragraph boundaries; adding them would need a full re-index. A distance window works on the current data.
3. **Same day:** in a post with day sections, a destination name is in the **same day section** (today's rule, unchanged).

**Removed:** "the name appears anywhere in the post," and "the name appears outside any day section."

**Destination names** stay as today (`contextNames`: the city plus every override area containing it, e.g. "Madeira" and "Funchal").

**W is measured, not guessed.** C1 compares W = 300, 600 and 1,000 on the real index, and C2 uses the smallest that meets both targets.

**If no W meets both targets:** ship the **best W anyway** and record the gap in the evidence. Best = the highest precision among the W values within the recall limit; if none is within it, the W with the smallest recall loss. No extra rule.

## Stored results must be recomputed

Destination results **never expire**. They're stored under `writermap:<basePlaceCacheVersion>:…` (`writerdestination.go`).
- Add a **matcher version** to that key (e.g. `writermap:v3:m2:…`), so every destination recomputes its matches on its next visit.
- **Don't** bump `basePlaceCacheVersion`. That would also re-fetch every Wikidata gazetteer, which is slow for big cities (see nearby-rings).

## Measuring (before and after)

Run on a **temp copy** of `launch-check.db`, offline (gazetteers from its cache, no new Wikidata calls where cached).

| Destination | Why |
|---|---|
| **Lisbon** | The bug: a capital mentioned in passing |
| **Madeira / Funchal** | The 6a baseline (≥90%); must not regress |
| **Porto** | A second big city, mentioned in passing |

Per destination and per rule (today, W = 300 / 600 / 1,000):
- **Totals:** mentions, posts, pins with ≥1 writer.
- **Precision:** 40 random mentions, each judged "right place" or "wrong place" from the post. One sample per destination, drawn with a fixed seed, so before and after judge the same mentions where both rules keep them.
- **Recall loss:** of today's mentions judged *right*, how many the new rule drops.

The sample sheet lists post title, URL, place, its QID, the text window and the verdict. **The agent judges every mention**, reading the public post with the app's `UserAgent`, and writes a one-line reason per verdict. The owner spot-checks the sheet.

## Decisions (owner, 2026-10-13)

| Decision | Choice |
|---|---|
| W | **Measure** 300 / 600 / 1,000; use the smallest that meets both targets |
| Precision target | **≥90%** for Lisbon, Madeira and Porto |
| Acceptable recall loss | **≤10%** of today's right mentions per destination |
| If no W meets both | **Take the best W anyway** (defined above) and record the gap; don't stop |
| Who judges the samples | **The agent alone**, with a reason per verdict; the owner spot-checks |
| Destinations measured | **Lisbon, Madeira / Funchal, Porto** |
| Refreshing stored results | **Matcher version in the result key**; gazetteers stay cached, no warm-up job |

## Risks

- **Guides that name the city only in the introduction** lose places further down. Their title usually names the city, which rule 1 covers. C1 measures what's left.
- **Multi-city posts without day headings** (e.g. "Portugal in 10 days") keep only places near a city's name. This is the intended trade.
- **Title in another language:** the title is matched against the same context names as the body (Lisbon and Lisboa are both destination names today; C1 confirms).

## Done when

- São Vicente (Lisbon) no longer lists the 4 Madeira posts; a test reproduces the case first.
- Lisbon, Madeira and Porto: ≥90% precision and ≤10% recall loss on the same 40-mention samples, or the best W shipped with the gap recorded.
- Existing 6a acceptance tests still pass ("Bom Jesus do Monte" in a Braga post, the São Jorge / Monte / Ponta do Sol / Teleférico pins).
- Stored destination results recompute after the deploy; the gazetteer cache isn't refetched.
- `make test lint build` passes; evidence in `docs/tdd/writer-context.tdd.md`.
