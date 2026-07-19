use oxpdf::{AnnotationCreateRequest, AnnotationRemoveRequest, AnnotationSubtype, open};

mod support;

#[test]
fn creates_reads_removes_and_reopens_an_annotation() {
    let source = open(&support::fixture_pdf(&["ANNOTATION"], None)).expect("fixture opens");
    let created = source
        .create_annotation(AnnotationCreateRequest {
            page_index: 0,
            subtype: AnnotationSubtype::Text,
            rect: [10.0, 10.0, 90.0, 40.0],
            contents: "Created note".into(),
            quad_points: Vec::new(),
            uri: String::new(),
        })
        .expect("annotation creates");
    assert!(created.verification.passed);
    assert!(created.verification.reparsed);
    assert!(created.verification.appearance_reachable);

    let after_create = open(&created.bytes).expect("created output reopens");
    let annotation = after_create
        .annotations()
        .expect("created annotation reads")
        .into_iter()
        .next()
        .expect("created annotation exists");
    assert_eq!(annotation.subtype, "Text");
    assert_eq!(annotation.contents.as_deref(), Some("Created note"));

    let removed = after_create
        .remove_annotation(AnnotationRemoveRequest {
            annotation_index: annotation.index,
        })
        .expect("created annotation removes");
    assert!(removed.verification.passed);
    assert!(removed.verification.reparsed);
    assert!(removed.verification.no_dangling_references);

    assert!(
        open(&removed.bytes)
            .expect("removed output reopens")
            .annotations()
            .expect("removed annotations read")
            .is_empty()
    );
}
