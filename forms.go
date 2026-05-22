package oxpdf

// Field describes a PDF form field.
type Field struct {
	Name   string
	Type   string
	Value  string
	Status string
}

// Fields lists AcroForm fields when the backing engine can expose them.
func (d *Document) Fields() ([]Field, error) {
	return nil, unsupported("form field listing awaits stable binas form metadata")
}

// TextFields lists text fields only.
func (d *Document) TextFields() ([]Field, error) {
	return nil, unsupported("text field listing awaits stable binas form metadata")
}

// Fill fills supported form fields and returns rewritten PDF bytes.
func (d *Document) Fill(values map[string]string) ([]byte, error) {
	return nil, unsupported("form filling awaits OxPDF field appearance guardrails")
}

// SetCheckbox sets a checkbox field value.
func (d *Document) SetCheckbox(name string, checked bool) ([]byte, error) {
	return nil, unsupported("checkbox filling awaits OxPDF field appearance guardrails")
}

// Annotation describes a page annotation.
type Annotation struct {
	Kind     string
	Contents string
}

// Annotations lists common page annotations when available.
func (p *Page) Annotations() ([]Annotation, error) {
	return nil, unsupported("annotation listing awaits stable binas annotation metadata")
}
