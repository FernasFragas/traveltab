# TravelTab Writers' Map

> **Refined October 6, 2026 — "always ready", AI summaries in the MVP.** The map is useful instantly for any searched location. Plan: [`tasks/plan.md`](../../tasks/plan.md) · tasks: [`tasks/todo.md`](../../tasks/todo.md) · evidence: [overlap results](../../tasks/blog-overlap-results.md). Earlier versions (community map; offline consensus map; live-per-search map; on-demand excerpts) are in git history.

## Problem Statement
How might we show any traveler, for any place they search, what is worth seeing and what experienced travel writers say about it — instantly, in English, with links back to the writers?

## Recommended Direction
**Search any destination, get today's page with one change: the Waze map becomes a writers' map that is always ready.** Weather, guide, planner and stays stay as they are.

**The map has three layers, so it is never empty and never waits on blogs:**

| Layer | What the user sees | Source | Ready |
| --- | --- | --- | --- |
| **1. Base** | Notable places with English description and photo; the TravelTab day plan | Wikidata nearby places (CC0), Commons photos, OSM tiles, the existing planner | Any location; one ~2–3 s query on the first-ever search of a place, then cached |
| **2. Writers** | Pins writers mention, writer count, links to their posts, writers' itineraries | A **local index of every post** on the allowlisted blogs (~20,400 posts), synced monthly by the server | Instant lookup; no blog requests during a search |
| **3. Summaries** (in MVP, ships when LLM Gateway is ready) | An English "what writers say" summary per place, labelled *AI summary* | Monthly job on the server → **LLM Gateway** (owner's own project, free to TravelTab) | Places with writers, after the monthly run |

**The writers index stores place names, not text.** For each post: title, link, language, and the place-name phrases with their positions and day headings. Searching a destination matches its Wikidata places against the index — a database lookup.

**Summaries replace excerpts.** The panel shows no quoted blog text. Once a month the server re-reads the posts that mention each place, sends the relevant passages (English or Portuguese) to LLM Gateway, and stores one 2–3 sentence English summary. Post text stays in memory only.

**Until LLM Gateway is ready:** panels show description, photo, "Mentioned by N writers" and the writers' post links (with language label). Nothing else waits on it.

**Itineraries:** your TravelTab day plan as numbered pins; writers' "Day 1 / Dia 1" posts as routes, pre-parsed at sync. **"Plan with these places"** opens the TravelTab planner with the writer's stops included.

**Journey:** search any place → map shows notable places and writers' pins at once → open a pin: description, photo, writers' summary, links → save to *Want to visit* → plan the trip, see the day plan on the map → try a writer's itinerary in the planner.

## Key Assumptions to Validate
- [x] **Blogs cover the same places** — 8–11 blogs per Portuguese destination; matcher precision 37/40. [Results](../../tasks/blog-overlap-results.md)
- [x] **Open data answers any location fast** — Wikidata "within 30 km of Funchal": 2.4 s, 200 places, 183 with photos, 194 with English descriptions (October 5).
- [ ] **A names-only index stays small** — spike on two full blogs; projected size for 20k posts must fit the Fly volume (proposed ≤300 MB).
- [ ] **Wikidata names match writers as well as OSM + Wikidata did** — re-run the 40-match precision check (proposed ≥90% correct place).
- [ ] **Base pins are worth visiting, not just notable** — airports and municipalities rank high; check a type filter on 4 destinations.
- [ ] **Monthly server sync fits the $5 goal** — measure initial crawl bandwidth/time and monthly delta.
- [ ] **AI summaries are faithful and useful** — never tested: the claim-extraction check stalled for lack of a model. Hand-check 20 Madeira summaries against their posts (no invented facts; PT sources summarized correctly).
- [ ] **LLM Gateway handles the monthly volume** — places with writers × posts per place; measure once the gateway exists.
- [ ] **Writers' itineraries parse reliably** — 20 known posts.
- [ ] **Bloggers accept indexing** — track blocks and takedown requests.

## MVP Scope
**In:**
- New map card on **every** destination; Waze removed; "View larger map" kept.
- Base layer: Wikidata places within ≈40 km (islands/regions larger), filtered to visitable types, max ~200 ranked by writer count then Wikipedia sitelinks; description, Commons photo with credit; warmed when a city is picked from search suggestions.
- Writers index: allowlist of 12 blogs; initial crawl, then a monthly sync (changed posts, deleted posts, Wikidata refresh); names, positions, day headings only.
- Writer pins above base pins; panel with writer count, writer links (language label) and the AI summary when it exists.
- **AI summaries** via LLM Gateway in the monthly job — built and tested against a fake; switched on when the gateway is ready.
- Day plan layer; writers' itinerary layers; "Plan with these places".
- *Want to visit* (`localStorage`).
- **Caches never expire.** The monthly run updates rows in place (no history, no duplicates) so the database stays bounded; takedowns and deleted posts still purge.
- Honest states: base only (no writers here), summary not yet available, a writer's site unavailable.

## Not Doing (and Why)
- **Fetching blogs during a search or when a panel opens** — the index and stored summaries answer instantly; blogs are contacted only by the monthly sync.
- **Storing or showing quoted blog text** — names, positions and AI summaries only.
- **Portuguese text in the UI** — all reader-facing text is English; PT writers appear as links until summarized.
- **Human review of every summary** — impossible at this scale; labelled *AI summary*, with a sample check before launch and an override to hide one.
- **Automatic source discovery, paid APIs** — hand-edited allowlist, free sources only (LLM Gateway is the owner's own, at no cost to TravelTab).
- **Blog photos or full posts** — "all rights reserved".
- **Frozen copies of writers' itineraries** — "Plan with these places" re-plans with today's forecast instead.
- **Notifying bloggers before launch** — decided no; blocks and takedowns are honored.
- **User comments, accounts, likes, moderation** — later.
- **Waze fallback** — removed everywhere.

## Decisions (October 6, 2026)
- **Search area:** ≈40 km for cities; islands/regions larger via overrides; cap ~200 places ranked by fame.
- **Caches:** never expire; refreshed monthly by updating rows in place.
- **Language:** everything shown is English; summaries translate PT sources.
- **AI:** in the MVP, on the server, through LLM Gateway. **Blocked on that separate project** — the rest ships without it.
- **Writers' itineraries:** "Plan with these places" into the existing planner; no frozen snapshot.
- **Bloggers:** not notified before launch.
- **Budget:** US$5/month maximum, everything included.

## Open Questions
- **When is LLM Gateway ready, and what is its API?** Sets when PR 9c can start.
- **Launch date / hour budget:** not set (≈89 h estimated).
