# Clickable "Your day plan" pins — Tasks

Source of truth for progress. Read [the plan](dayplan-pins-plan.md) first.

**How to read this file**
- Tasks run in order P1 → P5, by **one agent**: P2–P4 all edit `map-dayplan.js`.
- **Owns** = the only files the task may edit. None overlap with the open Writers' Map tasks in [todo.md](todo.md); `map-panel.js` stays untouched.
- Start from an up-to-date `main` (`git fetch` and check `git status -sb`).
- TDD per [AGENTS.md](../AGENTS.md) §5: see each test fail for the intended reason first.
- No commits or pushes. The owner commits.

**User journeys**
- *As a traveller looking at my day plan on the map, I want to click a green pin and see which stop it is, so that I know what I'm looking at without scrolling to the list.*
- *As a traveller, I want to expand a day-plan pin into the full place panel, so that I can read about that place the same way as any other pin.*
- *As a keyboard user, I want to open and close a day-plan pin's popup with Enter and Escape, so that I don't lose my place on the map.*

---

### P1. Put the stop's details in the markup
- [ ] Done

**Owns:** `views/itinerary_body.go.tpl` (the `data-plan-stop` line only), `internal/adapters/httpserver/dayplan_test.go`

- Add `data-qid="{{ $stop.ID }}"`, `data-kind="{{ $stop.Kind }}"`, `data-description="{{ $stop.Description }}"` to `<li data-plan-stop>`.
- Test: a rendered plan has the attributes, HTML-escaped; empty values render as empty attributes.

**Verify:** `go test ./internal/adapters/httpserver/ -run DayPlan`

### P2. Read the new fields
- [ ] Done

**Owns:** `public/redesign/map-dayplan.js`, `tasks/redesign/checks/map-dayplan.test.mjs`

- `readDays` also keeps `qid`, `kind`, `description` and the `[data-stop-distance]` text for each stop.
- Update the existing "reports valid stops" test to the new shape.

**Verify:** `node --test tasks/redesign/checks/map-dayplan.test.mjs`

### P3. Click a pin → popup
- [ ] Done

**Owns:** `public/redesign/map-dayplan.js`, `tasks/redesign/checks/map-dayplan.test.mjs`, `public/redesign/map-panel.css` (new rules only, if needed)

- Keep `context` from `add()` (not only `context.draw`).
- Each pin gets `onClick` → `context.draw.popup('pin-popup', [lon, lat], element, { onClose })` and `aria-label="Day N, stop K: <name>"`.
- Popup: name · "Day N · Stop K" + kind · description · distance. Uses `.map-popup*` classes.
- Escape closes the popup and refocuses the pin. Redraws (day toggle, plan change, layer off) clear the popup.
- Tests: a click calls `popup` with the right content; empty fields are left out; Escape clears and refocuses; the toggle clears the popup.

**Verify:** `node --test tasks/redesign/checks/map-dayplan.test.mjs`

### P4. Details → place panel
- [ ] Done

**Owns:** `public/redesign/map-dayplan.js`, `tasks/redesign/checks/map-dayplan.test.mjs`

- Show **Details** only when `context.features` has a place with the stop's `qid`.
- Click → close the popup, then `context.selectPlace(qid)` (`map-panel.js` already loads the panel on `travelmap:place-selected`).
- Tests: Details present and calls `selectPlace` for a known QID; absent for an unknown or empty QID.

**Verify:** `node --test tasks/redesign/checks/map-dayplan.test.mjs`

### P5. Browser check and evidence
- [ ] Done

**Owns:** `tasks/redesign/checks/map-dayplan.browser.mjs`, `docs/tdd/dayplan-pins.tdd.md` (new)

- Browser check in `cmd/design-preview`: generate a plan, click a green pin → popup; Details → panel shows the place; Escape → focus on the pin. Run for MapLibre and Leaflet (`data-map-engine="leaflet"`), at 375 and 1440 px, console clean.
- Write the TDD evidence report (journeys, RED/GREEN excerpts, the guarantees table, coverage).

**Verify:** the browser check passes; `make test lint build` passes.
