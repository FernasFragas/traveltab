# Writers' Map Index Spike — Results

**Verdict:** keep the phrase/QID/position schema for PR 5, with the planned five-word sub-span cap. The measured names-only prototype is well below the proposed size and lookup limits. **PR 5 remains gated on the 55-place coverage check:** the committed research records only the aggregate of 55, not the place names needed to compare against Wikidata nearby plus filters.

Run October 6, 2026. The reproducible runner is [`writers-map-spike/run.py`](writers-map-spike/run.py). It streamed the public WordPress API responses through memory with the identified `TravelTab-research/0.1 (blog overlap check)` user agent and a 1.5-second delay between pages. It stopped on access blocks. No post body, title, or URL was written to the database or output. Only matched QIDs/names, post IDs, and offsets were indexed. The temporary database and downloaded candidate-name cache were removed after measurement.

## Measurements

| Measure | Result | Proposed bar | Outcome |
| --- | ---: | ---: | --- |
| Public posts processed | 4,800 total: viajecomigo 4,207; Salt in Our Hair 593 | Two full blogs | Met |
| Unique Madeira Wikidata candidates | 357 QIDs, 593 folded en/pt names | — | — |
| Indexed name occurrences | 2,892 | — | — |
| SQLite size | 679,936 bytes (0.68 MB decimal) | — | — |
| Rows per 1,000 posts | 602.5 | — | — |
| Linear projection to 20,400 posts | 2.89 MB | ≤300 MB | Pass |
| Warm indexed phrase lookup, p50 of 101 | 0.034 ms (`madeira`, up to 200 rows) | ≤200 ms | Pass |
| Precision sample | 37/40 = 92.5% in the recorded research | ≥90% | Pass on recorded evidence; not reproduced |
| 55 research places returned by nearby query + filters | Not measurable from retained files | Record count | Unavailable; PR 5 gate remains |

The size benchmark used a temporary SQLite table keyed by post ID, QID, folded phrase, and character position, with phrase and QID lookup indexes. It stores mention occurrences, not post content. The size projection is a linear extrapolation from these two blogs; it does not include extra rows for generated sub-spans. Keeping sub-spans capped at five words bounds that additional growth.

## Evidence limits and next step

The recorded precision result is in [`blog-overlap-results.md`](blog-overlap-results.md): 37 of 40 hand-reviewed matches were the right place after manual alias fixes. The 40 individual samples are not present, so this spike could not rerun the same precision check against the current candidate list. The ≥90% threshold is supported by the saved aggregate, not by a new sample.

The recorded deep run reports 55 Madeira places mentioned by at least three blogs, but it does not retain their names or a Wikidata candidate fixture. Two-blog crawling cannot reconstruct the 55-place set. Restore a names-only list of those 55 places or repeat the cross-blog match, then count how many survive the nearby query and type/infrastructure filters before PR 5 proceeds.

**Design decision:** retain the current phrase/QID/position shape; the measured footprint leaves substantial room under 300 MB, and lookup is far below 200 ms. The available evidence does not yet establish candidate coverage, so do not treat the whole PR 2 acceptance as passed.
