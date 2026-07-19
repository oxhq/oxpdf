#![forbid(unsafe_code)]

//! A deliberately small OxPDF consumer facade over `binas-pdf`.
//! Only Binas-verified page workflows are exposed; rendering and compatibility layers stay out of scope.

pub use binas_pdf::{
    Annotation, AnnotationContentsMutationOutcome, AnnotationContentsMutationReport,
    AnnotationContentsMutationRequest, AnnotationContentsMutationVerification,
    AnnotationCreateRequest, AnnotationLifecycleOutcome, AnnotationLifecycleReport,
    AnnotationLifecycleVerification, AnnotationRemoveRequest, AnnotationSubtype, AppearanceStatus,
    BatchTextEditOutcome, BatchTextEditPlan, BatchTextEditReport, BatchTextEditRequest,
    BatchTextEditVerification, BlankPageSize, ButtonChoiceMutationRequest,
    ButtonFieldMutationOutcome, ButtonFieldMutationReport, ButtonFieldMutationVerification,
    CanonicalizeOutcome, CanonicalizeReport, CanonicalizeVerification, CapabilityDecision,
    CheckboxFieldMutationRequest, CmsParseStatus, CmsValidation, DecodeParams, DecryptionOutcome,
    DecryptionReport, DecryptionVerification, DigestMatchStatus, DocumentCapabilityProfile,
    DocumentInfoMetadata, DocumentInfoUpdate, DocumentStructureOutcome, DocumentStructureReport,
    DocumentStructureVerification, EmbeddedAttachment, EmbeddedAttachmentUpdate,
    EncodedImageReplacementRequest, EncryptionMetadata, EncryptionOutcome, EncryptionReport,
    EncryptionVerification, ExternalSignatureFieldOptions, ExternalSignaturePlan,
    ExternalSignaturePlanDescriptor, FilteredEditOutcome, FilteredEditReport,
    FilteredEditVerification, FilteredTextEditRequest, FontEditOutcome, FontEditReport,
    FontEditVerification, FontTextEditRequest, FormField, FormFieldCreateRequest, FormFieldKind,
    FormFieldRemoveRequest, FormLifecycleOutcome, FormLifecycleReport, FormLifecycleVerification,
    FormValueMutationOutcome, FormValueMutationReport, FormValueMutationRequest,
    FormValueMutationVerification, FreeTextAppearanceOutcome, FreeTextAppearanceReport,
    FreeTextAppearanceRequest, FreeTextAppearanceVerification, ImageColorSpace, ImageDecodeParams,
    ImageFilter, ImageMaskPolicy, ImageReplacementOutcome, ImageReplacementReport,
    ImageReplacementRequest, ImageReplacementVerification, ImageXObjectInventoryEntry,
    IncrementalEditOutcome, IncrementalEditReport, IncrementalEditVerification,
    IncrementalTextEditRequest, InlineImageColorSpace, InlineImageFilter,
    InlineImageInventoryEntry, InlineImageReplacementOutcome, InlineImageReplacementReport,
    InlineImageReplacementRequest, InlineImageReplacementVerification, InspectResult,
    JavaScriptAction, JavaScriptActionInventory, NamedDestination, NamedDestinationUpdate,
    OcrParseLimits, OcrPlacedText, OcrTextBox, OcrTextLayerOutcome, OcrTextLayerPlan,
    OcrTextLayerReport, OcrTextLayerRequest, OcrTextLayerVerification, OpenOptions,
    OperationCapability, OutlineCreateRequest, OutlineItem, OutlineRemoveRequest,
    OverlayStampOutcome, OverlayStampReport, OverlayStampRequest, OverlayStampVerification,
    PageCompositionPlacement, PageCompositionRequest, PageGeometry, PageLabel, PageLabelSpec,
    PageLabelStyle, PageLabelUpdate, PageOperationOutcome, PageOperationReport,
    PageOperationVerification, PageTransform, PdfError, PdfFilter, PlannedTextEdit,
    PublicKeyDecryptionOutcome, PublicKeyEncryptionMethod, PublicKeyEncryptionOptions,
    PublicKeyEncryptionOutcome, PublicKeyEncryptionVerification, QueryMatch, RawFlateImageSamples,
    RevocationStatus, SignatureCryptoStatus, SignatureInspection, SignatureTrustOptions,
    StandardEncryptionOptions, StandardEncryptionRevision, StreamFilterMetadata,
    StreamInventoryEntry, StreamMutationOutcome, StreamMutationReport, StreamMutationRequest,
    StreamMutationVerification, StreamObjectRef, SurgicalEditOutcome, SurgicalEditReport,
    SurgicalEditVerification, SurgicalTextEditRequest, TextExtraction, TextFieldAppearanceOutcome,
    TextFieldAppearanceReport, TextFieldAppearanceRequest, TextFieldAppearanceVerification,
    TextOverlayOutcome, TextOverlayReport, TextOverlayRequest, TextOverlayVerification, TextSpan,
    TimestampInspection, TimestampStatus, TrustStatus, ValidationResult, XfaDatasetField,
    XfaDatasetMutationOutcome, XfaDatasetMutationReport, XfaDatasetMutationVerification,
    XfaDatasetSetRequest, XfaDynamicReport, XfaPacket, XfaReplaceOutcome, XfaReplaceReport,
    XfaReplaceRequest, XfaReplaceVerification, XfaTemplateDatasetMapping, XmpMetadata,
    XmpMetadataUpdate,
};
pub use binas_pdf::{parse_alto_xml, parse_ocr_json};

