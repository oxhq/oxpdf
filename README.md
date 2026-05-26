# OxPDF

OxPDF is an early Go-native PDF package backed by `binas`.

The project is in active development. The initial focus is a conservative API
for opening, inspecting, profiling, and safely editing PDFs without shelling out
to external commands.

## Public support matrix

OxPDF fails closed with `ErrUnsupported` for known PDF features or operations
outside the supported surface. Callers can use `errors.Is(err,
oxpdf.ErrUnsupported)` to distinguish those boundaries from generic I/O or parse
errors.

| Area | Supported API | Explicit unsupported boundary |
| --- | --- | --- |
| Opening and validation | `Open`, `OpenFile`, `OpenBytes`, `WithPassword`, `WithStrictParsing`, `Validate`, `Profile`, `Bytes`, `Header`, `NumPages`, `Page` | Encrypted PDFs without a usable password, wrong passwords, unsupported parser features, and out-of-range page indexes fail with structured errors; page-tree navigation can read direct and simple/Flate object-stream page dictionaries but is not a full object graph API. |
| Document structure | `Metadata`, `XMPMetadata`, `SetMetadata`, `Security`, `SecurityWithSignatureTrustPolicy`, `Trailer`, `Catalog`, `Xref`, page boxes, page rotation | Filtered XMP metadata streams and malformed or unsupported structural representations return `ErrUnsupported` where the data cannot be read safely; metadata writes are limited to verified incremental updates for simple table-xref, unencrypted PDFs; signature trust policy checks use only caller-provided roots/intermediates and do not claim revocation, timestamp, legal, or viewer-policy trust. |
| Navigation and document assets | `OutlineItems`, `NamedDestinations`, `PageLabels`, `JavaScriptActions`, `JavaScriptNameTreeActions`, `Attachments`, `AttachmentNameTree`, `Streams`, `DecodedStream`, `ImageXObjectStreams`, `ImageXObjects`, `InlineImages` | Unsupported outline actions, destination forms, name-tree forms, embedded-file filters, stream filters, and complex image data return `ErrUnsupported`; JavaScript and attachment name-tree APIs are limited to catalog-reachable direct name-tree references; stream and image APIs stay narrow to proven filters and direct metadata. |
| Text | `FindText`, page `ExtractText`, page `ExtractTextRuns`, `TextEditability`, `ReplaceText`, `ReplaceTextOccurrence`, `RemoveText` | Page-scoped extraction is only exposed where mapping can be proven; text runs expose selectable text-show order and current text position only, not words, lines, bounding boxes, OCR, or reflow; occurrence replacement keeps verification checks; text removal currently fails closed because empty replacement cannot prove selectable replacement text. |
| OCR text layers | `PlanOCRTextLayer`, `EmbedOCRTextLayer` | OCR text-layer APIs require caller-provided text, boxes, and confidence. OxPDF does not run OCR, call OCR binaries or services, or infer page layout; invalid text-layer inputs return `ErrUnsupported`. |
| AcroForm fields | `Fields`, `TextFields`, `Fill`, `SetCheckbox`, `SetButtonChoice` | Missing fields, unsupported field encodings, unsafe fill candidates, pushbuttons, invalid button states, and non-checkbox targets return `ErrUnsupported`. |
| Annotations | page `Annotations`, `SetAnnotationContents` with optional appearance regeneration | Only annotations reported as safely editable are mutated; unsupported subtypes, missing indexes, and unsupported appearance regeneration return `ErrUnsupported`. |
| Signatures | `PlanIncrementalReSigning`, `ApplyIncrementalReSigning` | Incremental re-signing requires an existing supported signature dictionary, caller-provided external signer callback, reserved contents capacity, and byte-range digest proof; it does not accept private key material or claim legal trust, revocation, timestamp, or viewer acceptance. |
| XFA | `XFAPackets`, `XFADatasetFields`, `XFATemplateDatasetMappings`, `XFASemantics`, `SetXFADatasetField` | Static dataset edits are supported only when verification succeeds; dynamic XFA, ambiguous dataset paths, and renderer-dependent XFA semantics return `ErrUnsupported`. |
| Writing | `NewWriter`, `AddBlankPage`, `InsertBlankPage`, `Bytes`, `WriteFile`, `PageSizeA4`, `PageSizeLetter` | Copying existing pages with `AddPage`, inserting existing pages with `InsertPage`, merging documents with `Append`, and writing an empty writer return `ErrUnsupported`. |
| Utilities | `ParsePageRange` | Invalid page-range syntax returns an error; indexes are bounded by the supplied page count. |
