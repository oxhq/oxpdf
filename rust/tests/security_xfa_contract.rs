use oxpdf::{
    CmsParseStatus, SignatureTrustOptions, TrustStatus, XfaDatasetSetRequest, XfaDynamicReport,
    XfaReplaceRequest, open,
};

#[test]
fn reads_encryption_metadata_and_signature_inspection_through_binas() {
    let encrypted = open(&encrypted_fixture()).expect("encrypted fixture opens through binas-pdf");
    let metadata = encrypted
        .encryption_metadata()
        .expect("encryption metadata reads");
    assert!(metadata.encrypted);
    assert_eq!(metadata.filter.as_deref(), Some("Standard"));
    assert_eq!(metadata.revision, Some(4));
    assert_eq!(metadata.permissions, Some(-4));

    let signatures = open(&signature_fixture())
        .expect("signature fixture opens through binas-pdf")
        .signatures()
        .expect("signatures inspect");
    assert_eq!(signatures.len(), 1);
    assert_eq!(signatures[0].signer_name.as_deref(), Some("Signer"));
    assert!(signatures[0].covers_current_file);
    assert_eq!(signatures[0].signed_bytes_sha256.len(), 64);
    assert!(!signatures[0].cms_verified);
}

#[test]
fn forwards_native_signature_trust_options_without_adding_a_policy() {
    let fixture = signature_fixture_with_cms(&cms_without_certificate());
    let document = open(&fixture).expect("signature fixture opens through binas-pdf");

    assert_eq!(
        document.signatures().expect("default signatures inspect")[0]
            .cms
            .trust_status,
        TrustStatus::NotRequested
    );

    let signatures = document
        .signatures_with_options(&SignatureTrustOptions {
            roots_der: vec![b"caller-supplied-root".to_vec()],
            ..SignatureTrustOptions::default()
        })
        .expect("native trust inputs inspect without ambient trust");
    assert_eq!(signatures.len(), 1);
    assert_eq!(signatures[0].cms.parse_status, CmsParseStatus::Parsed);
    assert_eq!(
        signatures[0].cms.trust_status,
        TrustStatus::SignerCertificateMissing
    );
}

#[test]
fn rejects_invalid_signature_byte_ranges_with_explicit_options() {
    let document = open(&classic_pdf(
        &[
            b"<< /Type /Catalog /Pages 2 0 R >>".to_vec(),
            b"<< /Type /Pages /Kids [3 0 R] /Count 1 >>".to_vec(),
            b"<< /Type /Page /Parent 2 0 R >>".to_vec(),
            b"<< /Type /Sig /ByteRange [1 0 0 0] /Contents <00> >>".to_vec(),
        ],
        "",
    ))
    .expect("invalid signature metadata does not prevent opening");

    assert!(
        document
            .signatures_with_options(&SignatureTrustOptions::default())
            .is_err()
    );
}

#[test]
fn reads_xfa_packets_and_static_dataset_fields_through_binas() {
    let document = open(&static_xfa_fixture()).expect("XFA fixture opens through binas-pdf");

    let packets = document.xfa_packets().expect("XFA packets read");
    assert_eq!(packets.len(), 1);
    assert_eq!(packets[0].label, "datasets");
    assert_eq!(packets[0].root_element.as_deref(), Some("datasets"));
    assert!(!packets[0].unsafe_xml);

    let fields = document
        .xfa_dataset_fields()
        .expect("static XFA fields read");
    assert_eq!(
        fields
            .iter()
            .map(|field| (field.path.as_str(), field.value.as_str()))
            .collect::<Vec<_>>(),
        [("form.name", "Alice & Bob"), ("form.address.city", "TJ")]
    );
}

#[test]
fn exposes_read_only_static_xfa_template_dataset_mappings() {
    let document = open(&mapped_xfa_fixture()).expect("mapped XFA fixture opens through binas-pdf");

    let mappings = document
        .xfa_template_dataset_mappings()
        .expect("static template mappings read");
    assert_eq!(
        mappings
            .iter()
            .map(|mapping| {
                (
                    mapping.field_name.as_str(),
                    mapping.dataset_path.as_str(),
                    mapping.value.as_str(),
                )
            })
            .collect::<Vec<_>>(),
        [
            ("form.account.id", "form.account.id", "INV-7"),
            ("email", "form.payer.email", "payer@example.test"),
        ]
    );
    assert!(
        mappings
            .iter()
            .all(|mapping| mapping.template_packet_index == 0 && mapping.dataset_packet_index == 1)
    );
    assert_eq!(
        open(&dynamic_xfa_fixture())
            .expect("dynamic XFA fixture opens through binas-pdf")
            .xfa_template_dataset_mappings()
            .expect_err("dynamic XFA mappings refuse renderer semantics")
            .code
            .as_str(),
        "unsafe_rewrite"
    );
}

