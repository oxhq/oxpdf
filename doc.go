// Package oxpdf provides Go-native PDF inspection and manipulation helpers.
//
// The package is intentionally conservative: operations that cannot be proven
// through the backing PDF engine return structured unsupported errors instead
// of silently falling back to lossy behavior.
package oxpdf

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/oxhq/binas/pkg/core"
	"github.com/oxhq/binas/pkg/pdfapi"
)

var (
	// ErrUnsupported reports a PDF feature or OxPDF operation that is known but
	// not yet supported by the backing engine.
	ErrUnsupported = errors.New("oxpdf: unsupported")

	// ErrPageIndexOutOfRange reports a zero-based page index outside the document.
	ErrPageIndexOutOfRange = errors.New("oxpdf: page index out of range")
)

// Open reads a PDF from r.
func Open(r io.Reader, opts ...OpenOption) (*Document, error) {
	input, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return OpenBytes(input, opts...)
}

// OpenFile reads a PDF from path.
func OpenFile(path string, opts ...OpenOption) (*Document, error) {
	input, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return OpenBytes(input, opts...)
}

// OpenBytes parses PDF bytes and returns an inspection-oriented document.
func OpenBytes(input []byte, opts ...OpenOption) (*Document, error) {
	cfg := openConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	apiOpts := pdfapi.Options{Password: cfg.password}
	tree, err := pdfapi.Inspect(input, apiOpts)
	if err != nil {
		return nil, classifyParseError(err)
	}
	root, ok := tree.Node(tree.Root)
	if !ok {
		return nil, errors.New("oxpdf: parsed PDF tree has no root node")
	}
	return &Document{
		input:   bytes.Clone(input),
		tree:    tree,
		root:    root,
		options: apiOpts,
	}, nil
}

// Document is an opened PDF document.
type Document struct {
	input   []byte
	tree    *core.Tree
	root    core.Node
	options pdfapi.Options
}

// Bytes returns a copy of the original document bytes.
func (d *Document) Bytes() []byte {
	if d == nil {
		return nil
	}
	return bytes.Clone(d.input)
}

// Header returns the PDF header line, for example "%PDF-1.7".
func (d *Document) Header() string {
	if d == nil {
		return ""
	}
	if header, ok := d.root.Meta["header"].(string); ok {
		return header
	}
	if value, ok := d.root.Value.(map[string]any); ok {
		if header, ok := value["header"].(string); ok {
			return header
		}
	}
	return ""
}

// NumPages returns the page count reported by the backing parser.
func (d *Document) NumPages() int {
	if d == nil {
		return 0
	}
	value, _ := d.root.Value.(map[string]any)
	pages, _ := value["pages"].(int)
	return pages
}

// Page returns a zero-based page handle.
func (d *Document) Page(index int) (*Page, error) {
	if d == nil {
		return nil, ErrPageIndexOutOfRange
	}
	if index < 0 || index >= d.NumPages() {
		return nil, fmt.Errorf("%w: %d", ErrPageIndexOutOfRange, index)
	}
	return &Page{doc: d, index: index}, nil
}

// Metadata returns document information known to the current backing engine.
func (d *Document) Metadata() Metadata {
	if d == nil {
		return Metadata{}
	}
	metadata := parseMetadata(d.input)
	metadata.Header = d.Header()
	return metadata
}

// Tree returns a copy of the parsed binas tree for advanced inspection.
func (d *Document) Tree() core.Tree {
	if d == nil || d.tree == nil {
		return core.Tree{}
	}
	return *d.tree
}

// Page is a zero-based PDF page handle.
type Page struct {
	doc   *Document
	index int
}

// Index returns the zero-based page index.
func (p *Page) Index() int {
	if p == nil {
		return -1
	}
	return p.index
}

func classifyParseError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case bytes.Contains([]byte(msg), []byte("unsupported PDF:")),
		bytes.Contains([]byte(msg), []byte("encrypted PDF")),
		bytes.Contains([]byte(msg), []byte("password")):
		return fmt.Errorf("%w: %s", ErrUnsupported, msg)
	default:
		return err
	}
}

func unsupported(reason string) error {
	return fmt.Errorf("%w: %s", ErrUnsupported, reason)
}
