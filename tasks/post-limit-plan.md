# Writers index: cap at 100 posts

**Status:** planned 2026-10-11, not started. Tasks: [post-limit-todo.md](post-limit-todo.md).

**Goal:** the writers index holds **at most 100 posts in total**: the **most popular posts of each active blog**, re-chosen on every monthly sync. Posts that drop out are purged.

**When:** after PR 7 merges and **before 8b** (live measurement), since it changes crawl volume and coverage. It owns PR 5 files (`internal/writermap/sync.go`, `internal/adapters/api/wordpress.go`), which no open Writers' Map task edits. See [todo.md](todo.md).

## Owner decisions (2026-10-11)
- **100 in total**, across all blogs.
- **The most popular posts of each blog.**
- **Rolling:** each monthly run re-picks the set; posts that fall out are purged.

## Assumptions: confirm in L1
- **Even share:** 11 active blogs → **9 posts each** (99). If a blog has fewer usable posts, its unused share goes to the next blogs in allowlist order, so the cap is still 100. Blocked or removed sources get no share.
- **`MaxIndexedPosts = 100`** is a constant, not an environment variable (nothing asked for configurability).

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

## Design (after L1)
1. **Selection (`wordpress.go`):** a new client method returns a blog's post IDs ranked by the chosen signal. It is paged, uses the existing host limiter, robots.txt check and user agent, and stores no post text.
2. **Sync (`sync.go`):**
   - compute each blog's share
   - fetch content only for the chosen IDs, through the existing `UpsertPost`/`ExtractPost`
   - purge indexed posts no longer chosen, using the existing per-post delete
   - the cursor-based incremental crawl goes away, since every run re-picks the set
3. **Recompute:** the existing `SummaryStep` runs after the sync, so stored destination results refresh.
4. **Docs:** update the writers' map section of `docs/maintenance.md` (100-post cap, how posts are picked, purge on drop-out).
