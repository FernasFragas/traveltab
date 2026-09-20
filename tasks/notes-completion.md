# MVP completion pass — 2026-09-20

All 17 implementation tasks are complete in the working tree. This pass preserved the existing
work and finished the interrupted follow-ups. v2 remains deferred until the MVP ships.

## Changes and evidence

- Wikimedia parallel lookups were incomplete. A sequential helper restored compilation and
  produced the expected concurrency assertion failure (`1 is not greater than 1`). The bounded
  implementation passes race, ordering, failure-order and cancellation tests.
- Forecast fetches overlap place fetches, cancel on early failure and show an outage note.
  Assertion-based RED/GREEN evidence is in [Build notes](notes-planner-build.md).
- Three city tests now require a full plan and known landmarks. With an empty `PlaceTypes`
  supplied through a temporary Go overlay, all three fail on assertions; the real map is untouched.
- Kyoto's three-day plan stranded Arashiyama despite an unused bamboo grove nearby. The new city
  regression failed on a one-stop day and missing Q23579173. Grouping now rescues such days from
  unused top-20 candidates within 3 km. Grouping and city tests pass. Genuinely isolated places
  can still stand alone when no nearby candidate exists; the one-place-town case remains valid.
- The complete route renders identical plans when all three providers fail, with both fresh and
  expired caches. `TestPlanRoute_CachedPlanSurvivesAllSourcesOffline` is a guard test, not claimed
  as original RED evidence. Local fixture requests measured below 1 ms in its focused run.
- Input, weather display, stop labels and contrast fixes are documented in the
  [UI validation notes](notes-ui-validation.md).
- Docker failed on ARM with `gcc: unrecognized command-line option '-m64'`. The Dockerfile now
  downloads Go for `TARGETARCH`, matching its C compiler. The native ARM image builds successfully.
  `.dockerignore` excludes local secrets, SQLite files, build outputs and screenshot artifacts.

## Final verification

```text
make test    PASS: all packages, -race -count=1
make lint    PASS: 0 issues
make build   PASS
docker build -q -t traveltab .   PASS (native ARM)
```

The final image built as `sha256:acc516ac6552157247e8f083591015f6733352b4039bd314e38b2810dabac351`.
Browser evidence covers the real templates/HTMX with fixture providers at 375 px. Recorded
city acceptance tests exercise the real source clients and planner without network requests.
Only the three documented `LIVE_API_TESTS=1` tests skip in the default suite.

## Release boundaries and known limits

- Human review of generated place classifications, commit/PR, remote CI and deployment remain
  release steps. No commit, push or deployment was performed by this completion pass.
- Historical test-first evidence is incomplete for some original tasks; this pass does not claim
  to reconstruct it. The new defects were reproduced before their fixes.
- The browser fixture deliberately blanks the map embed. Production maps and every external
  provider were not revalidated. Commons photos and CDN HTMX loaded in the browser check.
- The Kyoto hotel fixture was recorded around older coordinates, so it does not establish hotel
  availability at every newly computed plan center. Stay radius and fallback behavior have
  separate tests. Walk distances are straight-line estimates; some city days remain long.
- Local Docker validation used ARM. Remote CI still needs to validate its own target environment.
