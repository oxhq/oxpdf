use oxpdf::{
    ImageColorSpace, InlineImageColorSpace, InlineImageFilter, InlineImageInventoryEntry,
    JavaScriptActionInventory, PageGeometry, PdfFilter, StreamObjectRef, open,
};

#[test]
fn reads_inherited_page_geometry_through_binas() {
    let document = open(&classic(&[
        "<< /Type /Catalog /Pages 2 0 R >>",
        "<< /Type /Pages /Kids [3 0 R] /Count 1 /MediaBox [0 0 400 500] /CropBox [10 20 390 480] /BleedBox [11 21 389 479] /TrimBox [12 22 388 478] /ArtBox [13 23 387 477] /Rotate -90 >>",
        "<< /Type /Page /Parent 2 0 R >>",
    ]))
    .expect("fixture opens");

    assert_eq!(
        document.page_geometry(0).expect("geometry reads"),
        PageGeometry {
            media_box: [0.0, 0.0, 400.0, 500.0],
            crop_box: [10.0, 20.0, 390.0, 480.0],
            bleed_box: Some([11.0, 21.0, 389.0, 479.0]),
            trim_box: Some([12.0, 22.0, 388.0, 478.0]),
            art_box: Some([13.0, 23.0, 387.0, 477.0]),
            rotation_degrees: 270,
        }
    );
}

#[test]
fn inventories_filtered_and_image_stream_metadata_without_decoding() {
    let document = open(&classic(&[
        "<< /Type /Catalog >>",
        "<< /Length 7 /Filter /FlateDecode >>\nstream\nencoded\nendstream",
        "<< /Type /XObject /Subtype /Image /Length 1 >>\nstream\nA\nendstream",
    ]))
    .expect("fixture opens");

    let streams = document.streams().expect("stream metadata reads");
    assert_eq!(streams.len(), 2);
    assert_eq!(
        streams[0].object,
        StreamObjectRef {
            object_number: 2,
            object_generation: 0,
        }
    );
    assert_eq!(streams[0].encoded_length, 7);
    assert_eq!(streams[0].filter_chain.len(), 1);
    assert_eq!(streams[0].filter_chain[0].filter, PdfFilter::FlateDecode);
    assert!(!streams[0].image_xobject);
    assert_eq!(streams[1].encoded_length, 1);
    assert!(streams[1].filter_chain.is_empty());
    assert!(streams[1].image_xobject);
}

#[test]
fn inventories_image_xobject_metadata_without_exposing_its_bytes() {
    let document = open(&classic(&[
        "<< /Type /Catalog >>",
        "<< /Type /XObject /Subtype /Image /Width 2 /Height 1 /ColorSpace /DeviceRGB /Filter /FlateDecode /Length 1 >>\nstream\nA\nendstream",
    ]))
    .expect("fixture opens");

    let images = document.image_xobjects().expect("image metadata reads");
    assert_eq!(images.len(), 1);
    assert_eq!(
        images[0].object,
        StreamObjectRef {
            object_number: 2,
            object_generation: 0,
        }
    );
    assert_eq!(images[0].width, 2);
    assert_eq!(images[0].height, 1);
    assert_eq!(images[0].color_space, Some(ImageColorSpace::DeviceRgb));
    assert_eq!(images[0].filter_chain.len(), 1);
    assert_eq!(images[0].filter_chain[0].filter, PdfFilter::FlateDecode);
}

#[test]
fn inventories_inline_image_metadata_and_refuses_content_arrays() {
    let content = "q BI /W 1 /H 1 /BPC 8 /CS /RGB ID\nabc\nEI Q";
    let stream = format!(
        "<< /Length {} >>\nstream\n{content}\nendstream",
        content.len()
    );
    let document = open(&classic(&[
        "<< /Type /Catalog /Pages 2 0 R >>",
        "<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
        "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] /Contents 4 0 R >>",
        &stream,
    ]))
    .expect("fixture opens");

    assert_eq!(
        document.inline_images().expect("inline metadata reads"),
        vec![InlineImageInventoryEntry {
            page_index: 0,
            image_index: 0,
            width: 1,
            height: 1,
            color_space: InlineImageColorSpace::Rgb,
            filter: InlineImageFilter::Raw,
            encoded_byte_length: 3,
        }]
    );

    let unsupported = open(&classic(&[
        "<< /Type /Catalog /Pages 2 0 R >>",
        "<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
        "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] /Contents [4 0 R] >>",
        "<< /Length 0 >>\nstream\n\nendstream",
    ]))
    .expect("array fixture opens");
    assert_eq!(
        unsupported
            .inline_images()
            .expect_err("content arrays must fail closed")
            .code
            .as_str(),
        "unsupported_feature"
    );
}

#[test]
fn inventories_javascript_actions_without_execution_and_refuses_unreadable_scripts() {
    let document = open(&classic(&[
        "<< /Type /Catalog /Pages 2 0 R /Names 4 0 R >>",
        "<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
        "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] >>",
        "<< /JavaScript 5 0 R >>",
        "<< /Names [(Reachable) 6 0 R] >>",
        "<< /S /JavaScript /JS (app.alert('reachable')) >>",
        "<< /S /JavaScript /JS (app.alert('orphan')) >>",
    ]))
    .expect("fixture opens");

    let inventory: JavaScriptActionInventory = document
        .javascript_actions()
        .expect("JavaScript action text inventories");
    assert_eq!(
        inventory
            .direct
            .iter()
            .map(|action| (
                action.object_number,
                action.name.as_deref(),
                action.script.as_str()
            ))
            .collect::<Vec<_>>(),
        [
            (6, None, "app.alert('reachable')"),
            (7, None, "app.alert('orphan')"),
        ]
    );
    assert_eq!(
        inventory
            .name_tree
            .iter()
            .map(|action| (
                action.object_number,
                action.name.as_deref(),
                action.script.as_str()
            ))
            .collect::<Vec<_>>(),
        [(6, Some("Reachable"), "app.alert('reachable')")]
    );

    let malformed = open(&classic(&[
        "<< /Type /Catalog /Pages 2 0 R >>",
        "<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
        "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] >>",
        "<< /S /JavaScript /JS <FF> >>",
    ]))
    .expect("malformed-script fixture opens");
    assert_eq!(
        malformed
            .javascript_actions()
            .expect_err("unreadable JavaScript must fail closed")
            .code
            .as_str(),
        "unsupported_feature"
    );
}

fn classic(objects: &[&str]) -> Vec<u8> {
    let mut bytes = b"%PDF-1.7\n".to_vec();
    let mut offsets = Vec::with_capacity(objects.len());
    for (index, object) in objects.iter().enumerate() {
        offsets.push(bytes.len());
        bytes.extend_from_slice(format!("{} 0 obj\n{object}\nendobj\n", index + 1).as_bytes());
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
