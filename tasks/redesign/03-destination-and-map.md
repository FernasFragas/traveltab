# Task 03 — Destination banner, current weather, and map

Status: implemented for the tasks 01–03 milestone. See `handoff-03.md` and [browser evidence](../artifacts/redesign/milestone-01-03/README.md). Later component redesign and final verification remain separate tasks.

## Owned files

- `views/weather_display.go.tpl`, `views/weather_card.go.tpl`
- New `views/map_card.go.tpl`
- New `public/redesign/destination.css`
- `tasks/redesign/handoff-03.md`

Read the [plan](../../docs/redesign-plan.md), [contracts](contracts.md), and the reference image.

## Deliverables

- [x] Turn `weather_display` into the destination banner only; extract the current map into `map_card` for the overview slot owned by task 01.
- [x] Desktop hero: approximately 58% photograph / 42% cream information panel, roughly 220–260px tall at full desktop width. Use the supplied Lisbon photo with a crop resembling the reference. Include optional small italic editorial caption when configured.
- [x] Information panel: large black serif city name, one short tagline, horizontal weather metrics separated by fine rules, optional short editorial line and country label. Do not show configuration-dependent copy for unknown destinations.
- [x] Weather metrics: golden sun or appropriate condition icon, blue humidity and wave icons, dark values, quieter labels. Use actual current values and conditions. Replace the old weather-dependent card backgrounds and blur styles.
- [x] Mobile: destination photo with readable city-name overlay, weather metrics beneath on a solid background. Reserve space for photo fallback and long city names.
- [x] Map: compact edge-to-edge rounded frame with the existing live embed, descriptive iframe title, visible map attribution, and a working View larger map link. Target approximately 240px desktop and 140–180px mobile height. The surrounding style should match the pale reference; document the external embed's internal-style limitation.
- [x] Keep third-party zoom controls usable; do not draw decorative controls on top of real controls. Use `.Presentation.MapURL` for the external action.
- [x] Missing/failed hero photo should keep the city and weather layout intact with a warm neutral fallback, not another destination's photograph.

## Verification and handoff

Inspect desktop/mobile crops, long city names, photo failure, nil hero, current weather labels, iframe title, and map link behavior. Verify no inline styles override shared tokens. Provide screenshots when runnable and document which external map behavior was actually checked in `handoff-03.md`.
