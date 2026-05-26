package oxpdf

import (
	"fmt"

	binaspdf "github.com/oxhq/binas/pkg/adapters/pdf"
	"github.com/oxhq/binas/pkg/core"
)

const (
	// OCRTextLayerTextSourceCallerProvided means OxPDF embeds text supplied by
	// the caller and does not run OCR itself.
	OCRTextLayerTextSourceCallerProvided = "caller-provided"
)

// OCRTextLayerBox identifies where caller-provided OCR text should be placed.
type OCRTextLayerBox struct {
	XMin float64
	YMin float64
	XMax float64
	YMax float64
}

// OCRTextLayerInput supplies an explicit OCR text-layer item.
//
// OxPDF does not run OCR, call external binaries, or call external services.
// Text must come from the caller's OCR pipeline.
type OCRTextLayerInput struct {
	PageIndex  int
	Text       string
	Box        OCRTextLayerBox
	Confidence float64
}

// OCRTextLayerBoundary describes the dependency boundary for OCR text-layer
// operations.
type OCRTextLayerBoundary struct {
	TextSource              string
	RunsOCR                 bool
	RequiresExternalBinary  bool
	RequiresExternalService bool
}

// OCRTextLayerPlan reports a validated caller-provided OCR text-layer item
// without rewriting the PDF.
type OCRTextLayerPlan struct {
	PageIndex    int
	Text         string
	Box          OCRTextLayerBox
	Confidence   float64
	PlannedOnly  bool
	FallbackUsed bool
	FallbackKind string
	FallbackMode string
	Boundary     OCRTextLayerBoundary
}

// OCRTextLayerVerification reports verification for an embedded OCR text layer.
type OCRTextLayerVerification struct {
	ReparseOK     bool
	NewSelectable bool
	PageUnchanged bool
}

// PlanOCRTextLayer validates a caller-provided OCR text layer without rewriting
// the document.
//
// This is an explicit text-layer fallback. It does not recognize text from page
// images, invoke OCR binaries, or call OCR services.
func (d *Document) PlanOCRTextLayer(input OCRTextLayerInput) (OCRTextLayerPlan, error) {
	if d == nil {
		return OCRTextLayerPlan{}, unsupported("missing document")
	}
	plan, err := binaspdf.PlanExplicitOCRTextLayer(d.input, mapOCRTextLayerInput(input))
	if err != nil {
		return OCRTextLayerPlan{}, unsupported(err.Error())
	}
	return mapOCRTextLayerPlan(plan), nil
}

// EmbedOCRTextLayer embeds caller-provided OCR text as a selectable invisible
// text layer and returns rewritten PDF bytes.
//
// This is an explicit text-layer fallback. It does not recognize text from page
// images, invoke OCR binaries, or call OCR services.
func (d *Document) EmbedOCRTextLayer(input OCRTextLayerInput) ([]byte, OCRTextLayerVerification, error) {
	if d == nil {
		return nil, OCRTextLayerVerification{}, unsupported("missing document")
	}
	out, _, verification, err := binaspdf.ApplyExplicitOCRTextLayer(d.input, mapOCRTextLayerInput(input))
	if err != nil {
		return nil, OCRTextLayerVerification{}, unsupported(err.Error())
	}
	if !verification.ReparseOK || !verification.NewSelectable || !verification.PageUnchanged {
		return nil, OCRTextLayerVerification{}, unsupported(fmt.Sprintf("OCR text-layer embed did not satisfy verification: %+v", verification))
	}
	return out, mapOCRTextLayerVerification(verification), nil
}

func mapOCRTextLayerInput(input OCRTextLayerInput) binaspdf.OCRTextLayerOptions {
	return binaspdf.OCRTextLayerOptions{
		PageIndex: input.PageIndex,
		Text:      input.Text,
		Box: binaspdf.OCRTextLayerBox{
			XMin: input.Box.XMin,
			YMin: input.Box.YMin,
			XMax: input.Box.XMax,
			YMax: input.Box.YMax,
		},
		Confidence: input.Confidence,
	}
}

func mapOCRTextLayerPlan(plan binaspdf.OCRTextLayerPlan) OCRTextLayerPlan {
	return OCRTextLayerPlan{
		PageIndex:   plan.PageIndex,
		Text:        plan.Text,
		Box:         mapOCRTextLayerBox(plan.Box),
		Confidence:  plan.Confidence,
		PlannedOnly: boolAnyValue(plan.Report.Meta, "planned_only"),
		FallbackUsed: plan.Report.FallbackUsed ||
			(plan.Policy.Fallback == "ocr_text_layer" && plan.Policy.Mode == "explicit"),
		FallbackKind: plan.Policy.Fallback,
		FallbackMode: plan.Policy.Mode,
		Boundary:     callerProvidedOCRTextLayerBoundary(),
	}
}

func mapOCRTextLayerBox(box binaspdf.OCRTextLayerBox) OCRTextLayerBox {
	return OCRTextLayerBox{
		XMin: box.XMin,
		YMin: box.YMin,
		XMax: box.XMax,
		YMax: box.YMax,
	}
}

func mapOCRTextLayerVerification(verification core.Verification) OCRTextLayerVerification {
	return OCRTextLayerVerification{
		ReparseOK:     verification.ReparseOK,
		NewSelectable: verification.NewSelectable,
		PageUnchanged: verification.PageUnchanged,
	}
}

func callerProvidedOCRTextLayerBoundary() OCRTextLayerBoundary {
	return OCRTextLayerBoundary{
		TextSource:              OCRTextLayerTextSourceCallerProvided,
		RunsOCR:                 false,
		RequiresExternalBinary:  false,
		RequiresExternalService: false,
	}
}

func boolAnyValue(values map[string]any, key string) bool {
	if values == nil {
		return false
	}
	value, _ := values[key].(bool)
	return value
}
