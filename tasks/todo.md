# Community Map Task List

Source of truth for implementation progress: this checklist. Read [the plan](plan.md) and [product scope](../docs/ideas/community-map.md) first.

**Status:** planned only; no task is implemented by creating this file. Human review of the plan is required before implementation. Public visitors do not sign in: names are attribution and the current browser owns personal state. Fly bill clarification is excluded.

**Allocation:** 58 task hours + 10 contingency + 2 planning/review = 70 hours. Individual values are timeboxes for reassessment, not promises. If a task exceeds two hours or five files, split it here before expanding implementation. Checkpoints follow every three tasks. Evidence is recorded in future files under `tasks/community-map/`; these paths do not imply evidence already exists.

**Verification conventions:** commands run from the repository root. `make build` is the repository build command. Use the listed affected-package tests; new behavior needs meaningful cases, and a command matching zero intended tests is not evidence. Document-only tasks use document review instead of invented runtime tests. The final full suite/race/lint checks belong to Task 36. A browser or live-provider check cannot be marked complete based only on HTTP fixtures.

## Task 1: Prove map interaction in the local preview

- [ ] Complete Task 1

**Description:** Build a disposable, clearly labeled preview of selectable pins before changing the production map.

**Acceptance criteria:**
- [ ] The preview supports pan/zoom, pin selection and a small place panel at 375 px and 1440 px.
- [ ] Repeated destination swaps create one map instance and release the previous instance.
- [ ] Automated runs use local tile fixtures; record library loading, attribution and keyboard findings.

**Verification:**
- [ ] Tests pass: `go test ./cmd/design-preview`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Open the local experiment, select a pin, change destination twice and check keyboard focus.

**Dependencies:** None

**Files likely touched:**
- `cmd/design-preview/community_map.go`
- `cmd/design-preview/community_map_test.go`
- `cmd/design-preview/main.go`
- `public/redesign/community-map.js`
- `tasks/community-map/evidence.md`

**Estimated scope:** Medium (5 files); 1.5-hour timebox.

## Task 2: Prove exact-place and nearby-photo lookup

- [ ] Complete Task 2

**Description:** Test whether existing Wikimedia machinery can support arbitrary pins rather than only cities.

**Acceptance criteria:**
- [ ] A known landmark resolves to a confidently matched photo; ambiguous matches are rejected.
- [ ] An ordinary business and a place lacking an exact image return nearby candidates with source, license and distance, or an explicit empty result.
- [ ] Timeouts do not prevent pin creation; live samples and fixture results are recorded separately.

**Verification:**
- [ ] Tests pass: `go test ./internal/adapters/api`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Inspect the three sampled locations and verify nearby images are not labeled as exact matches.

**Dependencies:** None

**Files likely touched:**
- `internal/application/placephoto.go`
- `internal/adapters/api/placephoto.go`
- `internal/adapters/api/placephoto_test.go`
- `internal/adapters/api/testdata/community_photos.json`
- `tasks/community-map/evidence.md`

**Estimated scope:** Medium (5 files); 2-hour timebox.

## Task 3: Prove immutable itinerary persistence

- [ ] Complete Task 3

**Description:** Use a temporary SQLite database to demonstrate a published plan and independent copy without changing live trip pages.

**Acceptance criteria:**
- [ ] Persist and reload a versioned plan preserving stop IDs, coordinates and day order, including past travel dates.
- [ ] Copying produces a new ID with identical route content; modifying the copy cannot change the original.
- [ ] A changed forecast and unavailable planner cannot affect a stored snapshot.

**Verification:**
- [ ] Tests pass: `go test ./internal/adapters/sqlite`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Inspect two saved records in a temporary database and document the freeze/copy result.

**Dependencies:** None

**Files likely touched:**
- `internal/application/savedplan.go`
- `internal/adapters/sqlite/savedplan.go`
- `internal/adapters/sqlite/savedplan_test.go`
- `internal/adapters/sqlite/community_migrations.go`
- `tasks/community-map/evidence.md`

**Estimated scope:** Medium (5 files); 1.5-hour timebox.

## Checkpoint: After Tasks 1–3

- [ ] All affected tests pass and `make build` succeeds; do not rerun unchanged checks without a reason.
- [ ] Demonstrate the completed feature path in a disposable local environment and record evidence.
- [ ] Review map/photo/snapshot evidence and revise unknowns before expanding the implementation.

## Task 4: Document browser ownership behavior

- [ ] Complete Task 4

**Description:** Make the confirmed browser-only model explicit for implementation and verification.

**Acceptance criteria:**
- [ ] Public posting asks only for a display name, with no registration, email or identity-provider dependency.
- [ ] Names are unverified attribution, never authorization; an opaque browser credential owns edits, saves, copied plans and like state.
- [ ] The confirmed browser-only model has no recovery link: clearing cookies or changing devices loses access to private state and author controls without deleting public posts.

**Verification:**
- [ ] Review the identity contract against the confirmed browser-only decision (documentation task; no new runtime tests).
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Walk through same-name visitors, a new device and cleared cookies; make the resulting behavior explicit.

**Dependencies:** None

**Files likely touched:**
- `tasks/community-map/identity.md`
- `tasks/community-map/evidence.md`

**Estimated scope:** Small (2 files); 1-hour timebox.

## Task 5: Measure the current resource baseline

- [ ] Complete Task 5

**Description:** Measure local runtime memory and a representative fixture workload without inspecting bills or changing Fly resources.

**Acceptance criteria:**
- [ ] Record idle, peak and post-load memory plus toolchain, workload and machine details.
- [ ] Run cached reads and cold-provider simulations with bounded concurrency, separating synthetic latency from live service behavior.
- [ ] State whether 512 MB is a testable target; do not claim production fit from idle measurements or macOS results alone.

