# Task 01 handoff — shell and shared foundation

Status: implemented for the tasks 01–03 milestone. Tasks 04 and 05 remain deferred; their existing component presentation is retained behind the new shell.

## Files

- `views/index.go.tpl`: compact responsive wordmark/header, labeled destination search and persistent error/status regions, one shared content composition, persistent footer with existing real links and source credits, mobile bottom navigation. Styles and deferred scripts register in the contracted order. Existing page title/SEO metadata are preserved.
- `views/content_fragment.go.tpl`: destination banner, sticky section shortcuts, four unique section targets, equal-column overview, map/form order, nil-planner explanation, and the contracted component calls. No footer duplication after searches.
- `views/shell_section_links.go.tpl`: shared icon-and-label anchors for the sticky shortcuts and bottom bar.
- `public/styles.css`: shared palette/font/spacing tokens, 1180px cream shell, shared controls, responsive composition, navigation/focus/safe-area styles. Removed obsolete global, weather, hotel, and exploratory planner rules. Bounded legacy trip/stay/video blocks remain until tasks 04–05 supply replacements; standalone itinerary/stay backgrounds preserve their former text contrast after extraction.
- `public/redesign/navigation.js`: one-time delegated navigation setup, shared active state during scrolling, fresh target discovery after HTMX/OOB/history updates, measured sticky offsets, mobile menu/Escape handling, native anchors with keyboard focus, and destination search errors/announcements.

## Verification

The milestone passed `go test ./...`, JavaScript syntax and whitespace checks, and real-browser
checks at 375, 768, 1100, and 1440px. See the [recorded evidence](../artifacts/redesign/milestone-01-03/README.md)
for screenshots, navigation/focus, overflow, HTMX behavior, and the checks still outstanding.

## Integration and remaining verification

- The integrator supplies the minimal `itinerary_body`/`plan_response` extraction, nil-safe stays, video heading, and empty registered future component asset files. These are compatibility wiring for this milestone; itinerary accordions, redesigned form/stays/videos, and click-to-load video behavior belong to tasks 04–05.
- Task 02 supplies `HX-Retarget: #destination-search-error` and `HX-Reswap: innerHTML` on failed search. Navigation allows that error swap without replacing current destination content, and also handles network/timeout failures.
- Task 03 supplies the hero/weather/map partials and destination stylesheet. Shared layout places the map first on desktop and the form first below 900px; mobile header/bottom navigation activates below 640px.
- Provider-controlled map appearance and the legacy tasks 04–05 surfaces remain known visual differences from the reference. The shell does not claim the complete redesign is finished.
