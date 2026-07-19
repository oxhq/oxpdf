# Release Readiness

This project should not be called release-ready from source-repo checks alone.
Use the gates below to name the highest proof level actually completed.

## Rust Cutover Status

The Rust OxPDF facade is an unreleased migration slice, not a replacement for
the Go module yet. It consumes the local Binas Rust crate through a path
dependency while both repositories are under active development. That proves a
direct in-process integration only; it is not a reproducible package, consumer,
hosted-CI, or release proof.

Before a Rust OxPDF release claim:

- Binas must publish, or be available from a clean immutable ref, at the exact
  `binas-pdf` revision used by OxPDF.
- Replace the local path dependency with that package version or immutable ref.
- Run the Rust contract suite, an independent downstream Rust consumer, and
  hosted checks against those exact revisions.
- Exercise the Go rollback window before retiring the Go module.

The local Rust slice is checked from `rust/` with:

```powershell
cargo fmt --all -- --check
cargo test -p oxpdf
cargo clippy -p oxpdf --all-targets -- -D warnings
```

## Local Gate

Run from the repository root:

```powershell
go test ./...
go vet ./...
```

Passing this gate means the current checkout builds, tests, and vets locally on
this machine. It does not prove consumer installability, hosted CI, tagged
release behavior, registry/module availability, or deployed runtime behavior.

## Consumer Gate

Before claiming consumer proof, install or replace `github.com/oxhq/oxpdf` from
a separate downstream Go module and exercise the newly exposed APIs against
representative PDFs. Record the consuming module path, command output, and any
replace directive used.

## Hosted CI Gate

Before claiming hosted CI proof, confirm the target branch or pull request has
completed hosted checks successfully. Record the CI provider, branch or PR, run
URL, commit SHA, and check names.

## Release Gate

Before claiming release proof, verify the intended tag/module publish path from
a clean checkout. Record the tag, commit SHA, module version resolution, release
artifact or registry evidence, and rollback path.

## Semver Tag Preflight

For the first public module cut, use `v0.1.0` unless a previous semver tag is
found remotely. Do not tag from a dirty worktree. Before creating the tag:

```powershell
git status --short --branch
git ls-remote --tags origin
gh release list --repo oxhq/oxpdf --limit 20
go list -m -versions github.com/oxhq/oxpdf
go test ./...
go vet ./...
```

If `develop` remains the candidate branch while `main` is the default branch,
either merge the candidate commit to `main` and tag that merge commit, or record
why the release is intentionally tagged from `develop`. After tagging, prove
module availability from outside this repository with:

```powershell
go list -m github.com/oxhq/oxpdf@v0.1.0
```

## Current Boundary

`go test ./...` plus `go vet ./...` evaluate the existing Go implementation.
The Rust cutover has its own local gate above; neither local gate proves the
other implementation. Consumer, hosted CI, migration/rollback, and release
proof remain separate levels.

## Historical P8.2/P9.3 Audit Stamp - 2026-05-26

This record is historical only: `go.mod` now uses Binas `v0.2.0`, and the Rust
cutover has not reached a package, consumer, hosted, or release gate.

Current source checkout:

- Local branch: `develop` at `8119e586bbde6baa0ac8c4ad37dba018d98011e8`.
- GitHub default branch: `main`.
- Remote heads: `develop` at `8119e586bbde6baa0ac8c4ad37dba018d98011e8`;
  `main` at `b2eea0d070b58726ae95e5bdb629a02f91d36c11`.
- Hosted CI: `develop` commit
  `8119e586bbde6baa0ac8c4ad37dba018d98011e8` has a completed successful
  GitHub Actions `CI` run:
  `https://github.com/oxhq/oxpdf/actions/runs/26469102712`.

Dynamic XFA boundary:

- OxPDF is pinned to `github.com/oxhq/binas v0.1.1`, which resolves to
  upstream tag commit `142c7e9a1d8cd0b41d66c6f3bd541b061f694e92`.
- The current feasible XFA surface is static packet/dataset inspection and
  verified static dataset field edits.
- Dynamic XFA remains unsupported: renderer-dependent XFA semantics must keep
  returning `ErrUnsupported` rather than claiming rendered or layout-aware form
  behavior.

Release artifact proof:

- `git ls-remote --tags origin` returned no OxPDF tags.
- `gh release list --repo oxhq/oxpdf --limit 20` returned no OxPDF releases.
- `gh api repos/oxhq/oxpdf/releases --jq 'length'` returned `0`.
- `go list -m -versions github.com/oxhq/oxpdf` returned no semver versions.
- `go list -m -json github.com/oxhq/oxpdf@latest` resolves only the untagged
  pseudo-version `v0.0.0-20260522184146-b2eea0d070b5` from `main`.

Release remains blocked on a human-approved tag target. If `develop` is the
approved release candidate, the smallest release command is:

```powershell
git tag -a v0.1.0 8119e586bbde6baa0ac8c4ad37dba018d98011e8 -m "oxpdf v0.1.0"
```

Do not run that command until the branch/tag decision is approved; follow with
the tag push, release creation, and outside-repo module resolution proof before
claiming release artifact proof.
