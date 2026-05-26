# OxPDF Private Feature Inventory

This is an internal implementation artifact. Keep it out of public docs unless
it is rewritten as user-facing documentation.

## Current Backing Reality

- Current dependency: `github.com/oxhq/binas v0.1.1`.
- OxPDF uses `github.com/oxhq/binas/pkg/pdfapi` for inspect, validate, profile,
  text query, and verified text rewrite without shelling out.

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

- Implemented first: `NewWriter`, `AddBlankPage`, `InsertBlankPage`,
  `WriteFile`, `Bytes`, and `ParsePageRange` for canonical blank PDFs and
  page-range parsing with parse-after-write tests.
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

- `Fields()` and `TextFields()` expose read-only field metadata from released
  `binas` for stable field names, values, types, flags, status, blockers, and
  options.
- `Page.Annotations()` exposes read-only annotation metadata from released
  `binas` for subtype, decoded contents/title, status, blockers, rectangles,
  colors, border, flags, and appearance presence.
- `Fill()` and `SetCheckbox()` wrap released `binas` field edits with reparse,
  field-value, and NeedAppearances verification. Broader radio/button helpers
  still need explicit state semantics before public expansion.
- `SetButtonChoice()` exposes exact-state button/radio changes while
  `SetCheckbox()` is restricted to checkbox-like buttons with one non-Off state.
- `SetAnnotationContents()` wraps released `binas` annotation contents edits for
  annotations whose appearance status is `approximate_supported`; appearance
  regeneration is explicit opt-in and verified.
- Profile-level form and annotation boundaries are exposed through released
  `pdfapi.Profile`: field counts, fillable counts, annotation counts, editable
  annotation counts, and blocker counts.
- Backing candidates in binas for future stable APIs: `ListFormFields`,
  `ApplyFormFieldEdit`, `ListAnnotationCandidates`, and annotation content edit
  helpers.
- pypdf evidence: `_doc_common.py`, `_writer.py`, `annotations/*`,
  `generic/_appearance_stream.py`, `tests/test_forms.py`, and
  `tests/test_annotations.py`.
- Gap: broader form/annotation appearance handling still needs more corpus proof
  before exposing removal/flattening or richer widget/annotation mutation APIs.

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
