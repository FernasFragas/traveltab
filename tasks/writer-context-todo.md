# Writer post context — Tasks

Source of truth for progress. Read [the plan](writer-context-plan.md) first.

**Status (2026-10-13):** C1–C4 done. W = 1,000 shipped as the best W; the 90% precision and 10% recall targets are **not met** (Lisbon worst). See the TDD report's **Outcome**.

**How to read this file**
- Tasks run in order C1 → C4. Decisions are recorded in the plan; no owner input is needed between tasks.
- **Owns** = the only files the task may edit.
- Start from an up-to-date branch (`git fetch`, `git status -sb`).
- TDD per [AGENTS.md](../AGENTS.md) §5: see each test fail for the intended reason first.
- Measurements use a **temp copy** of `launch-check.db`. Any live request (reading a public post to judge a mention) uses the app's `UserAgent`, never personal details. `make test` stays offline.
- No commits or pushes. The owner commits.

**User journeys**
- *As a traveller opening a Lisbon place, I want its posts to be about Lisbon, so that I don't read Madeira guides for a Lisbon parish.*
- *As a traveller opening a Madeira place, I want to keep the posts I get today, so that the fix doesn't empty the map.*

---

### C1. Measure today's rule and the candidates
- [x] Done (2026-10-13): totals, three judged 40-mention samples, W = 1,000 chosen as best W (no W met both targets).

**Owns:** `docs/tdd/writer-context.tdd.md`, a temporary measuring test or command (deleted afterwards)

- For Lisbon, Madeira/Funchal and Porto, on a temp copy of `launch-check.db`:
  - today's totals: mentions, posts, pins with ≥1 writer;
  - the same totals with subject + nearby at W = 300, 600 and 1,000.
- Draw **40 random mentions per destination** from today's rule (fixed seed). Write a sample sheet: post title, URL, place, QID, text window, which candidate rules keep it.
- **Judge** each mention right or wrong from the public post, with a one-line reason. The owner spot-checks.
- From the verdicts, report per rule: precision, and recall loss (right mentions dropped).
- **Pick W:** the smallest that meets ≥90% precision and ≤10% recall loss for all three. If none does, the best W as defined in the plan, with the gap recorded.

**Verify:** the evidence doc has the totals table, the three judged samples, precision and recall loss per rule, and the chosen W with the reason.

### C2. Subject-or-nearby context rule
- [x] Done (2026-10-13): title, nearby (W = 1,000, inclusive, either side, overlap counts) or same day; RED/GREEN in the TDD report.

**Owns:** `internal/writermap/match.go`, `match_test.go`

- Replace `inDestinationContext` with: title subject, **or** a context phrase within W runes, **or** the same day section.
- Pass the title length (in runes, plus the line break) from `IndexedPost.Title` into the check.
- Tests (RED first):
  - **São Vicente case:** a post titled about Madeira, with "Lisbon" far from "São Vicente" → no Lisbon mention;
  - title names the destination, the place is far down → counts;
  - destination within W of the place, not in the title → counts;
  - destination just beyond W → doesn't count;
  - day sections: same day counts; destination only in the introduction → doesn't count (today it does);
  - existing 6a acceptance tests unchanged and passing.

**Verify:** `go test ./internal/writermap/...`; `go test -cover ./internal/writermap/` ≥ 80%.

### C3. Recompute stored destination results
- [x] Done (2026-10-13): key prefix `writermap:v3:m2:`; old results ignored, `basePlaceCacheVersion` unchanged.

**Owns:** `internal/adapters/sqlite/writerdestination.go` (the key prefix only), `writerdestination_test.go`

- Add a matcher version to `writerDestinationKeyPrefix` (e.g. `writermap:v3:m2:`). Keep `basePlaceCacheVersion` unchanged.
- `WriterDestinations` still lists areas from the new keys only.
- Tests (RED first): a result saved under the old prefix isn't returned; the key still carries the base-place version.

**Verify:** `go test ./internal/adapters/sqlite/ -run WriterDestination`

### C4. Check it end to end and write the evidence
- [x] Done (2026-10-13): runtime replay matches; São Vicente (Lisbon) has no Madeira post; `make test lint build` passed. Targets not met, gap recorded.

**Owns:** `docs/tdd/writer-context.tdd.md`

- Re-run C1's totals and the same 40-mention samples with the chosen W. Judge any mention the new rule keeps that wasn't judged before.
- Run the app on a temp copy of the database (`DB_PATH=…`, `make run`), open Lisbon and click São Vicente: no Madeira posts. Open Funchal: writer pins and posts still there.
- Finish the TDD report: journeys, RED/GREEN excerpts, guarantees table, before/after numbers, coverage, gaps.

**Verify:** ≥90% precision and ≤10% recall loss (or the gap recorded); `make test lint build` passes.
