use oxpdf::{BlankPageSize, create_blank_pdf, open};

#[test]
fn creates_and_opens_two_blank_pages() {
    let bytes = create_blank_pdf(&[
        BlankPageSize {
            width: 595.0,
            height: 842.0,
        },
        BlankPageSize {
            width: 612.0,
            height: 792.0,
        },
    ])
    .expect("blank PDF creates through binas-pdf");

    let document = open(&bytes).expect("blank PDF opens through binas-pdf");
    assert_eq!(
        document.inspect().expect("blank PDF inspects").page_count,
        2
    );
    assert_eq!(document.pages().expect("blank pages enumerate").len(), 2);
}
