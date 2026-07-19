use oxpdf::{CapabilityDecision, DocumentCapabilityProfile, OpenOptions, open, open_with_options};
mod support;

#[test]
fn opens_inspects_reads_metadata_enumerates_pages_and_extracts_text() {
    let document = open(&support::fixture_pdf(
        &["ALPHA", "BETA"],
        Some(("Rust cutover", "OxHQ")),
    ))
    .expect("fixture opens through binas-pdf");

    let inspection = document.inspect().expect("inspection succeeds");
    assert_eq!(inspection.page_count, 2);
    assert_eq!(inspection.object_count, 7);

    let metadata = document.metadata().expect("metadata reads");
    assert_eq!(metadata.title.as_deref(), Some("Rust cutover"));
    assert_eq!(metadata.author.as_deref(), Some("OxHQ"));

    let pages = document.pages().expect("pages enumerate");
    assert_eq!(
        pages.iter().map(|page| page.index()).collect::<Vec<_>>(),
        [0, 1]
    );

    let text = document.extract_text().expect("text extracts");
    assert_eq!(
        text.spans
            .iter()
            .map(|span| (span.page_index, span.text.as_str()))
            .collect::<Vec<_>>(),
        [(0, "ALPHA"), (1, "BETA")]
    );
}

#[test]
fn validates_and_returns_binas_capabilities_without_interpreting_them() {
    let document = open(&support::fixture_pdf(&["ALPHA", "BETA"], None)).expect("fixture opens");
    let inspection = document.inspect().expect("inspection succeeds");
    let validation = document.validate().expect("validation succeeds");
    let profile = document.capability_profile().expect("profile succeeds");

    assert!(validation.valid);
    assert_eq!(validation.object_count, inspection.object_count);
    assert_eq!(validation.page_count, inspection.page_count);
    assert_eq!(profile.profile_version, DocumentCapabilityProfile::VERSION);
    assert_eq!(profile.object_count, validation.object_count);
    assert_eq!(profile.page_count, validation.page_count);
    assert_eq!(
        profile
            .operation("canonicalize")
            .expect("canonicalize capability")
            .decision,
        CapabilityDecision::Conditional
    );
}

#[test]
fn explicit_repair_recovers_a_broken_xref_without_weakening_strict_open() {
    let mut input = support::fixture_pdf(&["RECOVERED"], None);
    let marker = b"startxref\n";
    let offset_start = input
        .windows(marker.len())
        .rposition(|window| window == marker)
        .expect("fixture has startxref")
        + marker.len();
    let offset_end = offset_start
        + input[offset_start..]
            .iter()
            .position(|byte| *byte == b'\n')
            .expect("startxref offset terminates");
    input[offset_start..offset_end].fill(b'0');

    let strict_error = open(&input).expect_err("strict open refuses a broken xref");
    assert_eq!(strict_error.code.as_str(), "invalid_syntax");

    let repaired = open_with_options(&input, OpenOptions { repair: true })
        .expect("explicit repair recovers the object graph");
    assert_eq!(
        repaired
            .inspect()
            .expect("repaired PDF inspects")
            .page_count,
        1
    );
    assert_eq!(
        repaired
            .query_text_all("RECOVERED")
            .expect("repaired text queries")
            .len(),
        1
    );

    assert!(
        open_with_options(
            b"%PDF-1.7\ntrailer\n<< /Size 1 >>\n",
            OpenOptions { repair: true }
        )
        .is_err(),
        "repair must still reject input without a usable object graph"
    );
}
