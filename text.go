package oxpdf

import (
	"github.com/oxhq/binas/pkg/adapters/pdf"
	"github.com/oxhq/binas/pkg/core"
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
	if d == nil || d.tree == nil {
		return nil, nil
	}
	nodes := d.tree.Query(core.Match{Kind: pdf.KindTextShow, Text: text})
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
	if p == nil || p.doc == nil || p.doc.tree == nil {
		return "", nil
	}
	if p.index != 0 {
		return "", unsupported("page-scoped text extraction awaits page-to-stream mapping in binas")
	}
	var out string
	for _, node := range p.doc.tree.Query(core.Match{Kind: pdf.KindTextShow}) {
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
	out, _, _, err := pdf.ApplyCanonicalEdit(
		d.input,
		core.Match{Kind: pdf.KindTextShow, Text: oldText},
		core.Mutation{Replace: newText},
		[]core.Invariant{
			core.InvariantReparse,
			core.InvariantOldGone,
			core.InvariantNewSelectable,
			core.InvariantPageUnchanged,
			core.InvariantNoFallbackUsed,
		},
	)
	if err != nil {
		return nil, classifyParseError(err)
	}
	return out, nil
}
