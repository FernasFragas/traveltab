# Writers' posts by city: tasks

Read [the plan](writer-posts-city-plan.md) first. No commits or pushes; the owner commits. Measure on a **temp copy** of `launch-check.db`, offline.

**User journeys**
- *As a traveller opening the Louvre in Paris, I want only Paris posts, so that I don't read a Toronto guide.*
- *As a traveller opening a Madeira place, I want to keep the posts I get today, so that the fix doesn't empty the map.*

---

### P1. Measure candidate N
- [x] **Owns:** `docs/tdd/writer-posts-city.tdd.md`, a temporary measuring command (deleted after)
- For Paris, Lisbon, Madeira/Funchal and Porto: mentions, posts and pins at today's rule and N = 2, 3, 5.
- Draw 40 mentions per destination (fixed seed); judge each right or wrong with a reason; reuse the writer-context samples where they exist.
- Pick the smallest N meeting ≥90% precision and ≤25% recall loss (precision first), or the best N.
- **Verify:** sample sheet and table in the report; owner confirms N.

### P1b. Local names for the destination
- [x] **Owns:** `internal/writermap/source.go` (`contextNames`), `source_test.go`, `writerdata/overrides.json`, `writerdata/embed_test.go`
- Add `destination_aliases` to overrides (Lisbon → Lisboa) and append them in `contextNames`.
- Test first: a Lisbon destination returns "Lisboa" as a context name.
- **Verify:** test fails, then passes; the embedded file still parses.

### P2. Failing test (RED)
- [x] **Owns:** `internal/writermap/match_test.go`
- Post with "Paris" once, not in the title, "Louvre" 50 chars away: not listed. Post with Paris in the title: listed. Post with Paris N times: listed.
- **Verify:** fails for the intended reason.

### P3. Post-level rule (GREEN)
- [x] **Owns:** `internal/writermap/match.go`
- Add the "about the destination" check ahead of `inDestinationContext`.
- **Verify:** P2 passes; existing match tests pass; coverage ≥80%.

### P4. Recompute stored results
- [x] **Owns:** `internal/adapters/sqlite/writerdestination.go`, `writerdestination_test.go`
- Matcher version in the result key; test that an old key is ignored.
- **Verify:** test passes; gazetteer cache untouched.

### P5. Check locally and report
- [x] **Owns:** `docs/tdd/writer-posts-city.tdd.md`
- Run the app on a copy of `launch-check.db` with `WRITERS_MAP=1`; open Paris, Louvre Museum: no Toronto or Ponta Delgada posts.
- `make test build`; `make lint` needs a golangci-lint built with Go 1.27.
- **Verify:** evidence report with real command output.
