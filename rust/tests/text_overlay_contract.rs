use oxpdf::{TextOverlayRequest, open};

mod support;

#[test]
fn places_selectable_text_and_reopens_through_binas() {
    let document = open(&support::fixture_pdf(&["original"], None)).expect("fixture opens");

    let outcome = document
        .place_text_overlay(TextOverlayRequest {
            page_index: 0,
            text: "APPROVED".into(),
            x: 12.0,
            y: 24.0,
            font_size: 12.0,
        })
        .expect("text overlay succeeds");

    assert!(outcome.verification.passed);
    assert!(outcome.verification.reparsed);
    assert!(outcome.verification.form_stream_matches);
    assert!(outcome.verification.placements_match);
    assert!(outcome.verification.font_resource_matches);
    assert!(outcome.verification.text_selectable);
    assert!(outcome.verification.no_dangling_references);

    let reopened = open(&outcome.bytes).expect("text-overlay output reopens");
    assert_eq!(
        reopened
            .inspect()
            .expect("reopened document inspects")
            .page_count,
        1
    );
    assert_eq!(
        reopened
            .query_text_all("original")
            .expect("original page text remains queryable")
            .len(),
        1
    );
    assert_eq!(
        reopened
            .query_text_all("APPROVED")
            .expect("overlay text remains queryable through its Form XObject")
            .len(),
        1
    );
}
