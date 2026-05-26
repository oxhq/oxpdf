package oxpdf

import (
	"bytes"
	"errors"
	"fmt"
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

func TestFindTextMapsPageAndExtractsTextBeyondFirstPage(t *testing.T) {
	doc, err := OpenBytes(multiPageTextPDF("First page", "Second page"))
	if err != nil {
		t.Fatal(err)
	}
	matches, err := doc.FindText("Second page")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("FindText() len = %d, want 1", len(matches))
	}
	if matches[0].Page != 1 {
		t.Fatalf("FindText() Page = %d, want 1", matches[0].Page)
	}
	page, err := doc.Page(1)
	if err != nil {
		t.Fatal(err)
	}
	text, err := page.ExtractText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "Second page" {
		t.Fatalf("Page(1).ExtractText() = %q, want %q", text, "Second page")
	}
	first, err := doc.Page(0)
	if err != nil {
		t.Fatal(err)
	}
	text, err = first.ExtractText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "First page" {
		t.Fatalf("Page(0).ExtractText() = %q, want %q", text, "First page")
	}
}

func TestExtractTextRunsPreservesOrderAndPosition(t *testing.T) {
	content := "BT\n/F1 12 Tf\n72 720 Td\n(Top) Tj\n18 TL\nT*\n(Bottom) Tj\n1 0 0 1 144 700 Tm\n(Moved) Tj\nET"
	doc, err := OpenBytes(contentPageTextPDF(content))
	if err != nil {
		t.Fatal(err)
	}
	page, err := doc.Page(0)
	if err != nil {
		t.Fatal(err)
	}

	runs, err := page.ExtractTextRuns()
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 3 {
		t.Fatalf("ExtractTextRuns() len = %d, want 3: %+v", len(runs), runs)
	}
	want := []TextRun{
		{Page: 0, Text: "Top", X: 72, Y: 720, Font: "F1", FontSize: 12, Operator: "Tj"},
		{Page: 0, Text: "Bottom", X: 72, Y: 702, Font: "F1", FontSize: 12, Operator: "Tj"},
		{Page: 0, Text: "Moved", X: 144, Y: 700, Font: "F1", FontSize: 12, Operator: "Tj"},
	}
	for i := range want {
		assertTextRun(t, runs[i], want[i])
	}
}

func TestExtractTextRunsFiltersToRequestedPage(t *testing.T) {
	doc, err := OpenBytes(multiPageContentTextPDF(
		"BT\n/F1 12 Tf\n72 720 Td\n(First) Tj\nET",
		"BT\n/F1 9 Tf\n30 40 Td\n(Second A) Tj\n12 TL\nT*\n(Second B) Tj\nET",
	))
	if err != nil {
		t.Fatal(err)
	}
	page, err := doc.Page(1)
	if err != nil {
		t.Fatal(err)
	}

	runs, err := page.ExtractTextRuns()
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 2 {
		t.Fatalf("ExtractTextRuns() len = %d, want 2: %+v", len(runs), runs)
	}
	assertTextRun(t, runs[0], TextRun{Page: 1, Text: "Second A", X: 30, Y: 40, Font: "F1", FontSize: 9, Operator: "Tj"})
	assertTextRun(t, runs[1], TextRun{Page: 1, Text: "Second B", X: 30, Y: 28, Font: "F1", FontSize: 9, Operator: "Tj"})
}

func TestReplaceTextOccurrenceSelectsExactMatchIndex(t *testing.T) {
	doc, err := OpenBytes(multiPageTextPDF("Repeated", "Repeated"))
	if err != nil {
		t.Fatal(err)
	}
	out, err := doc.ReplaceTextOccurrence("Repeated", "Changed", 1)
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenBytes(out)
	if err != nil {
		t.Fatal(err)
	}
	oldMatches, err := reopened.FindText("Repeated")
	if err != nil {
		t.Fatal(err)
	}
	if len(oldMatches) != 1 || oldMatches[0].Page != 0 {
		t.Fatalf("FindText(old) = %+v, want one match on page 0", oldMatches)
	}
	newMatches, err := reopened.FindText("Changed")
	if err != nil {
		t.Fatal(err)
	}
	if len(newMatches) != 1 || newMatches[0].Page != 1 {
		t.Fatalf("FindText(new) = %+v, want one match on page 1", newMatches)
	}
}

func TestReplaceTextOccurrenceInvalidIndexesFailClosed(t *testing.T) {
	doc, err := OpenBytes(textPDF("Only once"))
	if err != nil {
		t.Fatal(err)
	}
	for _, index := range []int{-1, 1} {
		if _, err := doc.ReplaceTextOccurrence("Only once", "Changed", index); !errors.Is(err, ErrUnsupported) {
			t.Fatalf("ReplaceTextOccurrence(index %d) error = %v, want ErrUnsupported", index, err)
		}
	}
}