**Verification:**
- [ ] Tests pass: `go test ./cmd/loadtest-server`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Run the local load fixture and record memory; exercise no public tile servers or production writes.

**Dependencies:** None

**Files likely touched:**
- `loadtests/community-baseline.js`
- `loadtests/README.md`
- `tasks/community-map/evidence.md`

**Estimated scope:** Medium (3 files); 1.5-hour timebox.

## Task 6: Wire the optional community module

- [ ] Complete Task 6

**Description:** Provide narrow application ports, an incremental migration runner and dependency wiring while keeping existing pages functional.

**Acceptance criteria:**
- [ ] Community dependencies follow existing adapter/setter patterns; existing Storage mocks do not require unrelated community methods.
- [ ] Versioned migrations are transactional and restart-safe, preserve existing tables, and adopt the snapshot migration from Task 3.
- [ ] Disabled or unconfigured community dependencies preserve the current dashboard; fixtures never leak into production.

**Verification:**
- [ ] Tests pass: `go test ./internal/adapters/sqlite ./internal/adapters/httpserver ./cmd/design-preview`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Start with community disabled and compare home, search and shared-trip pages with the baseline.

**Dependencies:** Task 3

**Files likely touched:**
- `internal/application/community.go`
- `internal/adapters/sqlite/community_migrations.go`
- `internal/adapters/sqlite/database.go`
- `internal/adapters/httpserver/community.go`
- `cmd/web/main.go`

**Estimated scope:** Medium (5 files); 1.5-hour timebox.

## Checkpoint: After Tasks 4–6

- [ ] All affected tests pass and `make build` succeeds; do not rerun unchanged checks without a reason.
- [ ] Demonstrate the completed feature path in a disposable local environment and record evidence.
- [ ] Review the first five tasks' experiments and the anonymous identity contract with the owner; re-estimate the remaining hours. Billing clarification is explicitly excluded.

## Task 7: Establish anonymous visitor ownership

- [ ] Complete Task 7

**Description:** Assign a durable opaque browser credential without presenting a sign-in flow or treating the chosen name as identity.

**Acceptance criteria:**
- [ ] A random opaque credential identifies a server-side visitor; the client cannot select its owner ID and stored credentials are protected against disclosure.
- [ ] The persistent cookie uses Secure in production, HttpOnly and appropriate SameSite settings; lost/expired credentials create a new owner rather than claiming old content by name.
- [ ] Two browsers with the same display name remain distinct; restart preserves valid credentials and records, and reads do not unnecessarily allocate visitors.

**Verification:**
- [ ] Tests pass: `go test ./internal/adapters/sqlite ./internal/adapters/httpserver`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Create visitor state in two isolated browsers using the same name; restart and verify ownership stays separate.

**Dependencies:** Task 4, Task 6

**Files likely touched:**
- `internal/application/visitor.go`
- `internal/adapters/sqlite/visitors.go`
- `internal/adapters/sqlite/visitors_test.go`
- `internal/adapters/httpserver/visitor.go`
- `internal/adapters/httpserver/visitor_test.go`

**Estimated scope:** Medium (5 files); 1.5-hour timebox.

## Task 8: Protect the admin workspace

- [ ] Complete Task 8

**Description:** Provide owner-only access to moderation without introducing accounts or sign-in for public contributors.

**Acceptance criteria:**
- [ ] An admin-only entry verifies a deployment-provided secret verifier and establishes a separate protected session; absent configuration disables administration.
- [ ] Failed login attempts are bounded and admin session expiry/logout work; secrets never appear in HTML, logs or the repository.
- [ ] Visitors cannot grant themselves admin status through a name, cookie field or request parameter.

**Verification:**
- [ ] Tests pass: `go test ./internal/adapters/httpserver ./internal/config`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Try valid, invalid and missing admin configuration without changing the anonymous contribution flow.

**Dependencies:** Task 6, Task 7

**Files likely touched:**
- `internal/adapters/httpserver/admin_auth.go`
- `internal/adapters/httpserver/admin_auth_test.go`
- `internal/config/env.go`
- `internal/config/env_test.go`
- `views/admin_login.go.tpl`

**Estimated scope:** Medium (5 files); 1.5-hour timebox.

## Task 9: Enforce visitor write boundaries

- [ ] Complete Task 9

**Description:** Apply shared ownership and request-protection checks before enabling public writes, and expose browser-owned personal navigation.

**Acceptance criteria:**
- [ ] Mutations enforce CSRF/origin protection and server-side visitor ownership; knowing a name or object ID never permits an edit.
- [ ] Private visitor responses are not publicly cached, and expired credentials cannot access old personal state.
- [ ] Personal navigation explains browser-local access where relevant; existing destination navigation remains intact and no public sign-in control is introduced.

**Verification:**
- [ ] Tests pass: `go test ./internal/adapters/httpserver`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Try two same-name browsers plus an admin session; verify cross-owner and forged writes fail.

**Dependencies:** Task 7, Task 8

**Files likely touched:**
- `internal/adapters/httpserver/visitor.go`
- `internal/adapters/httpserver/visitor_test.go`
- `internal/adapters/httpserver/community.go`
- `views/personal.go.tpl`
- `views/index.go.tpl`

**Estimated scope:** Medium (5 files); 1-hour timebox.

## Checkpoint: After Tasks 7–9

- [ ] All affected tests pass and `make build` succeeds; do not rerun unchanged checks without a reason.
- [ ] Demonstrate the completed feature path in a disposable local environment and record evidence.
- [ ] Report deviations; continue within the approved plan unless scope, deadline, budget or an external action needs a new decision.

