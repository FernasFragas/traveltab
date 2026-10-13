# Writers' posts: only posts about the searched city

**Status:** implemented; owner confirmed N=3. Evidence: [writer-posts-city.tdd.md](../docs/tdd/writer-posts-city.tdd.md). Spike: [writer-posts-city-results.md](writer-posts-city-results.md). Tasks: [writer-posts-city-todo.md](writer-posts-city-todo.md).

**Goal:** "View writers' posts" under a place lists only posts about that place's destination. The Louvre in Paris no longer lists a Toronto or Ponta Delgada guide.

**Owns:** `internal/writermap/match.go`, `match_test.go`, `internal/writermap/source.go` (`contextNames` only), `writerdata/overrides.json`, `internal/adapters/sqlite/writerdestination.go` (the result key only), `writerdestination_test.go`, `docs/tdd/writer-posts-city.tdd.md`.

## Fix

A post counts for a destination only if **it is about it**:
- **Title** names the destination, **or**
- the post names it **≥ N times** (any destination name, e.g. Lisbon and Lisboa).

Then today's rules decide which places in that post count (nearby W = 1,000, same day section). Both rules stay as they are.

**N is measured, not guessed:** compare 2, 3 and 5; take the smallest that meets the targets.

## Decisions (owner, 2026-10-13)

| Question | Choice |
|---|---|
| Which names count | The destination's **own names** (`contextNames`), no extra areas |
| Local spellings (Lisboa) | **In scope**: a small curated list in `writerdata/overrides.json` (`destination_aliases`, e.g. Lisbon → Lisboa) added to the destination names. Only the measured cities get entries; others unchanged |
| Targets | **Favour precision**: ≥90% precision, up to **25%** recall loss |
| If no N meets both | Ship the best N (highest precision within 25% loss), record the gap |

## Stored results must be recomputed

Add a **matcher version** to the result key, as writer-context did, so destinations recompute on next visit. Don't bump `basePlaceCacheVersion` (it would refetch gazetteers).

## Risks
See the spike. Main one: single-mention guides of the right city drop out. The measured recall loss shows how many.

## Done when
- A test reproduces the screenshot first: a post with Paris once, not in the title, near "Louvre", is **not** listed for Paris.
- Lisbon posts titled or written with "Lisboa" count as Lisbon posts.
- Paris, Lisbon, Madeira and Porto meet the targets, or the best N ships with the gap recorded.
- Existing writer-context tests still pass.
- `make test lint build` passes; evidence in `docs/tdd/writer-posts-city.tdd.md`.
