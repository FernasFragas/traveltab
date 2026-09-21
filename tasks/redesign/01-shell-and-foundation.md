# Task 01 — Shell, navigation, and shared visual foundation

Status: implemented for the tasks 01–03 milestone. See `handoff-01.md` and [browser evidence](../artifacts/redesign/milestone-01-03/README.md). Later component redesign and final verification remain separate tasks.

## Owned files

- `views/index.go.tpl`, `views/content_fragment.go.tpl`
- `public/styles.css`
- New `public/redesign/navigation.js`
- Optional new header/footer partials prefixed `shell_`
- `tasks/redesign/handoff-01.md`

## Deliverables

- [x] Recreate the central reference's cream dashboard shell, compact wordmark, restrained border/shadow, and compact header. Target a 1120–1200px maximum desktop width with approximately 20–24px internal gutters.
- [x] Define the shared tokens/classes from the contracts. Use an editorial serif for brand and section headings, Inter for labels/body. Keep the dense reference rhythm without shrinking readable body text to poster scale.
- [x] Move duplicated full-page/search content into the shared fragment composition. Register every component CSS/JS file in the agreed order.
- [x] Build the desktop header with functional section links and right-aligned pill-shaped destination search. Mobile gets a compact brand/menu row and full-width search underneath. Add a visible or accessible label, focus state, spinner, and an error region.
- [x] Build the four icon-and-label section anchors, with terracotta active underline and sticky behavior. Use real anchors and scroll offsets. Update active state after replacement of the content fragment.
- [x] Build the mobile bottom bar seen in the reference, adapted to actual sections: Overview, Itinerary, Stays, Videos. Avoid dead Home/Saved/Profile controls. Add safe-area and content-bottom padding. Both navigation instances share active state.
- [x] Compose the hero, overview, itinerary, stays, and video slots exactly as contracted. Header/footer stay outside destination swaps. Desktop map/form order and mobile form/map order must match the plan.
- [x] Keep the footer compact with real existing links and source credits; do not add placeholder legal or social destinations.
- [x] Remove obsolete global visual rules after confirming component replacements own their styles. Do not edit component partials owned by other tasks.

## Verification and handoff

Check shell layout at 375, 768, and 1440px; keyboard navigation and skip link; anchor IDs and clear active state; no header overlap or bottom-nav occlusion. Integration render depends on the other templates, so report missing deliveries rather than claiming a complete render. Write `handoff-01.md` with exact files, checks, and remaining integration needs.