/// Creates a Binas-verified blank PDF using the default bounded engine.
pub fn create_blank_pdf(pages: &[BlankPageSize]) -> Result<Vec<u8>, PdfError> {
    binas_pdf::PdfEngine::default().create_blank_pdf(pages)
}

/// Opens PDFs through the default bounded Binas engine.
pub fn open(input: &[u8]) -> Result<Document, PdfError> {
    open_with_options(input, OpenOptions::default())
}

/// Opens PDFs through the default bounded Binas engine with explicit options.
pub fn open_with_options(input: &[u8], options: OpenOptions) -> Result<Document, PdfError> {
    Ok(Document {
        inner: binas_pdf::PdfEngine::default().open(input, options)?,
    })
}

/// Opens Standard Security input with a caller-provided password.
pub fn open_with_password(input: &[u8], password: &str) -> Result<Document, PdfError> {
    Ok(Document {
        inner: binas_pdf::PdfEngine::default().open_with_password(
            input,
            password,
            OpenOptions::default(),
        )?,
    })
}

/// Opens public-key encrypted input with caller-provided DER certificate and PKCS#8 key bytes.
pub fn open_with_public_key(
    input: &[u8],
    recipient_certificate_der: &[u8],
    recipient_private_key_pkcs8_der: &[u8],
) -> Result<Document, PdfError> {
    Ok(Document {
        inner: binas_pdf::PdfEngine::default().open_with_public_key(
            input,
            recipient_certificate_der,
            recipient_private_key_pkcs8_der,
            OpenOptions::default(),
        )?,
    })
}

/// A PDF opened by the Binas Rust engine.
#[derive(Clone, Debug)]
pub struct Document {
    inner: binas_pdf::PdfDocument,
}

impl Document {
    /// Returns structural information reported by Binas.
    pub fn inspect(&self) -> Result<InspectResult, PdfError> {
        self.inner.inspect()
    }

    /// Returns Binas's direct structural validation result without applying a policy.
    pub fn validate(&self) -> Result<ValidationResult, PdfError> {
        self.inner.validate()
    }

    /// Returns Binas's capability profile without scoring or interpreting it.
    pub fn capability_profile(&self) -> Result<DocumentCapabilityProfile, PdfError> {
        self.inner.capability_profile()
    }

    /// Explicitly canonicalizes this document through Binas; opening never rewrites it.
    pub fn canonicalize(&self) -> Result<CanonicalizeOutcome, PdfError> {
        self.inner.canonicalize()
    }

    /// Returns the PDF Info dictionary metadata when present.
    pub fn metadata(&self) -> Result<DocumentInfoMetadata, PdfError> {
        binas_pdf::read_document_info(&self.inner)
    }

    /// Returns the catalog XMP packet when Binas can decode and validate its XML stream.
    pub fn xmp_metadata(&self) -> Result<Option<XmpMetadata>, PdfError> {
        binas_pdf::read_xmp_metadata(&self.inner)
    }

    /// Applies Binas's verified PDF Info dictionary update.
    pub fn update_document_info(
        &self,
        update: DocumentInfoUpdate,
    ) -> Result<DocumentStructureOutcome, PdfError> {
        self.inner.update_document_info(update)
    }

