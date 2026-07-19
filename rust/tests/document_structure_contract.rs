use std::collections::BTreeMap;

use oxpdf::{
    DocumentInfoUpdate, EmbeddedAttachmentUpdate, NamedDestinationUpdate, OutlineCreateRequest,
    OutlineRemoveRequest, PageLabelSpec, PageLabelStyle, PageLabelUpdate, XmpMetadataUpdate, open,
};

#[test]
fn reads_info_and_xmp_metadata_through_binas() {
    let document = open(&metadata_pdf()).expect("fixture opens");

    let info = document.metadata().expect("Info metadata reads");
    assert_eq!(info.title.as_deref(), Some("Rust cutover"));
    assert_eq!(info.author.as_deref(), Some("OxHQ"));

    let xmp = document
        .xmp_metadata()
        .expect("XMP metadata reads")
        .expect("fixture has XMP");
    assert_eq!(
        xmp.xml,
        b"<?xpacket begin=''?><x:xmpmeta xmlns:x='adobe:ns:meta/'/>"
    );
}

#[test]
fn reads_bounded_ascii_hex_filtered_xmp_metadata_through_binas() {
    let document = open(&filtered_metadata_pdf()).expect("filtered XMP fixture opens");

    assert_eq!(
        document
            .xmp_metadata()
            .expect("filtered XMP metadata reads")
            .expect("fixture has XMP")
            .xml,
        b"<?xpacket begin=''?><x:xmpmeta xmlns:x='adobe:ns:meta/'/>"
    );
}

#[test]
fn updates_document_info_and_reopens_through_binas() {
    let document = open(&metadata_pdf()).expect("fixture opens");
    let outcome = document
        .update_document_info(DocumentInfoUpdate {
            entries: BTreeMap::from([("Title".into(), Some("Updated cutover".into()))]),
        })
        .expect("Info update succeeds");
    assert!(outcome.verification.passed);
    assert!(outcome.verification.reparsed);
    assert!(outcome.verification.requested_state_matches);
    assert!(outcome.verification.unknown_entries_preserved);

    let reopened = open(&outcome.bytes).expect("updated PDF reopens");
    let info = reopened.metadata().expect("Info metadata reads");
    assert_eq!(info.title.as_deref(), Some("Updated cutover"));
    assert_eq!(info.author.as_deref(), Some("OxHQ"));
}

#[test]
fn updates_xmp_metadata_and_reopens_through_binas() {
    let document = open(&metadata_pdf()).expect("fixture opens");
    let xml =
        b"<?xpacket begin=''?><x:xmpmeta xmlns:x='adobe:ns:meta/'><updated/></x:xmpmeta>".to_vec();
    let outcome = document
        .update_xmp_metadata(XmpMetadataUpdate {
            xml: Some(xml.clone()),
        })
        .expect("XMP update succeeds");
    assert!(outcome.verification.passed);
    assert!(outcome.verification.reparsed);
    assert!(outcome.verification.requested_state_matches);
    assert!(outcome.verification.catalog_reachable);

    let reopened = open(&outcome.bytes).expect("updated PDF reopens");
    assert_eq!(
        reopened
            .xmp_metadata()
            .expect("XMP metadata reads")
            .expect("XMP remains present")
            .xml,
        xml
    );
}

#[test]
fn reads_page_labels_navigation_and_attachment_metadata_through_binas() {
    let document = open(&structure_pdf()).expect("fixture opens");

    assert_eq!(
        document
            .page_labels()
            .expect("page labels read")
            .into_iter()
            .map(|label| label.label)
            .collect::<Vec<_>>(),
        ["page-4", "page-5"]
    );
    assert_eq!(
        document
            .named_destinations()
            .expect("destinations read")
            .into_iter()
            .map(|destination| (destination.name, destination.page_index))
            .collect::<Vec<_>>(),
        [("chapter-one".into(), 0)]
    );
    assert_eq!(
        document
            .outlines()
            .expect("outlines read")
            .into_iter()
            .map(|item| (item.title, item.destination_name, item.depth))
            .collect::<Vec<_>>(),
        [("Chapter one".into(), Some("chapter-one".into()), 0)]
    );
    let attachments = document.embedded_attachments().expect("attachments read");
    assert_eq!(attachments.len(), 1);
    assert_eq!(attachments[0].name, "evidence.txt");
    assert_eq!(attachments[0].size, 4);
    assert_eq!(attachments[0].object_number, 9);
}

