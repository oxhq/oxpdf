# OxPDF Rust cutover slice

This crate consumes `binas-pdf` directly through a development path dependency. That proves
local development wiring only; it is not a published package or registry/release-consumer proof.
It exposes open, inspect, Info and bounded supported-filter XMP metadata, zero-based page enumeration,
effective inherited page geometry, structured text extraction, catalog-reachable page
labels/destinations/outlines/embedded-file metadata, and Binas-verified
copy/extract/insert/merge/transform/composition page workflows. Each workflow returns Binas's
`PageOperationOutcome` and its verification report. Page creation is limited to Binas's
`create_blank_pdf`; `Document::insert_blank_page` derives a one-page PDF from it and uses the
verified page-insertion workflow. There is no general writer/builder, drawing API, or raw
page-tree manipulation.

`Document::page_geometry` returns the effective page-tree inherited media, crop, bleed, trim,
and art boxes plus normalized rotation for one zero-based page. It is a reader, not a page-tree
editing or layout API.

Exact text query and the source-proven `surgical_text_edit` are exposed directly. Batch edits use
an explicit Binas plan/apply cycle bound to the source bytes and verified after writing; there is
no automatic fallback, reflow policy, or text-layout abstraction. Font/CMap edits and every broader
rewrite mode remain separate future methods.

Info and XMP updates return Binas's document-structure verification outcome. XMP reads use
Binas's bounded supported-filter decoder; replacing an existing filtered XMP stream remains
fail-closed. There is no generic metadata abstraction in this slice.

Page-label and named-destination updates are direct Binas operations. Outline create, child-create,
and remove use Binas's named-destination lifecycle. A general navigation-tree model and custom
destination parsing remain outside this slice.

Embedded attachment update is a direct Binas operation. `embedded_attachments` exposes only
catalog-reachable metadata, and `read_embedded_attachment_bytes` returns bounded decoded bytes
only for an exact inventory entry; stale, forged, or ambiguous entries are rejected. Generic
attachment-tree traversal remains outside this slice.

Raw and Binas-encoded image XObject replacement are exposed directly.
`Document::image_xobjects` lists indirect image-XObject object references, dimensions, and
color-space/filter metadata without reading or decoding image bytes. There is no image
abstraction, alpha conversion, or OxPDF format auto-detection. Three narrowly bounded readers are
available: `read_jpeg_xobject_bytes` returns opaque bytes only for an exact inventory entry whose
stream is one direct `/DCTDecode` JPEG with matching declared dimensions;
`read_jpx_xobject_bytes` returns opaque bytes only for an exact direct `/JPXDecode` JPEG 2000
stream with parsed dimensions matching the inventory/dictionary and no decode parameters or masks;
and `read_raw_flate_image_samples` returns unconverted samples only for an exact, unmasked direct
`/FlateDecode` DeviceGray/RGB/CMYK image with 8-bit components, no decode transforms, and an
exact decoded-length match. None of these readers decodes pixels, converts color spaces, follows
masks, or introduces a generic image abstraction. `Document::inline_images` lists metadata only
when each page has at most one directly referenced, unfiltered content stream. It does not expose
bytes, decode filters, follow resources, accept content-stream arrays, or extract images.

Direct stream mutation accepts Binas-addressed decoded bytes. `Document::streams` lists parsed
stream object references, encoded lengths, filter/decode-parameter metadata, and a narrow
image-XObject marker without decoding or copying stream bytes. There is no decoder selection,
generic image extraction, or stream abstraction.

`Document::read_decoded_stream` reads one explicitly addressed `StreamObjectRef` through Binas's
bounded supported filter chain. It is not generic object traversal or arbitrary decoder access.

Inline-image replacement accepts Binas-addressed encoded bytes and caller-specified image
metadata. Inventory and replacement remain bounded to direct page content; there is no decoding,
filter auto-detection, or media abstraction.

Text overlay places printable ASCII with the built-in Helvetica font on one page at explicit finite
`x`/`y` coordinates and a positive size. Reopened exact text queries include the overlay through
its Form XObject. Style, alternate fonts, rotation, and multi-page overlay abstractions remain
outside this slice.

It also exposes Binas-resolved AcroForm fields and annotations, verified incremental
`set_form_field_value` and `set_annotation_contents` mutations, and direct Binas form-field
create/remove/flatten and annotation create/remove outcomes. `set_checkbox_field` requires one
proven checkbox on-state; `set_button_field_choice` requires an exact proven radio-button
appearance state. There is no OxPDF form model, widget-layout engine, annotation renderer,
appearance policy, generic drawing model, popup/layout logic, or appearance abstraction.

Read-only encryption metadata, default signature inspection, XFA packet metadata, and static
XFA dataset fields are exposed directly. `open_with_password` explicitly opens Standard Security
input with a caller-provided password; `open_with_public_key` explicitly opens public-key input
with caller-provided DER certificate and PKCS#8 private-key bytes. External-signature preparation
returns Binas's prepared bytes, digest, and descriptor; callers apply CMS only through that Binas
plan. Standard-password and public-key encryption/decryption return Binas's direct outcomes with
built-in re-open verification. There is no key storage, certificate management, signer execution,
trust/revocation policy, or network behavior. Static packet replacement and dataset-field set are
the only XFA writes exposed, along with exact static dataset-field removal; dynamic XFA, renderer
semantics, selectors, and custom XML parsing are unsupported.

`Document::xfa_template_dataset_mappings` is read-only and static-XFA-only. It maps an exact
template field path or enclosing simple named-subform path only when it identifies one dataset
leaf; ambiguous and unmatched fields are omitted. It is not a selector, renderer, layout, or
write API.

OCR accepts caller-provided bounded JSON or ALTO box data and delegates text-layer planning and
application to Binas. It has no renderer, OCR service, layout inference, or fallback abstraction.

It has no Go bridge, CLI bridge, renderer, or compatibility claim. The page workflows
fail closed for Binas's unsupported encrypted/signed inputs, widget annotations without
their AcroForm, page/page-tree dependencies, invalid selections, and configured limits.
Structure readers likewise fail closed for malformed graphs, malformed XMP, unsupported or
over-budget XMP stream filters, and unsupported name-tree or destination forms.
