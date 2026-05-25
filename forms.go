package oxpdf

import (
	"fmt"
	"sort"

	binaspdf "github.com/oxhq/binas/pkg/adapters/pdf"
)

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
	if d == nil {
		return nil, unsupported("missing document")
	}
	out := d.Bytes()
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		next, _, verification, err := binaspdf.ApplyFormFieldEdit(out, key, values[key])
		if err != nil {
			return nil, unsupported(err.Error())
		}
		if !verification.ReparseOK || !verification.FieldValueSet || !verification.NeedAppearancesSet {
			return nil, unsupported(fmt.Sprintf("field %q edit did not satisfy verification", key))
		}
		out = next
	}
	return out, nil
}

// SetCheckbox sets a checkbox field value.
func (d *Document) SetCheckbox(name string, checked bool) ([]byte, error) {
	if d == nil {
		return nil, unsupported("missing document")
	}
	value := "Off"
	if checked {
		value = "Yes"
		fields, err := d.Fields()
		if err != nil {
			return nil, err
		}
		for _, field := range fields {
			if field.Name != name {
				continue
			}
			for _, state := range field.ButtonStates {
				if state != "Off" {
					value = state
					break
				}
			}
			break
		}
	}
	return d.Fill(map[string]string{name: value})
}

// Annotation describes a page annotation.
type Annotation struct {
	Kind          string
	Contents      string
	Name          string
	Title         string
	Modified      string
	Status        string
	Blockers      []string
	Rect          []float64
	Color         []float64
	Border        []float64
	Flags         []string
	HasAppearance bool
}

// Annotations lists annotation metadata for a page.
func (p *Page) Annotations() ([]Annotation, error) {
	if p == nil || p.doc == nil {
		return nil, nil
	}
	annotations, err := binaspdf.ListAnnotationCandidates(p.doc.input)
	if err != nil {
		return nil, classifyParseError(err)
	}
	out := make([]Annotation, 0, len(annotations))
	for _, annotation := range annotations {
		if annotation.PageIndex != nil && *annotation.PageIndex != p.index {
			continue
		}
		out = append(out, mapAnnotation(annotation))
	}
	return out, nil
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

func mapAnnotation(annotation binaspdf.AnnotationCandidateMetadata) Annotation {
	return Annotation{
		Kind:          annotation.Subtype,
		Contents:      decodePDFTextBytes([]byte(annotation.Contents)),
		Name:          decodePDFTextBytes([]byte(annotation.Name)),
		Title:         decodePDFTextBytes([]byte(annotation.Title)),
		Modified:      annotation.Modified,
		Status:        annotation.AppearanceGenerationStatus,
		Blockers:      append([]string(nil), annotation.AppearanceGenerationBlockers...),
		Rect:          append([]float64(nil), annotation.Rect...),
		Color:         append([]float64(nil), annotation.Color...),
		Border:        append([]float64(nil), annotation.Border...),
		Flags:         append([]string(nil), annotation.FlagNames...),
		HasAppearance: annotation.HasAppearance,
	}
}
