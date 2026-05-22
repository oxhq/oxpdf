package oxpdf

import (
	"github.com/oxhq/binas/pkg/pdfapi"
)

// TextMatch describes a selectable text occurrence found by the backing parser.
type TextMatch struct {
	Page     int
	Text     string
	Kind     string
	Encoding string
}

// TextEditability reports whether verified text replacement is currently safe.
type TextEditability struct {
	ReplaceTextSupported bool
	UnsupportedReasons   []string
}

// FindText returns selectable text nodes that exactly match text.
func (d *Document) FindText(text string) ([]TextMatch, error) {
	if d == nil {
		return nil, nil
	}
	nodes, err := pdfapi.QueryText(d.input, pdfapi.TextSelector{Text: text}, d.options)
	if err != nil {
		return nil, classifyParseError(err)
	}
	matches := make([]TextMatch, 0, len(nodes))
	for _, node := range nodes {
		encoding, _ := node.Meta["encoding"].(string)
		matches = append(matches, TextMatch{
			Page:     -1,
			Text:     text,
			Kind:     node.Kind,
			Encoding: encoding,
		})
	}
	return matches, nil
}

// ExtractText returns selectable text discovered in this page's content streams.
//
// The current binas adapter does not map text nodes back to page objects yet, so
// page 0 exposes document-level selectable text and later pages fail closed.
func (p *Page) ExtractText() (string, error) {
	if p == nil || p.doc == nil {
		return "", nil
	}
	if p.index != 0 {
		return "", unsupported("page-scoped text extraction awaits page-to-stream mapping in binas")
	}
	nodes, err := pdfapi.QueryText(p.doc.input, pdfapi.TextSelector{}, p.doc.options)
	if err != nil {
		return "", classifyParseError(err)
	}
	var out string
	for _, node := range nodes {
		if out != "" {
			out += "\n"
		}
		out += node.Value.(string)
	}
	return out, nil
}

// TextEditability returns conservative replacement support for this document.
func (d *Document) TextEditability() TextEditability {
	profile := d.Profile()
	return TextEditability{
		ReplaceTextSupported: profile.Editable,
		UnsupportedReasons:   profile.UnsupportedReasons,
	}
}

// ReplaceText performs a verified selectable-text rewrite.
func (d *Document) ReplaceText(oldText, newText string) ([]byte, error) {
	if d == nil {
		return nil, unsupported("missing document")
	}
	out, _, _, err := pdfapi.EditText(
		d.input,
		pdfapi.TextSelector{Text: oldText},
		pdfapi.TextReplacement{Replace: newText},
		pdfapi.Options{
			Password:      d.options.Password,
			SignatureMode: d.options.SignatureMode,
			Rewrite:       pdfapi.RewriteModeCanonical,
			Verify: []string{
				"reparse",
				"old-gone",
				"new-selectable",
				"page-count-unchanged",
				"no-fallback",
			},
		},
	)
	if err != nil {
		return nil, classifyParseError(err)
	}
	return out, nil
}
