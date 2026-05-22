# OxPDF Private Corpus Manifest

This manifest tracks compatibility targets. It is not a blind pypdf test port.

| Fixture | Source | v0.1 Expectation | Purpose |
| --- | --- | --- | --- |
| synthetic blank page | OxPDF tests | pass | Reader, profile, validate, writer parse-after-write. |
| synthetic selectable text | OxPDF tests | pass | `FindText`, `ExtractText`, verified `ReplaceText`. |
| `hello-world.pdf` | `C:\Users\garae\Documents\pypdf\resources\hello-world.pdf` | pass | Ordinary open from bytes/file, `NumPages()==1`, no security boundary. |
| `two-different-pages.pdf` | `C:\Users\garae\Documents\pypdf\resources\two-different-pages.pdf` | pass | Multi-page traversal, `NumPages()==2`, `Page(0)` and `Page(1)`. |
| `metadata.pdf` | `C:\Users\garae\Documents\pypdf\resources\metadata.pdf` | pass | Document info dictionary: title, author, subject, keywords, dates. |
| `missing_info.pdf` | `C:\Users\garae\Documents\pypdf\resources\missing_info.pdf` | pass | Missing `/Info` is not an error; metadata should be empty. |
| `encrypted-file.pdf` | `C:\Users\garae\Documents\pypdf\resources\encrypted-file.pdf` | unsupported | Structured encrypted/security refusal, no panic or generic parse failure. |
| `r2-user-password.pdf` | `C:\Users\garae\Documents\pypdf\resources\encryption\r2-user-password.pdf` | unsupported | Named Standard Security boundary case for v0.6. |
| `pdflatex-forms.pdf` | `C:\Users\garae\Documents\pypdf\resources\pdflatex-forms.pdf` | profile-only | AcroForm presence; field APIs remain v0.4. |
| `libreoffice-form.pdf` | `C:\Users\garae\Documents\pypdf\resources\libreoffice-form.pdf` | profile-only | Richer real-world form and annotation pressure. |
| `commented.pdf` | `C:\Users\garae\Documents\pypdf\resources\commented.pdf` | profile-only | Annotation boundary before `Page.Annotations()`. |
| `box.pdf` | `C:\Users\garae\Documents\pypdf\resources\box.pdf` | pass | Page box fallback smoke: all boxes resolve to `[0 0 60 60]`, rotation `0`. |
| `indirect-rotation.pdf` | `C:\Users\garae\Documents\pypdf\resources\indirect-rotation.pdf` | pass | Five pages with indirect `/Rotate` resolving to `0` and media box `[0 0 612 792]`. |

Next concrete step: use `commented.pdf`, `pdflatex-forms.pdf`, and
`libreoffice-form.pdf` as profile-only boundary tests before exposing stable
forms/annotation APIs.
