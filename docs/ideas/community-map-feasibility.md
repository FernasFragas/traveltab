# Community Map: Initial Feasibility Assessment

**Superseded October 5, 2026:** the product was rethought as a blog-seeded consensus map ([one-pager](community-map.md)). The hosting, Leaflet/OSM tile, Wikimedia attribution and cost notes below still apply; the effort estimate and scope do not.

**Historical assessment:** the user subsequently removed public accounts/sign-in, confirmed browser-only ownership with display names, and excluded Fly billing clarification. The account estimates and billing prerequisites below describe the earlier scope and are superseded by [the implementation plan](../../tasks/plan.md). Do not require billing investigation to begin unrelated implementation. No prototypes have yet validated the revised allocation.

Assessed September 26, 2026. Target: full agreed scope by October 2, with 70 development/review hours and a US$5/month total running-cost ceiling. Product scope remains defined in [the one-pager](community-map.md).

## Assessment

**The full scope is a stretch within 70 hours, and budget feasibility is not yet demonstrated.** The optimistic effort estimate fits narrowly; the upper estimate does not. The current repository configuration appears too expensive at published Fly.io rates. A smaller Machine could fit an illustrative low-traffic budget, subject to memory testing and actual billing details.

This is a code inspection and documentation review, not a completed prototype or load test. No production settings were changed. No features have been removed from the agreed scope.

## Existing Foundations and Missing Work

| Area | Evidence and implication |
| --- | --- |
| Hosting | `fly.toml` configures Madrid, one shared CPU, 1 GB RAM, auto-stop off, and a mounted persistent volume. It does not establish deployed Machine count or provisioned volume size. |
| Data | `internal/adapters/sqlite/database.go` provides persistence for cached content and analytics. Community tables, ownership, migrations, and transactional duplicate merges are new work. |
| Accounts | `internal/adapters/httpserver/server.go` has session middleware, but no account/login or admin authorization routes. A session alone is not an account system. |
| Map | `views/map_card.go.tpl` is an iframe. Map interactions, viewport queries, pin creation, and the mobile place panel are new work. |
| Photos | `internal/adapters/api/wikimedia.go` supplies landmark coordinates and image references. `destinationphoto.go` provides attribution/filtering machinery but matches cities. Arbitrary place matching, nearby-image selection, and review state need additional implementation. |
| Itineraries | `internal/planner/types.go` has serializable plan structures. `internal/adapters/httpserver/trip.go` regenerates plans from request parameters. Published snapshots, ownership, listing saved copies, and copying without regeneration are new work. |

## Effort Estimate

These are engineering estimates for focused work in the existing Go/HTMX/SQLite application, not measured completion times. They assume simple interfaces, one account sign-in method, no general itinerary editor, no photo uploads, and no extra product features. Verification covers all agreed behaviors; account provider setup remains a dependency to validate.

| Work | Hours |
| --- | ---: |
| Feasibility experiments, provider/account decisions, memory and billing checks | 4–6 |
| Accounts, persistent sessions, ownership and admin authorization | 8–12 |
| Place schema, interactive map, creation, comments and mobile place panel | 10–14 |
| Place likes and personal “Want to visit” list | 4–6 |
| Exact/nearby photo lookup, attribution, preview selection and review workflow | 8–12 |
| Published itinerary snapshots, attachment, preview and independent copies | 6–10 |
| Reports, admin corrections and duplicate merges preserving related data | 8–12 |
| Integration tests, mobile checks, migration/restore checks and deployment | 8–12 |
| Subtotal | 56–84 |
| Additional contingency | 10 |
| **Total** | **66–94** |

Seventy hours leaves only four hours beyond the optimistic total. Treat October 2 as a stretch target until the first experiments reduce uncertainty. Complete foundations before dependent features: accounts and schema → contributions/likes/saves; place identity → photos; saved plan snapshots → itinerary attachments; all related data → merge verification.

## Operating-Cost Check

Fly announces pricing effective **October 1**, before launch. Its displayed base rate is $2.19/month for shared-CPU-1x/256 MB, with extra RAM at $6/GB-month. Illustrative arithmetic: 1 GB costs $6.69; 512 MB costs $3.69. These are published-rate calculations, not verified Madrid account quotes. [Fly pricing update](https://fly.io/pricing-update/)

An example 512 MB configuration with a 1 GB volume ($0.15) and 5 GB Europe egress ($0.10) totals **$3.94/month** at those rates. Assumes one Machine, shared IPv4, snapshots within the free allowance, and no other billable resources. Calendar length, regional rates, taxes, domain renewal and existing API charges need reconciliation with the total ceiling. Traffic is an assumption, not a forecast. [Fly resource pricing](https://docs.fly.io/about/pricing/)

Confirm the actual bill, all allocated Machines/volumes/IPs, account credits, and peak RAM before changing resources. The local Fly CLI is unavailable, so deployed state was not inspected. No hard spending cutoff has been verified: a low estimate does not enforce a $5 maximum. [Fly cost management](https://docs.fly.io/about/cost-management/)

Proposed service approach, subject to validation:

- Keep the existing Go application and SQLite volume; avoid adding a separate database service.
- Use [Leaflet](https://leafletjs.com/) for map interaction. OpenStreetMap standard tiles are a possible initial source for ordinary interactive viewing, with attribution, caching and valid browser referrers. They are best-effort and may block unsuitable usage; they are not guaranteed free infrastructure at arbitrary scale. Keep the tile source configurable. [Tile policy](https://operations.osmfoundation.org/policies/tiles/)
- Reuse Wikimedia image metadata and attribution. Direct image embedding is possible but discouraged by Commons; handle missing images and do not assume permanent availability. The selected nearby image must remain explicitly distinguished from an exact-place photo. [Commons reuse guidance](https://commons.wikimedia.org/wiki/Commons:Reusing_content_outside_Wikimedia)
- Validate one sign-in method before estimating it as zero additional provider cost. Do not silently add paid email or authentication services.

## First Six Hours: Evidence Needed

1. **Hours 0–1: deployment and budget.** Inspect actual billing/resources and establish launch usage assumptions. Determine whether the $5 ceiling includes existing domain/API expenses and whether a smaller Machine is viable.
2. **Hours 1–3: map/photo experiment.** Show selectable pins in the current page. Test an identifiable landmark, an ordinary business, and a place with no exact photo. Demonstrate nearby candidates with credits and distance, or an honest no-photo result. Use local tile fixtures for automated checks rather than repeatedly fetching public tiles.
3. **Hours 3–4: itinerary experiment.** Persist and copy a plan, then verify that changing the forecast or copying it cannot alter the published version.
4. **Hours 4–6: account and resource experiment.** Confirm the selected sign-in flow, enforce author/admin separation, and measure memory with a representative workload. Do not claim a memory fit based only on idle use.

Re-estimate the remaining scope after these experiments. Proceed on the full October 2 target only if remaining implementation, verification and contingency fit the remaining 64 hours, and the operating-cost model fits the ceiling with explicit usage limits. If not, report the exact conflict for a scope, date or budget decision.

## Remaining Evidence

- Actual Fly invoice/upcoming invoice, resource counts, and volume capacity.
- Supported launch traffic and measured memory under that workload.
- Working arbitrary-place and nearby-photo lookup.
- Account-provider choice and configuration readiness.
- Passing saved-plan/copy experiment.

No prototype or production measurements have been performed as part of this initial assessment.
