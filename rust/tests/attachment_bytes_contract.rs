use oxpdf::{BlankPageSize, EmbeddedAttachmentUpdate, create_blank_pdf, open};

#[test]
fn reads_exact_attachment_inventory_bytes_and_rejects_a_stale_entry() {
    let data = b"bounded attachment bytes".to_vec();
    let blank = create_blank_pdf(&[BlankPageSize {
        width: 100.0,
        height: 100.0,
    }])
    .expect("blank fixture creates");
    let added = open(&blank)
        .expect("blank fixture opens")
        .update_embedded_attachment(EmbeddedAttachmentUpdate {
            name: "evidence.txt".into(),
            data: Some(data.clone()),
        })
        .expect("attachment adds");
    let document = open(&added.bytes).expect("attachment output opens");
    let attachment = document
        .embedded_attachments()
        .expect("attachment inventory reads")
        .remove(0);

    assert_eq!(
        document
            .read_embedded_attachment_bytes(&attachment)
            .expect("exact inventory entry reads"),
        data
    );

    let mut stale = attachment;
    stale.name = "missing.txt".into();
    assert_eq!(
        document
            .read_embedded_attachment_bytes(&stale)
            .expect_err("stale inventory entry is rejected")
            .code
            .as_str(),
        "selection_not_found"
    );
}