#[test]
fn updates_a_page_label_and_reopens_through_binas() {
    let document = open(&structure_pdf()).expect("fixture opens");
    let outcome = document
        .update_page_label(PageLabelUpdate {
            page_index: 1,
            spec: Some(PageLabelSpec {
                style: Some(PageLabelStyle::UpperRoman),
                prefix: "section-".into(),
                start: 3,
            }),
        })
        .expect("page-label update succeeds");
    assert!(outcome.verification.passed);
    assert!(outcome.verification.reparsed);
    assert!(outcome.verification.requested_state_matches);
    assert!(outcome.verification.catalog_reachable);

    let reopened = open(&outcome.bytes).expect("updated PDF reopens");
    assert_eq!(
        reopened
            .page_labels()
            .expect("page labels read")
            .into_iter()
            .map(|label| label.label)
            .collect::<Vec<_>>(),
        ["page-4", "section-III"]
    );
}

#[test]
fn updates_a_named_destination_and_reopens_through_binas() {
    let document = open(&structure_pdf()).expect("fixture opens");
    let outcome = document
        .update_named_destination(NamedDestinationUpdate {
            name: "chapter-one".into(),
            page_index: Some(1),
        })
        .expect("named-destination update succeeds");
    assert!(outcome.verification.passed);
    assert!(outcome.verification.reparsed);
    assert!(outcome.verification.requested_state_matches);
    assert!(outcome.verification.catalog_reachable);

    let reopened = open(&outcome.bytes).expect("updated PDF reopens");
    assert_eq!(
        reopened
            .named_destinations()
            .expect("destinations read")
            .into_iter()
            .map(|destination| (destination.name, destination.page_index))
            .collect::<Vec<_>>(),
        [("chapter-one".into(), 1)]
    );
}

#[test]
fn updates_an_embedded_attachment_and_reopens_through_binas() {
    let document = open(&structure_pdf()).expect("fixture opens");
    let data = b"replacement evidence".to_vec();
    let outcome = document
        .update_embedded_attachment(EmbeddedAttachmentUpdate {
            name: "evidence.txt".into(),
            data: Some(data.clone()),
        })
        .expect("embedded-attachment update succeeds");
    assert!(outcome.verification.passed);
    assert!(outcome.verification.reparsed);
    assert!(outcome.verification.requested_state_matches);
    assert!(outcome.verification.catalog_reachable);

    let reopened = open(&outcome.bytes).expect("updated PDF reopens");
    assert_eq!(
        reopened
            .embedded_attachments()
            .expect("attachments read")
            .into_iter()
            .map(|attachment| (attachment.name, attachment.size))
            .collect::<Vec<_>>(),
        [("evidence.txt".into(), data.len())]
    );
}

#[test]
fn creates_children_and_removes_an_outline_subtree_through_binas() {
    let document = open(&structure_pdf()).expect("fixture opens");
    let created = document
        .create_outline(OutlineCreateRequest {
            title: "Parent".into(),
            destination_name: "chapter-one".into(),
        })
        .expect("top-level outline creates");
    assert!(created.verification.passed);
    assert!(created.verification.reparsed);

    let document = open(&created.bytes).expect("created-outline PDF reopens");
    let parent_index = document
        .outlines()
        .expect("outlines read")
        .iter()
        .position(|item| item.title == "Parent")
        .expect("parent outline exists");
    let child = document
        .create_child_outline(
            parent_index,
            OutlineCreateRequest {
                title: "Child".into(),
                destination_name: "chapter-one".into(),
            },
        )
        .expect("child outline creates");
    assert!(child.verification.passed);
    assert!(child.verification.reparsed);

    let document = open(&child.bytes).expect("child-outline PDF reopens");
    let parent_index = document
        .outlines()
        .expect("outlines read")
        .iter()
        .position(|item| item.title == "Parent")
        .expect("parent outline remains");
    let removed = document
        .remove_outline(OutlineRemoveRequest {
            outline_index: parent_index,
        })
        .expect("outline subtree removes");
    assert!(removed.verification.passed);
    assert!(removed.verification.reparsed);
    assert!(removed.verification.catalog_reachable);

    let reopened = open(&removed.bytes).expect("removed-outline PDF reopens");
    assert_eq!(
        reopened
            .outlines()
            .expect("outlines read")
            .into_iter()
            .map(|item| (item.title, item.depth))
            .collect::<Vec<_>>(),
        [("Chapter one".into(), 0)]
    );
}

