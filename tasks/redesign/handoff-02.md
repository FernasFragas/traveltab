# Task 02 handoff — presentation data and assets

Status: implemented for the tasks 01–03 milestone.

## Delivered

- `internal/adapters/httpserver/presentation.go`: HTTP-only view decoration, embedded manifest, destination identity lookup (Lisbon + Portugal/PT), safe coordinate map URLs, validated YouTube IDs and thumbnail/watch URLs, current-condition icons, and optional property photographs matched by destination, name, and exact coordinates. Duplicate property identities deliberately yield no image.
- `internal/adapters/httpserver/presentation_assets/manifest.json` and `public/images/destinations/lisbon-alfama.jpg`: local 1280×488 real Alfama skyline with dome and Tagus River, source/credit/license/alt/dimensions recorded. Photo by Qiqritiq, [Wikimedia source](https://commons.wikimedia.org/wiki/File:Alfama,_Lisboa_-_2006.jpg), [CC BY-SA 3.0](https://creativecommons.org/licenses/by-sa/3.0/). Wikimedia thumbnail; no local image edits. The manifest ships in the binary and the existing Docker image copies public assets.
- `server.go`: presentation on full page and destination fragment routes; search errors retarget the persistent error region and preserve the last destination.
- `trip.go`: presentation on shared pages; inclusive `EndLabel`; optional `StayCards`; all rendered plan success/validation/provider-error branches use `plan_response` with existing status and push-URL semantics.
- `presentation_test.go` and `redesign_response_test.go`: identity isolation, asset dimensions/provenance, conservative stay matching, date boundaries, video validation/escaping, map coordinate safety, condition icons, full/search/shared rendering, three plan roots, failure empty states, search error targeting, status and headers.

Planner/provider types, cached JSON, export semantics, and booking checkout dates are unchanged. No request-time image provider was added. There are no verified property photos in the initial manifest, so stays keep absent images rather than fabricated photography.

## Checks and dependencies

`go test ./...` passes after the existing footer test was updated to require one credited footer on full pages and none inside search fragments. Real browser HTMX checks passed for 200, 400, and 500 plan responses and failed searches; see [milestone evidence](../artifacts/redesign/milestone-01-03/README.md).

The integrator supplied minimal `plan_response` and `itinerary_body` compatibility templates so this milestone runs before task 04. `EndLabel`, `StayCards`, and enriched video metadata are ready for the future task 04/05 templates; those visual/interaction redesigns remain pending. The presentation contract additionally exposes `ConditionIcon`, and image `License`/`LicenseURL` for accurate icons and visible licensing.
