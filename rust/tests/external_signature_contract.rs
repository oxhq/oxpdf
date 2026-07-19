use oxpdf::{ExternalSignatureFieldOptions, ExternalSignaturePlan, open};

mod support;

#[test]
fn prepares_an_external_signature_plan_through_binas() {
    let document = open(&support::fixture_pdf(&["original"], None)).expect("fixture opens");

    let plan = document
        .prepare_external_signature_with_field(
            1024,
            ExternalSignatureFieldOptions {
                field_name: Some("Approval".into()),
                page_index: 0,
                rect: [12.0, 24.0, 48.0, 36.0],
            },
        )
        .expect("external signature preparation succeeds");

    assert_eq!(plan.digest_algorithm, "sha256");
    assert_eq!(plan.digest_to_sign.len(), 32);
    assert_eq!(plan.reserved_cms_bytes, 1024);
    assert_ne!(plan.signature_object_number, 0);
    assert_ne!(plan.field_object_number, 0);

    let descriptor = plan.descriptor();
    assert_eq!(descriptor.digest_to_sign, plan.digest_to_sign);
    assert_eq!(descriptor.byte_range, plan.byte_range);
    assert_eq!(descriptor.reserved_cms_bytes, 1024);

    let reopened = open(&plan.bytes).expect("prepared PDF reopens");
    assert_eq!(
        reopened
            .inspect()
            .expect("prepared PDF inspects")
            .page_count,
        1
    );

    let reloaded = ExternalSignaturePlan::from_prepared_pdf(plan.bytes, descriptor)
        .expect("plan descriptor reloads through Binas");
    assert_eq!(reloaded.digest_algorithm, "sha256");
    assert_eq!(reloaded.digest_to_sign.len(), 32);
}

#[test]
fn prepares_the_default_signature_field_and_refuses_an_empty_reservation() {
    let document = open(&support::fixture_pdf(&["original"], None)).expect("fixture opens");

    let error = document
        .prepare_external_signature(0)
        .expect_err("an empty CMS reservation is unsafe");
    assert_eq!(error.code.as_str(), "unsafe_rewrite");

    let plan = document
        .prepare_external_signature(1024)
        .expect("default external signature preparation succeeds");
    assert_eq!(plan.reserved_cms_bytes, 1024);
    assert_ne!(plan.field_object_number, 0);
    assert_eq!(
        open(&plan.bytes)
            .expect("prepared PDF reopens")
            .inspect()
            .expect("prepared PDF inspects")
            .page_count,
        1
    );
}