## Task 10: Let a contributor publish a place

- [ ] Complete Task 10

**Description:** Create one persistent shared place with its initial text contribution through a form and selected coordinates.

**Acceptance criteria:**
- [ ] A visitor can submit a display name, place name, finite valid coordinates and initial text without signing in; place and contribution commit atomically under that browser owner.
- [ ] Empty/oversized names or text and invalid coordinates display field errors; repeated submissions do not leave empty or duplicate records.
- [ ] The result opens the newly created public place; no image is required and text is rendered safely.

**Verification:**
- [ ] Tests pass: `go test ./internal/adapters/httpserver ./internal/adapters/sqlite`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Create a place at chosen coordinates, reload it, then retry the submission and simulate a failed write.

**Dependencies:** Task 6, Task 9

**Files likely touched:**
- `internal/adapters/sqlite/places.go`
- `internal/adapters/httpserver/places.go`
- `internal/adapters/httpserver/places_test.go`
- `views/place_create.go.tpl`
- `internal/adapters/httpserver/community.go`

**Estimated scope:** Medium (5 files); 2-hour timebox.

## Task 11: Suggest existing nearby places

- [ ] Complete Task 11

**Description:** Before publication, let contributors inspect existing nearby places and continue on the correct shared pin.

**Acceptance criteria:**
- [ ] The create form lists nearby existing places with names and distances; selecting one opens its current detail page instead of creating a duplicate.
- [ ] Coordinates alone never force two distinct neighboring businesses to merge; the user can still create a distinct place.
- [ ] Queries and result counts are bounded; unavailable suggestions do not erase the draft.

**Verification:**
- [ ] Tests pass: `go test ./internal/adapters/httpserver ./internal/adapters/sqlite`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Add two distinct nearby venues and verify both can exist while an obvious duplicate is suggested.

**Dependencies:** Task 10

**Files likely touched:**
- `internal/adapters/sqlite/places.go`
- `internal/adapters/httpserver/places.go`
- `internal/adapters/httpserver/places_test.go`
- `views/place_create.go.tpl`
- `public/redesign/community-map.js`

**Estimated scope:** Medium (5 files); 1.5-hour timebox.

## Task 12: Replace Waze with the community map

- [ ] Complete Task 12

**Description:** Integrate the proven map lifecycle with real viewport place queries and the existing destination flow.

**Acceptance criteria:**
- [ ] Home, destination search and shared-trip pages show pins for the current viewport without provider-photo calls per pin.
- [ ] Viewport results are bounded and paginated or clustered without ranking by likes; stale requests cannot replace a newer destination.
- [ ] Clicking the map opens creation at those coordinates; empty/error states and map attribution remain usable.

**Verification:**
- [ ] Tests pass: `go test ./internal/adapters/httpserver ./cmd/design-preview`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Pan, zoom, create a pin and switch cities rapidly; confirm old markers and event handlers are gone.

**Dependencies:** Task 1, Task 10, Task 11

**Files likely touched:**
- `internal/adapters/httpserver/places.go`
- `internal/adapters/httpserver/places_test.go`
- `internal/adapters/sqlite/places.go`
- `views/map_card.go.tpl`
- `public/redesign/community-map.js`

**Estimated scope:** Medium (5 files); 1.5-hour timebox.

## Checkpoint: After Tasks 10–12

- [ ] All affected tests pass and `make build` succeeds; do not rerun unchanged checks without a reason.
- [ ] Demonstrate the completed feature path in a disposable local environment and record evidence.
- [ ] Report deviations; continue within the approved plan unless scope, deadline, budget or an external action needs a new decision.

## Task 13: Open a place discussion

- [ ] Complete Task 13

**Description:** Deliver a public place detail panel and a direct place URL with readable contributions.

**Acceptance criteria:**
- [ ] Opening a pin or direct URL shows the same place, a photo placeholder and newest-first contributions.
- [ ] Contribution pages have a stable cursor and bounded size; an empty discussion remains usable.
- [ ] The panel supports keyboard opening/closing and small screens; tile failure leaves the place content accessible.

**Verification:**
- [ ] Tests pass: `go test ./internal/adapters/httpserver`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Open a direct place link at 375 px, paginate contributions and close the panel using the keyboard.

**Dependencies:** Task 12

**Files likely touched:**
- `internal/adapters/httpserver/places.go`
- `internal/adapters/httpserver/places_test.go`
- `views/place_panel.go.tpl`
- `public/redesign/community-map.js`
- `public/redesign/community-map.css`

**Estimated scope:** Medium (5 files); 1.5-hour timebox.

## Task 14: Let authors manage their comments

- [ ] Complete Task 14

**Description:** Implement publication, editing and deletion of each author's text contributions in the place discussion.

**Acceptance criteria:**
- [ ] Visitors enter a display name and publish immediately; their browser can edit/delete its own comments, with stable newest-first ordering.
- [ ] Another browser cannot change a comment or its owner, even using the same display name; names are unverified attribution. The author's browser cannot edit or restore an admin-deleted comment.
- [ ] Deleting the first comment does not delete the shared place or other people's contributions; empty states remain clear.

**Verification:**
- [ ] Tests pass: `go test ./internal/adapters/httpserver ./internal/adapters/sqlite`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Publish as one user, try editing as another, then delete the original contribution.

**Dependencies:** Task 9, Task 13

**Files likely touched:**
- `internal/adapters/sqlite/contributions.go`
- `internal/adapters/httpserver/contributions.go`
- `internal/adapters/httpserver/contributions_test.go`
- `views/place_panel.go.tpl`
- `internal/adapters/httpserver/community.go`