#[test]
fn reads_an_exact_static_xfa_dataset_field_and_refuses_missing_paths() {
    let document = open(&static_xfa_fixture()).expect("XFA fixture opens through binas-pdf");

    let field = document
        .xfa_dataset_field("form.address.city")
        .expect("exact static XFA field reads");
    assert_eq!(field.path, "form.address.city");
    assert_eq!(field.value, "TJ");
    assert_eq!(field.packet_index, 0);
    assert_eq!((field.object_number, field.object_generation), (5, 0));

    let error = document
        .xfa_dataset_field("form.address.country")
        .expect_err("missing XFA path fails closed");
    assert_eq!(error.code.as_str(), "selection_not_found");
}

#[test]
fn reports_absent_static_and_dynamic_xfa_without_rendering_or_mutation() {
    let absent: XfaDynamicReport = open(&classic_pdf(
        &[
            b"<< /Type /Catalog /Pages 2 0 R >>".to_vec(),
            b"<< /Type /Pages /Kids [3 0 R] /Count 1 >>".to_vec(),
            b"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] >>".to_vec(),
        ],
        "",
    ))
    .expect("non-XFA fixture opens through binas-pdf")
    .inspect_xfa_dynamic()
    .expect("absent XFA report reads");
    assert_eq!(
        absent,
        XfaDynamicReport {
            present: false,
            dynamic: false,
            static_packets: false,
            markers: vec![],
        }
    );

    let static_report = open(&static_xfa_fixture())
        .expect("static XFA fixture opens through binas-pdf")
        .inspect_xfa_dynamic()
        .expect("static XFA report reads");
    assert!(static_report.present);
    assert!(!static_report.dynamic);
    assert!(static_report.static_packets);
    assert!(static_report.markers.is_empty());

    let dynamic_report = open(&dynamic_xfa_fixture())
        .expect("dynamic XFA fixture opens through binas-pdf")
        .inspect_xfa_dynamic()
        .expect("dynamic XFA report reads");
    assert!(dynamic_report.present);
    assert!(dynamic_report.dynamic);
    assert!(!dynamic_report.static_packets);
    assert!(
        dynamic_report
            .markers
            .iter()
            .any(|marker| marker == r#"config: dynamicRender="required""#)
    );
}

#[test]
fn replaces_a_static_xfa_packet_with_reopen_verification() {
    let document = open(&static_xfa_fixture()).expect("XFA fixture opens through binas-pdf");
    let outcome = document
        .replace_xfa_text(XfaReplaceRequest {
            old_text: "Alice".into(),
            new_text: "Olive".into(),
            packet_index: 0,
        })
        .expect("static XFA packet replacement succeeds");
    assert!(outcome.verification.passed);
    assert!(outcome.verification.reparsed);
    assert!(outcome.verification.old_text_removed);
    assert!(outcome.verification.new_text_present);

    let reopened = open(&outcome.bytes).expect("replaced XFA PDF reopens through binas-pdf");
    assert_eq!(
        reopened
            .xfa_dataset_fields()
            .expect("static XFA fields read")[0]
            .value,
        "Olive & Bob"
    );
}

#[test]
fn sets_a_static_xfa_dataset_field_with_reopen_verification() {
    let document = open(&static_xfa_fixture()).expect("XFA fixture opens through binas-pdf");
    let outcome = document
        .set_xfa_dataset_field(XfaDatasetSetRequest {
            path: "form.name".into(),
            value: "Ana & Co".into(),
        })
        .expect("static XFA field set succeeds");
    assert!(outcome.verification.passed);
    assert!(outcome.verification.reparsed);
    assert!(outcome.verification.path_state_verified);

    let reopened = open(&outcome.bytes).expect("updated XFA PDF reopens through binas-pdf");
    let field = reopened
        .xfa_dataset_fields()
        .expect("static XFA fields read")
        .into_iter()
        .find(|field| field.path == "form.name")
        .expect("name field remains present");
    assert_eq!(field.value, "Ana & Co");
}

#[test]
fn removes_a_static_xfa_dataset_field_with_reopen_verification() {
    let document = open(&static_xfa_fixture()).expect("XFA fixture opens through binas-pdf");
    let outcome = document
        .remove_xfa_dataset_field("form.address.city")
        .expect("static XFA field removal succeeds");
    assert!(outcome.verification.passed);
    assert!(outcome.verification.reparsed);
    assert!(outcome.verification.path_state_verified);

    let reopened = open(&outcome.bytes).expect("updated XFA PDF reopens through binas-pdf");
    assert!(
        reopened
            .xfa_dataset_fields()
            .expect("static XFA fields read")
            .into_iter()
            .all(|field| field.path != "form.address.city")
    );
}

fn encrypted_fixture() -> Vec<u8> {
    classic_pdf(
        &[
            b"<< /Type /Catalog /Pages 2 0 R >>".to_vec(),
            b"<< /Type /Pages /Kids [3 0 R] /Count 1 >>".to_vec(),
            b"<< /Type /Page /Parent 2 0 R >>".to_vec(),
            b"<< /Filter /Standard /V 4 /R 4 /Length 128 /P -4 /EncryptMetadata false /StmF /StdCF /StrF /StdCF >>".to_vec(),
        ],
        " /Encrypt 4 0 R",
    )
}

fn signature_fixture() -> Vec<u8> {
    let placeholder = "9999999999";
    let mut bytes = classic_pdf(
        &[
            b"<< /Type /Catalog /Pages 2 0 R >>".to_vec(),
            b"<< /Type /Pages /Kids [3 0 R] /Count 1 >>".to_vec(),
            b"<< /Type /Page /Parent 2 0 R >>".to_vec(),
            format!(
                "<< /Type /Sig /Filter /Adobe.PPKLite /SubFilter /adbe.pkcs7.detached /Name (Signer) /ByteRange [0 {placeholder} {placeholder} {placeholder}] /Contents <{}> >>",
                "00".repeat(64)
            )
            .into_bytes(),
        ],
        "",
    );
    let marker = bytes
        .windows(b"/Contents <".len())
        .position(|window| window == b"/Contents <")
        .expect("signature contents marker");
    let gap_start = marker + b"/Contents ".len();
    let gap_end = gap_start
        + bytes[gap_start..]
            .iter()
            .position(|byte| *byte == b'>')
            .expect("signature contents terminator")
        + 1;
    let mut search = 0;
    for value in [gap_start, gap_end, bytes.len() - gap_end] {
        let position = bytes[search..]
            .windows(placeholder.len())
            .position(|window| window == placeholder.as_bytes())
            .expect("byte range placeholder")
            + search;
        bytes[position..position + placeholder.len()]
            .copy_from_slice(format!("{value:010}").as_bytes());
        search = position + placeholder.len();
    }
    bytes
}

fn signature_fixture_with_cms(cms: &[u8]) -> Vec<u8> {
    let placeholder = "9999999999";
    let mut bytes = classic_pdf(
        &[
            b"<< /Type /Catalog /Pages 2 0 R >>".to_vec(),
            b"<< /Type /Pages /Kids [3 0 R] /Count 1 >>".to_vec(),
            b"<< /Type /Page /Parent 2 0 R >>".to_vec(),
            format!(
                "<< /Type /Sig /ByteRange [0 {placeholder} {placeholder} {placeholder}] /Contents <{}> >>",
                "00".repeat(cms.len())
            )
            .into_bytes(),
        ],
        "",
    );
    let marker = bytes
        .windows(b"/Contents <".len())
        .position(|window| window == b"/Contents <")
        .expect("signature contents marker");
    let gap_start = marker + b"/Contents ".len();
    let gap_end = gap_start
        + bytes[gap_start..]
            .iter()
            .position(|byte| *byte == b'>')
            .expect("signature contents terminator")
        + 1;
    let mut search = 0;
    for value in [gap_start, gap_end, bytes.len() - gap_end] {
        let position = bytes[search..]
            .windows(placeholder.len())
            .position(|window| window == placeholder.as_bytes())
            .expect("byte range placeholder")
            + search;
        bytes[position..position + placeholder.len()]
            .copy_from_slice(format!("{value:010}").as_bytes());
        search = position + placeholder.len();
    }
    let encoded: String = cms.iter().map(|byte| format!("{byte:02X}")).collect();
    let contents_start = gap_start + 1;
    bytes[contents_start..contents_start + encoded.len()].copy_from_slice(encoded.as_bytes());
    bytes
}

fn cms_without_certificate() -> Vec<u8> {
    vec![
        0x30, 0x54, 0x06, 0x09, 0x2A, 0x86, 0x48, 0x86, 0xF7, 0x0D, 0x01, 0x07, 0x02, 0xA0, 0x47,
        0x30, 0x45, 0x02, 0x01, 0x03, 0x31, 0x0D, 0x30, 0x0B, 0x06, 0x09, 0x60, 0x86, 0x48, 0x01,
        0x65, 0x03, 0x04, 0x02, 0x01, 0x30, 0x0B, 0x06, 0x09, 0x2A, 0x86, 0x48, 0x86, 0xF7, 0x0D,
        0x01, 0x07, 0x01, 0x31, 0x24, 0x30, 0x22, 0x02, 0x01, 0x03, 0x80, 0x01, 0x01, 0x30, 0x0B,
        0x06, 0x09, 0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x01, 0x30, 0x0A, 0x06, 0x08,
        0x2A, 0x86, 0x48, 0xCE, 0x3D, 0x04, 0x03, 0x02, 0x04, 0x01, 0x00,
    ]
}

fn static_xfa_fixture() -> Vec<u8> {
    let dataset = br#"<xfa:datasets xmlns:xfa="http://www.xfa.org/schema/xfa-data/1.0/"><xfa:data><form><name>Alice &amp; Bob</name><address><city>TJ</city></address></form></xfa:data></xfa:datasets>"#;
    classic_pdf(
        &[
            b"<< /Type /Catalog /Pages 2 0 R /AcroForm 4 0 R >>".to_vec(),
            b"<< /Type /Pages /Kids [3 0 R] /Count 1 >>".to_vec(),
            b"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] >>".to_vec(),
            b"<< /XFA [(datasets) 5 0 R] >>".to_vec(),
            stream(dataset),
        ],
        "",
    )
}

fn mapped_xfa_fixture() -> Vec<u8> {
    let template = b"<template><field name=\"form.account.id\"/><subform name=\"form\"><subform name=\"payer\"><field name=\"email\"/></subform></subform></template>";
    let dataset = br#"<xfa:datasets xmlns:xfa="http://www.xfa.org/schema/xfa-data/1.0/"><xfa:data><form><account><id>INV-7</id></account><payer><email>payer@example.test</email></payer></form></xfa:data></xfa:datasets>"#;
    classic_pdf(
        &[
            b"<< /Type /Catalog /Pages 2 0 R /AcroForm 4 0 R >>".to_vec(),
            b"<< /Type /Pages /Kids [3 0 R] /Count 1 >>".to_vec(),
            b"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] >>".to_vec(),
            b"<< /XFA [(template) 5 0 R (datasets) 6 0 R] >>".to_vec(),
            stream(template),
            stream(dataset),
        ],
        "",
    )
}

fn dynamic_xfa_fixture() -> Vec<u8> {
    let config = b"<config><dynamicRender>required</dynamicRender></config>";
    classic_pdf(
        &[
            b"<< /Type /Catalog /Pages 2 0 R /AcroForm 4 0 R >>".to_vec(),
            b"<< /Type /Pages /Kids [3 0 R] /Count 1 >>".to_vec(),
            b"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] >>".to_vec(),
            b"<< /XFA [(config) 5 0 R] >>".to_vec(),
            stream(config),
        ],
        "",
    )
}

fn classic_pdf(objects: &[Vec<u8>], trailer_extra: &str) -> Vec<u8> {
    let mut bytes = b"%PDF-1.7\n".to_vec();
    let mut offsets = Vec::with_capacity(objects.len());
    for (index, object) in objects.iter().enumerate() {
        offsets.push(bytes.len());
        bytes.extend_from_slice(format!("{} 0 obj\n", index + 1).as_bytes());
        bytes.extend_from_slice(object);
        bytes.extend_from_slice(b"\nendobj\n");
    }
    let xref = bytes.len();
    let size = objects.len() + 1;
    bytes.extend_from_slice(format!("xref\n0 {size}\n0000000000 65535 f \n").as_bytes());
    for offset in offsets {
        bytes.extend_from_slice(format!("{offset:010} 00000 n \n").as_bytes());
    }
    bytes.extend_from_slice(
        format!(
            "trailer\n<< /Size {size} /Root 1 0 R{trailer_extra} >>\nstartxref\n{xref}\n%%EOF\n"
        )
        .as_bytes(),
    );
    bytes
}

fn stream(contents: &[u8]) -> Vec<u8> {
    [
        format!("<< /Length {} >>\nstream\n", contents.len()).into_bytes(),
        contents.to_vec(),
        b"\nendstream".to_vec(),
    ]
    .concat()
}
