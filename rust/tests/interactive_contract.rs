use oxpdf::{AnnotationContentsMutationRequest, FormValueMutationRequest, open};

#[test]
fn reads_and_edits_a_form_field_through_binas() {
    let document = open(&interactive_pdf()).expect("fixture opens");
    let fields = document.form_fields().expect("fields read");
    assert_eq!(fields.len(), 2);
    assert_eq!(fields[0].value.as_deref(), Some("one"));
    assert_eq!(fields[1].value.as_deref(), Some("two"));

    let outcome = document
        .set_form_field_value(FormValueMutationRequest {
            field_name: "dup".into(),
            value: "second updated".into(),
            match_index: 1,
        })
        .expect("field value updates");
    assert!(outcome.verification.passed);
    assert!(outcome.verification.prefix_preserved);
    assert!(outcome.verification.value_updated);
    assert_eq!(
        open(&outcome.bytes)
            .expect("output opens")
            .form_fields()
            .expect("output fields read")[1]
            .value
            .as_deref(),
        Some("second updated")
    );
}

#[test]
fn reads_and_edits_an_annotation_through_binas() {
    let document = open(&interactive_pdf()).expect("fixture opens");
    let annotations = document.annotations().expect("annotations read");
    assert_eq!(annotations.len(), 1);
    assert_eq!(annotations[0].subtype, "Text");
    assert_eq!(annotations[0].contents.as_deref(), Some("old"));

    let outcome = document
        .set_annotation_contents(AnnotationContentsMutationRequest {
            annotation_index: 0,
            contents: "Nota ✓".into(),
        })
        .expect("annotation contents update");
    assert!(outcome.verification.passed);
    assert!(outcome.verification.prefix_preserved);
    assert!(outcome.verification.contents_updated);
    assert_eq!(
        open(&outcome.bytes)
            .expect("output opens")
            .annotations()
            .expect("output annotations read")[0]
            .contents
            .as_deref(),
        Some("Nota ✓")
    );
}

fn interactive_pdf() -> Vec<u8> {
    classic_pdf(&[
        "<< /Type /Catalog /Pages 2 0 R /AcroForm 4 0 R >>",
        "<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
        "<< /Type /Page /Parent 2 0 R /Annots [7 0 R] >>",
        "<< /NeedAppearances true /Fields [5 0 R 6 0 R] >>",
        "<< /T (dup) /FT /Tx /V (one) >>",
        "<< /T (dup) /FT /Tx /V (two) >>",
        "<< /Type /Annot /Subtype /Text /Rect [0 0 10 10] /Contents (old) >>",
    ])
}

fn classic_pdf(objects: &[&str]) -> Vec<u8> {
    let mut bytes = b"%PDF-1.7\n".to_vec();
    let mut offsets = Vec::with_capacity(objects.len());
    for (index, object) in objects.iter().enumerate() {
        offsets.push(bytes.len());
        bytes.extend_from_slice(format!("{} 0 obj\n{object}\nendobj\n", index + 1).as_bytes());
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