**Estimated scope:** Medium (5 files); 1.5-hour timebox.

## Task 15: Like a place

- [ ] Complete Task 15

**Description:** Add a place-level like/unlike action and count, independent of comments and saved places.

**Acceptance criteria:**
- [ ] A visitor can like/unlike once per browser owner per place; concurrent retries cannot inflate the count.
- [ ] Counts and the current browser's state survive refresh/restart; new visitors can like without entering a name or signing in.
- [ ] No comment, photo or itinerary gets a like action, and map inclusion/order does not depend on likes.

**Verification:**
- [ ] Tests pass: `go test ./internal/adapters/httpserver ./internal/adapters/sqlite`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Like repeatedly from two tabs in one browser, unlike, then compare counts from a separate browser.

**Dependencies:** Task 9, Task 13

**Files likely touched:**
- `internal/adapters/sqlite/place_likes.go`
- `internal/adapters/httpserver/place_likes.go`
- `internal/adapters/httpserver/place_likes_test.go`
- `views/place_panel.go.tpl`
- `internal/adapters/httpserver/community.go`

**Estimated scope:** Medium (5 files); 1.5-hour timebox.

## Checkpoint: After Tasks 13–15

- [ ] All affected tests pass and `make build` succeeds; do not rerun unchanged checks without a reason.
- [ ] Demonstrate the completed feature path in a disposable local environment and record evidence.
- [ ] Report deviations; continue within the approved plan unless scope, deadline, budget or an external action needs a new decision.

## Task 16: Save a place for later

- [ ] Complete Task 16

**Description:** Add a personal Want to visit toggle to the place panel.

**Acceptance criteria:**
- [ ] Saving/unsaving is idempotent and persists per visitor independently of likes.
- [ ] One browser owner's saved state is never returned for another or cached into public fragments; a name cannot recover it.
- [ ] The UI confirms success and preserves the prior state on request failure.

**Verification:**
- [ ] Tests pass: `go test ./internal/adapters/httpserver ./internal/adapters/sqlite`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Save without liking, open a second isolated browser, then return to the first.

**Dependencies:** Task 9, Task 13

**Files likely touched:**
- `internal/adapters/sqlite/saved_places.go`
- `internal/adapters/httpserver/saved_places.go`
- `internal/adapters/httpserver/saved_places_test.go`
- `views/place_panel.go.tpl`
- `internal/adapters/httpserver/community.go`

**Estimated scope:** Medium (5 files); 1.5-hour timebox.

## Task 17: Browse Want to visit

- [ ] Complete Task 17

**Description:** Give each visitor a discoverable page for their browser-owned saved places.

**Acceptance criteria:**
- [ ] Personal navigation opens a private, bounded saved-place list with names and preview states.
- [ ] Each item opens its place, and removing an item updates the list and empty state.
- [ ] The list works on mobile and across destinations without adding places directly to an itinerary.

**Verification:**
- [ ] Tests pass: `go test ./internal/adapters/httpserver`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Save places from two destinations, open the list and remove the final item.

**Dependencies:** Task 16

**Files likely touched:**
- `internal/adapters/httpserver/saved_places.go`
- `internal/adapters/httpserver/saved_places_test.go`
- `views/saved_places.go.tpl`
- `views/personal.go.tpl`

**Estimated scope:** Medium (4 files); 1.5-hour timebox.

## Task 18: Show supplied exact-place photos

- [ ] Complete Task 18

**Description:** Promote the photo experiment into a cached, bounded lookup for a persistent place.

**Acceptance criteria:**
- [ ] Only a confident place match is labeled exact; ambiguous/no-result/error outcomes preserve a usable photo-free place.
- [ ] The panel displays image, author, license and source links; broken images show Photo unavailable.
- [ ] Cache identity includes the actual place identity/coordinates and uses negative/error handling without reusing city photos as exact matches.

**Verification:**
- [ ] Tests pass: `go test ./internal/adapters/api ./internal/adapters/sqlite ./internal/adapters/httpserver`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Inspect an exact match, ambiguous namesake and broken image without losing place comments.

**Dependencies:** Task 2, Task 6, Task 13

**Files likely touched:**
- `internal/adapters/api/placephoto.go`
- `internal/adapters/sqlite/placephoto.go`
- `internal/adapters/httpserver/place_photos.go`
- `internal/adapters/httpserver/place_photos_test.go`
- `views/place_panel.go.tpl`

**Estimated scope:** Medium (5 files); 2-hour timebox.

## Checkpoint: After Tasks 16–18

- [ ] All affected tests pass and `make build` succeeds; do not rerun unchanged checks without a reason.
- [ ] Demonstrate the completed feature path in a disposable local environment and record evidence.
- [ ] Report deviations; continue within the approved plan unless scope, deadline, budget or an external action needs a new decision.

## Task 19: Choose a nearby preview

- [ ] Complete Task 19

**Description:** Allow the creator to select a supplied nearby image when the place lacks an exact photo.

**Acceptance criteria:**
- [ ] The creation/result flow offers bounded nearby candidates with attribution and distance; skipping selection still publishes the pin.
- [ ] Only a server-issued candidate for that place can be selected; no upload field or arbitrary remote image URL is accepted.
- [ ] Selection appears immediately as Nearby photo and atomically creates a pending admin review item; retries do not duplicate it.

**Verification:**
- [ ] Tests pass: `go test ./internal/adapters/sqlite ./internal/adapters/httpserver`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Create a photo-free place, choose a nearby preview, and inspect the pending review state.