fn metadata_pdf() -> Vec<u8> {
    let xmp = b"<?xpacket begin=''?><x:xmpmeta xmlns:x='adobe:ns:meta/'/>";
    classic_pdf(
        &[
            b"<< /Type /Catalog /Pages 2 0 R /Metadata 6 0 R >>".to_vec(),
            b"<< /Type /Pages /Kids [3 0 R] /Count 1 >>".to_vec(),
            b"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] >>".to_vec(),
            b"<< >>".to_vec(),
            b"<< /Title (Rust cutover) /Author (OxHQ) >>".to_vec(),
            [
                format!(
                    "<< /Type /Metadata /Subtype /XML /Length {} >>\nstream\n",
                    xmp.len()
                )
                .into_bytes(),
                xmp.to_vec(),
                b"\nendstream".to_vec(),
            ]
            .concat(),
        ],
        " /Info 5 0 R",
    )
}

fn filtered_metadata_pdf() -> Vec<u8> {
    let xmp = b"<?xpacket begin=''?><x:xmpmeta xmlns:x='adobe:ns:meta/'/>";
    let encoded = ascii_hex(xmp);
    classic_pdf(
        &[
            b"<< /Type /Catalog /Pages 2 0 R /Metadata 4 0 R >>".to_vec(),
            b"<< /Type /Pages /Kids [3 0 R] /Count 1 >>".to_vec(),
            b"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] >>".to_vec(),
            [
                format!(
                    "<< /Type /Metadata /Subtype /XML /Filter /ASCIIHexDecode /Length {} >>\nstream\n",
                    encoded.len()
                )
                .into_bytes(),
                encoded,
                b"\nendstream".to_vec(),
            ]
            .concat(),
        ],
        "",
    )
}

fn ascii_hex(bytes: &[u8]) -> Vec<u8> {
    const HEX: &[u8; 16] = b"0123456789ABCDEF";
    let mut encoded = Vec::with_capacity(bytes.len() * 2 + 1);
    for &byte in bytes {
        encoded.push(HEX[usize::from(byte >> 4)]);
        encoded.push(HEX[usize::from(byte & 15)]);
    }
    encoded.push(b'>');
    encoded
}

fn structure_pdf() -> Vec<u8> {
    classic_pdf(
        &[
            b"<< /Type /Catalog /Pages 2 0 R /PageLabels 5 0 R /Names 6 0 R /Outlines 10 0 R >>"
                .to_vec(),
            b"<< /Type /Pages /Kids [3 0 R 4 0 R] /Count 2 >>".to_vec(),
            b"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] >>".to_vec(),
            b"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] >>".to_vec(),
            b"<< /Nums [0 << /S /D /P (page-) /St 4 >>] >>".to_vec(),
            b"<< /Dests 7 0 R /EmbeddedFiles 8 0 R >>".to_vec(),
            b"<< /Names [(chapter-one) [3 0 R /Fit]] >>".to_vec(),
            b"<< /Names [(evidence.txt) 9 0 R] >>".to_vec(),
            b"<< /Type /Filespec /F (evidence.txt) /EF << /F 12 0 R >> >>".to_vec(),
            b"<< /Type /Outlines /First 11 0 R /Last 11 0 R /Count 1 >>".to_vec(),
            b"<< /Title (Chapter one) /Parent 10 0 R /Dest /chapter-one >>".to_vec(),
            b"<< /Type /EmbeddedFile /Length 4 >>\nstream\nDATA\nendstream".to_vec(),
        ],
        "",
    )
}

fn classic_pdf(objects: &[Vec<u8>], trailer_extra: &str) -> Vec<u8> {
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
            "trailer\n<< /Size {} /Root 1 0 R{trailer_extra} >>\nstartxref\n{xref}\n%%EOF\n",
            objects.len() + 1
        )
        .as_bytes(),
    );
    bytes
}