func TestReplaceTextOccurrenceRejectsRemovalShape(t *testing.T) {
	doc, err := OpenBytes(textPDF("Only once"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := doc.ReplaceTextOccurrence("Only once", "", 0); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("ReplaceTextOccurrence(empty replacement) error = %v, want ErrUnsupported", err)
	}
}

func TestReplaceTextWithOptionsPreservesObjectStreams(t *testing.T) {
	input := objectStreamTextPDF()
	doc, err := OpenBytes(input)
	if err != nil {
		t.Fatal(err)
	}

	canonical, err := doc.ReplaceText("08-15-2024", "05-20-2026")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(canonical, []byte("/ObjStm")) {
		t.Fatalf("canonical ReplaceText preserved object stream container:\n%s", canonical)
	}

	doc, err = OpenBytes(input)
	if err != nil {
		t.Fatal(err)
	}
	preserved, err := doc.ReplaceTextWithOptions("08-15-2024", "05-20-2026", TextEditOptions{
		Rewrite: TextRewritePreserveStructure,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(preserved, []byte("/Type /ObjStm")) {
		t.Fatalf("preserve-structure ReplaceText lost object stream container:\n%s", preserved)
	}
	reopened, err := OpenBytes(preserved)
	if err != nil {
		t.Fatal(err)
	}
	xref := reopened.Xref()
	if !xref.HasObjectStream || xref.ObjectStreamCount != 1 {
		t.Fatalf("Xref() = %+v, want preserved object stream metadata", xref)
	}
	oldMatches, err := reopened.FindText("08-15-2024")
	if err != nil {
		t.Fatal(err)
	}
	if len(oldMatches) != 0 {
		t.Fatalf("FindText(old) = %+v, want none", oldMatches)
	}
	newMatches, err := reopened.FindText("05-20-2026")
	if err != nil {
		t.Fatal(err)
	}
	if len(newMatches) != 1 {
		t.Fatalf("FindText(new) len = %d, want 1", len(newMatches))
	}
}

func TestReplaceTextWithOptionsRejectsUnknownRewriteMode(t *testing.T) {
	doc, err := OpenBytes(textPDF("Only once"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := doc.ReplaceTextWithOptions("Only once", "Changed", TextEditOptions{Rewrite: "unknown"}); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("ReplaceTextWithOptions(unknown rewrite) error = %v, want ErrUnsupported", err)
	}
}

func TestRemoveTextFailsClosed(t *testing.T) {
	doc, err := OpenBytes(textPDF("Remove me"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := doc.RemoveText("Remove me"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("RemoveText() error = %v, want ErrUnsupported", err)
	}
}

func TestLocalPageTextFailsClosedForContentsArrays(t *testing.T) {
	input := textPDF("Array contents")
	input = bytes.Replace(input, []byte("/Contents 4 0 R"), []byte("/Contents [4 0 R]"), 1)
	_, err := localPageText(input, 0)
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("localPageText() error = %v, want ErrUnsupported", err)
	}
}

func textPDF(text string) []byte {
	return multiPageTextPDF(text)
}

func multiPageTextPDF(texts ...string) []byte {
	contents := make([]string, 0, len(texts))
	for _, text := range texts {
		contents = append(contents, "BT /F1 12 Tf 72 720 Td ("+text+") Tj ET")
	}
	return multiPageContentTextPDF(contents...)
}

func contentPageTextPDF(content string) []byte {
	return multiPageContentTextPDF(content)
}

func multiPageContentTextPDF(pageContents ...string) []byte {
	pages := make([]string, 0, len(pageContents))
	contents := make([]string, 0, len(pageContents))
	kids := ""
	fontObjectNumber := 3 + len(pageContents)*2
	for i, content := range pageContents {
		pageObjectNumber := 3 + i
		contentObjectNumber := 3 + len(pageContents) + i
		kids += itoa(pageObjectNumber) + " 0 R "
		pages = append(pages, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 "+itoa(fontObjectNumber)+" 0 R >> >> /Contents "+itoa(contentObjectNumber)+" 0 R >>")
		contents = append(contents, "<< /Length "+itoa(len(content))+" >>\nstream\n"+content+"\nendstream")
	}
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [" + kids + "] /Count " + itoa(len(pageContents)) + " >>",
	}
	objects = append(objects, pages...)
	objects = append(objects, contents...)
	objects = append(objects, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")
	return pdfObjects(objects...)
}

func assertTextRun(t *testing.T, got, want TextRun) {
	t.Helper()
	if got.Page != want.Page ||
		got.Text != want.Text ||
		got.X != want.X ||
		got.Y != want.Y ||
		got.Font != want.Font ||
		got.FontSize != want.FontSize ||
		got.Operator != want.Operator {
		t.Fatalf("TextRun = %+v, want %+v", got, want)
	}
}

func objectStreamTextPDF() []byte {
	content := []byte("BT\n(08\\05515\\0552024) Tj\nET\n")
	objectStreamData := "5 0 << /Fixture true >>"
	return pdfObjects(
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Page /Contents 3 0 R >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(content), content),
		fmt.Sprintf("<< /Type /ObjStm /N 1 /First 4 /Length %d >>\nstream\n%sendstream", len(objectStreamData), objectStreamData),
	)
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