**Dependencies:** Task 11, Task 18

**Files likely touched:**
- `internal/adapters/sqlite/placephoto.go`
- `internal/adapters/httpserver/place_photos.go`
- `internal/adapters/httpserver/place_photos_test.go`
- `views/place_photo_picker.go.tpl`
- `views/place_create.go.tpl`

**Estimated scope:** Medium (5 files); 2-hour timebox.

## Task 20: Review selected photos

- [ ] Complete Task 20

**Description:** Give admins a dedicated queue for nearby previews and the tools to resolve each selection.

**Acceptance criteria:**
- [ ] Only admins can list pending selections and approve, replace or remove the preview.
- [ ] Approval retains Nearby photo unless the admin explicitly confirms it depicts the actual place; removal restores the unavailable state.
- [ ] A stale review cannot overwrite a newer selection; resolution and actor are recorded once.

**Verification:**
- [ ] Tests pass: `go test ./internal/adapters/sqlite ./internal/adapters/httpserver`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Review as admin and non-admin, and resolve the same pending selection from two tabs.

**Dependencies:** Task 9, Task 19

**Files likely touched:**
- `internal/adapters/sqlite/photo_reviews.go`
- `internal/adapters/httpserver/admin_photos.go`
- `internal/adapters/httpserver/admin_photos_test.go`
- `views/admin_photos.go.tpl`
- `internal/adapters/httpserver/community.go`

**Estimated scope:** Medium (5 files); 1.5-hour timebox.

## Task 21: Save the displayed itinerary

- [ ] Complete Task 21

**Description:** Let a visitor persist exactly the plan currently displayed by the existing planner without signing in.

**Acceptance criteria:**
- [ ] Save operates on a server-issued reference bound to the rendered plan; it never silently regenerates from city/date parameters.
- [ ] Stored content preserves stops and day order with a schema version and owner, without embedding transient weather as current information.
- [ ] Expired/tampered references fail clearly while the displayed itinerary remains visible; saving creates a private plan.

**Verification:**
- [ ] Tests pass: `go test ./internal/adapters/httpserver ./internal/adapters/sqlite`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Generate, change the forecast fixture, then save and verify the original displayed route was retained.

**Dependencies:** Task 3, Task 9

**Files likely touched:**
- `internal/adapters/httpserver/trip.go`
- `internal/adapters/httpserver/saved_plans.go`
- `internal/adapters/httpserver/saved_plans_test.go`
- `views/itinerary_body.go.tpl`
- `internal/adapters/sqlite/savedplan.go`

**Estimated scope:** Medium (5 files); 1.5-hour timebox.

## Checkpoint: After Tasks 19–21

- [ ] All affected tests pass and `make build` succeeds; do not rerun unchanged checks without a reason.
- [ ] Demonstrate the completed feature path in a disposable local environment and record evidence.
- [ ] Report deviations; continue within the approved plan unless scope, deadline, budget or an external action needs a new decision.

## Task 22: Browse saved itineraries

- [ ] Complete Task 22

**Description:** Make browser-owned stored plans discoverable and viewable independently of the live planner.

**Acceptance criteria:**
- [ ] Personal navigation offers a private saved-plan list with stable links and an empty state.
- [ ] The owner can view a stored route when dates are past or external providers are unavailable.
- [ ] Private plans cannot be read by another user; views do not call the planner or relabel old forecast information as current.

**Verification:**
- [ ] Tests pass: `go test ./internal/adapters/httpserver`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Open a stored plan with providers disabled and verify a second browser cannot open its private URL.

**Dependencies:** Task 21

**Files likely touched:**
- `internal/adapters/httpserver/saved_plans.go`
- `internal/adapters/httpserver/saved_plans_test.go`
- `views/saved_plans.go.tpl`
- `views/saved_plan.go.tpl`
- `views/personal.go.tpl`

**Estimated scope:** Medium (5 files); 1.5-hour timebox.

## Task 23: Attach an itinerary recommendation

- [ ] Complete Task 23

**Description:** Let creators and commenters attach their own saved plan, either alone or alongside text.

**Acceptance criteria:**
- [ ] The contribution form can select an owned plan and explicitly publish an immutable snapshot with an inline preview/link.
- [ ] A display name is required; text-only, itinerary-only and combined submissions work for new/existing places, and a fully empty submission fails.
- [ ] Authors can replace/remove their own attachment; publishing does not expose unrelated private plans or mutate earlier published snapshots.

**Verification:**
- [ ] Tests pass: `go test ./internal/adapters/httpserver ./internal/adapters/sqlite`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Create an itinerary-only pin, add a combined contribution and remove its attachment as the author.

**Dependencies:** Task 10, Task 14, Task 22

**Files likely touched:**
- `internal/adapters/sqlite/contributions.go`
- `internal/adapters/httpserver/contributions.go`
- `internal/adapters/httpserver/contributions_test.go`
- `views/place_create.go.tpl`
- `views/place_panel.go.tpl`

**Estimated scope:** Medium (5 files); 2-hour timebox.

## Task 24: Copy a published itinerary

- [ ] Complete Task 24

**Description:** Turn a public itinerary recommendation into an independent saved plan owned by the current visitor.

**Acceptance criteria:**
- [ ] A visitor can preview a published snapshot and copy it into their browser-owned saved-plan list without signing in.
- [ ] The copy receives a new ID/owner and retains the route and source attribution; retries do not create accidental duplicates.
- [ ] Source removal or later source changes do not mutate an existing copy; no general itinerary editor is implied.

**Verification:**
- [ ] Tests pass: `go test ./internal/adapters/httpserver ./internal/adapters/sqlite`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Copy in a second browser, remove the source attachment from its author's browser, then verify the copy still opens unchanged.

