# Writers index: cap at 100 posts

**Status:** planned 2026-10-11; L1 done, owner decisions recorded, L2–L4 not started. Tasks: [post-limit-todo.md](post-limit-todo.md).

**Goal:** the writers index holds **at most 100 posts in total**: each blog's **most-commented destination posts**, split proportionally (see **Owner decisions**), re-chosen on every monthly sync. Posts that drop out are purged.

**When:** after PR 7 merges and **before 8b** (live measurement), since it changes crawl volume and coverage. It owns PR 5 files (`internal/writermap/sync.go`, `internal/adapters/api/wordpress.go`), which no open Writers' Map task edits. See [todo.md](todo.md).

## Owner decisions (2026-10-11)
- **100 in total**, across all blogs. Fixed constant `MaxIndexedPosts = 100`.
- **Rolling:** each monthly run re-picks the set; posts that fall out are purged.
- **After L1** ([results](post-limit-results.md)):
  1. **Signal:** only **destination posts**, ranked by the **recent-comments tally**, with the **newest** destination posts as fill.
  2. **Destination post** = a `search=` hit **whose title or URL slug names a known place**, from the matcher's existing place names (destination areas + Wikidata labels), not a hand-written list.
  3. **Share: proportional** to each blog's destination posts. Largest-remainder rounding, **minimum 1** for any blog with at least one destination post (assumed; owner may drop the minimum).
  4. **No readable comments** (Viaje Comigo, Portugalist: 404) → newest destination posts.
  5. **Lower coverage accepted**; L4 measures it against PR 2's 24/49 and the owner reviews before 8b.
- **After L3** (2026-10-11):
  6. **Blocked blog:** its indexed posts stay and **count inside the 100** (hard cap holds; pins survive a temporary 403). Permanent removal stays the `removed` takedown.
  7. **Sudden empty search:** a blog that had indexed posts but now returns **0 candidates** keeps its rows and logs a warning; retried next run.
- **After L4** (2026-10-11):
  8. **International blogs with 0 posts:** accepted. Slots follow cached destinations, so those blogs appear once their cities are cached.
  9. **Empty guard:** decision 7 triggers on **0 destination candidates** (after the title/URL filter), not 0 search hits.
  10. **Overshoot after a mid-run failure:** after the run, if the total is over 100, **trim** the lowest-ranked rows of the blogs that did update until it's 100.

## The hard part: "popular" has no free, standard signal
**The WordPress REST API exposes no view counts.** `orderby` supports date, modified, title, relevance and similar, but not popularity. Jetpack stats need the site owner's login. **L1 is a spike** to choose a signal from free public data:

| Candidate | How | Cost | Risk |
|---|---|---|---|
| **Recent comments tally** | Page `/wp/v2/comments?_fields=post` (newest first) and count comments per post | About 10 requests per blog for the last 1,000 comments | Comments may be closed or empty on some blogs, and favours recent posts |
| **Comment count per post** | `/wp/v2/comments?post=ID&per_page=1` → `X-WP-Total` | 1 request **per candidate post**, which means thousands on big blogs | Too slow unless candidates are pre-filtered |
| **Sticky posts** | `/wp/v2/posts?sticky=true` | 1 request | Editor-curated, often 0–5 posts |
| **Fallback: newest** | `orderby=date&order=desc` | 1 request | Not popularity |

**Recommendation going in:** a recent comments tally, with sticky posts first, falling back to newest when a blog has neither. L1 confirms this on the real blogs, and **the owner picks**.

## Main risk: coverage drops sharply
- PR 2 measured **24/49** Madeira places on the full index. Nine posts per blog will match far fewer.
- **International blogs** (nomadicmatt, theplanetd, earthtrekkers…): their most popular posts are probably **not about Portugal**, so their 9 slots may add **zero pins**.
- **Option for the owner (in L1):** take "most popular **among posts matching the destinations**" by using WP `search=` on destination names (Madeira, Açores, Lisboa…) before ranking. This adds 1–2 requests per destination per blog.
- L4 measures coverage before and after, so the trade-off is visible before launch.

## Design
1. **Client (`wordpress.go`), 3 small read methods**, all paged, using the existing limiter, robots.txt check and user agent; no post text:
   - `SearchPosts(term)` → IDs, titles, slugs, dates (`search=`, `_fields=id,title,slug,date,link`)
   - `RecentCommentCounts(pages)` → post ID → count, from `/wp/v2/comments?_fields=post` (newest first, ≤10 pages); a 404 means "no comments" for that blog, not an error
   - `PostsByID(ids)` → full posts for extraction, used only for the chosen IDs
2. **Selection (`internal/writermap/`, new `selection.go`): a pure function**, no I/O:
   - destination posts = search hits whose title or slug names a known place (matcher's place names, accent- and case-insensitive)
   - per-blog share: proportional, largest remainder, minimum 1, total exactly ≤100
   - within a blog: by comment count, then newest as fill and tie-break; no comments → newest
3. **Sync (`sync.go`):** search terms → candidates → comment counts → `Select` → fetch only the chosen posts → `UpsertPost`/`ExtractPost` → purge indexed posts not chosen. The cursor crawl and deleted-posts sweep go away (every run re-picks). Keep blocked/removed handling and `SummaryStep`.
4. **Search terms:** destination names and aliases from `writerdata/overrides.json` areas plus the cities the planner caches; 1 request per term per blog.
5. **Docs:** writers' map section of `docs/maintenance.md`.
