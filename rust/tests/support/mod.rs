pub fn fixture_pdf(pages: &[&str], metadata: Option<(&str, &str)>) -> Vec<u8> {
    assert!(!pages.is_empty());

    let first_page = 3;
    let first_content = first_page + pages.len();
    let mut objects = vec![
        b"<< /Type /Catalog /Pages 2 0 R >>".to_vec(),
        format!(
            "<< /Type /Pages /Kids [{}] /Count {} >>",
            (0..pages.len())
                .map(|index| format!("{} 0 R", first_page + index))
                .collect::<Vec<_>>()
                .join(" "),
            pages.len()
        )
        .into_bytes(),
    ];
    for index in 0..pages.len() {
        objects.push(
            format!(
                "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] /Contents {} 0 R >>",
                first_content + index
            )
            .into_bytes(),
        );
    }
    for text in pages {
        objects.push(stream(format!("BT ({text}) Tj ET").as_bytes()));
    }
    let info = metadata.map(|(title, author)| {
        objects.push(format!("<< /Title ({title}) /Author ({author}) >>").into_bytes());
        format!(" /Info {} 0 R", objects.len())
    });

    let mut bytes = b"%PDF-1.7\n".to_vec();
    let mut offsets = vec![0];
    for (index, object) in objects.iter().enumerate() {
        offsets.push(bytes.len());
        bytes.extend_from_slice(format!("{} 0 obj\n", index + 1).as_bytes());
        bytes.extend_from_slice(object);
        bytes.extend_from_slice(b"\nendobj\n");
    }
    let xref = bytes.len();
    bytes.extend_from_slice(format!("xref\n0 {}\n0000000000 65535 f \n", offsets.len()).as_bytes());
    for offset in offsets.into_iter().skip(1) {
        bytes.extend_from_slice(format!("{offset:010} 00000 n \n").as_bytes());
    }
    bytes.extend_from_slice(
        format!(
            "trailer\n<< /Size {} /Root 1 0 R{} >>\nstartxref\n{xref}\n%%EOF\n",
            objects.len() + 1,
            info.unwrap_or_default()
        )
        .as_bytes(),
    );
    bytes
}

fn stream(contents: &[u8]) -> Vec<u8> {
    [
        format!("<< /Length {} >>\nstream\n", contents.len()).into_bytes(),
        contents.to_vec(),
        b"\nendstream".to_vec(),
    ]
    .concat()
}
