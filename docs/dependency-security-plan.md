# Plan: Dependabot, Go 1.27, Trivy, govulncheck

## Revision after review — 2026-10-09

**This revision replaces the original OSV-Scanner work and steps 4–5 below; the rest still applies.**

- **OSV-Scanner removed.** govulncheck covers Go code more precisely and Trivy covers the
  image, so a third scanner mostly added duplicate findings. `.github/workflows/security.yml`
  is gone.
- **The custom downloader is replaced by the standard npm lockfile.**
  - `package-lock.json` stores each package's integrity hash, and `npm ci --ignore-scripts` checks it.
  - `make assets`, CI (`actions/setup-node`) and a `node` Docker stage install the packages.
  - `npm run vendor` copies only the five files the page loads into `public/vendor/` (git-ignored).
    The existing `public/` route serves them; `node_modules/` is never published or shipped.
  - A Dependabot browser-library PR needs no manual step.
- **Removed:** `cmd/vendorassets/`, `public/vendor/` (including `versions.json`) and
  `make vendor-assets`.
- **New tests:** `TestNewAppServer_ServesBrowserLibraries` checks that each `/vendor/` file the page loads is served;
  `TestNewAppServer_HidesOtherPackageFiles` checks that other package files return 404.
- **Unchanged:** Dependabot (all four ecosystems), cooldowns, grouping, manual merge,
  govulncheck, Trivy with the Security tab upload, and the Go 1.27.2 upgrade.

## Implementation status (first pass) — 2026-10-09

Repository implementation and local verification are complete. See
[the TDD and security evidence](tdd/dependabot-vendor-assets.tdd.md) for commands,
results, browser screenshots and the remaining owner rollout checks.

Corrections established during implementation:

- Go **1.27.2** replaces the planned 1.27.1: the live vulnerability database reports
  standard-library security fixes in 1.27.2. Vulnerable Go modules and required transitive
  dependencies were upgraded as well; `go mod tidy` alone does not upgrade them.
- Trivy `v0.36.0` resolves to commit `ed142fd0673e97e23eac54620cfb913e5ce36c25`;
  CodeQL `v4.38.3` resolves to `24c54180a607b1449ed407dd24f251e4e9147c8d`.
  The original hashes below were annotated tag objects, not commits.
  OSV's full commit is `a345acffa64b0eaede81a3d9aae6141214d9c8fc`.
- The runtime Docker stage applies available Debian updates to fix the base image's
  `perl-base` findings. No vulnerability exclusions were added.
- Dependabot's source confirms a manifest without a lockfile is supported.
- All active browser scripts that referenced CDN htmx were updated, including four
  scripts beyond the two named below.
- Scanner failure checks used an isolated scratch module/image with vulnerable `x/net`,
  avoiding branch changes or commits. All three failed for vulnerability findings.
- OSV SARIF upload is disabled on PRs and enabled on `main`/scheduled scans.

Still owner-only: enable the three GitHub security settings below, merge/push, and check
hosted Dependabot and Security results. No commits or pushes were made.

## Context

- **Problem:** nothing keeps dependencies current. Go is pinned to **1.23.7**, which is past end of life, in two places (`go.mod`, and `Dockerfile` via `wget`). golangci-lint is pinned at `v2.4.0`. The front-end libraries are hard-coded CDN URLs. Nothing scans for known vulnerabilities.
- **Outcome:** Dependabot opens PRs for Go modules, Actions, Docker images (including the Go toolchain) and front-end libraries. Three scanners cover what Dependabot doesn't:

| Tool | Scans | Where it runs |
|---|---|---|
| OSV-Scanner | `go.mod` and Go standard-library vulnerabilities | PRs (new issues only), plus `main` and weekly (Security tab) |
| govulncheck | Go vulnerabilities in code that is **actually called** | CI lint/test |
| Trivy | Built image: Debian packages and compiled binary | CI `docker` job (Security tab on `main`) |

- **Versions checked on 2026-10-08:** Go `1.27.1`, golangci-lint `v2.14.0`, osv-scanner-action `v2.6.0` (`a345acff…`), trivy-action `v0.36.0` (`a9c7b0f0…`), govulncheck `v1.8.0`.
- **Pin both scanner actions by full commit SHA.** trivy-action re-tagged its releases after a supply-chain attack in 2026. Dependabot keeps SHA pins updated and preserves the `# vX.Y.Z` comment.
- **Rules:** no commits, as `AGENTS.md` §6 requires. TDD applies only to step 5, which changes behaviour; the rest is config.