    /// Applies Binas's verified catalog XMP metadata update.
    pub fn update_xmp_metadata(
        &self,
        update: XmpMetadataUpdate,
    ) -> Result<DocumentStructureOutcome, PdfError> {
        self.inner.update_xmp_metadata(update)
    }

    /// Returns the encryption dictionary metadata without decrypting or applying policy.
    pub fn encryption_metadata(&self) -> Result<EncryptionMetadata, PdfError> {
        binas_pdf::inspect_encryption(&self.inner)
    }

    /// Applies Binas's standard-password encryption with its built-in re-open verification.
    pub fn encrypt_standard(
        &self,
        options: StandardEncryptionOptions,
    ) -> Result<EncryptionOutcome, PdfError> {
        self.inner.encrypt_standard(options)
    }

    /// Decrypts Standard Security input with Binas's built-in re-open verification.
    pub fn decrypt_to_plain(&self, password: &str) -> Result<DecryptionOutcome, PdfError> {
        self.inner.decrypt_to_plain(password)
    }

    /// Applies Binas's public-key encryption with caller-provided recipient certificate bytes.
    pub fn encrypt_public_key(
        &self,
        options: PublicKeyEncryptionOptions,
    ) -> Result<PublicKeyEncryptionOutcome, PdfError> {
        self.inner.encrypt_public_key(options)
    }

    /// Decrypts public-key input from caller-provided DER certificate and PKCS#8 key bytes.
    pub fn decrypt_public_key(
        &self,
        recipient_certificate_der: &[u8],
        recipient_private_key_pkcs8_der: &[u8],
    ) -> Result<PublicKeyDecryptionOutcome, PdfError> {
        self.inner
            .decrypt_public_key(recipient_certificate_der, recipient_private_key_pkcs8_der)
    }

    /// Returns signature inspection records using Binas's built-in defaults.
    pub fn signatures(&self) -> Result<Vec<SignatureInspection>, PdfError> {
        binas_pdf::inspect_signatures(&self.inner)
    }

    /// Returns signature inspection records using caller-provided, offline Binas trust inputs.
    pub fn signatures_with_options(
        &self,
        options: &SignatureTrustOptions,
    ) -> Result<Vec<SignatureInspection>, PdfError> {
        binas_pdf::inspect_signatures_with_options(&self.inner, options)
    }

    /// Prepares a Binas external-CMS signature plan with the default signature field options.
    pub fn prepare_external_signature(
        &self,
        reserved_cms_bytes: usize,
    ) -> Result<ExternalSignaturePlan, PdfError> {
        self.inner.prepare_external_signature(reserved_cms_bytes)
    }

    /// Prepares a Binas external-CMS signature plan with explicit signature field options.
    pub fn prepare_external_signature_with_field(
        &self,
        reserved_cms_bytes: usize,
        field_options: ExternalSignatureFieldOptions,
    ) -> Result<ExternalSignaturePlan, PdfError> {
        self.inner
            .prepare_external_signature_with_field(reserved_cms_bytes, field_options)
    }

    /// Returns resolved page labels in page order.
    pub fn page_labels(&self) -> Result<Vec<PageLabel>, PdfError> {
        binas_pdf::read_page_labels(&self.inner)
    }

    /// Applies Binas's verified page-label update.
    pub fn update_page_label(
        &self,
        update: PageLabelUpdate,
    ) -> Result<DocumentStructureOutcome, PdfError> {
        self.inner.update_page_label(update)
    }

    /// Returns named destinations reachable through the catalog name tree.
    pub fn named_destinations(&self) -> Result<Vec<NamedDestination>, PdfError> {
        binas_pdf::read_named_destinations(&self.inner)
    }

    /// Lists readable JavaScript action text without executing or rewriting it.
    pub fn javascript_actions(&self) -> Result<JavaScriptActionInventory, PdfError> {
        binas_pdf::read_javascript_actions(&self.inner)
    }

    /// Applies Binas's verified named-destination update.
    pub fn update_named_destination(
        &self,
        update: NamedDestinationUpdate,
    ) -> Result<DocumentStructureOutcome, PdfError> {
        self.inner.update_named_destination(update)
    }

    /// Returns the catalog outline items in document order.
    pub fn outlines(&self) -> Result<Vec<OutlineItem>, PdfError> {
        binas_pdf::read_outlines(&self.inner)
    }

