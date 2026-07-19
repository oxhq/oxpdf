use oxpdf::{FormFieldCreateRequest, FormFieldKind, FormFieldRemoveRequest, open};

mod support;

#[test]
fn creates_reads_removes_and_flattens_form_fields_with_reopens() {
    let source = open(&support::fixture_pdf(&["FORM"], None)).expect("fixture opens");
    let created = source
        .create_form_field(text_field("remove-me", 10.0, "Remove"))
        .expect("first field creates");
    assert!(created.verification.passed);
    assert!(created.verification.reparsed);

    let after_create = open(&created.bytes).expect("created output reopens");
    assert_eq!(
        after_create.form_fields().expect("created field reads")[0].name,
        "remove-me"
    );
    let with_remaining = after_create
        .create_form_field(text_field("flatten-me", 45.0, "Flatten"))
        .expect("second field creates");
    assert!(with_remaining.verification.passed);

    let after_second_create = open(&with_remaining.bytes).expect("second output reopens");
    let removed = after_second_create
        .remove_form_field(FormFieldRemoveRequest {
            field_name: "remove-me".into(),
            match_index: 0,
        })
        .expect("selected field removes");
    assert!(removed.verification.passed);
    assert!(removed.verification.reparsed);

    let before_flatten = open(&removed.bytes).expect("removed output reopens");
    let fields = before_flatten.form_fields().expect("remaining field reads");
    assert_eq!(fields.len(), 1);
    assert_eq!(fields[0].name, "flatten-me");

    let flattened = before_flatten
        .flatten_form_fields()
        .expect("remaining field flattens");
    assert!(flattened.verification.passed);
    assert!(flattened.verification.reparsed);
    assert_eq!(flattened.report.appearances_placed, 1);

    let reopened = open(&flattened.bytes).expect("flattened output reopens");
    assert!(
        reopened
            .form_fields()
            .expect("flattened fields read")
            .is_empty()
    );
}

fn text_field(name: &str, y: f64, value: &str) -> FormFieldCreateRequest {
    FormFieldCreateRequest {
        name: name.into(),
        page_index: 0,
        rect: [10.0, y, 90.0, y + 20.0],
        kind: FormFieldKind::Text,
        value: value.into(),
        options: Vec::new(),
    }
}
