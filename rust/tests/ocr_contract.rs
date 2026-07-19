use oxpdf::{OcrParseLimits, open, parse_alto_xml, parse_ocr_json};

mod support;

#[test]
fn parses_caller_boxes_applies_an_ocr_layer_and_reopens_through_binas() {
    let limits = OcrParseLimits::default();
    let request = parse_ocr_json(
        br#"{
            "page_index": 0,
            "source_width": 100,
            "source_height": 100,
            "boxes": [{
                "text": "Selectable OCR",
                "x": 10,
                "y": 20,
                "width": 50,
                "height": 10
            }]
        }"#,
        limits,
    )
    .expect("OCR JSON parses");
    let alto = parse_alto_xml(
        br#"<alto><Layout><Page WIDTH="100" HEIGHT="100"><PrintSpace><TextBlock><TextLine><String CONTENT="Selectable OCR" HPOS="10" VPOS="20" WIDTH="50" HEIGHT="10"/></TextLine></TextBlock></PrintSpace></Page></Layout></alto>"#,
        limits,
    )
    .expect("ALTO parses");
    assert_eq!(alto.as_slice(), std::slice::from_ref(&request));

    let document =
        open(&support::fixture_pdf(&["VISIBLE"], None)).expect("fixture opens through binas-pdf");
    let plan = document
        .plan_ocr_text_layer(request)
        .expect("OCR layer plans through binas-pdf");
    assert_eq!(plan.boxes.len(), 1);

    let outcome = document
        .apply_ocr_text_layer(&plan)
        .expect("OCR layer applies through binas-pdf");
    assert!(outcome.verification.passed);
    assert!(outcome.verification.page_count_unchanged);
    assert!(outcome.verification.text_selectable);
    assert!(outcome.verification.no_dangling_references);

    let reopened = open(&outcome.bytes).expect("OCR output reopens through binas-pdf");
    assert_eq!(
        reopened.inspect().expect("inspection succeeds").page_count,
        1
    );
    assert_eq!(
        reopened
            .query_text_all("Selectable OCR")
            .expect("OCR text is selectable")
            .len(),
        1
    );
}
