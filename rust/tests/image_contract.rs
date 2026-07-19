use oxpdf::{
    EncodedImageReplacementRequest, ImageColorSpace, ImageFilter, ImageMaskPolicy,
    ImageReplacementRequest, open,
};

#[test]
fn replaces_raw_and_encoded_image_xobjects_and_reopens_through_binas() {
    let raw = open(&image_pdf())
        .expect("image fixture opens through binas-pdf")
        .replace_image_xobject(ImageReplacementRequest {
            object_number: 5,
            object_generation: 0,
            encoded_bytes: vec![6, 5, 4, 3, 2, 1],
            width: 2,
            height: 1,
            bits_per_component: 8,
            color_space: ImageColorSpace::DeviceRgb,
            filter: ImageFilter::Raw,
            decode_params: None,
            mask_policy: ImageMaskPolicy::Reject,
        })
        .expect("raw image replacement succeeds");
    assert!(raw.verification.passed);
    assert!(raw.verification.object_reference_preserved);
    assert!(raw.verification.encoded_stream_matches);

    let encoded = open(&raw.bytes)
        .expect("raw replacement reopens")
        .replace_image_xobject_encoded(EncodedImageReplacementRequest {
            object_number: 5,
            object_generation: 0,
            encoded_bytes: jpeg(),
            mask_policy: ImageMaskPolicy::Reject,
        })
        .expect("encoded image replacement succeeds");
    assert!(encoded.verification.passed);
    assert!(encoded.verification.object_reference_preserved);
    assert_eq!(encoded.report.filter, ImageFilter::Jpeg);

    let reopened = open(&encoded.bytes).expect("encoded replacement reopens through binas-pdf");
    assert_eq!(
        reopened.inspect().expect("inspection succeeds").page_count,
        1
    );
    assert!(String::from_utf8_lossy(&encoded.bytes).contains("/DCTDecode"));
}

fn image_pdf() -> Vec<u8> {
    classic_pdf(&[
        b"<< /Type /Catalog /Pages 2 0 R >>".to_vec(),
        b"<< /Type /Pages /Kids [3 0 R] /Count 1 >>".to_vec(),
        b"<< /Type /Page /Parent 2 0 R /Resources << /XObject << /Im0 5 0 R >> >> /Contents 4 0 R /MediaBox [0 0 20 20] >>".to_vec(),
        stream(b"q 20 0 0 20 0 0 cm /Im0 Do Q", ""),
        stream(
            &[1, 2, 3, 4, 5, 6],
            " /Type /XObject /Subtype /Image /Width 2 /Height 1 /BitsPerComponent 8 /ColorSpace /DeviceRGB",
        ),
    ])
}

fn jpeg() -> Vec<u8> {
    vec![
        0xff, 0xd8, 0xff, 0xc0, 0x00, 0x11, 0x08, 0x00, 0x01, 0x00, 0x02, 0x03, 0x01, 0x11, 0x00,
        0x02, 0x11, 0x00, 0x03, 0x11, 0x00, 0xff, 0xda, 0x00, 0x0c, 0x03, 0x01, 0x00, 0x02, 0x00,
        0x03, 0x00, 0x00, 0x3f, 0x00, 0x00, 0xff, 0xd9,
    ]
}

fn stream(bytes: &[u8], entries: &str) -> Vec<u8> {
    [
        format!("<< /Length {}{entries} >>\nstream\n", bytes.len()).into_bytes(),
        bytes.to_vec(),
        b"\nendstream".to_vec(),
    ]
    .concat()
}

fn classic_pdf(objects: &[Vec<u8>]) -> Vec<u8> {
    let mut bytes = b"%PDF-1.7\n".to_vec();
    let mut offsets = Vec::with_capacity(objects.len());
    for (index, object) in objects.iter().enumerate() {
        offsets.push(bytes.len());
        bytes.extend_from_slice(format!("{} 0 obj\n", index + 1).as_bytes());
        bytes.extend_from_slice(object);
        bytes.extend_from_slice(b"\nendobj\n");
    }
    let xref = bytes.len();
    let size = objects.len() + 1;
    bytes.extend_from_slice(format!("xref\n0 {size}\n0000000000 65535 f \n").as_bytes());
    for offset in offsets {
        bytes.extend_from_slice(format!("{offset:010} 00000 n \n").as_bytes());
    }
    bytes.extend_from_slice(
        format!("trailer\n<< /Size {size} /Root 1 0 R >>\nstartxref\n{xref}\n%%EOF\n").as_bytes(),
    );
    bytes
}
