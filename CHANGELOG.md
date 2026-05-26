# Changelog

## Unreleased

### Added

- Opening and reader inspection APIs for `Open`, `OpenFile`, `OpenBytes`,
  document metadata, page geometry, validation, profile summaries, trailers,
  catalogs, xref summaries, and strict parsing.
- Writer APIs for creating blank PDFs, inserting blank pages, writing bytes or
  files, and parsing page ranges.
- Selectable text APIs for finding, page-scoped extraction, editability checks,
  verified replacement, occurrence-scoped replacement, and explicit fail-closed
  text removal.
- Form and annotation APIs for listing fields and annotations, filling text
  fields, setting checkbox/button states, and guarded annotation content edits.
- Read-only security, stream decoding, image XObject extraction, inline image
  extraction, richer XMP metadata, JavaScript action, attachment, named
  destination, nested outline, page label, and XFA inspection/editing surfaces.

### Proof Boundaries

- Current proof is source-repo local only unless a release note says otherwise.
- Local tests and vetting do not prove consumer installation, hosted CI,
  published module availability, release artifacts, signing, or downstream
  runtime behavior.
- Known intentional gaps remain around broad page copy/merge/manipulation,
  object-stream-backed navigation, dynamic XFA rendering, OCR, image
  replacement, encryption writes, and legal-grade signature trust validation.