    /// Creates a top-level outline through Binas's verified lifecycle operation.
    pub fn create_outline(
        &self,
        request: OutlineCreateRequest,
    ) -> Result<DocumentStructureOutcome, PdfError> {
        self.inner.create_outline(request)
    }

    /// Creates a child outline through Binas's verified lifecycle operation.
    pub fn create_child_outline(
        &self,
        parent_outline_index: usize,
        request: OutlineCreateRequest,
    ) -> Result<DocumentStructureOutcome, PdfError> {
        self.inner
            .create_child_outline(parent_outline_index, request)
    }

    /// Removes an outline subtree through Binas's verified lifecycle operation.
    pub fn remove_outline(
        &self,
        request: OutlineRemoveRequest,
    ) -> Result<DocumentStructureOutcome, PdfError> {
        self.inner.remove_outline(request)
    }

    /// Returns embedded-file metadata reachable through the catalog name tree.
    pub fn embedded_attachments(&self) -> Result<Vec<EmbeddedAttachment>, PdfError> {
        binas_pdf::read_embedded_attachments(&self.inner)
    }

    /// Reads bounded decoded bytes for one exact `embedded_attachments` inventory entry.
    pub fn read_embedded_attachment_bytes(
        &self,
        attachment: &EmbeddedAttachment,
    ) -> Result<Vec<u8>, PdfError> {
        binas_pdf::read_embedded_attachment_bytes(&self.inner, attachment)
    }

    /// Applies Binas's verified embedded-attachment update.
    pub fn update_embedded_attachment(
        &self,
        update: EmbeddedAttachmentUpdate,
    ) -> Result<DocumentStructureOutcome, PdfError> {
        self.inner.update_embedded_attachment(update)
    }

    /// Replaces one image XObject with caller-supplied raw or pre-encoded image data.
    pub fn replace_image_xobject(
        &self,
        request: ImageReplacementRequest,
    ) -> Result<ImageReplacementOutcome, PdfError> {
        self.inner.replace_image_xobject(request)
    }

    /// Replaces one image XObject from Binas-supported encoded image bytes.
    pub fn replace_image_xobject_encoded(
        &self,
        request: EncodedImageReplacementRequest,
    ) -> Result<ImageReplacementOutcome, PdfError> {
        self.inner.replace_image_xobject_encoded(request)
    }

    /// Replaces one Binas-addressed inline image from caller-supplied encoded bytes.
    pub fn replace_inline_image(
        &self,
        request: InlineImageReplacementRequest,
    ) -> Result<InlineImageReplacementOutcome, PdfError> {
        self.inner.replace_inline_image(request)
    }

    /// Places Binas's bounded selectable Helvetica text overlay on one page.
    pub fn place_text_overlay(
        &self,
        request: TextOverlayRequest,
    ) -> Result<TextOverlayOutcome, PdfError> {
        self.inner.place_text_overlay(request)
    }

    /// Places caller-provided Binas Form content as an explicit stamp on explicit page indices.
    pub fn place_overlay_stamp(
        &self,
        request: OverlayStampRequest,
    ) -> Result<OverlayStampOutcome, PdfError> {
        self.inner.place_overlay_stamp(request)
    }

    /// Returns XFA packet metadata that Binas can read safely.
    pub fn xfa_packets(&self) -> Result<Vec<XfaPacket>, PdfError> {
        binas_pdf::list_xfa_packets(&self.inner)
    }

    /// Reports Binas-detected dynamic or unsafe XFA markers without rendering or mutation.
    pub fn inspect_xfa_dynamic(&self) -> Result<XfaDynamicReport, PdfError> {
        binas_pdf::inspect_xfa_dynamic(&self.inner)
    }

    /// Returns fields from Binas-supported static XFA datasets.
    pub fn xfa_dataset_fields(&self) -> Result<Vec<XfaDatasetField>, PdfError> {
        binas_pdf::list_xfa_dataset_fields(&self.inner)
    }

    /// Returns Binas's read-only mappings for supported static XFA template fields and datasets.
    pub fn xfa_template_dataset_mappings(
        &self,
    ) -> Result<Vec<XfaTemplateDatasetMapping>, PdfError> {
        binas_pdf::list_xfa_template_dataset_mappings(&self.inner)
    }

    /// Returns one exact field from a Binas-supported static XFA dataset.
    pub fn xfa_dataset_field(&self, path: &str) -> Result<XfaDatasetField, PdfError> {
        self.inner.get_xfa_dataset_field(path)
    }

