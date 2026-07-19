use oxpdf::{OverlayStampRequest, open};

mod support;

#[test]
fn places_a_direct_overlay_stamp_and_reopens_through_binas() {
    let input = support::fixture_pdf(&["ORIGINAL", "UNTOUCHED"], None);
    let outcome = open(&input)
        .expect("fixture opens")
        .place_overlay_stamp(OverlayStampRequest {
            page_indices: vec![0],
            form_content: b"0 0 10 10 re 1 0 0 rg f".to_vec(),
            bbox: [0.0, 0.0, 10.0, 10.0],
            transform: [1.0, 0.0, 0.0, 1.0, 12.0, 24.0],
            opacity: Some(0.5),
        })
        .expect("direct overlay stamp succeeds");

    assert_eq!(outcome.report.operation, "place_overlay_stamp");
    assert_eq!(outcome.report.pages_stamped, 1);
    assert_eq!(outcome.report.input_bytes, input.len());
    assert_eq!(outcome.report.output_bytes, outcome.bytes.len());
    assert!(outcome.verification.passed);
    assert!(outcome.verification.reparsed);
    assert!(outcome.verification.page_count_unchanged);
    assert!(outcome.verification.form_stream_matches);
    assert!(outcome.verification.placements_match);
    assert!(outcome.verification.no_dangling_references);

    let reopened = open(&outcome.bytes).expect("stamped output reopens");
    assert_eq!(reopened.inspect().expect("output inspects").page_count, 2);
    assert_eq!(
        reopened
            .query_text_all("ORIGINAL")
            .expect("original text queries")
            .len(),
        1
    );
    assert_eq!(
        reopened
            .query_text_all("UNTOUCHED")
            .expect("other page text queries")
            .len(),
        1
    );
}
