# Blog Overlap Check — Results

**Verdict: the consensus hook survives for all four candidate destinations — but only for well-known places.** Each destination is covered by 8–11 blogs, and 13–39 places per destination are mentioned by ≥3 independent blogs. Roughly 40% of mentioned places have a single source, so "hidden gems" will mostly show "1 blog", not consensus.

Run October 5, 2026. Tests the first assumption in [the one-pager](../docs/ideas/community-map.md). Scripts and raw report: [`blog-overlap/`](blog-overlap/).

**Follow-up, same day:** the second assumption (*overlapping mentions contain overlapping claims*) was attempted on Madeira and **could not be completed** — no usable extraction model. See [Claim Consensus](#claim-consensus-madeira--not-validated).

## Results

| Destination | Blogs | Posts | Places mentioned | ≥2 blogs | **≥3 blogs** | ≥4 blogs | Single-source |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Lisbon | 9 | 49 | 97 | 51 | **29** | 21 | 46 |
| Porto | 11 | 52 | 63 | 29 | **20** | 13 | 34 |
| Madeira | 8 | 52 | 77 | 52 | **39** | 23 | 25 |
| Azores (São Miguel) | 8 | 51 | 36 | 21 | **13** | 12 | 15 |

**Most-agreed places:** Alfama, Bairro Alto (9 blogs) · Castelo de São Jorge (8) · Sé do Porto, Livraria Lello, Bolhão, São Bento (6) · Cabo Girão, Pico Ruivo (7) · Furnas (8), Sete Cidades (7).

**Madeira is the strongest candidate** — highest ≥3 count and lowest single-source share (32%). Azores has the thinnest long tail.

## Precision Check (40 random matches, hand-reviewed)
- [x] **37/40 correct place.** 3 wrong entities, all from Wikidata aliases (e.g. "Ribeira Grande" listed as an airport alias). Fixed and recounted.
- [ ] **~30/40 are recommendation context.** ~7 are the right place in a non-recommending context: addresses, hotel names, history passages, transport.
- [ ] **Only ~5/40 carry an explicit opinion within ±70 characters** ("the village we liked most", "entrance fees are hefty"). Mention ≠ claim — Ollama extraction must show whether consensus *claims* exist, not just consensus *mentions*.

## Findings That Change the Plan
- **VagaMundos blocks identified bots.** Its API answered a plain request earlier the same day, then returned 403 to `TravelTab-research` UA. Not circumvented. Also blocked or missing: viajarentreviagens.pt, lisbonlisboaportugal.com (403), alongdustyroads.com, travelingwithjosh.com (404). **Sources can switch off access; the source list must tolerate churn.**
- **Open WordPress APIs are common.** 13 of ~25 probed blogs expose `/wp-json/wp/v2/posts`; 12 contributed posts here (almadeviajante, viajecomigo, vortexmag, viagensasolta, portugalist, nomadicmatt, theplanetd, earthtrekkers, saltinourhair, travellemming, thecrowdedplanet, adventurouskate).
- **Entity resolution is real work.** Clérigos appeared as three places (church + tower, tower, church); the same name can refer to different churches. Expect a manual override file.
- **English names matter.** Before adding Wikidata labels, Sete Cidades scored 1 blog instead of 7 and Belém Tower/Livraria Lello/Ponte Luís I were undercounted.
- **WordPress search is shallow.** Relevance search ranks body mentions; Porto needed up to 150 results to find 10 title matches per blog.

## Method
1. **Gazetteer** (`gaz.py`): OSM places with a `wikidata` tag in a bounding box per destination (tourism, historic, natural, leisure, markets, places of worship, neighbourhoods/villages, bridges, lighthouses). Lisbon 565, Porto 298, Madeira 221, São Miguel 113.
2. **Names** (`wikidata.py`): OSM `name`/`name:en`/`name:pt`/alt names + Wikidata en/pt labels and aliases; manual fixes from the precision check.
3. **Posts** (`fetch.py`): up to 10 posts per blog per destination whose title names the destination; identified user agent, 1.5 s between requests.
4. **Matching** (`match.py`): accent-folded whole-word match; single-word names case-sensitive; generic words and destination names excluded; a name inside a longer matched name doesn't count; duplicate names merged. A place counts once per blog.

## Limitations
- **Mentions, not claims** — the claim-consensus follow-up was attempted and **could not be completed**; see [Claim Consensus](#claim-consensus-madeira--not-validated) below.
- **10 posts per blog per destination cap** — counts are a lower bound for prolific blogs.
- **Uneven blog fetch depth.** Porto used deeper search (up to 150 results); other destinations used the first 20 results per term. `fetch.py` now uses the deep search everywhere, so reruns may show higher counts.
- **Transient failures:** thecrowdedplanet (Lisbon) and nomadicsamuel (Lisbon, Porto) timed out; nomadicsamuel contributed no posts.
- **Gazetteer only knows places with Wikidata links** — restaurants, shops and small trails are missed (e.g. LX Factory was not matched).
- One reviewer, 40 samples: precision figures are indicative.

## Claim Consensus (Madeira) — Not Validated

**Verdict: the check could not be completed, so the second assumption is still unproven.** The corpus was rebuilt and is sound; **no claims were extracted**, because no extraction model was available at acceptable cost. Whether blogs agree on the same *advice* — not just the same places — remains unanswered. **Do not design the consensus card until this number exists.**

Attempted October 5, 2026. Tests the second assumption in [the one-pager](../docs/ideas/community-map.md).

### What did work

| Step | Result |
| --- | --- |
| Madeira gazetteer | **221 places** — Overpass 504'd/timed out on all 3 mirrors, succeeded on retry |
| Wikidata aliases | avg **1.25 → 2.66** per place (English names matter, as before) |
| Deep fetch | **231 posts** / 12 blogs, vs 235 in the earlier run |

Per blog: vortexmag 66 · viajecomigo 64 · almadeviajante 24 · earthtrekkers 23 (partial) · saltinourhair 22 · portugalist 18 · viagensasolta 7 · theplanetd 4 · travellemming 3 · nomadicmatt, thecrowdedplanet, adventurouskate 0.

Earth Trekkers timed out on the re-fetch too (23 posts; 27 expected). The error is a read timeout, not a block.

### Why extraction did not run

`qwen3.6:35b` — the model the earlier trial used — **is no longer installed**; only `:cloud` models remain. `glm-5.3:cloud` was tried with the user's approval and is **unusable here**:

| Post | Places | Output tokens | Time | Verified claims |
| --- | ---: | ---: | ---: | ---: |
| almadeviajante 31897 | 4 | 1,029 | 9 s | 0 |
| almadeviajante 93336 | 5 | 36,900 | 299 s | 0 |
| almadeviajante 31834 | 7 | 34,477 | 270 s | 0 |
| viajecomigo 7023 | 25 | 56,650 | 439 s | **0** |
| viajecomigo 78874 | 17 | 53,992 | 358 s | **0** |
| viajecomigo 57593 | 13 | 1,093 | 8 s | 0 |

Two failure modes:
- **The schema is ignored.** The prompt asks for `{"claims":[…]}`; the model repeatedly returned a place-keyed object (`{"Coroa": [], "Monte": [], …}`), so nothing was ever parsed as a claim. A stricter prompt *and* shape normalisation did not fix it.
- **Reasoning cannot be suppressed.** `think:false` is ignored — 34k–57k output tokens per post, nearly all visible chain-of-thought.

Projected full run: **~19 hours** for 231 posts, against 60–90 min for the local model.

### Still unknown

- **Places with ≥1 agreed statement (≥2 blogs) — the headline number. Not measured.**
- The advice-vs-logistics type split, disagreement counts, single-blog tip yield.
- The 50-claim hand check and 10-statement check — there are no claims to check.
- Best prior evidence is still the 2-post trial: 6 claims from one Salt in Our Hair hiking post, 5/6 quotes verbatim, mostly practical facts (costs, booking) rather than opinions.

### Limitations

- **Extraction would have used a cloud model, not local Ollama** — a deliberate deviation from the "free/open sources plus local AI only" rule, to avoid a 23 GB download. It did not work. No blog text was retained or republished.
- The deep-run *mention* counts were **reproduced** on the rebuilt 231-post corpus (re-checked October 5): 102 places mentioned, **55 by ≥3 blogs, 41 by ≥4**, 27 single-source — the same ≥3/≥4 figures as the 235-post run. They come from the place matcher, not from extraction.
- `extract.py` and `consensus.py` now carry cloud-compatibility patches (a JSON-only instruction, a tolerant parser, enum coercion, plus a new `stats.py`). These are no-ops for a schema-honouring local model.
- One destination, one reviewer, unvalidated tooling.

### Next Check

1. **`ollama pull qwen3.6:35b`** — the only blocking step. Then `extract.py` (~60–90 min), `consensus.py`, and the hand checks.
2. **If consensus fails but tips survive, say so:** the product pivots from "consensus" to "best tips from writers, with sources".
3. Proposed thresholds (suggested, **not agreed with the user**): consensus viable at ≥15 places with ≥1 agreed statement; extraction trustworthy at ≥85% quote-check pass and ≤2/50 misattributed.
