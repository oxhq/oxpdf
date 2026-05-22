package oxpdf

import "github.com/oxhq/binas/pkg/core"

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
	boundaries := boolMap(valueMap(value["boundaries"]))
	xrefValue := valueMap(value["xref"])
	profile := Profile{
		Header:     d.Header(),
		Pages:      d.NumPages(),
		Boundaries: boundaries,
		Xref: XrefProfile{
			HasTable:          boolValue(xrefValue["has_table"]),
			HasStream:         boolValue(xrefValue["has_stream"]),
			HasObjectStream:   boolValue(xrefValue["has_object_stream"]),
			ObjectCount:       intValue(xrefValue["object_count"]),
			StreamCount:       intValue(xrefValue["stream_count"]),
			ObjectStreamCount: intValue(xrefValue["object_stream_count"]),
		},
	}
	profile.UnsupportedReasons = unsupportedReasons(boundaries, profile.Xref)
	profile.Editable = len(profile.UnsupportedReasons) == 0
	profile.Fillable = boundaries["has_acroform"]
	if profile.Editable {
		profile.RewriteRecommendation = "canonical"
	} else {
		profile.RewriteRecommendation = "inspect-only"
	}
	return profile
}

// Validate reparses the original bytes and returns an OxPDF verification summary.
func (d *Document) Validate() (Verification, error) {
	if d == nil {
		return Verification{}, nil
	}
	tree, err := d.adapter.Parse(d.input, core.ParseOptions{Strict: true})
	if err != nil {
		return Verification{}, classifyParseError(err)
	}
	root, ok := tree.Node(tree.Root)
	if !ok {
		return Verification{}, nil
	}
	doc := &Document{input: d.input, tree: tree, root: root, adapter: d.adapter}
	profile := doc.Profile()
	return Verification{
		ReparseOK:          true,
		PageCount:          profile.Pages,
		UnsupportedReasons: profile.UnsupportedReasons,
	}, nil
}

func unsupportedReasons(boundaries map[string]bool, xref XrefProfile) []string {
	var reasons []string
	if boundaries["has_encrypt"] {
		reasons = append(reasons, "encrypted PDF")
	}
	if boundaries["has_signature"] {
		reasons = append(reasons, "signed PDF")
	}
	if boundaries["has_xfa"] {
		reasons = append(reasons, "XFA form")
	}
	if xref.HasStream {
		reasons = append(reasons, "xref stream")
	}
	return reasons
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
