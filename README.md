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
| Opening and validation | `Open`, `OpenFile`, `OpenBytes`, `WithPassword`, `WithStrictParsing`, `Validate`, `Profile`, `Bytes`, `Header`, `NumPages`, `Page` | Encrypted PDFs without a usable password, wrong passwords, unsupported parser features, and out-of-range page indexes fail with structured errors. |
| Document structure | `Metadata`, `XMPMetadata`, `Security`, `Trailer`, `Catalog`, `Xref`, page boxes, page rotation | Filtered XMP metadata streams and malformed or unsupported structural representations return `ErrUnsupported` where the data cannot be read safely. |
| Navigation and document assets | `OutlineItems`, `NamedDestinations`, `PageLabels`, `JavaScriptActions`, `Attachments`, `Streams`, `ImageXObjectStreams` | Unsupported outline actions, destination forms, name-tree forms, and embedded-file filters return `ErrUnsupported`; stream inventories expose unsupported stream metadata instead of decoding every filter. |
| Text | `FindText`, page `ExtractText`, `TextEditability`, `ReplaceText` | Page-scoped extraction is only exposed where the backing parser can prove the mapping; unsupported parser/editability cases fail with `ErrUnsupported` instead of rewriting blindly. |
| AcroForm fields | `Fields`, `TextFields`, `Fill`, `SetCheckbox`, `SetButtonChoice` | Missing fields, unsupported field encodings, unsafe fill candidates, pushbuttons, invalid button states, and non-checkbox targets return `ErrUnsupported`. |
| Annotations | page `Annotations`, `SetAnnotationContents` with optional appearance regeneration | Only annotations reported as safely editable are mutated; unsupported subtypes, missing indexes, and unsupported appearance regeneration return `ErrUnsupported`. |
| XFA | `XFAPackets`, `XFADatasetFields`, `XFATemplateDatasetMappings`, `XFASemantics`, `SetXFADatasetField` | Static dataset edits are supported only when verification succeeds; dynamic XFA, ambiguous dataset paths, and renderer-dependent XFA semantics return `ErrUnsupported`. |
| Writing | `NewWriter`, `AddBlankPage`, `InsertBlankPage`, `Bytes`, `WriteFile`, `PageSizeA4`, `PageSizeLetter` | Copying existing pages with `AddPage`, inserting existing pages with `InsertPage`, merging documents with `Append`, and writing an empty writer return `ErrUnsupported`. |
| Utilities | `ParsePageRange` | Invalid page-range syntax returns an error; indexes are bounded by the supplied page count. |
