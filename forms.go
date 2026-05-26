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
	field, err := d.buttonField(name)
	if err != nil {
		return nil, err
	}
	onState, ok := checkboxOnState(field)
	if !ok {
		return nil, unsupported(fmt.Sprintf("field %q is not a checkbox-like button", name))
	}
	value := "Off"
	if checked {
		value = onState
	}
	return d.Fill(map[string]string{name: value})
}

// SetButtonChoice sets a button/radio field to an exact exported appearance state.
func (d *Document) SetButtonChoice(name string, state string) ([]byte, error) {
	if d == nil {
		return nil, unsupported("missing document")
	}
	field, err := d.buttonField(name)
	if err != nil {
		return nil, err
	}
	if containsString(field.TypeFlags, "pushbutton") {
		return nil, unsupported(fmt.Sprintf("field %q is a pushbutton", name))
	}
	if !containsString(field.ButtonStates, "Off") {
		return nil, unsupported(fmt.Sprintf("field %q has no Off state", name))
	}
	if !containsString(field.ButtonStates, state) {
		return nil, unsupported(fmt.Sprintf("field %q does not allow state %q", name, state))
	}
	return d.Fill(map[string]string{name: state})
}

// Annotation describes a page annotation.
type Annotation struct {
	Index         int
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

// AnnotationContentsEditOptions configures annotation content edits.
type AnnotationContentsEditOptions struct {
	RegenerateAppearance bool
}

// AnnotationContentsEditVerification reports verification for an annotation edit.
type AnnotationContentsEditVerification struct {
	ReparseOK             bool
	ContentsUpdated       bool
	PageUnchanged         bool
	AppearanceRegenerated bool
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

// SetAnnotationContents updates a supported annotation's contents.
func (d *Document) SetAnnotationContents(index int, contents string, opts ...AnnotationContentsEditOptions) ([]byte, AnnotationContentsEditVerification, error) {
	if d == nil {
		return nil, AnnotationContentsEditVerification{}, unsupported("missing document")
	}
	annotations, err := binaspdf.ListAnnotationCandidates(d.input)
	if err != nil {
		return nil, AnnotationContentsEditVerification{}, classifyParseError(err)
	}
	var target *binaspdf.AnnotationCandidateMetadata
	for i := range annotations {
		if annotations[i].Index == index {
			target = &annotations[i]
			break
		}
	}
	if target == nil {
		return nil, AnnotationContentsEditVerification{}, unsupported(fmt.Sprintf("no annotation matches index %d", index))
	}
	if target.AppearanceGenerationStatus != "approximate_supported" || len(target.AppearanceGenerationBlockers) > 0 {
		return nil, AnnotationContentsEditVerification{}, unsupported(fmt.Sprintf("annotation %d is not safely editable", index))
	}
	editOptions := []binaspdf.AnnotationContentsEditOptions{}
	if len(opts) > 0 && opts[0].RegenerateAppearance {
		editOptions = append(editOptions, binaspdf.AnnotationContentsEditOptions{RegenerateAppearance: true})
	}
	out, _, verification, err := binaspdf.ApplyAnnotationContentsEdit(d.input, index, contents, editOptions...)
	if err != nil {
		return nil, AnnotationContentsEditVerification{}, unsupported(err.Error())
	}
	if !verification.ReparseOK || !verification.ContentsUpdated || !verification.PageUnchanged {
		return nil, AnnotationContentsEditVerification{}, unsupported(fmt.Sprintf("annotation %d edit did not satisfy verification", index))
	}
	if len(opts) > 0 && opts[0].RegenerateAppearance && !verification.AppearanceRegenerated {
		return nil, AnnotationContentsEditVerification{}, unsupported(fmt.Sprintf("annotation %d appearance was not regenerated", index))
	}
	return out, AnnotationContentsEditVerification{
		ReparseOK:             verification.ReparseOK,
		ContentsUpdated:       verification.ContentsUpdated,
		PageUnchanged:         verification.PageUnchanged,
		AppearanceRegenerated: verification.AppearanceRegenerated,
	}, nil
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

func (d *Document) buttonField(name string) (Field, error) {
	fields, err := d.Fields()
	if err != nil {
		return Field{}, err
	}
	for _, field := range fields {
		if field.Name != name {
			continue
		}
		if field.Type != "Btn" {
			return Field{}, unsupported(fmt.Sprintf("field %q is not a button", name))
		}
		if field.Status != "supported" || len(field.Blockers) > 0 {
			return Field{}, unsupported(fmt.Sprintf("field %q is not safely settable", name))
		}
		return field, nil
	}
	return Field{}, unsupported(fmt.Sprintf("no AcroForm field matches %q", name))
}

func checkboxOnState(field Field) (string, bool) {
	if containsString(field.TypeFlags, "radio") || containsString(field.TypeFlags, "pushbutton") {
		return "", false
	}
	if !containsString(field.ButtonStates, "Off") {
		return "", false
	}
	onState := ""
	for _, state := range field.ButtonStates {
		if state == "Off" {
			continue
		}
		if onState != "" {
			return "", false
		}
		onState = state
	}
	return onState, onState != ""
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func mapAnnotation(annotation binaspdf.AnnotationCandidateMetadata) Annotation {
	return Annotation{
		Index:         annotation.Index,
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
