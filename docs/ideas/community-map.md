# TravelTab Community Map

## Problem Statement
How might we help travelers find worthwhile places through community recommendations, existing place photos, and itineraries they can view and copy?

## Recommended Direction
Replace the Waze embed with an interactive community map. Each place has one shared pin. Its first public contributor creates the place and starts its discussion; subsequent visitors add their experiences to the same place. The primary user is a traveler deciding where to visit, and the main success moment is finding a worthwhile place.

Opening a pin reveals TravelTab-supplied photos, comments and reviews, and attached TravelTab itineraries. Contributors enter a display name without signing in and can write text and attach an itinerary together, but cannot upload photos. Likes apply to places, with one like per browser owner per place and the option to remove it. Show the like count in the place detail panel and contributions newest first. All places remain discoverable regardless of like count. A like expresses appreciation, not a verified visit; a separate “Want to visit” list records intent to go.

The central journey is: **discover a place → explore firsthand experiences → save it to “Want to visit” → optionally copy an attached itinerary.** Contributions publish immediately, with reports handled through an admin review queue. Contributors control their own comments and itinerary attachments; admins can delete any comment and control shared place details, photo decisions, and duplicate merges.

Names are unverified attribution. The current browser controls author edits, likes, saved places and copied itineraries. Clearing its cookies, switching devices or credential expiry loses access to this personal state without deleting public contributions. This browser-only behavior is explicitly confirmed; no recovery links or cross-device sync are planned. Admin access remains protected separately.

## Delivery Constraints
- **Deadline:** October 2, 2026, for the full agreed scope below. This is the requested target, not a verified delivery estimate; six days remain from the September 26 planning date.
- **Budget:** prefer free services, with a hard ceiling of **US$5/month in total running costs**, including hosting, maps, photo services, and storage.
- **Available effort:** 70 development/review hours. Current hosting is Fly.io; deployed resources and actual billing remain to be confirmed.
- Check technical/schedule feasibility early. Fly bill clarification is excluded at the user's request; the monthly ceiling remains unverified and billing investigation is not an implementation prerequisite. Do not silently reduce scope or exceed the budget.
- [Implementation plan](../../tasks/plan.md) and [task checklist](../../tasks/todo.md): 58 task hours, 10 contingency and 2 planning/review hours. This is a provisional allocation, not a verified delivery estimate. The [initial assessment](community-map-feasibility.md) predates removal of public sign-in.

## Key Assumptions to Validate
- [ ] **Travelers will contribute without an established audience.** Invite initial public contributors and observe whether they publish their first place without assistance. A lack of contributions could leave the map unhelpful.
- [ ] **Contributions help people choose places.** Test discovery sessions and ask what participants would visit and which contribution influenced that choice.
- [ ] **Place likes help travelers choose somewhere worthwhile.** Compare discovery sessions with and without visible counts, checking whether counts aid decisions or cause useful new places to be overlooked.
- [ ] **Attached itineraries provide useful context.** Measure itinerary previews and copies from place discussions, and ask whether the copied route fits the traveler's interests and schedule.
- [ ] **Public contributions create enough local coverage.** Track destinations with useful contributions and empty-map visits; global availability does not guarantee global coverage.
- [ ] **Supplied photos provide useful context without misleading visitors.** Test exact-place matching and nearby-photo selection; track unsuitable selections and admin review workload.

## MVP Scope
- Public browsing and contributions across destinations without accounts or sign-in. A display name is required when publishing a comment or itinerary recommendation; personal state belongs to the current browser.
- Create a named place at a selected map location, checking nearby existing places first to avoid duplicate pins.
- Publish an initial contribution with text, a TravelTab itinerary, or both; prevent empty submissions. The first contribution comes from the person adding the place.
- Open a place panel with supplied photos, comments/reviews, itinerary attachments, and place like count.
- Look for an existing photo of the actual place and preserve source attribution. When none is available, offer nearby photos for the contributor to select as the preview. The selected image appears immediately with a “Nearby photo” label and enters a separate admin photo-review queue.
- Admins can approve, replace, or remove the selected preview. Keep the nearby label unless the image is confirmed to depict the place itself. A pin still publishes with “Photo unavailable” if no suitable image exists or is selected.
- Add further named contributions, displayed newest first, and like or unlike the place once per browser owner. This does not guarantee one vote per person across browsers.
- Save and remove places in a personal “Want to visit” list, independently of likes. Adding a place directly to an itinerary is outside this release.
- Preview an attached itinerary and create an independent copy.
- Preserve the published itinerary's stops and day order. Current TravelTab links can change day order with the forecast, so attachments need a saved version rather than only the existing dynamic link.
- Let authors edit/delete their own comments and itinerary attachments. Publish contributions immediately and send reports to a simple admin queue, initially managed by the project owner.
- Let admins delete any comment, whether reported or not, with a confirmation and recorded moderation reason. Deletion removes the comment and its attached recommendation from public discussion without deleting the place, other comments, likes, saved places or previously copied itineraries. The original contributor cannot edit or restore an admin-deleted comment.
- Let anyone suggest corrections or report duplicate places. Only admins change shared place names/locations or merge pins. Merges preserve contributions, unique place likes, and saved-place references.
- For destinations without contributions, show an invitation to add the first place.

## Not Doing (and Why)
- **Public accounts, sign-in, recovery links or cross-device sync** — contributors enter a name and use the confirmed browser-only ownership model.
- **User photo uploads** — TravelTab supplies existing photos; contributors may select a suggested nearby preview when needed.
- **Adding places directly to itineraries** — the “Want to visit” list serves the initial save-for-later need; copying attached itineraries remains in scope.
- **Approval before every contribution is published** — immediate publication with report review is the agreed moderation model.
- **Contributor edits to shared place details** — admins handle changes that affect everyone.
- **Likes on individual contributions** — the agreed interaction is appreciation of a place.
- **Contributor leaderboards or follower feeds** — finding worthwhile places is the primary goal.
- **Popularity-controlled map visibility** — new and lesser-known places must remain discoverable.
- **Star ratings** — written experiences and place likes test the initial value with less complexity.
- **A separate pin for every visitor** — contributions should accumulate around a shared place.
- **A dedicated itinerary-discovery feed or route recommendation engine** — attached plans are sufficient to test whether routes help place discovery.
- **Turn-by-turn navigation** — outside the discovery experience.
- **Automatic publication of generated recommendations** — community contributions should come from people.

## Open Questions
The product questions above are resolved. The early implementation experiments must establish map/photo behavior, immutable itinerary copying and resource requirements, then re-estimate whether the full scope fits October 2 and the available hours. Fly billing clarification is intentionally deferred. The implementation plan awaits human review before coding begins.