    /// Replaces text in one Binas-supported static XFA packet.
    pub fn replace_xfa_text(
        &self,
        request: XfaReplaceRequest,
    ) -> Result<XfaReplaceOutcome, PdfError> {
        self.inner.replace_xfa_text(request)
    }

    /// Sets one field in a Binas-supported static XFA dataset.
    pub fn set_xfa_dataset_field(
        &self,
        request: XfaDatasetSetRequest,
    ) -> Result<XfaDatasetMutationOutcome, PdfError> {
        self.inner.set_xfa_dataset_field(request)
    }

    /// Removes one field from a Binas-supported static XFA dataset.
    pub fn remove_xfa_dataset_field(
        &self,
        path: &str,
    ) -> Result<XfaDatasetMutationOutcome, PdfError> {
        self.inner.remove_xfa_dataset_field(path)
    }

    /// Plans a Binas OCR text layer from caller-provided bounding boxes.
    pub fn plan_ocr_text_layer(
        &self,
        request: OcrTextLayerRequest,
    ) -> Result<OcrTextLayerPlan, PdfError> {
        self.inner.plan_ocr_text_layer(request)
    }

    /// Applies a Binas OCR text-layer plan to this exact source document.
    pub fn apply_ocr_text_layer(
        &self,
        plan: &OcrTextLayerPlan,
    ) -> Result<OcrTextLayerOutcome, PdfError> {
        self.inner.apply_ocr_text_layer(plan)
    }

    /// Returns AcroForm field metadata that Binas can resolve safely.
    pub fn form_fields(&self) -> Result<Vec<FormField>, PdfError> {
        binas_pdf::list_form_fields(&self.inner)
    }

    /// Applies Binas's verified incremental form-value mutation.
    pub fn set_form_field_value(
        &self,
        request: FormValueMutationRequest,
    ) -> Result<FormValueMutationOutcome, PdfError> {
        self.inner.set_form_field_value(request)
    }

    /// Sets a Binas-proven checkbox state without inferring an appearance policy.
    pub fn set_checkbox_field(
        &self,
        request: CheckboxFieldMutationRequest,
    ) -> Result<ButtonFieldMutationOutcome, PdfError> {
        self.inner.set_checkbox_field(request)
    }

    /// Selects an exact Binas-proven radio-button appearance state.
    pub fn set_button_field_choice(
        &self,
        request: ButtonChoiceMutationRequest,
    ) -> Result<ButtonFieldMutationOutcome, PdfError> {
        self.inner.set_button_field_choice(request)
    }

    /// Explicitly regenerates one Binas-supported text-field appearance.
    pub fn regenerate_text_field_appearance(
        &self,
        request: TextFieldAppearanceRequest,
    ) -> Result<TextFieldAppearanceOutcome, PdfError> {
        self.inner.regenerate_text_field_appearance(request)
    }

    /// Creates one Binas-supported form field with Binas-managed widget appearances.
    pub fn create_form_field(
        &self,
        request: FormFieldCreateRequest,
    ) -> Result<FormLifecycleOutcome, PdfError> {
        self.inner.create_form_field(request)
    }

    /// Removes one Binas-addressed form field and its reachable widgets.
    pub fn remove_form_field(
        &self,
        request: FormFieldRemoveRequest,
    ) -> Result<FormLifecycleOutcome, PdfError> {
        self.inner.remove_form_field(request)
    }

    /// Flattens all Binas-supported form fields into verified page content.
    pub fn flatten_form_fields(&self) -> Result<FormLifecycleOutcome, PdfError> {
        self.inner.flatten_form_fields()
    }

    /// Returns page annotations that Binas can resolve safely.
    pub fn annotations(&self) -> Result<Vec<Annotation>, PdfError> {
        binas_pdf::list_annotations(&self.inner)
    }

    /// Applies Binas's verified incremental annotation-contents mutation.
    pub fn set_annotation_contents(
        &self,
        request: AnnotationContentsMutationRequest,
    ) -> Result<AnnotationContentsMutationOutcome, PdfError> {
        self.inner.set_annotation_contents(request)
    }

    /// Explicitly regenerates one Binas-supported FreeText annotation appearance.
    pub fn regenerate_free_text_appearance(
        &self,
        request: FreeTextAppearanceRequest,
    ) -> Result<FreeTextAppearanceOutcome, PdfError> {
        self.inner.regenerate_free_text_appearance(request)
    }

