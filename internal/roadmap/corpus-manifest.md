# OxPDF Private Corpus Manifest

This manifest tracks compatibility targets. It is not a blind pypdf test port.

| Fixture | Source | v0.1 Expectation | Purpose |
| --- | --- | --- | --- |
| synthetic blank page | OxPDF tests | pass | Reader, profile, validate, writer parse-after-write. |
| synthetic two-page blank PDF | OxPDF tests | structure pass | Classic trailer `/Size` and `/Root`, catalog `/Pages`, and xref table summary. |
| synthetic selectable text | OxPDF tests | pass | `FindText`, `ExtractText`, verified `ReplaceText`. |
| `hello-world.pdf` | `C:\Users\garae\Documents\pypdf\resources\hello-world.pdf` | pass | Ordinary open from bytes/file, `NumPages()==1`, no security boundary. |
| `two-different-pages.pdf` | `C:\Users\garae\Documents\pypdf\resources\two-different-pages.pdf` | pass | Multi-page traversal, `NumPages()==2`, `Page(0)` and `Page(1)`. |
| `metadata.pdf` | `C:\Users\garae\Documents\pypdf\resources\metadata.pdf` | pass | Document info dictionary: title, author, subject, keywords, dates; no XMP returns empty. |
| `missing_info.pdf` | `C:\Users\garae\Documents\pypdf\resources\missing_info.pdf` | pass | Missing `/Info` is not an error; metadata should be empty. |
| `encrypted-file.pdf` | `C:\Users\garae\Documents\pypdf\resources\encrypted-file.pdf` | unsupported/security pass | Structured encrypted/security refusal, no panic or generic parse failure. |
| `r2-user-password.pdf` | `C:\Users\garae\Documents\pypdf\resources\encryption\r2-user-password.pdf` | security/password pass | Named Standard Security metadata: Standard, V=1, R=2, Length=40; opens with the known user password and fails closed on the wrong one. |
| `pdflatex-forms.pdf` | `C:\Users\garae\Documents\pypdf\resources\pdflatex-forms.pdf` | profile-only | AcroForm presence; Unicode field-name decoding still belongs in the backing API. |
| `libreoffice-form.pdf` | `C:\Users\garae\Documents\pypdf\resources\libreoffice-form.pdf` | fields/fill/profile pass | Richer real-world form and annotation pressure: 8 fillable fields, 4 text fields, text fill, checkbox set/unset, 9 blocked annotations. |
| `commented.pdf` | `C:\Users\garae\Documents\pypdf\resources\commented.pdf` | annotations/edit/profile pass | Annotation listing/editing: 6 annotations, decoded UTF-16BE contents/title, status/blocker metadata, supported content edits for indexes 0/2/4. |
| `commented-xmp.pdf` | `C:\Users\garae\Documents\pypdf\resources\commented-xmp.pdf` | XMP pass | Read-only XMP packet extraction and `tiff:Artist` parsing. |
| `issue-914-xmp-data.pdf` | `C:\Users\garae\Documents\pypdf\resources\issue-914-xmp-data.pdf` | XMP pass | Read-only XMP packet extraction and UTC-normalized `xmp:ModifyDate`. |
| `issue-297.pdf` | `C:\Users\garae\Documents\pypdf\resources\issue-297.pdf` | JavaScript/direct action pass | Broken-xref fixture that still exposes one direct `/S /JavaScript` action with literal `/JS`; opens as a sparse metadata handle only. |
| synthetic no-space attachment name | OxPDF tests | attachment/name pass | Direct `/Filespec /F(...) /UF(...)` attachment name parsing without whitespace before the literal string. |
| `attachment.pdf` | `C:\Users\garae\Documents\pypdf\resources\attachment.pdf` | attachment/direct filespec pass | Direct file attachment annotation through `/Filespec` -> `/EF /F` embedded file stream, zlib `FlateDecode`, decoded `jpeg.pdf` payload. |
| `jpeg.pdf` | `C:\Users\garae\Documents\pypdf\resources\jpeg.pdf` | stream inventory pass | Five stream nodes, including DCT image XObject metadata marked `pass_through_image` without claiming extraction. |
| `reportlab-inline-image.pdf` | `C:\Users\garae\Documents\pypdf\resources\reportlab-inline-image.pdf` | stream inventory pass | Editable reversible stream filter chain `[ASCII85Decode FlateDecode]`; no inline-image extraction claim. |
| `outline-without-title.pdf` | `C:\Users\garae\Documents\pypdf\resources\outline-without-title.pdf` | named destinations/outlines pass | Direct `/Names` -> `/Dests` name tree with 15 `/XYZ` destinations and a top-level outline linked list with 9 `GoTo` named-destination actions. |
| `box.pdf` | `C:\Users\garae\Documents\pypdf\resources\box.pdf` | pass | Page box fallback smoke: all boxes resolve to `[0 0 60 60]`, rotation `0`. |
| `indirect-rotation.pdf` | `C:\Users\garae\Documents\pypdf\resources\indirect-rotation.pdf` | pass | Five pages with indirect `/Rotate` resolving to `0` and media box `[0 0 612 792]`. |
| synthetic static XFA | OxPDF tests | XFA pass | XFA packet listing, static dataset field listing, template/dataset mappings, XML-escaped static dataset update. |
| synthetic dynamic/ambiguous XFA | OxPDF tests | unsupported/XFA pass | Dynamic XFA and ambiguous dataset edits fail closed with `ErrUnsupported`. |

Next concrete step: keep existing-page writer operations guarded until binas
exports page graph copy/merge support; continue tightening page-range and field
state edge cases as corpus pressure grows.
