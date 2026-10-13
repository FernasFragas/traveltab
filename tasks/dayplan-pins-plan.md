# Clickable "Your day plan" pins

**Status:** plan only, not started. Tasks: [dayplan-pins-todo.md](dayplan-pins-todo.md).

**Goal:** the green numbered day-plan pins behave like the writers' and notable pins:
- **click → popup** with the stop's details;
- **"Details" → expands** into the side place panel, when the stop is a place the map knows.

**Independent of the remaining Writers' Map work.** 7f (which owned `map-dayplan.js`) is done. This work owns `public/redesign/map-dayplan.js`, `views/itinerary_body.go.tpl` (the `data-plan-stop` line only), `public/redesign/map-panel.css` (new popup rules only, if any), their tests and `docs/tdd/dayplan-pins.tdd.md`. It does not touch `map-panel.js`.

## Today

- `map-dayplan.js` draws each stop as a `<button class="plan-pin">` with **no `onClick`**, and no `aria-label` (only a marker `title`). Clicking does nothing.
- It reads stops from the itinerary markup: `data-name`, `data-lat`, `data-lon` (`views/itinerary_body.go.tpl:63`). It does **not** read the stop's Wikidata ID, although `planner.Place.ID` has it.
- The writers'/notable pins (`map-panel.js`) already do what we want: `onClick` → `context.draw.popup('pin-popup', …)` → **Details** → `context.selectPlace(qid)` → the place panel loads over htmx.
- Both engines (MapLibre in `writers-map.js`, Leaflet in `writers-map-leaflet.js`) already support `onClick` on markers and `draw.popup`. **No engine changes needed.**

## Design

**Popup content** (reuse the `.map-popup` classes, so it looks like the other pins):
- **Name** (bold)
- **"Day N · Stop K"** + kind (Indoors / Outdoors / …)
- Description, when there is one
- Distance from the previous stop, when there is one
- **Details** button, only when the stop's QID is one of the map's places. Otherwise the popup has no button.

**Behaviour**
- One popup at a time: use the same popup id as `map-panel.js` (`'pin-popup'`), so opening one replaces the other.
- **Escape** closes the popup and puts focus back on the pin. Clicking the map outside closes it (the engines already do this).
- The pin gets `aria-label="Day N, stop K: <name>"`.
- Switching the day toggle or regenerating the plan closes any open plan popup (the layer is cleared and redrawn).

**Data:** add `data-qid`, `data-kind`, `data-description` to `<li data-plan-stop>`. The distance is already in the markup (`[data-stop-distance]`); read it from there. No new endpoint.

## Assumptions to confirm

1. **"Expandable" = popup first, then "Details" opens the side panel.** That matches the other pins. The alternative is clicking a pin to scroll to and open that day in the itinerary list below.
2. **Stops that aren't on the map get a popup without "Details".** The panel endpoint only knows the destination's map places. Planner stops usually are, but not always.
3. **No photo in the popup**, the same as the other pins. The itinerary card already shows it.

## Done when

- Clicking or pressing Enter on a green pin opens the popup; Details opens the place panel; Escape returns focus to the pin.
- Works on MapLibre and Leaflet, at 375 and 1440 px, with a clean console (`cmd/design-preview`).
- `node --test` checks and the Go template test pass; `make test lint build` passes.
- Evidence in `docs/tdd/dayplan-pins.tdd.md`.
