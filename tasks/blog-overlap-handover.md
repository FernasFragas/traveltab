# Handover: Finish the Madeira Claim-Consensus Research

> **Superseded October 5, 2026 — do not run.** The product no longer uses AI extraction: the writers' map shows verbatim excerpts fetched live ([plan](plan.md)). Kept for the record and in case offline AI enrichment is revisited.

**Your job:** answer one question with evidence. **Do travel blogs agree on the same *advice* about Madeira places, not just name the same places?** Run local Ollama extraction and consensus over all Madeira posts, hand-check 50 claims, and write up the results. Do not build product code.

Written October 5, 2026. Product context: [one-pager](../docs/ideas/community-map.md). Earlier evidence: [overlap results](blog-overlap-results.md).

## Why This Matters
TravelTab plans a map whose place panels show a **consensus card** ("6 blogs mention this · 5 say arrive before 10am"), every claim linked to its source post. The overlap check proved blogs *mention* the same places. Only ~5/40 sampled mentions had an explicit opinion nearby. If blogs mostly list names without advice, the consensus card is thin and the plan must change before any UI is built.

## Status

| Step | State |
| --- | --- |
| Overlap check, 4 destinations (capped at 10 posts/blog) | ✅ Done — [results](blog-overlap-results.md) |
| Deep fetch: all Madeira posts | ✅ Ran — **data lost**, must re-fetch (≈10–15 min) |
| Ollama extraction trial (2 posts) | ✅ Done — findings below |
| Ollama extraction, all posts | ⛔ Stopped by the user partway; output lost |
| Consensus grouping | ❌ Not run |
| 50-claim hand check | ❌ Not done |
| Write-up | ❌ Not done |

**Nothing is committed.** Uncommitted: `tasks/blog-overlap/`, `tasks/blog-overlap-results.md`, this file, edits to `docs/ideas/community-map*.md`. Do not commit unless the user asks.

## Findings So Far

### Deep fetch (Madeira, all posts vs. 10-post cap)
- **235 posts** kept (from 2,670 scanned), vs. 52 when capped.
- Places named by ≥3 blogs: **40 → 55**; ≥4 blogs: **24 → 41**; single-source: 25 → 28.
- Single-source places promoted to ≥3 blogs: Jardim do Mar, Paul do Mar, Monte Palace (plus noisy "Achada", "Nazaré").
- **Some new matches are noise:** "Fernandes", "Infante", "Casas Próximas", "São Sebastiao". Generic or ambiguous single words; treat counts as an upper bound until cleaned.
- Kept per blog: vortexmag 66, viajecomigo 64, earthtrekkers 27 (**timed out — partial**), almadeviajante 24, saltinourhair 22, portugalist 18, viagensasolta 7, theplanetd 4, travellemming 3; nomadicmatt, thecrowdedplanet, adventurouskate 0.

### Ollama trial (2 posts — the only extraction results that exist)
- **Speed:** 19 s for a post with 4 places (865 prompt tokens in, 656 out) on an M5 Pro / 48 GB, `qwen3.6:35b` (MoE, Q4_K_M). Full run estimate: **~60–90 min** for 235 posts.
- **Output quality looked good:** 6 claims from a Salt in Our Hair hiking post, all specific and correct:
  - Pico do Arieiro–Pico Ruivo (PR1) costs €10.50 per person; €7 with a certified tour company.
  - Trails like 25 Fontes and Ponta de São Lourenço require advance booking via the SIMplifica portal.
- **The quote check works:** 5/6 quotes verified verbatim; 1 was flagged because the model stitched two sentences with "...". Flagged claims are excluded from consensus by design.
- **Most claims were practical facts (cost, booking), not opinions.** These are still valuable card content. Record the type split: it tells us whether the card is "advice" or "logistics".
- **Restaurant posts are skipped correctly** when they mention no gazetteer place (viajecomigo "Jaket" post).
- **The same claim is attached to several places** (the PR1 fee went to both Arieiro and Ruivo). Expect inflated per-place counts from multi-place sentences; account for it.

## How to Run

