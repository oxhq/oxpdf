use oxpdf::{AppearanceStatus, FreeTextAppearanceRequest, TextFieldAppearanceRequest, open};

#[test]
fn regenerates_a_text_field_appearance_and_reopens() {
    let input = text_field_pdf();
    let outcome = open(&input)
        .expect("fixture opens")
        .regenerate_text_field_appearance(TextFieldAppearanceRequest {
            field_name: "name".into(),
            value: "Created".into(),
            match_index: 0,
        })
        .expect("text field appearance regenerates");

    assert_eq!(outcome.report.appearance_status, AppearanceStatus::Created);
    assert!(outcome.verification.passed);
    assert!(outcome.verification.prefix_preserved);
    assert!(outcome.verification.page_count_unchanged);
    assert!(outcome.verification.value_updated);
    assert!(outcome.verification.appearance_updated);
    assert!(outcome.verification.appearance_reachable);
    assert!(outcome.verification.no_dangling_references);
    assert!(outcome.verification.revision_incremented);

    let reopened = open(&outcome.bytes).expect("output reopens");
    assert_eq!(
        reopened.form_fields().expect("fields read")[0]
            .value
            .as_deref(),
        Some("Created")
    );
}

#[test]
fn regenerates_a_free_text_appearance_and_reopens() {
    let input = free_text_pdf();
    let outcome = open(&input)
        .expect("fixture opens")
        .regenerate_free_text_appearance(FreeTextAppearanceRequest {
            annotation_index: 0,
            contents: "Created note".into(),
        })
        .expect("FreeText appearance regenerates");

    assert_eq!(outcome.report.appearance_status, AppearanceStatus::Created);
    assert!(outcome.verification.passed);
    assert!(outcome.verification.prefix_preserved);
    assert!(outcome.verification.page_count_unchanged);
    assert!(outcome.verification.contents_updated);
    assert!(outcome.verification.appearance_updated);
    assert!(outcome.verification.appearance_reachable);
    assert!(outcome.verification.no_dangling_references);
    assert!(outcome.verification.revision_incremented);

    let reopened = open(&outcome.bytes).expect("output reopens");
    assert_eq!(
        reopened.annotations().expect("annotations read")[0]
            .contents
            .as_deref(),
        Some("Created note")
    );
}

fn text_field_pdf() -> Vec<u8> {
    pdf(vec![
        b"<< /Type /Catalog /Pages 2 0 R /AcroForm 4 0 R >>".to_vec(),
        b"<< /Type /Pages /Kids [3 0 R] /Count 1 >>".to_vec(),
        b"<< /Type /Page /Parent 2 0 R /Annots [6 0 R] >>".to_vec(),
        b"<< /DA (/Helv 12 Tf) /DR << /Font << /Helv 7 0 R >> >> /Fields [5 0 R] >>".to_vec(),
        b"<< /T (name) /FT /Tx /V (Old) /Kids [6 0 R] >>".to_vec(),
        b"<< /Type /Annot /Subtype /Widget /Parent 5 0 R /Rect [10 20 110 40] >>".to_vec(),
        b"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>".to_vec(),
    ])
}

fn free_text_pdf() -> Vec<u8> {
    pdf(vec![
        b"<< /Type /Catalog /Pages 2 0 R >>".to_vec(),
        b"<< /Type /Pages /Kids [3 0 R] /Count 1 /Resources << /Font << /F1 5 0 R >> >> >>"
            .to_vec(),
        b"<< /Type /Page /Parent 2 0 R /Annots [4 0 R] >>".to_vec(),
        b"<< /Type /Annot /Subtype /FreeText /Rect [0 0 120 24] /DA (/F1 10 Tf) /Contents (Old) >>"
            .to_vec(),
        b"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>".to_vec(),
    ])
}

fn pdf(objects: Vec<Vec<u8>>) -> Vec<u8> {
    let mut bytes = b"%PDF-1.7\n".to_vec();
    let mut offsets = Vec::with_capacity(objects.len());
    for (index, object) in objects.iter().enumerate() {
        offsets.push(bytes.len());
        bytes.extend_from_slice(format!("{} 0 obj\n", index + 1).as_bytes());
        bytes.extend_from_slice(object);
        bytes.extend_from_slice(b"\nendobj\n");
    }
    let xref = bytes.len();
    bytes.extend_from_slice(
        format!("xref\n0 {}\n0000000000 65535 f \n", objects.len() + 1).as_bytes(),
    );
    for offset in offsets {
        bytes.extend_from_slice(format!("{offset:010} 00000 n \n").as_bytes());
    }
    bytes.extend_from_slice(
        format!(
            "trailer\n<< /Size {} /Root 1 0 R >>\nstartxref\n{xref}\n%%EOF\n",
            objects.len() + 1
        )
        .as_bytes(),
    );
    bytes
}
