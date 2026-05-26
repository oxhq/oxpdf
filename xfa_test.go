package oxpdf

import (
	"errors"
	"strings"
	"testing"
)

func TestXFAPacketsAndDatasetFields(t *testing.T) {
	doc, err := OpenBytes(staticXFAPDF())
	if err != nil {
		t.Fatal(err)
	}

	packets, err := doc.XFAPackets()
	if err != nil {
		t.Fatal(err)
	}
	if len(packets) != 2 {
		t.Fatalf("XFAPackets() len = %d, want 2", len(packets))
	}
	if packets[0].Label != "template" || packets[0].Kind != "template" || packets[0].RootElement != "template" {
		t.Fatalf("template packet = %+v", packets[0])
	}
	if packets[1].Label != "datasets" || packets[1].Kind != "datasets" || packets[1].RootElement != "xfa:datasets" {
		t.Fatalf("datasets packet = %+v", packets[1])
	}

	fields, err := doc.XFADatasetFields()
	if err != nil {
		t.Fatal(err)
	}
	wantFields := []XFADatasetField{
		{Path: "form1.payer.name", Value: "David", PacketIndex: 1, Label: "datasets"},
		{Path: "form1.payer.email", Value: "david@example.test", PacketIndex: 1, Label: "datasets"},
	}
	if !sameXFADatasetFields(fields, wantFields) {
		t.Fatalf("XFADatasetFields() = %+v, want %+v", fields, wantFields)
	}

	mappings, err := doc.XFATemplateDatasetMappings()
	if err != nil {
		t.Fatal(err)
	}
	if len(mappings) != 2 {
		t.Fatalf("XFATemplateDatasetMappings() len = %d, want 2: %+v", len(mappings), mappings)
	}
	if mappings[0].FieldName != "name" || mappings[0].DatasetPath != "form1.payer.name" || mappings[0].Value != "David" {
		t.Fatalf("mapping 0 = %+v", mappings[0])
	}
}

func TestXFASemanticsAndDatasetFieldUpdate(t *testing.T) {
	doc, err := OpenBytes(staticXFAPDF())
	if err != nil {
		t.Fatal(err)
	}

	semantics, err := doc.XFASemantics()
	if err != nil {
		t.Fatal(err)
	}
	if semantics.Classification != "static_datasets_template" || !semantics.DatasetSemanticEditsSupported || semantics.RequiresRendering {
		t.Fatalf("XFASemantics() = %+v", semantics)
	}

	out, verification, err := doc.SetXFADatasetField("form1.payer.name", "Ana & Co")
	if err != nil {
		t.Fatal(err)
	}
	if !verification.ReparseOK || !verification.OldTextRemoved || !verification.NewSelectable {
		t.Fatalf("verification = %+v", verification)
	}
	updated, err := OpenBytes(out)
	if err != nil {
		t.Fatal(err)
	}
	fields, err := updated.XFADatasetFields()
	if err != nil {
		t.Fatal(err)
	}
	assertXFADatasetFieldValue(t, fields, "form1.payer.name", "Ana & Co")
	if strings.Contains(string(out), "Ana & Co<") {
		t.Fatalf("raw XFA output did not XML-escape replacement value")
	}
	if !strings.Contains(string(out), "Ana &amp; Co") {
		t.Fatalf("raw XFA output does not contain escaped replacement value")
	}
}

func TestXFADatasetFieldUpdateRejectsAmbiguousAndDynamicXFA(t *testing.T) {
	doc, err := OpenBytes(ambiguousXFAPDF())
	if err != nil {
		t.Fatal(err)
	}
	fields, err := doc.XFADatasetFields()
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 2 || fields[0].Path != "form.field" || fields[1].Path != "form.field" {
		t.Fatalf("ambiguous XFADatasetFields() = %+v, want duplicate form.field paths", fields)
	}
	if _, _, err := doc.SetXFADatasetField("form.field", "new"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("SetXFADatasetField(ambiguous) error = %v, want ErrUnsupported", err)
	} else if !strings.Contains(err.Error(), "ambiguous") || !strings.Contains(err.Error(), "2 matches") {
		t.Fatalf("SetXFADatasetField(ambiguous) error = %v, want explicit ambiguous match count", err)
	}

	dynamic, err := OpenBytes(dynamicXFAPDF())
	if err != nil {
		t.Fatal(err)
	}
	semantics, err := dynamic.XFASemantics()
	if err != nil {
		t.Fatal(err)
	}
	if !semantics.RequiresRendering || semantics.DatasetSemanticEditsSupported || semantics.RefusalReason == "" {
		t.Fatalf("dynamic XFASemantics() = %+v", semantics)
	}
	if _, _, err := dynamic.SetXFADatasetField("name", "Ana"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("SetXFADatasetField(dynamic) error = %v, want ErrUnsupported", err)
	}
}

func assertXFADatasetFieldValue(t *testing.T, fields []XFADatasetField, path string, want string) {
	t.Helper()
	for _, field := range fields {
		if field.Path == path {
			if field.Value != want {
				t.Fatalf("XFA field %q value = %q, want %q", path, field.Value, want)
			}
			return
		}
	}
	t.Fatalf("XFA field %q not found in %+v", path, fields)
}

func sameXFADatasetFields(a, b []XFADatasetField) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func staticXFAPDF() []byte {
	return xfaPDF(`[(template) (<template><subform name="form1"><subform name="payer"><field name="name"/><field name="email"/></subform></subform></template>) (datasets) (<xfa:datasets xmlns:xfa="http://www.xfa.org/schema/xfa-data/1.0/"><xfa:data><form1><payer><name>David</name><email>david@example.test</email></payer></form1></xfa:data></xfa:datasets>)]`)
}

func ambiguousXFAPDF() []byte {
	return xfaPDF(`[(datasets) (<datasets><data><form><field>one</field><field>two</field></form></data></datasets>)]`)
}

func dynamicXFAPDF() []byte {
	return xfaPDF(`[(template) (<template><subform layout="flowed"><field name="name"/></subform></template>) (datasets) (<datasets><data><name>Alice</name></data></datasets>)]`)
}

func xfaPDF(xfa string) []byte {
	pages := []PageSize{PageSizeLetter}
	input := blankPDF(pages)
	oldRoot := "<< /Type /Catalog /Pages 2 0 R >>"
	newRoot := "<< /Type /Catalog /Pages 2 0 R /AcroForm << /XFA " + xfa + " >> >>"
	return []byte(strings.Replace(string(input), oldRoot, newRoot, 1))
}
