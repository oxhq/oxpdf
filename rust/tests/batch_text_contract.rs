use oxpdf::{BatchTextEditRequest, SurgicalTextEditRequest, open};

mod support;

#[test]
fn plans_applies_and_reopens_two_surgical_text_edits() {
    let document = open(&support::fixture_pdf(&["alpha", "bravo"], None))
        .expect("fixture opens through binas-pdf");
    let plan = document
        .plan_batch_text_edits(BatchTextEditRequest {
            edits: vec![edit("bravo", "BRAVO"), edit("alpha", "omega")],
        })
        .expect("two source-bound edits plan");
    assert_eq!(plan.edits.len(), 2);

    let outcome = document
        .apply_batch_text_edits(plan)
        .expect("batch edit applies");
    assert_eq!(outcome.report.mode, "surgical");
    assert_eq!(outcome.report.edit_count, 2);
    assert!(outcome.verification.passed);
    assert!(outcome.verification.reparse_ok);
    assert!(outcome.verification.replacements_selectable);

    let reopened = open(&outcome.bytes).expect("batch output reopens through binas-pdf");
    assert_eq!(
        reopened
            .extract_text()
            .expect("batch output text extracts")
            .spans
            .into_iter()
            .map(|span| span.text)
            .collect::<Vec<_>>(),
        ["omega", "BRAVO"]
    );
}

fn edit(old_text: &str, replacement: &str) -> SurgicalTextEditRequest {
    SurgicalTextEditRequest {
        old_text: old_text.into(),
        replacement: replacement.into(),
        match_index: 0,
    }
}