## Steps

### 1. Upgrade Go to 1.27.1
- [x] `go.mod`: set `go 1.27`, add `toolchain go1.27.1`, then `go mod tidy`.
- [x] `Dockerfile`: builder becomes `FROM golang:1.27.1-bookworm AS builder`. Drop the `wget`/`tar`/`GOLANG_VERSION`/`PATH` lines and keep `CGO_ENABLED=1` (`gcc` is in the image). Dependabot can now bump the Go version through this tag.
- [x] `.github/workflows/ci.yml`: golangci-lint `v2.4.0` → `v2.14.0`. `setup-go` `go-version-file: go.mod` already reads the `toolchain` line.
- [x] `README.md:27` and `README.md:201`: change "Go 1.23" to "Go 1.27".
- [x] Fix any new lint findings. Don't change behaviour.
- Check: `make test lint build` and `docker build -t traveltab .` both pass.

### 2. Add `.github/dependabot.yml`
- [x] Add the file:
```yaml
version: 2
updates:
  - package-ecosystem: gomod
    directory: /
    schedule: { interval: weekly, day: monday }
    open-pull-requests-limit: 5
    cooldown: { semver-major-days: 14, semver-minor-days: 7, semver-patch-days: 3 }
    groups:
      go-minor-patch: { update-types: [minor, patch] }
      go-security: { applies-to: security-updates, patterns: ["*"] }

  - package-ecosystem: github-actions
    directory: /
    schedule: { interval: weekly, day: monday }
    cooldown: { default-days: 7 }
    groups:
      actions: { patterns: ["*"] }

  - package-ecosystem: docker          # golang:<ver>-bookworm and debian:bookworm-slim
    directory: /
    schedule: { interval: weekly, day: monday }
    cooldown: { default-days: 7 }

  - package-ecosystem: npm             # manifest only, see step 5
    directory: /
    schedule: { interval: weekly, day: monday }
    cooldown: { semver-major-days: 14, semver-minor-days: 7, semver-patch-days: 3 }
    groups:
      web-minor-patch: { update-types: [minor, patch] }
      web-security: { applies-to: security-updates, patterns: ["*"] }
```
- Cooldown applies only to version updates; security updates always open immediately. Docker and Actions accept `default-days` only.
- [ ] **Owner, in the web UI, not Claude:** Settings → Code security, turn on Dependency graph, Dependabot alerts and Dependabot security updates.

### 3. Add the scanners
- [x] **New `.github/workflows/security.yml`** for OSV-Scanner:
  - On `pull_request`, run `google/osv-scanner-action/.github/workflows/osv-scanner-reusable-pr.yml@a345acff… # v2.6.0`.
  - On `push` to `main` and on a `schedule` (weekly, Monday), run `osv-scanner-reusable.yml@a345acff… # v2.6.0`.
  - Permissions: `security-events: write`, `contents: read`, `actions: read`.
- [x] **`ci.yml` `test` job:** add a `Vulnerability check` step after Test, `go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...`. Pinning the version keeps the run reproducible, and it uses the Go version that `setup-go` installed.
- [x] **`ci.yml` `docker` job:** after `Build image`, run `aquasecurity/trivy-action@a9c7b0f06e461e9d4b4d1711f154ee024b8d7ab8 # v0.36.0` with:
  - `image-ref: traveltab`, `severity: CRITICAL,HIGH`, `ignore-unfixed: true`, `exit-code: '1'`
  - `format: sarif`, `output: trivy-results.sarif`, `limit-severities-for-sarif: true`
- [x] Next step: `github/codeql-action/upload-sarif@cee97f86b972a3ded3c58d04148362b193ddc518 # v4.38.3`, with `if: always() && github.event_name == 'push'` so findings reach the Security tab even when Trivy fails.
  - Upload only on pushes to `main`: Dependabot and fork PRs get a read-only token, so an upload from a PR would fail.
  - Add `security-events: write` to the `docker` job only.
- [x] `.trivyignore` only if needed: no exclusions were necessary; the final Trivy scan passes.

