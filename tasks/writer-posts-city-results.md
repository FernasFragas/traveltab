# Spike results: posts from other cities under a place

**Date:** 2026-10-13. No production code was changed. Plan: [writer-posts-city-plan.md](writer-posts-city-plan.md).

## Key point

**The matcher judges a post by one nearby "Paris", not by what the post is about.** A Ponta Delgada guide that says "…like the Louvre in Paris" passes the "nearby" rule, so it is listed under Paris's Louvre. Shrinking the window can't fix this, because the name really is next to the place.

## What the screenshot shows

"Louvre - Rivoli" lists 2 posts. Both are **not about Paris**:

| Post | Title names Paris? | "Paris" in post | "Louvre" in post | Gap |
|---|---|---|---|---|
| 1651 Alma de Viajante, Ponta Delgada | no | **1×** | 1× | 74 chars |
| 12726 The Planet D, Toronto | no | **1×** | 1× | 48 chars |

(Positions decoded from `writer_names` in `launch-check.db`; the posts are about São Miguel and Toronto.)

## Why they pass (`inDestinationContext`, `match.go`)

A mention counts if **any** of these holds:
1. a destination name is in the **title**: not here
2. a destination name is within **W = 1,000** characters: **this one**, both gaps are under 100
3. a destination name is in the **same day section**: not here

So rule 2 is the leak. It checks "is the word Paris close by", not "is this a Paris post".

Two causes stacked on the screenshot:
- **Wrong place:** "Louvre" is a Wikidata alias of the metro station as well as the museum. **Fixed earlier today** in `placeAliases` (shared alias goes to the place with more sitelinks). Those two posts still count for the **museum**.
- **Wrong city:** this spike, not fixed.

## How big is it (Paris, `launch-check.db`)

| Posts containing "Paris" | **926** |
|---|---|
| …with Paris in the title | 91 |
| …without it | **835** |
| …with exactly **one** Paris mention | **540** |

Sampled one-mention titles: a writing workshop, EU flight rights, an eSIM review, Mauritius, India, a Prague guide, "The 8 Best Old Towns in Europe". All pass today.

## Candidate rules (posts kept, of 926)

| Rule | Posts kept |
|---|---|
| Today (nearby anywhere) | 926 |
| Title only | 91 |
| Title **or** ≥2 mentions of the destination name | 386 |
| Title **or** ≥3 mentions | 260 |
| Title **or** ≥5 mentions | 171 |

Precision and recall are **not measured yet**; these are counts only.

## This is the known open gap

[writer-context-plan.md](writer-context-plan.md) added the title/nearby/same-day rule. It shipped with its targets **not met** (Lisbon precision 23.5%). It measured the window size W, which doesn't help here. What it didn't try is a **post-level** signal.

## Recommendation

Add a rule that the **post** must be about the destination: **title names it, or it names it at least N times**, then keep nearby/same-day for placing mentions within such posts. Measure N = 2, 3, 5 on the same 40-mention judged samples as before (Paris added; Lisbon, Madeira, Porto reused).

## Risks
- Guides that name the city once and still cover it (a "Weekend in Europe" list) lose recall. Accepted trade, measured.
- Names in other languages ("Lisboa") must count toward N, or Portuguese posts are dropped. Already a known gap in `contextNames`.
- Mentions of a city by a landmark only ("the Louvre", never "Paris") won't count. Acceptable.
