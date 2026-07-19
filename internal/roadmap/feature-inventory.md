# OxPDF Private Feature Inventory

This is an internal implementation artifact. Keep it out of public docs unless
it is rewritten as user-facing documentation.

## Current Backing Reality

- The Go implementation on `develop` depends on `github.com/oxhq/binas v0.2.0`.
- The Rust cutover branch is a separate, local Cargo workspace.  During the
  migration it consumes Binas through a direct path dependency on the
  `binas-pdf` crate; it must never introduce a CLI, C ABI, JSON, or Go bridge.
- A migrated Rust operation must call the Binas Rust API directly.  Recreating
  a raw-PDF parser in OxPDF is not a valid port, even if it reproduces a Go
  result for a narrow fixture.
- This branch is development-only until Binas has a published Rust package and
  OxPDF replaces the path dependency with that released package version.

## Rust Cutover Contract

The Go tests are the migration inventory, not automatic proof of feature
parity.  Port a public operation only when its Rust test exercises a real
`binas-pdf` document operation and proves the returned bytes or read result.

1. Core reader: `Open`, `OpenFile`, `OpenBytes`, strict/password options,
   header, page count, page handles, validation, and profile.
2. Read-only document data: metadata/XMP, page boxes/rotation, text/query,
   structure, navigation, streams/images, forms/annotations, XFA, and security.
3. Verified mutation: text, metadata, pages, forms/annotations, streams/images,
   OCR, XFA, navigation, and signatures.  Each method needs a Binas verification
   result plus a reopen/readback assertion where Binas exposes one.
4. Consumer proof: port the applicable Go tests into Rust integration tests,
   then add a separate downstream Rust consumer before a release claim.

Do not mark a row complete because a Binas unit test exists.  Completion needs
the OxPDF facade test, a fixture with known provenance, and a direct dependency
on the same Binas public API that downstream users will compile against.

## Reader

- Implemented first: `Open`, `OpenFile`, `OpenBytes`, `Document`, `NumPages`,
  `Page`, `Metadata`, `Profile`, `Validate`, page boxes, and rotation.
- `Trailer()`, `Catalog()`, and `Xref()` expose read-only structure summaries:
  classic trailer root/info/size references, direct catalog references, and
  released binas xref table/stream/object-stream counts.
- `WithStrictParsing()` is wired to the backing strict parser for malformed
  input checks such as missing EOF markers.
- Backing: `pdf.Adapter.Parse`, root node metadata, xref/boundary summary.
- pypdf evidence: `_reader.py`, `_doc_common.py`, `tests/test_reader.py`,
  and `tests/test_doc_common.py` cover opening, header, pages, metadata,
  encryption state, trailer/root, and malformed/xref behavior.
- Gap: compressed xref stream trailers, full object graph traversal, and
  inherited page-tree attributes beyond current corpus coverage.

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
- `Streams()` and `ImageXObjectStreams()` expose read-only stream inventory:
  stream byte spans, encoded/decoded length metadata, filter chains, decode
  parameters, image-XObject markers, and binas filter-capability flags.
- `Security()` exposes read-only encryption and signature boundary metadata
  without claiming signature trust, revocation, timestamp, or legal-grade
  validation.
- `WithPassword()` is proven against pypdf Standard Security fixtures for
  read/open behavior. Wrong passwords fail closed as `ErrUnsupported`.
- pypdf evidence: `filters.py`, `_encryption.py`, `_crypt_providers/*`,
  image helpers in `_page.py`, `tests/test_filters.py`, `tests/test_images.py`,
  and `tests/test_encryption.py`.
- Stream inventory does not claim image extraction or inline-image extraction;
  image streams with pass-through filters are identified as boundaries only.
- Encrypt, image extraction, image replacement, public-key encryption, AESV3,
  and legal-grade trust are not exposed.

## Metadata And Navigation

- `Metadata()` reads the document info dictionary for common title, author,
  subject, keywords, producer, creator, and date fields.
- `XMPMetadata()` exposes read-only XMP packet XML plus small parsed fields
  currently proven for `tiff:Artist` and UTC-normalized `xmp:ModifyDate`.
- `XFAPackets()`, `XFADatasetFields()`, `XFATemplateDatasetMappings()`,
  `XFASemantics()`, and `SetXFADatasetField()` expose static XFA dataset
  inspection/editing. Dynamic XFA rendering and ambiguous dataset edits fail
  closed.
- `JavaScriptActions()` exposes direct JavaScript action dictionaries with
  literal or hex `/JS` payloads. Broken-xref documents may open sparsely when
  this metadata is readable, but no full document parse is claimed for those
  sparse handles.
- `Attachments()` exposes direct file-spec attachment payloads backed by
  embedded-file streams with no filter or `FlateDecode`; attachment names are
  decoded from `/UF` or `/F` without requiring whitespace before the string.
- `NamedDestinations()` exposes read-only named destinations from direct `/Dests`
  name trees with `/XYZ` arrays and resolvable page object references.
- `OutlineItems()` exposes a flat, read-only top-level outline list from direct
  outline linked lists with `GoTo` actions targeting named destinations.
- `PageLabels()` exposes one label per page, defaulting to one-based decimal
  labels and parsing direct `/PageLabels /Nums` entries for decimal, roman, and
  alphabetic styles.
- pypdf evidence: `xmp.py`, `_page_labels.py`, `_doc_common.py`,
  `_writer.py`, `generic/_files.py`, `tests/test_xmp.py`,
  `tests/test_page_labels.py`, and `tests/test_javascript.py`.
- Nested outline trees, broad name-tree attachments, JavaScript name-tree
  actions, recursive `/PageLabels /Kids`, and object-stream-backed navigation
  fixtures remain roadmap items until object graph operations are exposed at a
  stable backing boundary.
