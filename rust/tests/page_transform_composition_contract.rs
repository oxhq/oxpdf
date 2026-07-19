use oxpdf::{PageCompositionPlacement, PageCompositionRequest, PageTransform, open};

mod support;

#[test]
fn transforms_selected_pages_and_reopens_through_binas() {
    let document = open(&support::fixture_pdf(&["ALPHA", "BETA"], None))
        .expect("fixture opens through binas-pdf");

    let outcome = document
        .transform_pages(
            &[0],
            PageTransform {
                rotation_degrees: Some(90),
                translate: Some([12.0, 24.0]),
                ..PageTransform::default()
            },
        )
        .expect("page transform succeeds");
    assert!(outcome.verification.passed);
    assert!(outcome.verification.reparsed);
    assert!(outcome.verification.page_count_matches);

    let reopened = open(&outcome.bytes).expect("transformed PDF reopens through binas-pdf");
    assert_eq!(
        reopened.inspect().expect("inspection succeeds").page_count,
        2
    );
    assert_eq!(
        reopened.query_text_all("ALPHA").expect("text reads").len(),
        1
    );
    assert_eq!(
        reopened.query_text_all("BETA").expect("text reads").len(),
        1
    );
}

#[test]
fn composes_a_source_page_and_reopens_through_binas() {
    let target = open(&support::fixture_pdf(&["TARGET"], None))
        .expect("target fixture opens through binas-pdf");
    let source = open(&support::fixture_pdf(&["SOURCE"], None))
        .expect("source fixture opens through binas-pdf");

    let outcome = target
        .compose_page(
            &source,
            PageCompositionRequest {
                target_page_index: 0,
                source_page_index: 0,
                transform: [1.0, 0.0, 0.0, 1.0, 10.0, 20.0],
                placement: PageCompositionPlacement::Overlay,
            },
        )
        .expect("page composition succeeds");
    assert!(outcome.verification.passed);
    assert!(outcome.verification.reparsed);
    assert!(outcome.verification.no_dangling_references);

    let reopened = open(&outcome.bytes).expect("composed PDF reopens through binas-pdf");
    assert_eq!(
        reopened.inspect().expect("inspection succeeds").page_count,
        1
    );
    assert_eq!(
        reopened.query_text_all("TARGET").expect("text reads").len(),
        1
    );
    assert!(String::from_utf8_lossy(&outcome.bytes).contains("SOURCE"));
}
