use std::io::Write;

use flate2::{Compression, write::ZlibEncoder};
use oxpdf::{ImageColorSpace, RawFlateImageSamples, open};

#[test]
fn reads_exact_jpeg_inventory_and_rejects_stale_or_non_jpeg_entries() {
    let jpeg = jpeg();
    let document = open(&image_pdf("/DCTDecode", &jpeg)).expect("JPEG fixture opens");
    let entry = document
        .image_xobjects()
        .expect("JPEG inventory reads")
        .remove(0);

    assert_eq!(
        document
            .read_jpeg_xobject_bytes(&entry)
            .expect("exact JPEG inventory entry reads"),
        jpeg
    );

    let mut stale = entry;
    stale.width = 3;
    assert_eq!(
        document
            .read_jpeg_xobject_bytes(&stale)
            .expect_err("stale JPEG inventory entry is rejected")
            .code
            .as_str(),
        "selection_not_found"
    );

    let non_jpeg = open(&image_pdf("/FlateDecode", b"A")).expect("non-JPEG fixture opens");
    let entry = non_jpeg
        .image_xobjects()
        .expect("non-JPEG inventory reads")
        .remove(0);
    assert_eq!(
        non_jpeg
            .read_jpeg_xobject_bytes(&entry)
            .expect_err("non-JPEG inventory entry is rejected")
            .code
            .as_str(),
        "unsupported_feature"
    );
}

#[test]
fn reads_exact_jpx_inventory_and_rejects_stale_or_non_jpx_entries() {
    let jpx = jpx();
    let document = open(&image_pdf("/JPXDecode", &jpx)).expect("JPX fixture opens");
    let entry = document
        .image_xobjects()
        .expect("JPX inventory reads")
        .remove(0);

    assert_eq!(
        document
            .read_jpx_xobject_bytes(&entry)
            .expect("exact JPX inventory entry reads"),
        jpx
    );

    let mut stale = entry;
    stale.height = 2;
    assert_eq!(
        document
            .read_jpx_xobject_bytes(&stale)
            .expect_err("stale JPX inventory entry is rejected")
            .code
            .as_str(),
        "selection_not_found"
    );

    let non_jpx = open(&image_pdf("/FlateDecode", b"A")).expect("non-JPX fixture opens");
    let entry = non_jpx
        .image_xobjects()
        .expect("non-JPX inventory reads")
        .remove(0);
    assert_eq!(
        non_jpx
            .read_jpx_xobject_bytes(&entry)
            .expect_err("non-JPX inventory entry is rejected")
            .code
            .as_str(),
        "unsupported_feature"
    );
}

#[test]
fn reads_exact_raw_flate_inventory_as_unconverted_samples() {
    let samples = vec![10, 20, 30, 40, 50, 60];
    let document =
        open(&image_pdf("/FlateDecode", &flate(&samples))).expect("raw-Flate fixture opens");
    let entry = document
        .image_xobjects()
        .expect("raw-Flate inventory reads")
        .remove(0);

    assert_eq!(
        document
            .read_raw_flate_image_samples(&entry)
            .expect("exact raw-Flate inventory entry reads"),
        RawFlateImageSamples {
            object: entry.object,
            width: 2,
            height: 1,
            color_space: ImageColorSpace::DeviceRgb,
            samples,
        }
    );

    let jpeg = open(&image_pdf("/DCTDecode", &jpeg())).expect("JPEG fixture opens");
    let entry = jpeg
        .image_xobjects()
        .expect("JPEG inventory reads")
        .remove(0);
    assert_eq!(
        jpeg.read_raw_flate_image_samples(&entry)
            .expect_err("non-Flate entry is rejected")
            .code
            .as_str(),
        "unsupported_feature"
    );
}

fn flate(bytes: &[u8]) -> Vec<u8> {
    let mut encoder = ZlibEncoder::new(Vec::new(), Compression::default());
    encoder.write_all(bytes).expect("sample compression writes");
    encoder.finish().expect("sample compression finishes")
}

fn image_pdf(filter: &str, bytes: &[u8]) -> Vec<u8> {
    classic_pdf(&[
        b"<< /Type /Catalog >>".to_vec(),
        stream(
            bytes,
            &format!(
                " /Type /XObject /Subtype /Image /Width 2 /Height 1 /BitsPerComponent 8 /ColorSpace /DeviceRGB /Filter {filter}"
            ),
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

fn jpx() -> Vec<u8> {
    let mut bytes = vec![0xff, 0x4f, 0xff, 0x51, 0x00, 0x2f, 0x00, 0x00];
    for value in [2_u32, 1, 0, 0, 2, 1, 0, 0] {
        bytes.extend_from_slice(&value.to_be_bytes());
    }
    bytes.extend_from_slice(&3_u16.to_be_bytes());
    for _ in 0..3 {
        bytes.extend_from_slice(&[7, 1, 1]);
    }
    bytes.extend_from_slice(&[0xff, 0xd9]);
    bytes
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
