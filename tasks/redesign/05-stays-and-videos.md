# Task 05 — Accommodation and travel video cards

Status: pending. Can start against frozen contracts; task 02 supplies enriched view data.

## Owned files

- `views/stay_card.go.tpl`, `views/video.go.tpl`
- New `public/redesign/discovery.css`, `public/redesign/discovery.js`
- `tasks/redesign/handoff-05.md`

Read the [plan](../../docs/redesign-plan.md) and [contracts](contracts.md).

## Deliverables

- [ ] `stay_card` becomes a section body (no `id="stays"` wrapper), accepting `*tripPlanView` or nil. Include Where to stay heading and a right-aligned Check prices button using the existing dated booking URL.
- [ ] Render `.StayCards` as three compact image-left cards per desktop row, wrapping additional results; one column on mobile. Include serif name, known stars, actual accommodation type, and a real website action. Use a restrained, correctly sized placeholder for absent photos and preserve photo credits when available.
- [ ] Treat known stars as property classification, not customer review scores. Hide unknown stars; do not invent prices, boutique/luxury labels, or availability.
- [ ] Preserve unavailable-stays notes, empty results, and pre-plan states. Do not hide the section anchor or booking action solely because images are missing.
- [ ] `video` becomes a section body using `.Presentation.Videos`, headed Explore through video with brief supporting copy. Match compact image-left thumbnail/title cards, approximately four across desktop and one across mobile.
- [ ] If more than four videos exist, use a working View all videos expansion control with `aria-expanded`; preserve access to the complete returned list. Do not add a dead link to an unbuilt page.
- [ ] Activate playback on request with a labeled embedded player or accessible watch link, consistent with card layout. Load no video iframe until activated; include an iframe title and a visible external fallback. Reserve dimensions to prevent disruptive jumps.
- [ ] No invented runtime labels; unavailable thumbnails use a stable fallback. Handle empty videos with a useful section message.
- [ ] Use delegated handlers that survive both search and plan updates; task 01 alone registers the script.

## Verification and handoff

Check 0/1/6 stays; absent photos, stars, and websites; unavailable stay service with a valid booking link; 0/1/10 videos; long titles; thumbnail failure; keyboard playback; expansion; and destination changes. Confirm nil template inputs render without panics. Report checks and data-driven visual substitutions in `handoff-05.md`.
