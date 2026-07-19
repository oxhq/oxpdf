use oxpdf::{SurgicalTextEditRequest, open};

mod support;

#[test]
fn queries_exact_matches_through_binas() {
    let document = open(&support::fixture_pdf(&["ALPHA", "BETA", "ALPHA"], None))
        .expect("fixture opens through binas-pdf");

    let matches = document
        .query_text_all("ALPHA")
        .expect("exact matches query");
    assert_eq!(matches.len(), 2);
    assert_eq!(
        matches
            .iter()
            .map(|matched| matched.match_index)
            .collect::<Vec<_>>(),
        [0, 1]
    );
    assert_eq!(
        document
            .query_text("ALPHA", 1)
            .expect("second match queries"),
        matches[1]
    );
}

#[test]
fn surgically_edits_and_reopens_through_binas() {
    let document =
        open(&support::fixture_pdf(&["ALPHA"], None)).expect("fixture opens through binas-pdf");

    let outcome = document
        .surgical_text_edit(SurgicalTextEditRequest {
            old_text: "ALPHA".into(),
            replacement: "OMEGA".into(),
            match_index: 0,
        })
        .expect("direct-source surgical edit succeeds");

    assert!(outcome.verification.passed);
    assert!(outcome.verification.reparse_ok);
    assert!(outcome.verification.page_count_unchanged);
    assert!(outcome.verification.replacement_selectable);

    let reopened = open(&outcome.bytes).expect("edited PDF reopens through binas-pdf");
    assert!(
        reopened
            .query_text_all("ALPHA")
            .expect("old text queries after reopen")
            .is_empty()
    );
    assert_eq!(
        reopened
            .query_text("OMEGA", 0)
            .expect("replacement queries after reopen")
            .text,
        "OMEGA"
    );
}
