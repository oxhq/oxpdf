mod support;

use oxpdf::{PageOperationOutcome, open};

#[test]
fn page_workflows_return_binas_verified_outputs() {
    let source = open(&support::fixture_pdf(&["ALPHA", "BETA"], None)).expect("source opens");

    let copied = source.copy_pages(&[1, 0]).expect("pages copy");
    assert_verified(&copied, "copy_pages", &["BETA", "ALPHA"]);

    let extracted = source.extract_pages(&[1]).expect("page extracts");
    assert_verified(&extracted, "extract_pages", &["BETA"]);

    let destination =
        open(&support::fixture_pdf(&["ONE", "THREE"], None)).expect("destination opens");
    let inserted = destination
        .insert_pages(1, &source, &[0])
        .expect("page inserts");
    assert_verified(&inserted, "insert_pages", &["ONE", "ALPHA", "THREE"]);

    let merged = destination
        .merge_pages(&[&source])
        .expect("documents merge");
    assert_verified(&merged, "merge_pages", &["ONE", "THREE", "ALPHA", "BETA"]);
}

fn assert_verified(outcome: &PageOperationOutcome, operation: &str, expected_text: &[&str]) {
    assert_eq!(outcome.report.operation, operation);
    assert_eq!(outcome.report.output_pages, expected_text.len());
    assert!(outcome.verification.passed);
    assert!(outcome.verification.reparsed);
    assert!(outcome.verification.page_count_matches);
    assert!(outcome.verification.no_dangling_references);

    let document = open(&outcome.bytes).expect("Binas output reopens");
    let text = document.extract_text().expect("Binas output text extracts");
    assert_eq!(
        text.spans
            .iter()
            .map(|span| span.text.as_str())
            .collect::<Vec<_>>(),
        expected_text
    );
}
