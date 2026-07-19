use oxpdf::{StreamMutationRequest, open};

mod support;

#[test]
fn mutates_a_stream_and_reopens_through_binas() {
    let document = open(&support::fixture_pdf(&["original"], None)).expect("fixture opens");

    let outcome = document
        .mutate_stream(StreamMutationRequest {
            object_number: 4,
            object_generation: 0,
            decoded_bytes: b"BT (replacement) Tj ET".to_vec(),
        })
        .expect("stream mutation succeeds");

    assert!(outcome.verification.passed);
    assert!(outcome.verification.reparsed);
    assert!(outcome.verification.decoded_stream_matches);
    assert!(outcome.verification.no_dangling_references);

    let reopened = open(&outcome.bytes).expect("mutated output reopens");
    assert_eq!(
        reopened
            .query_text("replacement", 0)
            .expect("replacement is queryable")
            .text,
        "replacement"
    );
}
