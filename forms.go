package oxpdf

import binaspdf "github.com/oxhq/binas/pkg/adapters/pdf"

// Field describes a PDF form field.
type Field struct {
	Name         string
	Type         string
	Value        string
	DefaultValue string
	Status       string
	ReadOnly     bool
	Required     bool
	NoExport     bool
	Flags        []string
	TypeFlags    []string
	Options      []string
	ButtonStates []string
	Blockers     []string
}

// Fields lists AcroForm fields with conservative fillability metadata.
func (d *Document) Fields() ([]Field, error) {
	if d == nil {
		return nil, nil
	}
	fields, err := binaspdf.ListFormFields(d.input)
	if err != nil {
		return nil, classifyParseError(err)
	}
	out := make([]Field, 0, len(fields))
	for _, field := range fields {
		out = append(out, mapFormField(field))
	}
	return out, nil
}

// TextFields lists text fields only.
func (d *Document) TextFields() ([]Field, error) {
	fields, err := d.Fields()
	if err != nil {
		return nil, err
	}
	textFields := make([]Field, 0, len(fields))
	for _, field := range fields {
		if field.Type == "Tx" {
			textFields = append(textFields, field)
		}
	}
	return textFields, nil
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

func mapFormField(field binaspdf.FormFieldMetadata) Field {
	return Field{
		Name:         field.Name,
		Type:         field.FieldType,
		Value:        stringValue(field.Value),
		DefaultValue: stringValue(field.DefaultValue),
		Status:       field.FillStatus,
		ReadOnly:     field.ReadOnly,
		Required:     field.Required,
		NoExport:     field.NoExport,
		Flags:        append([]string(nil), field.FlagNames...),
		TypeFlags:    append([]string(nil), field.TypeFlagNames...),
		Options:      append([]string(nil), field.Options...),
		ButtonStates: append([]string(nil), field.ButtonStates...),
		Blockers:     append([]string(nil), field.FillBlockers...),
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
