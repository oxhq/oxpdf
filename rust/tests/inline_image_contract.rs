use oxpdf::{InlineImageColorSpace, InlineImageFilter, InlineImageReplacementRequest, open};

mod support;

#[test]
fn replaces_an_inline_image_and_reopens_through_binas() {
    let content =
        "before) Tj ET q BI /W 1 /H 1 /BPC 8 /CS /RGB ID\n\u{1}\u{2}\u{3}\nEI Q BT (after";
    let document = open(&support::fixture_pdf(&[content], None)).expect("fixture opens");

    let outcome = document
        .replace_inline_image(InlineImageReplacementRequest {
            page_index: 0,
            image_index: 0,
            encoded_bytes: vec![9, 8, 7],
            width: 1,
            height: 1,
            bits_per_component: 8,
            color_space: InlineImageColorSpace::Rgb,
            filter: InlineImageFilter::Raw,
        })
        .expect("inline image replacement succeeds");

    assert!(outcome.verification.passed);
    assert!(outcome.verification.reparsed);
    assert!(outcome.verification.surrounding_bytes_preserved);
    assert!(outcome.verification.encoded_bytes_match);
    assert!(outcome.verification.metadata_matches);
    assert!(outcome.verification.content_reference_preserved);
    assert!(outcome.verification.no_dangling_references);

    let reopened = open(&outcome.bytes).expect("mutated output reopens");
    assert_eq!(
        reopened
            .query_text_all("after")
            .expect("surrounding text remains queryable")
            .len(),
        1
    );
}