### 4. Answer to "How can Dependabot track the CDN libraries?"
Dependabot only reads manifest files (`package.json`, `go.mod` and similar). It can't see URLs in a template. The CDN URLs also carry SRI hashes, so a version bump would break them anyway. The fix is to **list the libraries in a `package.json` and serve them from `public/vendor/`**, which step 5 does. Node isn't needed at build or run time.

### 5. Track and self-host the front-end libraries (TDD)
**Journey:** as the maintainer, I want Dependabot to propose Bootstrap, bootstrap-icons and htmx updates, so that front-end libraries don't go stale unnoticed.

- [x] Root `package.json` (`"private": true`), pinned exactly: `bootstrap 5.2.3`, `bootstrap-icons 1.11.3`, `htmx.org 1.9.11`. No `node_modules`. Dependabot reads only the manifest. **Check during implementation** that Dependabot runs without a lockfile. If it doesn't, commit a `package-lock.json` from `npm install --package-lock-only`.
- [x] **New `cmd/vendorassets`**, matching the other `cmd/*` tools. It reads `package.json`, downloads the fixed file list from `cdn.jsdelivr.net/npm/<pkg>@<ver>/…` into `public/vendor/`, and writes `public/vendor/versions.json`. Files:
  - `bootstrap/dist/css/bootstrap.min.css`
  - `bootstrap-icons/font/bootstrap-icons.min.css`, plus `font/fonts/bootstrap-icons.woff2` and `.woff`
  - `htmx.org/dist/htmx.min.js`
- [x] `Makefile`: add a `vendor-assets` target that runs it.
- [x] `views/index.go.tpl:12,13,23`: point at `/vendor/...`. Drop the `integrity` and `crossorigin` attributes, since the files are now same-origin.
- [x] **Test, written RED first:** `TestVendoredAssets_MatchPackageJSON` checks that `versions.json` equals the `package.json` versions and that every listed file exists. A Dependabot PR therefore fails CI until someone runs `make vendor-assets` on its branch. That failure is the reminder.
- [x] Unit test for `cmd/vendorassets` against `httptest`, with no live calls. Coverage ≥80%.
- [x] Browser checks: `tasks/redesign/checks/lib/browser.mjs:110,143` and `browser-smoke.mjs:52,126` intercept `https://unpkg.com/htmx.org@1.9.11`. Change them to the local `/vendor/` path, or drop the interception.
- [x] Runtime check in `cmd/design-preview`: page renders, console is clean, layout correct at 375 and 1440 px.
- [x] Evidence report: `docs/tdd/dependabot-vendor-assets.tdd.md`.
- Bootstrap stays and is tracked like the other two libraries, as decided.

### 6. Document
- [x] `docs/maintenance.md`: new "Dependency and vulnerability checks" section covering:
  - What each tool covers (the table above).
  - How to handle a failed scan.
  - A Dependabot `package.json` PR needs `make vendor-assets` run on its branch.
  - A Go minor upgrade (for example 1.28) arrives as a Docker PR, and the `go`/`toolchain` lines in `go.mod` are raised by hand in the same PR.
  - Not tracked: Google Fonts.

## Verification
1. `make test lint build` passes.
2. `docker build -t traveltab .` passes, and Trivy passes locally: `docker run --rm -v /var/run/docker.sock:/var/run/docker.sock aquasec/trivy image --severity CRITICAL,HIGH --ignore-unfixed traveltab`.
3. `go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...` reports no called vulnerabilities.
4. YAML parses for `dependabot.yml`, `ci.yml` and `security.yml` (`actionlint` if installed).
5. The scanners can fail: on a scratch branch, pin `golang.org/x/net` to a version with a known vulnerability. OSV-Scanner, govulncheck and Trivy should all go red. Then revert.
6. After the owner merges and pushes, Insights → Dependency graph → Dependabot shows every ecosystem with no config errors, and the Security tab lists OSV results.

## Risks
- **The first runs will be noisy.** The scanners will probably flag existing vulnerabilities (for example `x/net v0.38.0`) until step 1's tidy and the first Dependabot batch land. That's why steps 1 and 2 come first.
- **Go version drift.** Dependabot's Docker PR can move the build to Go 1.28 while `go.mod` still says 1.27. That builds fine, but CI then uses 1.27. The maintenance note covers raising the lines by hand.
- **Local Go is 1.26.2.** `toolchain go1.27.1` makes `go` download 1.27.1 automatically, unless `GOTOOLCHAIN=local` is set.
- **No auto-merge.** Every Dependabot PR is merged by hand.
