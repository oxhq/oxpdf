package oxpdf

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
)

func TestPlanOCRTextLayerReportsCallerProvidedBoundary(t *testing.T) {
	doc, err := OpenBytes(ocrTextLayerTestPDF())
	if err != nil {
		t.Fatal(err)
	}

	plan, err := doc.PlanOCRTextLayer(OCRTextLayerInput{
		PageIndex:  0,
		Text:       "External OCR text",
		Box:        OCRTextLayerBox{XMin: 10, YMin: 20, XMax: 110, YMax: 45},
		Confidence: 0.82,
	})
	if err != nil {
		t.Fatal(err)
	}

	if plan.PageIndex != 0 || plan.Text != "External OCR text" || plan.Confidence != 0.82 {
		t.Fatalf("plan = %+v, want caller-provided OCR metadata", plan)
	}
	if !plan.PlannedOnly || !plan.FallbackUsed || plan.FallbackKind != "ocr_text_layer" || plan.FallbackMode != "explicit" {
		t.Fatalf("plan fallback = %+v, want planned explicit OCR text-layer fallback", plan)
	}
	if plan.Boundary.TextSource != "caller-provided" || plan.Boundary.RunsOCR || plan.Boundary.RequiresExternalBinary || plan.Boundary.RequiresExternalService {
		t.Fatalf("plan boundary = %+v, want caller-provided text with no OCR binary/service run by OxPDF", plan.Boundary)
	}
}

func TestEmbedOCRTextLayerMakesCallerProvidedTextSelectable(t *testing.T) {
	doc, err := OpenBytes(ocrTextLayerTestPDF())
	if err != nil {
		t.Fatal(err)
	}

	output, verification, err := doc.EmbedOCRTextLayer(OCRTextLayerInput{
		PageIndex:  0,
		Text:       "External OCR text",
		Box:        OCRTextLayerBox{XMin: 10, YMin: 20, XMax: 110, YMax: 45},
		Confidence: 0.82,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !verification.ReparseOK || !verification.NewSelectable || !verification.PageUnchanged {
		t.Fatalf("verification = %+v, want reparsed selectable text layer with unchanged page count", verification)
	}

	reopened, err := OpenBytes(output)
	if err != nil {
		t.Fatal(err)
	}
	matches, err := reopened.FindText("External OCR text")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("FindText(embedded OCR text) len = %d, want 1", len(matches))
	}
}

func TestOCRTextLayerInvalidInputFailsClosed(t *testing.T) {
	doc, err := OpenBytes(ocrTextLayerTestPDF())
	if err != nil {
		t.Fatal(err)
	}

	_, err = doc.PlanOCRTextLayer(OCRTextLayerInput{
		PageIndex:  0,
		Text:       "",
		Box:        OCRTextLayerBox{XMin: 10, YMin: 20, XMax: 110, YMax: 45},
		Confidence: 0.82,
	})
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("PlanOCRTextLayer(empty text) error = %v, want ErrUnsupported", err)
	}
}

func ocrTextLayerTestPDF() []byte {
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /Resources << >> >>",
	}
	var body bytes.Buffer
	offsets := []int{0}
	for i, object := range objects {
		offsets = append(offsets, body.Len())
		fmt.Fprintf(&body, "%d 0 obj\n%s\nendobj\n", i+1, object)
	}
	var out bytes.Buffer
	out.WriteString("%PDF-1.7\n")
	base := out.Len()
	out.Write(body.Bytes())
	xrefAt := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n", len(offsets))
	out.WriteString("0000000000 65535 f \n")
	for i := 1; i < len(offsets); i++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", base+offsets[i])
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets), xrefAt)
	return out.Bytes()
}
