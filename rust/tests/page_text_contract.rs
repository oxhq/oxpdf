mod support;

use oxpdf::open;

#[test]
fn extracts_only_the_requested_zero_based_page_text() {
    let document = open(&support::fixture_pdf(&["ALPHA", "BETA", "GAMMA"], None))
        .expect("fixture opens through binas-pdf");

    let extraction = document.extract_page_text(1).expect("page text extracts");
    assert_eq!(
        extraction
            .spans
            .iter()
            .map(|span| (span.page_index, span.text.as_str()))
            .collect::<Vec<_>>(),
        [(1, "BETA")]
    );
    assert!(extraction.warnings.is_empty());
}

#[test]
fn rejects_a_page_outside_the_document() {
    let document = open(&support::fixture_pdf(&["ALPHA"], None)).expect("fixture opens");

    let error = document
        .extract_page_text(1)
        .expect_err("out-of-range page is rejected");
    assert_eq!(error.code.as_str(), "selection_not_found");
}
