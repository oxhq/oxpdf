package oxpdf

import (
	"bytes"
	"strings"

	"github.com/oxhq/binas/pkg/core"
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
	pageIndexes := pageObjectIndexByNumber(d.input)
	localPages := localPagesContainingText(d.input, text)
	for _, node := range nodes {
		encoding, _ := node.Meta["encoding"].(string)
		page := -1
		if mapped, ok := nodePageIndex(node, pageIndexes); ok {
			page = mapped
		} else if len(localPages) == 1 {
			page = localPages[0]
		}
		matches = append(matches, TextMatch{
			Page:     page,
			Text:     text,
			Kind:     node.Kind,
			Encoding: encoding,
		})
	}
	return matches, nil
}

// ExtractText returns selectable text discovered in this page's content streams.
func (p *Page) ExtractText() (string, error) {
	if p == nil || p.doc == nil {
		return "", nil
	}
	nodes, err := pdfapi.QueryText(p.doc.input, pdfapi.TextSelector{}, p.doc.options)
	if err != nil {
		return "", classifyParseError(err)
	}
	pageText := make([][]string, p.doc.NumPages())
	pageIndexes := pageObjectIndexByNumber(p.doc.input)
	allMapped := true
	for _, node := range nodes {
		text, ok := node.Value.(string)
		if !ok {
			continue
		}
		pageIndex, ok := nodePageIndex(node, pageIndexes)
		if !ok || pageIndex < 0 || pageIndex >= len(pageText) {
			allMapped = false
			break
		}
		pageText[pageIndex] = append(pageText[pageIndex], text)
	}
	if allMapped && len(pageText[p.index]) > 0 {
		return strings.Join(pageText[p.index], "\n"), nil
	}
	local, err := localPageText(p.doc.input, p.index)
	if err != nil {
		return "", err
	}
	return local, nil
}

func nodePageIndex(node core.Node, pageIndexes map[int]int) (int, bool) {
	number, ok := intMeta(node.Meta, "page_object_number")
	if !ok {
		return 0, false
	}
	index, ok := pageIndexes[number]
	return index, ok
}

func pageObjectIndexByNumber(input []byte) map[int]int {
	pages := parsePageObjects(input)
	out := make(map[int]int, len(pages))
	for i, page := range pages {
		out[page.number] = i
	}
	return out
}

func localPagesContainingText(input []byte, text string) []int {
	pages := parsePageObjects(input)
	out := make([]int, 0, 1)
	for i := range pages {
		pageText, err := localPageText(input, i)
		if err == nil && pageText == text {
			out = append(out, i)
		}
	}
	return out
}

func localPageText(input []byte, pageIndex int) (string, error) {
	pages := parsePageObjects(input)
	if pageIndex < 0 || pageIndex >= len(pages) {
		return "", ErrPageIndexOutOfRange
	}
	raw, ok := directNameValue(pages[pageIndex].dict, "Contents")
	if !ok {
		return "", nil
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return "", nil
	}
	if raw[0] == '[' {
		return "", unsupported("page text extraction supports only a single direct /Contents stream")
	}
	ref, ok := parsePDFRef(raw)
	if !ok {
		return "", unsupported("page text extraction supports only direct /Contents stream references")
	}
	object, ok := indirectObjectByNumber(parseIndirectObjects(input), ref.number, ref.gen)
	if !ok {
		return "", unsupported("page text extraction content stream object not found")
	}
	dict, ok := objectDictionary(object.body)
	if !ok {
		return "", unsupported("page text extraction content stream dictionary not found")
	}
	if filter := streamFilterName(dict); filter != "" {
		return "", unsupported("page text extraction supports only unfiltered content streams")
	}
	stream, ok := streamBytesAfterDictionary(object.body, dict)
	if !ok {
		return "", unsupported("page text extraction content stream not found")
	}
	text, ok := localContentStreamText(stream)
	if !ok {
		return "", unsupported("page text extraction supports only simple text-show operators")
	}
	return strings.Join(text, "\n"), nil
}

func localContentStreamText(content []byte) ([]string, bool) {
	tokens, ok := contentTokens(content)
	if !ok {
		return nil, false
	}
	out := make([]string, 0)
	for i, token := range tokens {
		switch token {
		case "Tj", "'", `"`:
			if i == 0 {
				return nil, false
			}
			text, ok := parsePDFTextValue([]byte(tokens[i-1]))
			if !ok {
				return nil, false
			}
			out = append(out, text)
		case "TJ":
			if i == 0 {
				return nil, false
			}
			text, ok := localTextArrayValue(tokens[i-1])
			if !ok {
				return nil, false
			}
			out = append(out, text)
		}
	}
	return out, true
}

func contentTokens(input []byte) ([]string, bool) {
	tokens := make([]string, 0)
	for i := 0; i < len(input); {
		i = skipPDFSpace(input, i)
		if i >= len(input) {
			break
		}
		start := i
		switch input[i] {
		case '(':
			end, ok := scanLiteralEnd(input, i)
			if !ok {
				return nil, false
			}
			tokens = append(tokens, string(input[start:end+1]))
			i = end + 1
		case '<':
			if i+1 < len(input) && input[i+1] == '<' {
				return nil, false
			}
			endRel := bytes.IndexByte(input[i+1:], '>')
			if endRel == -1 {
				return nil, false
			}
			i += endRel + 2
			tokens = append(tokens, string(input[start:i]))
		case '[':
			end, ok := scanArrayEnd(input, i)
			if !ok {
				return nil, false
			}
			tokens = append(tokens, string(input[start:end+1]))
			i = end + 1
		default:
			for i < len(input) && !isPDFSpaceByte(input[i]) && !isPDFDelimiterByte(input[i]) {
				i++
			}
			if i == start {
				return nil, false
			}
			tokens = append(tokens, string(input[start:i]))
		}
	}
	return tokens, true
}

func localTextArrayValue(token string) (string, bool) {
	raw := bytes.TrimSpace([]byte(token))
	if len(raw) == 0 || raw[0] != '[' {
		return "", false
	}
	tokens, ok := contentTokens(raw[1 : len(raw)-1])
	if !ok {
		return "", false
	}
	var out strings.Builder
	for _, token := range tokens {
		text, ok := parsePDFTextValue([]byte(token))
		if ok {
			out.WriteString(text)
		}
	}
	return out.String(), true
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