Run everything from a **working directory outside the repo** (fetched posts are third-party copyrighted text; don't commit them). Scripts live in `tasks/blog-overlap/`.

```bash
S=~/personal-projects/traveltab/tasks/blog-overlap
mkdir -p ~/traveltab-overlap-run && cd ~/traveltab-overlap-run

python3 $S/gaz.py          # OSM gazetteer → gazetteer.json (no args = fresh run; Overpass may 429/504 — mirrors are built in, retry)
python3 $S/wikidata.py     # adds Wikidata en/pt names + manual alias fixes (slow on purpose: 2 s/batch, honors Retry-After)
python3 $S/fetch_deep.py   # all Madeira posts → deep/<blog>.json (resumable; skips blogs already fetched)
ollama list | grep qwen3.6 # model must be present; Ollama server on localhost:11434
python3 $S/extract.py 5    # smoke test on 5 posts → claims.jsonl, claims_done.txt
python3 $S/extract.py      # full run (resumable via claims_done.txt); run in background, ~60–90 min
python3 $S/consensus.py    # → consensus.json + summary lines
ollama stop qwen3.6:35b    # free 23 GB when done
```

**Before the full run:** rerun Earth Trekkers, which timed out. Delete `deep/www.earthtrekkers.com.json` and rerun `fetch_deep.py`.

### What the scripts do
- **`extract.py`:** finds gazetteer places in each post, sends ±500 chars around each mention (≤14k chars) with a JSON schema. Each claim has `place`, `type` (verdict/timing/cost/access/duration/warning/tip), `stance` (go/skip/mixed/neutral), `claim_en`, `quote`. It then sets `quote_ok` (verbatim, accent/whitespace-normalized, ≥15 chars) and `place_ok`.
- **`consensus.py`:** per place with verified claims from ≥2 blogs, the model groups claims into `statements` and `disagreements`. **The script keeps a group only if its claim ids span ≥2 distinct blogs**; the model's grouping is not trusted on its own.

## What to Produce

### 1. Numbers (from `consensus.py` output plus your own counts)
- [ ] Claims total, verified share (`quote_ok && place_ok`), and split by `type` and `stance`.
- [ ] Places with ≥1 verified claim; from ≥2 blogs; from ≥3 blogs.
- [ ] **Places with ≥1 agreed statement (≥2 blogs)** — the headline number.
- [ ] Places with a disagreement.
- [ ] Single-blog places with a usable tip (the "local tip" card for lesser-known places).

### 2. Hand check: 50 random verified claims
For each, compare `claim_en` to `quote` and the post. Classify:
- **Correct:** the author says this about this place.
- **Wrong place:** the claim is about another place in the same sentence.
- **Overstated/misattributed:** `claim_en` says more than the quote.
- **Not advice:** history, an address, or a passing mention.

Also hand-check **10 consensus statements**: does each source quote really support the statement?

### 3. Write-up
- Add a **"Claim consensus (Madeira)"** section to [`blog-overlap-results.md`](blog-overlap-results.md): verdict first, then numbers, hand-check table, top 10 statements with blog counts, limitations.
- Update the second unchecked assumption in [the one-pager](../docs/ideas/community-map.md) ("Overlapping mentions contain overlapping claims") and the misattribution assumption, with links to the evidence.
- Keep it skimmable: key point first, short bullets, bold for the answer.

## Proposed Pass/Fail Thresholds
These are **suggestions, not agreed with the user.** State them in the write-up and flag them for confirmation.
- **Consensus is viable:** ≥15 Madeira places have ≥1 agreed statement from ≥2 blogs.
- **Extraction is trustworthy:** ≥85% of claims pass the quote check, and ≤2/50 hand-checked claims are misattributed or about the wrong place.
- **"Local tip" fallback is viable:** most single-blog places yield ≥1 usable claim.

If consensus fails but tips pass, say so: the product could pivot from "consensus" to "best tips from writers, with sources".

## Rules
- **Identified user agent only** (`TravelTab-research/0.1 …`, already set). Keep the delays. **Never work around a 403/block.** VagaMundos blocks bots; leave it out.
- **Free and open sources plus local AI only.** No paid APIs, no cloud models; the `:cloud` models in `ollama list` are not local, so don't use them.
- **Don't republish blog content.** Quotes stay in evidence files; don't commit fetched posts.
- **Don't change product code** or `tasks/plan.md`/`todo.md`. Re-planning comes after this research.
- **Report honestly:** partial runs, timeouts and noisy matches go in the write-up's limitations.

## Known Issues
- **Noisy single-word places** (see deep-fetch list above). Consider adding them to `STOP` in `extract.py` and noting it.
- **`gaz.py` resumes only when given an argument** (`python3 gaz.py x` reuses `gazetteer.json`); with no args it starts fresh.
- **`fetch.py`/`match.py` are the capped overlap check** (all 4 destinations); not needed for this task.
- **Multi-place sentences** produce duplicate claims across places; consider deduping by `(blog, quote)` when counting.
- **`report.json`** is the capped-run output; the deep-run numbers above were computed ad hoc and aren't in a file.
