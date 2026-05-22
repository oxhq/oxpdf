package oxpdf

import "github.com/oxhq/binas/pkg/pdfapi"

// Profile summarizes what OxPDF can safely do with a document.
type Profile struct {
	Editable              bool
	Fillable              bool
	RewriteRecommendation string
	UnsupportedReasons    []string
	Header                string
	Pages                 int
	Boundaries            map[string]bool
	Xref                  XrefProfile
	Text                  TextProfile
	Streams               StreamProfile
	Forms                 FormProfile
	Annotations           AnnotationProfile
}

// XrefProfile summarizes cross-reference structure exposed by the backing parser.
type XrefProfile struct {
	HasTable          bool
	HasStream         bool
	HasObjectStream   bool
	ObjectCount       int
	StreamCount       int
	ObjectStreamCount int
}

// TextProfile summarizes selectable-text support.
type TextProfile struct {
	NodeCount     int
	EditableCount int
	CanEdit       bool
}

// StreamProfile summarizes stream/filter support.
type StreamProfile struct {
	TotalCount       int
	EditableCount    int
	PassThroughCount int
	UnsupportedCount int
	FilterCounts     map[string]int
	CapabilityCounts map[string]int
}

// FormProfile summarizes AcroForm support without exposing fill APIs yet.
type FormProfile struct {
	HasAcroForm   bool
	FieldCount    int
	FillableCount int
	BlockerCount  int
}

// AnnotationProfile summarizes annotation support without exposing edit APIs yet.
type AnnotationProfile struct {
	Present       bool
	Count         int
	EditableCount int
	BlockerCount  int
}

// Verification reports the result of reparsing and boundary checks.
type Verification struct {
	ReparseOK          bool
	PageCount          int
	UnsupportedReasons []string
}

// Profile returns the current high-level editability profile.
func (d *Document) Profile() Profile {
	if d == nil {
		return Profile{}
	}
	value, _ := d.root.Value.(map[string]any)
	report, err := pdfapi.Profile(d.input, pdfapi.ProfileOptions{Options: d.options})
	if err != nil {
		return Profile{
			Header:                d.Header(),
			Pages:                 d.NumPages(),
			UnsupportedReasons:    []string{err.Error()},
			RewriteRecommendation: "inspect-only",
		}
	}
	boundaries := map[string]bool{
		"has_encrypt":          report.Markers.Encrypted,
		"has_signature":        report.Markers.Signed,
		"has_acroform":         report.Markers.AcroForm,
		"has_xfa":              report.Markers.XFA,
		"has_annotations":      report.Markers.Annotations,
		"has_font_markers":     report.Markers.Fonts,
		"has_cmap_markers":     report.Markers.CMaps,
		"has_tounicode_cmap":   report.Markers.ToUnicode,
		"has_cid_font_markers": report.Markers.CIDFonts,
	}
	xrefValue := valueMap(value["xref"])
	profile := Profile{
		Editable:              report.Editable || (report.Valid && len(report.UnsupportedReasons) == 0),
		Fillable:              report.Fillable,
		RewriteRecommendation: string(report.RewriteRecommendation),
		UnsupportedReasons:    append([]string(nil), report.UnsupportedReasons...),
		Header:                d.Header(),
		Pages:                 d.NumPages(),
		Boundaries:            boundaries,
		Xref: XrefProfile{
			HasTable:          boolValue(xrefValue["has_table"]),
			HasStream:         boolValue(xrefValue["has_stream"]),
			HasObjectStream:   boolValue(xrefValue["has_object_stream"]),
			ObjectCount:       intValue(xrefValue["object_count"]),
			StreamCount:       intValue(xrefValue["stream_count"]),
			ObjectStreamCount: intValue(xrefValue["object_stream_count"]),
		},
		Text: TextProfile{
			NodeCount:     report.Text.NodeCount,
			EditableCount: report.Text.EditableCount,
			CanEdit:       report.Text.CanEdit,
		},
		Streams: StreamProfile{
			TotalCount:       report.Streams.TotalCount,
			EditableCount:    report.Streams.EditableCount,
			PassThroughCount: report.Streams.PassThroughCount,
			UnsupportedCount: report.Streams.UnsupportedCount,
			FilterCounts:     report.Streams.FilterCounts,
			CapabilityCounts: report.Streams.CapabilityCounts,
		},
		Forms: FormProfile{
			HasAcroForm:   report.Forms.HasAcroForm,
			FieldCount:    report.Forms.FieldCount,
			FillableCount: report.Forms.FillableCount,
			BlockerCount:  report.Forms.BlockerCount,
		},
		Annotations: AnnotationProfile{
			Present:       report.Annotations.Present,
			Count:         report.Annotations.Count,
			EditableCount: report.Annotations.EditableCount,
			BlockerCount:  report.Annotations.BlockerCount,
		},
	}
	return profile
}

// Validate reparses the original bytes and returns an OxPDF verification summary.
func (d *Document) Validate() (Verification, error) {
	if d == nil {
		return Verification{}, nil
	}
	verification, err := pdfapi.Validate(d.input, d.options)
	if err != nil {
		return Verification{}, classifyParseError(err)
	}
	profile := d.Profile()
	return Verification{
		ReparseOK:          verification.ReparseOK,
		PageCount:          profile.Pages,
		UnsupportedReasons: profile.UnsupportedReasons,
	}, nil
}

func valueMap(value any) map[string]any {
	if m, ok := value.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func boolMap(value map[string]any) map[string]bool {
	out := make(map[string]bool, len(value))
	for k, v := range value {
		out[k] = boolValue(v)
	}
	return out
}

func boolValue(value any) bool {
	v, _ := value.(bool)
	return v
}

func intValue(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	default:
		return 0
	}
}