**Dependencies:** Task 22, Task 23

**Files likely touched:**
- `internal/adapters/sqlite/savedplan.go`
- `internal/adapters/httpserver/saved_plans.go`
- `internal/adapters/httpserver/saved_plans_test.go`
- `views/saved_plan.go.tpl`
- `views/place_panel.go.tpl`

**Estimated scope:** Medium (5 files); 1.5-hour timebox.

## Checkpoint: After Tasks 22–24

- [ ] All affected tests pass and `make build` succeeds; do not rerun unchanged checks without a reason.
- [ ] Demonstrate the completed feature path in a disposable local environment and record evidence.
- [ ] Report deviations; continue within the approved plan unless scope, deadline, budget or an external action needs a new decision.

## Task 25: Report content

- [ ] Complete Task 25

**Description:** Provide public report entry points for places, comments and photos without requiring pre-publication approval.

**Acceptance criteria:**
- [ ] A visitor can submit a bounded reason against an existing target without an account; visitor identifiers are not exposed publicly.
- [ ] Duplicate/rate-limited reports receive a clear response and cannot automatically hide content; no contact email is required.
- [ ] Reports enter the admin queue while ordinary contributions continue publishing immediately.

**Verification:**
- [ ] Tests pass: `go test ./internal/adapters/httpserver ./internal/adapters/sqlite`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Report each supported target anonymously and confirm the content stays visible pending review.

**Dependencies:** Task 13, Task 14, Task 18

**Files likely touched:**
- `internal/adapters/sqlite/reports.go`
- `internal/adapters/httpserver/reports.go`
- `internal/adapters/httpserver/reports_test.go`
- `views/place_panel.go.tpl`
- `internal/adapters/httpserver/community.go`

**Estimated scope:** Medium (5 files); 1.5-hour timebox.

## Task 26: Moderate community content

- [ ] Complete Task 26

**Description:** Give admins a compact moderation workspace with report resolution and the ability to find and delete any comment, whether reported or not. Comment deletion is an explicit action separate from temporarily hiding content.

**Acceptance criteria:**
- [ ] Only admins can resolve reports or delete any reported/unreported comment. Deletion requires confirmation, a moderation reason and request protection; record actor, target and time.
- [ ] Deletion removes the comment and attached recommendation from public discussion, preserving the place, other contributions, likes, saved places and previously copied itineraries. Repeated deletion is safe; author edits, hide/restore and later merges cannot revive it.
- [ ] Report dismissal and temporary hide/restore actions still work independently of deletion; hiding a place preserves references and existing private copies, and public pages show an appropriate unavailable state.

**Verification:**
- [ ] Tests pass: `go test ./internal/adapters/httpserver ./internal/adapters/sqlite`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Delete reported and unreported comments as admin, including a place's first comment and one with an itinerary. Verify public removal, retained copies/place data, blocked non-admin deletion and blocked stale author edits; also dismiss a report and hide/restore another target.

**Dependencies:** Task 9, Task 25, Task 24

**Files likely touched:**
- `internal/adapters/sqlite/reports.go`
- `internal/adapters/httpserver/admin_reports.go`
- `internal/adapters/httpserver/admin_reports_test.go`
- `views/admin_reports.go.tpl`
- `internal/adapters/httpserver/community.go`

**Estimated scope:** Medium (5 files); 1.5-hour timebox.

## Task 27: Review shared-place corrections

- [ ] Complete Task 27

**Description:** Route suggested place-name/location changes through admins instead of granting creators shared editing rights.

**Acceptance criteria:**
- [ ] Anyone can propose a bounded correction or duplicate reference, and the suggestion reaches the admin queue.
- [ ] Only an admin can apply a validated name/location change; the actor and previous value are recorded.
- [ ] A location change invalidates inappropriate photo matches and refreshes map placement without moving contributions to a new identity.

**Verification:**
- [ ] Tests pass: `go test ./internal/adapters/httpserver ./internal/adapters/sqlite`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Suggest a correction as a visitor, apply it as admin and verify map/photo behavior.

**Dependencies:** Task 11, Task 20, Task 26

**Files likely touched:**
- `internal/adapters/httpserver/place_corrections.go`
- `internal/adapters/httpserver/place_corrections_test.go`
- `internal/adapters/sqlite/places.go`
- `views/place_panel.go.tpl`
- `views/admin_reports.go.tpl`

**Estimated scope:** Medium (5 files); 1.5-hour timebox.

## Checkpoint: After Tasks 25–27

- [ ] All affected tests pass and `make build` succeeds; do not rerun unchanged checks without a reason.
- [ ] Demonstrate the completed feature path in a disposable local environment and record evidence.
- [ ] Report deviations; continue within the approved plan unless scope, deadline, budget or an external action needs a new decision.

## Task 28: Preserve data during a duplicate merge

- [ ] Complete Task 28

**Description:** Implement a transactional merge operation on the complete community data model before exposing it to admins.

**Acceptance criteria:**
- [ ] Merging moves contributions, reports and photo-review references to the survivor, retaining an audit record and old-ID alias.
- [ ] Overlapping browser owners produce one like and one save each; published snapshots and independent copies retain their contents.
- [ ] Failure rolls back the entire merge; repeated/self/cyclic/concurrent merges are handled without orphan records.

**Verification:**
- [ ] Tests pass: `go test -race ./internal/adapters/sqlite`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Run a temporary-database merge with overlapping likes/saves and an injected failure, then inspect all references.

**Dependencies:** Task 15, Task 16, Task 20, Task 24, Task 27

