# OxPDF Private Feature Inventory

This is an internal implementation artifact. Keep it out of public docs unless
it is rewritten as user-facing documentation.

## Current Backing Reality

- Roadmap target dependency: `github.com/oxhq/binas/pkg/pdfapi`.
- Current live binas module exposes `github.com/oxhq/binas/pkg/adapters/pdf`
  and `github.com/oxhq/binas/pkg/core`; `pkg/pdfapi` is not present yet.
- OxPDF therefore keeps its public API in package `oxpdf` and hides the current
  adapter behind document/profile/text methods that can later move to `pdfapi`
  without changing callers.

## Reader

- Implemented first: `Open`, `OpenFile`, `OpenBytes`, `Document`, `NumPages`,
  `Page`, `Metadata`, `Profile`, `Validate`, page boxes, and rotation.
- Backing: `pdf.Adapter.Parse`, root node metadata, xref/boundary summary.
- pypdf evidence: `_reader.py`, `_doc_common.py`, `tests/test_reader.py`,
  and `tests/test_doc_common.py` cover opening, header, pages, metadata,
  encryption state, trailer/root, and malformed/xref behavior.
- Gap: trailer/root structured summaries, inherited page-tree attributes beyond
  current corpus coverage, and password-open support need a higher-level binas
  API.

## Writer And Pages

- Implemented first: `NewWriter`, `AddBlankPage`, `Bytes` for canonical blank
  PDFs with parse-after-write tests.
- Guarded as unsupported: `AddPage`, `InsertPage`, `Append`.
- pypdf evidence: `_writer.py`, `_page.py`, `tests/test_writer.py`,
  `tests/test_page.py`, and `tests/test_merger.py` define the eventual append,
  merge, split, rotate, crop, scale, transform, and page-copy target surface.
- Gap: page graph copy/merge/split/rotate/crop/scale requires binas page tree
  writer support before OxPDF can honestly claim pypdf-like manipulation.

## Text

- Implemented first: `FindText`, page-0 `ExtractText`, `TextEditability`,
  `ReplaceText`.
- Backing: `pdf.KindTextShow`, `pdf.ApplyCanonicalEdit`, binas verification.
- pypdf evidence: `_page.py`, `_text_extraction/*`, `_cmap.py`, `_font.py`,
  `tests/test_text_extraction.py`, `tests/test_cmap.py`, and
  `docs/meta/scope-of-pypdf.md`. pypdf owns selectable extraction and removal;
  verified replacement remains the OxPDF differentiator.
- Boundary: text extraction is selectable text only, not OCR; page-scoped text
  mapping is partial until binas exposes page-to-stream metadata.

## Forms And Annotations

- Public placeholders exist with structured unsupported errors.
- Backing candidates in binas: `ListFormFields`, `ApplyFormFieldEdit`,
  `ListAnnotationCandidates`, and annotation content edit helpers.
- pypdf evidence: `_doc_common.py`, `_writer.py`, `annotations/*`,
  `generic/_appearance_stream.py`, `tests/test_forms.py`, and
  `tests/test_annotations.py`.
- Gap: OxPDF needs field/annotation appearance guardrails before exposing these
  as a stable ergonomic API.

## Filters, Images, And Security

- Current profile reports encryption, signatures, XFA, xref streams, object
  streams, filters, CMaps, and related boundaries surfaced by binas.
- pypdf evidence: `filters.py`, `_encryption.py`, `_crypt_providers/*`,
  image helpers in `_page.py`, `tests/test_filters.py`, `tests/test_images.py`,
  and `tests/test_encryption.py`.
- Password open/encrypt, image extraction, image replacement, public-key
  encryption, AESV3, and legal-grade trust are not exposed.

## Metadata And Navigation

- Current metadata is header-only.
- pypdf evidence: `xmp.py`, `_page_labels.py`, `_doc_common.py`,
  `_writer.py`, `generic/_files.py`, `tests/test_xmp.py`,
  `tests/test_page_labels.py`, and `tests/test_javascript.py`.
- Document info, XMP, page labels, outlines, named destinations, attachments,
  and JavaScript actions remain roadmap items until object graph operations are
  exposed at a stable backing boundary.
