mod support;

use oxpdf::{BlankPageSize, PageGeometry, open};

#[test]
fn inserts_a_blank_page_in_the_middle_without_reordering_text() {
    let document = open(&support::fixture_pdf(&["BEFORE", "AFTER"], None))
        .expect("fixture opens through binas-pdf");

    let outcome = document
        .insert_blank_page(
            1,
            BlankPageSize {
                width: 200.0,
                height: 300.0,
            },
        )
        .expect("middle blank page insertion succeeds");
    assert!(outcome.verification.passed);

    let reopened = open(&outcome.bytes).expect("output reopens through binas-pdf");
    assert_eq!(reopened.inspect().expect("output inspects").page_count, 3);
    assert_eq!(page_text(&reopened, 0), ["BEFORE"]);
    assert!(page_text(&reopened, 1).is_empty());
    assert_eq!(page_text(&reopened, 2), ["AFTER"]);
    assert_eq!(
        reopened
            .page_geometry(1)
            .expect("blank page geometry reads"),
        PageGeometry {
            media_box: [0.0, 0.0, 200.0, 300.0],
            crop_box: [0.0, 0.0, 200.0, 300.0],
            bleed_box: None,
            trim_box: None,
            art_box: None,
            rotation_degrees: 0,
        }
    );
}

fn page_text(document: &oxpdf::Document, index: usize) -> Vec<String> {
    document
        .extract_page_text(index)
        .expect("page text extracts")
        .spans
        .iter()
        .map(|span| span.text.clone())
        .collect()
}
