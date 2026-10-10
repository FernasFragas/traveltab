# Dependency security and self-hosted browser libraries

Date: 2026-10-09. Source: [dependency-security-plan.md](../dependency-security-plan.md),
particularly step 5. All changes remain uncommitted.

## Journeys

- As the maintainer, I want Dependabot to propose Bootstrap, bootstrap-icons and htmx
  updates, so that browser libraries do not go stale unnoticed.
- As the maintainer, I want a browser-library update to need no manual step, so that an
  urgent patch is merged as soon as CI passes.
- As a visitor, I want the existing styles, icons and planning interactions to load from
  the application, so that they work without a runtime CDN dependency.

**Revision 2026-10-09:** the first pass used a custom downloader (`cmd/vendorassets`) with
committed files and a version marker. After review it was replaced by `package-lock.json`
and `npm ci --ignore-scripts`, and OSV-Scanner was removed. The downloader's tests went
with it; the evidence below is for the replacement.

## RED → GREEN

With the downloader and `public/vendor/` removed, the new server test was run first:

```sh
go test ./internal/adapters/httpserver/ -run TestNewAppServer_ServesBrowserLibraries
```

```text
        	Error:      	Not equal:
        	            	expected: 200
        	            	actual  : 404
        	Test:       	TestNewAppServer_ServesBrowserLibraries
        	Messages:   	/vendor/htmx.org/dist/htmx.min.js
FAIL	weatherservice/internal/adapters/httpserver	0.530s
```

