use std::io::Write;

use flate2::{Compression, write::ZlibEncoder};
use oxpdf::{FilteredTextEditRequest, FontTextEditRequest, IncrementalTextEditRequest, open};

mod support;

#[test]
fn incrementally_edits_direct_text_and_reopens() {
    let input = support::fixture_pdf(&["short"], None);
    let outcome = open(&input)
        .expect("fixture opens")
        .incremental_text_edit(IncrementalTextEditRequest {
            old_text: "short".into(),
            replacement: "a much longer value".into(),
            match_index: 0,
        })
        .expect("direct incremental edit succeeds");

    assert_eq!(outcome.report.mode, "incremental");
    assert!(outcome.verification.passed);
    assert!(outcome.verification.prefix_preserved);
    assert!(outcome.verification.page_count_unchanged);
    assert!(outcome.verification.replacement_selectable);
    assert!(outcome.verification.revision_incremented);
    assert_eq!(&outcome.bytes[..input.len()], input.as_slice());
    assert_reopens(&outcome.bytes, "short", "a much longer value");
}

#[test]
fn edits_flate_filtered_text_and_reopens() {
    let input = filtered_pdf(b"BT (short) Tj ET");
    let outcome = open(&input)
        .expect("fixture opens")
        .filtered_text_edit(FilteredTextEditRequest {
            old_text: "short".into(),
            replacement: "a much longer value".into(),
            match_index: 0,
        })
        .expect("filtered incremental edit succeeds");

    assert_eq!(outcome.report.mode, "filtered_incremental");
    assert!(outcome.verification.passed);
    assert!(outcome.verification.prefix_preserved);
    assert!(outcome.verification.page_count_unchanged);
    assert!(outcome.verification.decoded_stream_verified);
    assert!(outcome.verification.filter_metadata_preserved);
    assert!(outcome.verification.replacement_selectable);
    assert!(outcome.verification.revision_incremented);
    assert_eq!(&outcome.bytes[..input.len()], input.as_slice());
    assert_reopens(&outcome.bytes, "short", "a much longer value");
}

#[test]
fn edits_tounicode_font_text_and_reopens() {
    let input = font_pdf();
    let outcome = open(&input)
        .expect("fixture opens")
        .font_text_edit(FontTextEditRequest {
            old_text: "Ω".into(),
            replacement: "中😀".into(),
            match_index: 0,
        })
        .expect("font incremental edit succeeds");

    assert_eq!(outcome.report.mode, "font_incremental");
    assert_eq!(outcome.report.encoded_glyph_bytes, 3);
    assert!(outcome.verification.passed);
    assert!(outcome.verification.prefix_preserved);
    assert!(outcome.verification.page_count_unchanged);
    assert!(outcome.verification.decoded_stream_verified);
    assert!(outcome.verification.replacement_selectable);
    assert!(outcome.verification.revision_incremented);
    assert_eq!(&outcome.bytes[..input.len()], input.as_slice());
    assert_reopens(&outcome.bytes, "Ω", "中😀");
}

fn assert_reopens(bytes: &[u8], old: &str, replacement: &str) {
    let reopened = open(bytes).expect("output reopens");
    assert_eq!(
        reopened.inspect().expect("output inspects").xref_revisions,
        2
    );
    assert!(
        reopened
            .query_text_all(old)
            .expect("old text queries")
            .is_empty()
    );
    assert_eq!(
        reopened
            .query_text_all(replacement)
            .expect("replacement text queries")
            .len(),
        1
    );
}

fn filtered_pdf(content: &[u8]) -> Vec<u8> {
    let mut encoder = ZlibEncoder::new(Vec::new(), Compression::default());
    encoder.write_all(content).expect("content compresses");
    let encoded = encoder.finish().expect("stream finishes");
    write_pdf(vec![
        b"<< /Type /Catalog /Pages 2 0 R >>".to_vec(),
        b"<< /Type /Pages /Kids [3 0 R] /Count 1 >>".to_vec(),
        b"<< /Type /Page /Parent 2 0 R /Contents 4 0 R >>".to_vec(),
        stream(&encoded, " /Filter /FlateDecode"),
    ])
}

fn font_pdf() -> Vec<u8> {
    write_pdf(vec![
        b"<< /Type /Catalog /Pages 2 0 R >>".to_vec(),
        b"<< /Type /Pages /Kids [3 0 R] /Count 1 /Resources 5 0 R >>".to_vec(),
        b"<< /Type /Page /Parent 2 0 R /Contents 4 0 R >>".to_vec(),
        stream(b"BT /F1 12 Tf <0102> Tj ET", ""),
        b"<< /Font << /F1 6 0 R >> >>".to_vec(),
        b"<< /Type /Font /Subtype /Type0 /ToUnicode 7 0 R >>".to_vec(),
        stream(
            b"2 begincodespacerange <00> <ff> <0100> <01ff> endcodespacerange 5 beginbfchar <0102> <03a9> <03> <4e2d> <04> <4e2d> <0105> <d83dde00> <06> <0058> endbfchar",
            "",
        ),
    ])
}

fn stream(data: &[u8], entries: &str) -> Vec<u8> {
    [
        format!("<< /Length {}{entries} >>\nstream\n", data.len()).into_bytes(),
        data.to_vec(),
        b"\nendstream".to_vec(),
    ]
    .concat()
}

fn write_pdf(objects: Vec<Vec<u8>>) -> Vec<u8> {
    let mut bytes = b"%PDF-1.7\n".to_vec();
    let mut offsets = Vec::with_capacity(objects.len());
    for (index, object) in objects.iter().enumerate() {
        offsets.push(bytes.len());
        bytes.extend_from_slice(format!("{} 0 obj\n", index + 1).as_bytes());
        bytes.extend_from_slice(object);
        bytes.extend_from_slice(b"\nendobj\n");
    }
    let xref = bytes.len();
    bytes.extend_from_slice(
        format!("xref\n0 {}\n0000000000 65535 f \n", objects.len() + 1).as_bytes(),
    );
    for offset in offsets {
        bytes.extend_from_slice(format!("{offset:010} 00000 n \n").as_bytes());
    }
    bytes.extend_from_slice(
        format!(
            "trailer\n<< /Size {} /Root 1 0 R >>\nstartxref\n{xref}\n%%EOF\n",
            objects.len() + 1
        )
        .as_bytes(),
    );
    bytes
}
