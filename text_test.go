package oxpdf

import (
	"bytes"
	"testing"
)

func TestFindExtractAndReplaceSelectableText(t *testing.T) {
	input := textPDF("Invoice 1234")
	doc, err := OpenBytes(input)
	if err != nil {
		t.Fatal(err)
	}
	matches, err := doc.FindText("Invoice 1234")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("FindText() len = %d, want 1", len(matches))
	}
	page, err := doc.Page(0)
	if err != nil {
		t.Fatal(err)
	}
	text, err := page.ExtractText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "Invoice 1234" {
		t.Fatalf("ExtractText() = %q", text)
	}
	out, err := doc.ReplaceText("Invoice 1234", "Invoice 5678")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out, []byte("Invoice 1234")) {
		t.Fatal("old text still present after ReplaceText")
	}
	reopened, err := OpenBytes(out)
	if err != nil {
		t.Fatal(err)
	}
	replaced, err := reopened.FindText("Invoice 5678")
	if err != nil {
		t.Fatal(err)
	}
	if len(replaced) != 1 {
		t.Fatalf("FindText(replacement) len = %d, want 1", len(replaced))
	}
}

func textPDF(text string) []byte {
	content := "BT /F1 12 Tf 72 720 Td (" + text + ") Tj ET"
	var objects []string
	objects = append(objects,
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 5 0 R >> >> /Contents 4 0 R >>",
		"<< /Length "+itoa(len(content))+" >>\nstream\n"+content+"\nendstream",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	)
	return pdfObjects(objects...)
}

func pdfObjects(objects ...string) []byte {
	var body bytes.Buffer
	offsets := []int{0}
	for i, object := range objects {
		offsets = append(offsets, body.Len())
		body.WriteString(itoa(i+1) + " 0 obj\n" + object + "\nendobj\n")
	}
	var out bytes.Buffer
	out.WriteString("%PDF-1.7\n")
	base := out.Len()
	out.Write(body.Bytes())
	xrefAt := out.Len()
	out.WriteString("xref\n0 " + itoa(len(offsets)) + "\n")
	out.WriteString("0000000000 65535 f \n")
	for i := 1; i < len(offsets); i++ {
		out.WriteString(pad10(base+offsets[i]) + " 00000 n \n")
	}
	out.WriteString("trailer\n<< /Size " + itoa(len(offsets)) + " /Root 1 0 R >>\nstartxref\n" + itoa(xrefAt) + "\n%%EOF\n")
	return out.Bytes()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func pad10(n int) string {
	s := itoa(n)
	for len(s) < 10 {
		s = "0" + s
	}
	return s
}
