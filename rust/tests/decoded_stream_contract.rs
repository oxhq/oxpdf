use oxpdf::{StreamObjectRef, open};

#[test]
fn reads_one_decoded_stream_by_its_object_reference() {
    let document = open(&pdf_bytes(&[
        b"<< /Type /Catalog >>",
        b"<< /Length 11 /Filter /ASCIIHexDecode >>\nstream\n48656c6c6f>\nendstream",
    ]))
    .expect("fixture opens");

    assert_eq!(
        document
            .read_decoded_stream(StreamObjectRef {
                object_number: 2,
                object_generation: 0,
            })
            .expect("supported stream decodes"),
        b"Hello"
    );
}

#[test]
fn refuses_unsupported_or_malformed_stream_filters() {
    let document = open(&pdf_bytes(&[
        b"<< /Type /Catalog >>",
        b"<< /Length 4 /Filter /DCTDecode >>\nstream\njpeg\nendstream",
        b"<< /Length 1 /Filter [/FlateDecode 7] >>\nstream\nA\nendstream",
    ]))
    .expect("fixture opens");

    for (object_number, code) in [(2, "unsupported_feature"), (3, "invalid_syntax")] {
        let error = document
            .read_decoded_stream(StreamObjectRef {
                object_number,
                object_generation: 0,
            })
            .expect_err("unsafe or malformed filter must fail closed");
        assert_eq!(error.code.as_str(), code);
        assert_eq!(error.object, Some((object_number, 0)));
    }
}

fn pdf_bytes(objects: &[&[u8]]) -> Vec<u8> {
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