**Files likely touched:**
- `internal/adapters/sqlite/place_merge.go`
- `internal/adapters/sqlite/place_merge_test.go`
- `internal/adapters/sqlite/community_migrations.go`

**Estimated scope:** Medium (3 files); 2-hour timebox.

## Task 29: Let admins merge duplicate pins

- [ ] Complete Task 29

**Description:** Expose a reviewable survivor/source merge action and keep old place links usable.

**Acceptance criteria:**
- [ ] An admin preview identifies the survivor, affected contributions, likes/saves and preview-photo decision before committing.
- [ ] Only an admin can confirm the merge; a stale preview is rejected instead of applying unexpected changes.
- [ ] Old links resolve to the survivor, old pins disappear and saved lists show one entry without losing contributions.

**Verification:**
- [ ] Tests pass: `go test ./internal/adapters/httpserver ./internal/adapters/sqlite`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Preview and confirm a merge, then follow an old link as a visitor and inspect both users' saved lists.

**Dependencies:** Task 28

**Files likely touched:**
- `internal/adapters/httpserver/admin_places.go`
- `internal/adapters/httpserver/admin_places_test.go`
- `views/admin_places.go.tpl`
- `internal/adapters/httpserver/places.go`
- `internal/adapters/httpserver/community.go`

**Estimated scope:** Medium (5 files); 1.5-hour timebox.

## Task 30: Expose admin navigation

- [ ] Complete Task 30

**Description:** Make moderation work discoverable from one admin entry point with separate report and photo queues.

**Acceptance criteria:**
- [ ] Admin navigation links to comment moderation, reports, photo reviews, corrections and merge tools with pending counts where applicable; comment deletion is accessible without a prior report.
- [ ] Public visitors cannot access queues through navigation or direct requests, and queue counts do not leak publicly.
- [ ] Queue lists are bounded and support pending/resolved states without loading every record.

**Verification:**
- [ ] Tests pass: `go test ./internal/adapters/httpserver`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Delete an unreported comment, complete one report and review one photo from the admin entry page; verify relevant counts update.

**Dependencies:** Task 20, Task 26, Task 27, Task 29

**Files likely touched:**
- `internal/adapters/httpserver/admin.go`
- `internal/adapters/httpserver/admin_test.go`
- `views/admin.go.tpl`
- `views/personal.go.tpl`
- `internal/adapters/httpserver/community.go`

**Estimated scope:** Medium (5 files); 1.5-hour timebox.

## Checkpoint: After Tasks 28–30

- [ ] All affected tests pass and `make build` succeeds; do not rerun unchanged checks without a reason.
- [ ] Demonstrate the completed feature path in a disposable local environment and record evidence.
- [ ] Report deviations; continue within the approved plan unless scope, deadline, budget or an external action needs a new decision.

## Task 31: Add deterministic community journeys

- [ ] Complete Task 31

**Description:** Extend the existing preview with temporary persistent community data and fake visitor identities/providers.

**Acceptance criteria:**
- [ ] Fixtures include two browser owners using the same name, an admin, empty destinations, exact/nearby/missing photos and a published itinerary; no production authorization bypass is added.
- [ ] Real handlers can exercise create, like, save, attach, copy and review against temporary SQLite state.
- [ ] Restart/reset behavior and fixture labels are documented; prior dashboard scenarios remain available.

**Verification:**
- [ ] Tests pass: `go test ./cmd/design-preview`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Walk one traveler journey and one admin journey without external credentials or internet access.

**Dependencies:** Task 12, Task 17, Task 24, Task 30

**Files likely touched:**
- `cmd/design-preview/community.go`
- `cmd/design-preview/community_test.go`
- `cmd/design-preview/server.go`
- `cmd/design-preview/main.go`
- `tasks/community-map/verification.md`

**Estimated scope:** Medium (5 files); 1.5-hour timebox.

## Task 32: Verify mobile map journeys

- [ ] Complete Task 32

**Description:** Check the real browser experience, focusing on the map lifecycle and accessible place interactions.

**Acceptance criteria:**
- [ ] At 375 px and 1440 px, create/open/like/save/copy controls work without clipping, trapped focus or page-scroll interference.
- [ ] Rapid destination changes, panel reopening and browser back/forward do not duplicate handlers or show another destination's data.
- [ ] Keyboard navigation, loading/error states and supplied-photo attribution work with tiles or photos unavailable.

**Verification:**
- [ ] Tests pass: `go test ./cmd/design-preview ./internal/adapters/httpserver`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Run the new browser check using the repository's isolated-browser convention; retain screenshots and failures.

**Dependencies:** Task 31

**Files likely touched:**
- `tasks/community-map/browser-check.mjs`
- `public/redesign/community-map.js`
- `public/redesign/community-map.css`
- `views/place_panel.go.tpl`
- `tasks/community-map/verification.md`

**Estimated scope:** Medium (5 files); 1.5-hour timebox.

## Task 33: Verify community write boundaries

- [ ] Complete Task 33

**Description:** Exercise hostile and concurrent requests across the implemented features rather than merely retesting happy paths.

**Acceptance criteria:**
- [ ] Cross-browser edits, non-admin comment deletion, forged admin requests, CSRF, unsafe names/text/URLs and unauthorized snapshot access are rejected.
- [ ] Concurrent like/save/retry/merge/review operations preserve uniqueness, ownership and reference integrity; deletion racing an author edit or a merge cannot revive a deleted comment or remove independent itinerary copies.
- [ ] Write limits and pagination bounds work under repeated requests without blocking public reading; failures identify a focused follow-up before completion.