    /// Creates one Binas-supported annotation with its verified appearance lifecycle.
    pub fn create_annotation(
        &self,
        request: AnnotationCreateRequest,
    ) -> Result<AnnotationLifecycleOutcome, PdfError> {
        self.inner.create_annotation(request)
    }

    /// Removes one Binas-addressed annotation and its unreachable appearance objects.
    pub fn remove_annotation(
        &self,
        request: AnnotationRemoveRequest,
    ) -> Result<AnnotationLifecycleOutcome, PdfError> {
        self.inner.remove_annotation(request)
    }

    /// Returns stable zero-based page handles.
    pub fn pages(&self) -> Result<Vec<Page>, PdfError> {
        Ok((0..self.inspect()?.page_count).map(Page::new).collect())
    }

    /// Returns effective inherited geometry for one zero-based page index.
    pub fn page_geometry(&self, page_index: usize) -> Result<PageGeometry, PdfError> {
        self.inner.page_geometry(page_index)
    }

    /// Extracts the engine's structured text spans without layout inference.
    pub fn extract_text(&self) -> Result<TextExtraction, PdfError> {
        self.inner.extract_text_spans()
    }

    /// Extracts Binas's structured text spans and warnings for one zero-based page.
    pub fn extract_page_text(&self, page_index: usize) -> Result<TextExtraction, PdfError> {
        let page_count = self.inspect()?.page_count;
        if page_index >= page_count {
            return Err(PdfError {
                code: binas_pdf::PdfErrorCode::SelectionNotFound,
                message: format!("page index {page_index} exceeds page count {page_count}"),
                span: None,
                object: None,
            });
        }
        let extraction = self.inner.extract_text_spans()?;
        Ok(TextExtraction {
            spans: extraction
                .spans
                .into_iter()
                .filter(|span| span.page_index == page_index)
                .collect(),
            warnings: extraction
                .warnings
                .into_iter()
                .filter(|warning| warning.page_index == page_index)
                .collect(),
        })
    }

    /// Returns one exact text match by zero-based match index.
    pub fn query_text(&self, needle: &str, match_index: usize) -> Result<QueryMatch, PdfError> {
        self.inner.query_text(needle, match_index)
    }

    /// Returns every exact text match in Binas engine order.
    pub fn query_text_all(&self, needle: &str) -> Result<Vec<QueryMatch>, PdfError> {
        self.inner.query_text_all(needle)
    }

    /// Applies Binas's verified direct-source surgical text edit.
    pub fn surgical_text_edit(
        &self,
        request: SurgicalTextEditRequest,
    ) -> Result<SurgicalEditOutcome, PdfError> {
        self.inner.surgical_text_edit(request)
    }

    /// Appends a verified revision for direct, unfiltered text with an ASCII replacement.
    pub fn incremental_text_edit(
        &self,
        request: IncrementalTextEditRequest,
    ) -> Result<IncrementalEditOutcome, PdfError> {
        self.inner.incremental_text_edit(request)
    }

    /// Appends a verified revision for one Binas-supported filtered content stream.
    pub fn filtered_text_edit(
        &self,
        request: FilteredTextEditRequest,
    ) -> Result<FilteredEditOutcome, PdfError> {
        self.inner.filtered_text_edit(request)
    }

    /// Appends a verified revision by encoding through the selected font's ToUnicode CMap.
    pub fn font_text_edit(
        &self,
        request: FontTextEditRequest,
    ) -> Result<FontEditOutcome, PdfError> {
        self.inner.font_text_edit(request)
    }

    /// Plans Binas's source-bound surgical batch text edit without applying a fallback rewrite.
    pub fn plan_batch_text_edits(
        &self,
        request: BatchTextEditRequest,
    ) -> Result<BatchTextEditPlan, PdfError> {
        self.inner.plan_batch_text_edits(request)
    }

    /// Applies a Binas source-bound batch plan with its explicit post-write verification.
    pub fn apply_batch_text_edits(
        &self,
        plan: BatchTextEditPlan,
    ) -> Result<BatchTextEditOutcome, PdfError> {
        self.inner.apply_batch_text_edits(plan)
    }

    /// Applies Binas's verified direct decoded-stream mutation.
    pub fn mutate_stream(
        &self,
        request: StreamMutationRequest,
    ) -> Result<StreamMutationOutcome, PdfError> {
        self.inner.mutate_stream(request)
    }

