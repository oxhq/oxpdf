use oxpdf::open;

mod support;

#[test]
fn explicitly_canonicalizes_and_reopens_through_binas() {
    let input = support::fixture_pdf(&["ALPHA", "BETA"], None);
    let outcome = open(&input)
        .expect("fixture opens")
        .canonicalize()
        .expect("explicit canonicalization succeeds");

    assert_eq!(outcome.report.operation, "canonicalize");
    assert_eq!(outcome.report.mode, "canonical");
    assert_eq!(outcome.report.input_bytes, input.len());
    assert_eq!(outcome.report.output_bytes, outcome.bytes.len());
    assert!(outcome.verification.passed);
    assert!(outcome.verification.reparsed);
    assert!(outcome.verification.page_count_unchanged);
    assert!(outcome.verification.text_queries_available);
    assert!(outcome.verification.text_query_semantics_unchanged);

    let reopened = open(&outcome.bytes).expect("canonical output reopens");
    assert_eq!(reopened.inspect().expect("output inspects").page_count, 2);
    assert_eq!(
        reopened
            .extract_text()
            .expect("canonical output text extracts")
            .spans
            .iter()
            .map(|span| span.text.as_str())
            .collect::<Vec<_>>(),
        ["ALPHA", "BETA"]
    );
}