**Verification:**
- [ ] Tests pass: `go test -race ./internal/adapters/httpserver ./internal/adapters/sqlite`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Run the author/second-browser/admin attack matrix against the local fixture and inspect rendered escaping.

**Dependencies:** Task 30, Task 31

**Files likely touched:**
- `internal/adapters/httpserver/community_security_test.go`
- `internal/adapters/sqlite/community_concurrency_test.go`
- `internal/adapters/httpserver/community.go`
- `tasks/community-map/verification.md`

**Estimated scope:** Medium (4 files); 2-hour timebox.

## Checkpoint: After Tasks 31–33

- [ ] All affected tests pass and `make build` succeeds; do not rerun unchanged checks without a reason.
- [ ] Demonstrate the completed feature path in a disposable local environment and record evidence.
- [ ] Report deviations; continue within the approved plan unless scope, deadline, budget or an external action needs a new decision.

## Task 34: Rehearse migration and recovery

- [ ] Complete Task 34

**Description:** Prove the new user data survives deployment preparation and can be restored using a disposable database.

**Acceptance criteria:**
- [ ] Migrating a copy of the previous schema preserves cached data and creates required tables; rerunning migrations is harmless.
- [ ] A database-consistent backup restores visitor ownership, saved places, published plans and review decisions in a fresh process.
- [ ] The release runbook defines compatible rollback behavior and never drops user data to revert the UI.

**Verification:**
- [ ] Tests pass: `go test ./internal/adapters/sqlite`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Migrate, back up and restore a temporary pre-community database; compare counts and representative records.

**Dependencies:** Task 28, Task 31

**Files likely touched:**
- `internal/adapters/sqlite/community_migrations_test.go`
- `tasks/community-map/recovery.md`
- `tasks/community-map/verification.md`

**Estimated scope:** Medium (3 files); 1.5-hour timebox.

## Task 35: Measure the completed workload

- [ ] Complete Task 35

**Description:** Repeat resource measurements with realistic community reads/writes and provider failure simulations.

**Acceptance criteria:**
- [ ] Document a reproducible workload including map browsing, discussions, likes, saves, copying and admin actions at 10 concurrent fixture users.
- [ ] Record peak memory, latency and error rate with warm/cold data; investigate failures against declared thresholds and repeat only affected checks.
- [ ] Assess 512 MB headroom in a comparable Linux environment if available, recording uncertainty otherwise; do not resize Fly or inspect billing.

**Verification:**
- [ ] Tests pass: `go test ./cmd/loadtest-server`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Run the local workload, retain metrics and state precisely which production behaviors were not measured.

**Dependencies:** Task 5, Task 31, Task 33

**Files likely touched:**
- `loadtests/community.js`
- `cmd/loadtest-server/community.go`
- `cmd/loadtest-server/main.go`
- `loadtests/README.md`
- `tasks/community-map/verification.md`

**Estimated scope:** Medium (5 files); 1.5-hour timebox.

## Task 36: Prepare the release candidate

- [ ] Complete Task 36

**Description:** Document and validate the complete release configuration without changing production resources.

**Acceptance criteria:**
- [ ] Document admin secret configuration, visitor-cookie lifetime/recovery limitations, tile settings and feature enable/disable behavior without secrets.
- [ ] Full tests, race checks, lint, builds and container startup succeed; regression evidence covers existing search/planner/export flows.
- [ ] Record live admin-auth/photo smoke results separately from fixtures and list unresolved release blockers, retaining the full agreed scope.

**Verification:**
- [ ] Tests pass: `go test -race -count=1 ./...`.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Run make lint and make build, verify the container, then review every requirement against the release checklist.

**Dependencies:** Task 32, Task 33, Task 34, Task 35

**Files likely touched:**
- `README.md`
- `tasks/community-map/release.md`
- `tasks/community-map/verification.md`
- `.env.example`
- `Dockerfile`

**Estimated scope:** Medium (5 files); 1.5-hour timebox.

## Checkpoint: After Tasks 34–36

- [ ] All affected tests pass and `make build` succeeds; do not rerun unchanged checks without a reason.
- [ ] Demonstrate the completed feature path in a disposable local environment and record evidence.
- [ ] Obtain owner review of the concrete release candidate before Task 37; record unresolved limits, including unverified operating cost.

## Task 37: Release the reviewed community map

- [ ] Complete Task 37

**Description:** After the owner reviews the candidate, perform the documented deployment and verify the public flows.

**Acceptance criteria:**
- [ ] The reviewed candidate and migration/restore plan are approved before changing production; no infrastructure resize or paid service is bundled.
- [ ] Deployment preserves existing data; public smoke checks cover browsing, named posting, likes, saves, copying and protected admin review.
- [ ] Record deployed revision and outcomes; use the rehearsed rollback if release checks fail and never claim the deferred $5 ceiling was verified.

**Verification:**
- [ ] Verify Task 36's passing test/build evidence matches the reviewed revision; rerun affected checks only if the revision changed.
- [ ] Build succeeds: `make build` (reuse unchanged passing evidence for documentation/release-only steps).
- [ ] Manual check: Deploy the reviewed revision using the established Fly workflow, then run the documented public smoke checks without publishing fake community recommendations.

**Dependencies:** Task 36

**Files likely touched:**
- `tasks/community-map/release.md`
- `tasks/community-map/verification.md`

**Estimated scope:** Small (2 files); 1.5-hour timebox.

## Checkpoint: After Task 37

- [ ] All affected tests pass and `make build` succeeds; do not rerun unchanged checks without a reason.
- [ ] Demonstrate the completed feature path in a disposable local environment and record evidence.
- [ ] Record the public smoke outcome and deployed revision; close only requirements actually verified.