`npm install --package-lock-only --ignore-scripts` created the lockfile (four packages,
each with an `integrity` hash; `@popperjs/core` is Bootstrap's peer dependency).
`npm ci --ignore-scripts` installed them, and `server.go` briefly gained
`app.Static("/vendor", "./node_modules")`. The same command then passed:

```text
ok  	weatherservice/internal/adapters/httpserver	0.391s
```

```sh
go test -cover ./internal/adapters/httpserver/
```

```text
ok  	weatherservice/internal/adapters/httpserver	0.836s	coverage: 92.7% of statements
```

### Publish only the files the page loads

Serving all of `node_modules/` published about 2,585 package files. A follow-up review
asked for an explicit copy instead. The test came first:

```sh
go test ./internal/adapters/httpserver/ -run TestNewAppServer_HidesOtherPackageFiles
```

```text
--- FAIL: TestNewAppServer_HidesOtherPackageFiles (0.09s)
        	            	expected: 404
        	            	actual  : 200
FAIL	weatherservice/internal/adapters/httpserver	0.605s
```

The `/vendor` route was removed, and a `vendor` script in `package.json` now copies the five
files into git-ignored `public/vendor/`. `make assets` and the Docker `assets` stage run
`npm ci --ignore-scripts && npm run vendor`, and the image copies only `public/vendor/`.
Then `go test ./internal/adapters/httpserver/ -run 'TestNewAppServer_'` passed:

```text
ok  	weatherservice/internal/adapters/httpserver	0.434s
```

Coverage stayed at 92.7% (`go test -cover ./internal/adapters/httpserver/`).

| What is guaranteed | Test name | Unit/integration | Result |
|---|---|---|---|
| All five files the page loads are served from `/vendor/` with a body | `TestNewAppServer_ServesBrowserLibraries` | Integration, server + `public/vendor` | PASS |
| Other package files (e.g. `/vendor/bootstrap/package.json`) return 404 | `TestNewAppServer_HidesOtherPackageFiles` | Integration | PASS |
| Package contents match `package-lock.json` hashes | `npm ci` (not a Go test) | Tooling | PASS |
| The image holds only the five files; it serves them and 404s `package.json` | `find` and `curl` against `docker run traveltab` | Manual, container | 5 files; 200 ×5, 404 |
| CSS, both fonts, htmx and planning work at 375/1440 px without overflow or console errors | `tasks/redesign/checks/vendor-assets.mjs` | Browser, design preview | PASS |

**Known gap:** `npm ci` trusts what the npm registry published under each hash. It proves the
files are unchanged since the lockfile was written, not that the release is benign.

## Browser verification

An isolated headless Brave profile used debugging port 9437. The check starts and stops
`cmd/design-preview` itself:

```sh
CDP_PORT=9437 node tasks/redesign/checks/vendor-assets.mjs
```

```text
PASS 375px: local CSS, fonts, htmx, planning, no overflow
PASS 1440px: local CSS, fonts, htmx, planning, no overflow
PASS: no JavaScript exceptions or console errors
```

Recorded [browser results](artifacts/dependabot-vendor-assets/results.json),
[375 px screenshot](artifacts/dependabot-vendor-assets/375.png) and
[1440 px screenshot](artifacts/dependabot-vendor-assets/1440.png) were inspected.
Bootstrap CSS rules and icon font loading were checked, both font files returned 200,
and a three-day plan was generated through the actual local htmx script.

The updated `browser-smoke.mjs` and `interaction-regressions.mjs` also ran against a
preview started with `startPreview()` and `CDP_PORT=9437`. They passed at 375, 768, 1100
and 1440 pixels and passed planning/history/keyboard/video interaction checks:

```text
PASS default: interactions, reduced motion, 200% zoom, no JavaScript exceptions
PASS: actual history restoration without planner; no browser exceptions
```

All browser check scripts passed `node --check`. Photos, map and provider data in these
checks are fixtures; this does not verify live external providers.

### Browser checks on dependency PRs

`.github/workflows/browser.yml` runs `interaction-regressions.mjs` and `browser-smoke.mjs`
when a PR changes `package.json` or `package-lock.json`. The same sequence ran locally
(design preview on 8087, headless Chrome 155 via Brave on CDP 9227, `OUTPUT_DIR` in `.tmp`):

| Library state | interaction-regressions | browser-smoke |
|---|---|---|
| htmx 1.9.11 (current) | PASS, exit 0 | PASS at 375/768/1100/1440, exit 0 |
| htmx 2.0.11 (simulated Dependabot major bump, then restored) | PASS, exit 0 | PASS, exit 0 |
| Empty `htmx.min.js` (negative control, then `make assets`) | FAIL: timed out waiting for `window.htmx`; exit 1 | FAIL: timed out waiting for htmx; exit 1 |

The htmx 2 result means the checked interactions survive that upgrade; it doesn't prove
every htmx 2 change is harmless. The hosted run (Chrome with `--no-sandbox`) hasn't run yet.

## Security changes and verification

The initial govulncheck run found 19 reachable vulnerabilities in five modules and the
Go standard library. Go 1.27.1 therefore could not meet the plan's clean-scan requirement.
[GO-2026-6617](https://pkg.go.dev/vuln/GO-2026-6617), among the findings, is fixed in
Go 1.27.2 and `golang.org/x/net` 0.60.0. The implementation uses Go **1.27.2** in both
`go.mod` and Docker. Module upgrades use scanner-reported fixes and Go's required
transitive versions. No application Go source needed changing.

The initial image scan found seven fixable HIGH/CRITICAL Debian findings and 27 in the
Go binary. The runtime image now applies available Debian upgrades, fixing `perl-base`;
Go module updates fix the binary findings. There is no `.trivyignore`.

| Check | Result |
|---|---|
| `make test lint build` with golangci-lint 2.14.0 on PATH | PASS; race-enabled tests, zero lint issues, binary built |
| `docker build -t traveltab .` | PASS; Node 24.18.1 asset stage, Go 1.27.2 builder, patched Debian runtime |
| `go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...` | PASS; zero called vulnerabilities |
| `npm audit --audit-level=high` | PASS; found 0 vulnerabilities |
| Trivy 0.70.0 image scan, HIGH/CRITICAL, ignore-unfixed, exit-code 1 | PASS; zero findings for Debian and the Go binary (npm packages are no longer in the image) |
| actionlint 1.7.12 on `ci.yml` | PASS |
| `git diff --check` | PASS |

Final [full-suite output](artifacts/dependabot-vendor-assets/full-suite.txt).

Final scan logs: [govulncheck](artifacts/dependabot-vendor-assets/govulncheck.txt),
[Trivy](artifacts/dependabot-vendor-assets/trivy.txt) (first pass, before the npm change; the
npm-change image scan was rerun with the same command and passed).
Govulncheck reports one module-only advisory for the unimported, uncalled
`golang.org/x/crypto/openpgp` package (GO-2026-5932); this does not fail its reachability gate.
The scan results are observations against the databases available on the run date.

The final local image scan command matched the pinned Trivy action's default version:

```sh
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock -v /Users/fernando/personal-projects/traveltab/.tmp/trivy-cache:/root/.cache/ aquasec/trivy:0.70.0 image --severity CRITICAL,HIGH --ignore-unfixed --exit-code 1 traveltab
```

The full-suite command used
`PATH=/Users/fernando/personal-projects/traveltab/.tmp/security-bin:$PATH make test lint build`.
One intermediate run passed tests but formatting caught the temporary negative fixture;
that fixture was formatted, lint/build passed, and the final combined check was run after
the small downloader refactor. After the npm change, `make test lint build` was rerun and
passed (13 packages, 0 lint issues). No application lint suppressions were added.

## Proving the scanners fail

An isolated `.tmp/security-negative` module (ignored by Git and normal source scanning)
used Go 1.27.2 with `golang.org/x/net v0.38.0`. Its `main` calls
`(&http2.Transport{}).RoundTrip(request)`; the binary was built but never run. A scratch
image containing that binary was tagged `traveltab-security-negative`.
This substitutes a disposable fixture for the plan's scratch branch, preserving the
owner's working tree and the no-commit rule.

```sh
# From .tmp/security-negative:
go mod tidy
go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o app .
docker build -t traveltab-security-negative .
# From the repository root:
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock -v /Users/fernando/personal-projects/traveltab/.tmp/trivy-cache:/root/.cache/ aquasec/trivy:0.70.0 image --severity CRITICAL,HIGH --ignore-unfixed --exit-code 1 traveltab-security-negative
```

| Scanner | Expected failure observed | Evidence |
|---|---|---|
| govulncheck | Eight called vulnerabilities from two modules; scanner exit 3 (`go run` exits 1) | [log](artifacts/dependabot-vendor-assets/govulncheck-negative.txt) |
| npm audit | Scratch `.tmp/npm-audit-negative` lockfile with `lodash 4.17.4`: 1 critical; exit 1 | Command output only |
| Trivy | Six HIGH vulnerabilities; exit 1 | [log](artifacts/dependabot-vendor-assets/trivy-negative.txt) |

## Verified upstream details and remaining rollout

Dependabot's npm ecosystem updates `package.json` and `package-lock.json` together, so a
browser-library PR is complete: CI and the Docker build install the new version with
`npm ci --ignore-scripts`.

The GitHub tag APIs showed the plan's Trivy/CodeQL hashes were annotated tag objects.
The workflow pins their resolved commits:

- Trivy action 0.36.0: `ed142fd0673e97e23eac54620cfb913e5ce36c25`.
- CodeQL uploader 4.38.3: `24c54180a607b1449ed407dd24f251e4e9147c8d`.

Trivy uploads SARIF on pushes to `main` even when its findings cause a failure; PRs don't
upload, so fork/Dependabot read-only tokens are enough. Hosted behavior has not been
exercised locally.

**Owner-only, still pending:** enable Dependency graph, Dependabot alerts and Dependabot
security updates in the repository UI; merge/push; confirm all four Dependabot ecosystems
have no configuration errors and Trivy appears in the Security tab. No commit, push or
repository-settings mutation was performed.
