# Release Readiness

This project should not be called release-ready from source-repo checks alone.
Use the gates below to name the highest proof level actually completed.

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

## Current Boundary

As of this document, the known lightweight local gate is `go test ./...` plus
`go vet ./...`. Consumer, hosted CI, and release gates remain separate proof
levels and must be completed explicitly before they are claimed.
