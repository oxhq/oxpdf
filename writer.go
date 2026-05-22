package oxpdf

import (
	"bytes"
	"fmt"
)

// PageSize describes a blank page size in PDF points.
type PageSize struct {
	Width  float64
	Height float64
}

// Common page sizes.
var (
	PageSizeA4     = PageSize{Width: 595, Height: 842}
	PageSizeLetter = PageSize{Width: 612, Height: 792}
)

// Writer builds canonical PDFs for the subset currently supported by OxPDF.
type Writer struct {
	blankPages []PageSize
}

// NewWriter creates an empty PDF writer.
func NewWriter() *Writer {
	return &Writer{}
}

// AddBlankPage appends a blank page.
func (w *Writer) AddBlankPage(size PageSize) {
	if size.Width <= 0 || size.Height <= 0 {
		size = PageSizeLetter
	}
	w.blankPages = append(w.blankPages, size)
}

// AddPage is reserved for copying existing pages once binas exposes page graph writing.
func (w *Writer) AddPage(page *Page) error {
	if page == nil {
		return unsupported("missing page")
	}
	return unsupported("copying existing pages awaits binas page graph writer support")
}

// InsertPage is reserved for page insertion once binas exposes page graph writing.
func (w *Writer) InsertPage(index int, page *Page) error {
	return unsupported("page insertion awaits binas page graph writer support")
}

// Append is reserved for document merge once binas exposes page graph writing.
func (w *Writer) Append(doc *Document) error {
	return unsupported("document append awaits binas page graph writer support")
}

// Bytes writes a canonical PDF containing the configured blank pages.
func (w *Writer) Bytes() ([]byte, error) {
	if w == nil || len(w.blankPages) == 0 {
		return nil, unsupported("writer requires at least one supported page operation")
	}
	return blankPDF(w.blankPages), nil
}

func blankPDF(pages []PageSize) []byte {
	var body bytes.Buffer
	offsets := []int{0}
	writeObj := func(id int, content string) {
		offsets = append(offsets, body.Len())
		body.WriteString(fmt.Sprintf("%d 0 obj\n%s\nendobj\n", id, content))
	}
	kids := ""
	for i := range pages {
		pageID := 3 + i
		kids += fmt.Sprintf("%d 0 R ", pageID)
	}
	writeObj(1, "<< /Type /Catalog /Pages 2 0 R >>")
	writeObj(2, fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", kids, len(pages)))
	for i, size := range pages {
		writeObj(3+i, fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.0f %.0f] >>", size.Width, size.Height))
	}
	var out bytes.Buffer
	out.WriteString("%PDF-1.7\n")
	base := out.Len()
	out.Write(body.Bytes())
	xrefAt := out.Len()
	out.WriteString(fmt.Sprintf("xref\n0 %d\n", len(offsets)))
	out.WriteString("0000000000 65535 f \n")
	for i := 1; i < len(offsets); i++ {
		out.WriteString(fmt.Sprintf("%010d 00000 n \n", base+offsets[i]))
	}
	out.WriteString(fmt.Sprintf("trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets), xrefAt))
	return out.Bytes()
}
