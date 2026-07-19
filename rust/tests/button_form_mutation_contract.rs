use oxpdf::{ButtonChoiceMutationRequest, CheckboxFieldMutationRequest, open};

#[test]
fn mutates_proven_checkbox_and_radio_states_and_refuses_the_wrong_button_kind() {
    let input = button_pdf();
    let checked = open(&input)
        .expect("fixture opens")
        .set_checkbox_field(CheckboxFieldMutationRequest {
            field_name: "consent".into(),
            checked: true,
            match_index: 0,
        })
        .expect("checkbox updates");
    assert!(checked.verification.passed);
    assert_eq!(checked.report.selected_state, "Yes");

    let choice = open(&checked.bytes)
        .expect("checkbox output opens")
        .set_button_field_choice(ButtonChoiceMutationRequest {
            field_name: "plan".into(),
            state: "Pro".into(),
            match_index: 0,
        })
        .expect("radio selection updates");
    assert!(choice.verification.passed);
    assert_eq!(choice.report.widgets_affected, 2);

    let fields = open(&choice.bytes)
        .expect("button output opens")
        .form_fields()
        .expect("button fields read");
    assert_eq!(fields[0].value.as_deref(), Some("Yes"));
    assert_eq!(fields[1].value.as_deref(), Some("Pro"));

    let error = open(&input)
        .expect("fixture opens")
        .set_button_field_choice(ButtonChoiceMutationRequest {
            field_name: "consent".into(),
            state: "Yes".into(),
            match_index: 0,
        })
        .expect_err("checkbox does not accept radio selection");
    assert_eq!(error.code.as_str(), "unsafe_rewrite");
}

fn button_pdf() -> Vec<u8> {
    classic_pdf(&[
        "<< /Type /Catalog /Pages 2 0 R /AcroForm 4 0 R >>",
        "<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
        "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>",
        "<< /Fields [5 0 R 6 0 R] >>",
        "<< /FT /Btn /T (consent) /Kids [7 0 R] /V /Off >>",
        "<< /FT /Btn /Ff 32768 /T (plan) /Kids [8 0 R 9 0 R] /V /Off >>",
        "<< /Type /Annot /Subtype /Widget /Parent 5 0 R /AP << /N << /Off 10 0 R /Yes 11 0 R >> >> /AS /Off >>",
        "<< /Type /Annot /Subtype /Widget /Parent 6 0 R /AP << /N << /Off 12 0 R /Basic 13 0 R >> >> /AS /Off >>",
        "<< /Type /Annot /Subtype /Widget /Parent 6 0 R /AP << /N << /Off 14 0 R /Pro 15 0 R >> >> /AS /Off >>",
        form_stream(),
        form_stream(),
        form_stream(),
        form_stream(),
        form_stream(),
        form_stream(),
    ])
}

fn form_stream() -> &'static str {
    "<< /Type /XObject /Subtype /Form /BBox [0 0 10 10] /Length 0 >>\nstream\n\nendstream"
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