    /// Lists stream metadata without decoding or copying stream bytes.
    pub fn streams(&self) -> Result<Vec<StreamInventoryEntry>, PdfError> {
        binas_pdf::list_streams(&self.inner)
    }

    /// Decodes one explicitly selected stream through Binas's bounded filter chain.
    pub fn read_decoded_stream(&self, object: StreamObjectRef) -> Result<Vec<u8>, PdfError> {
        binas_pdf::read_decoded_stream(&self.inner, object)
    }

    /// Lists image-XObject metadata without decoding or copying image bytes.
    pub fn image_xobjects(&self) -> Result<Vec<ImageXObjectInventoryEntry>, PdfError> {
        binas_pdf::list_image_xobjects(&self.inner)
    }

    /// Lists inline-image metadata from one direct, unfiltered content stream per page.
    pub fn inline_images(&self) -> Result<Vec<InlineImageInventoryEntry>, PdfError> {
        binas_pdf::list_inline_images(&self.inner)
    }

    /// Reads opaque JPEG bytes for one exact direct-DCT image inventory entry.
    pub fn read_jpeg_xobject_bytes(
        &self,
        image: &ImageXObjectInventoryEntry,
    ) -> Result<Vec<u8>, PdfError> {
        binas_pdf::read_jpeg_xobject_bytes(&self.inner, image)
    }

    /// Reads opaque JPEG 2000 bytes for one exact direct-JPX image inventory entry.
    pub fn read_jpx_xobject_bytes(
        &self,
        image: &ImageXObjectInventoryEntry,
    ) -> Result<Vec<u8>, PdfError> {
        binas_pdf::read_jpx_xobject_bytes(&self.inner, image)
    }

    /// Reads raw samples for one exact, direct-Flate image inventory entry.
    pub fn read_raw_flate_image_samples(
        &self,
        image: &ImageXObjectInventoryEntry,
    ) -> Result<RawFlateImageSamples, PdfError> {
        binas_pdf::read_raw_flate_image_samples(&self.inner, image)
    }

    /// Copies the selected pages into a Binas-verified output PDF.
    pub fn copy_pages(&self, page_indices: &[usize]) -> Result<PageOperationOutcome, PdfError> {
        self.inner.copy_pages(page_indices)
    }

    /// Extracts the selected pages into a Binas-verified output PDF.
    pub fn extract_pages(&self, page_indices: &[usize]) -> Result<PageOperationOutcome, PdfError> {
        self.inner.extract_pages(page_indices)
    }

    /// Inserts selected pages from `source` at a zero-based insertion point.
    pub fn insert_pages(
        &self,
        at: usize,
        source: &Document,
        source_page_indices: &[usize],
    ) -> Result<PageOperationOutcome, PdfError> {
        self.inner
            .insert_pages(at, &source.inner, source_page_indices)
    }

    /// Inserts one Binas-verified blank page at a zero-based insertion point.
    pub fn insert_blank_page(
        &self,
        index: usize,
        size: BlankPageSize,
    ) -> Result<PageOperationOutcome, PdfError> {
        let blank = open(&create_blank_pdf(&[size])?)?;
        self.insert_pages(index, &blank, &[0])
    }

    /// Appends every page from each source document in order.
    pub fn merge_pages(&self, sources: &[&Document]) -> Result<PageOperationOutcome, PdfError> {
        let sources = sources
            .iter()
            .map(|source| &source.inner)
            .collect::<Vec<_>>();
        self.inner.merge_pages(&sources)
    }

    /// Applies Binas's bounded transform to the selected zero-based pages.
    pub fn transform_pages(
        &self,
        page_indices: &[usize],
        transform: PageTransform,
    ) -> Result<PageOperationOutcome, PdfError> {
        self.inner.transform_pages(page_indices, transform)
    }

    /// Paints a source page onto a target page through Binas's composition engine.
    pub fn compose_page(
        &self,
        source: &Document,
        request: PageCompositionRequest,
    ) -> Result<PageOperationOutcome, PdfError> {
        self.inner.compose_page(&source.inner, request)
    }
}

/// A zero-based page handle for read-only enumeration.
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub struct Page {
    index: usize,
}

impl Page {
    fn new(index: usize) -> Self {
        Self { index }
    }

    /// Returns the zero-based page index.
    pub fn index(self) -> usize {
        self.index
    }
}
