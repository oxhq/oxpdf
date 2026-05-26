package oxpdf

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"path/filepath"
	"testing"
)

func TestDocumentTrailerAndCatalogSyntheticPDF(t *testing.T) {
	doc, err := OpenBytes(blankPDF([]PageSize{PageSizeLetter, PageSizeA4}))
	if err != nil {
		t.Fatal(err)
	}

	trailer, ok := doc.Trailer()
	if !ok {
		t.Fatal("Trailer() ok = false, want true")
	}
	if trailer.Size != 5 {
		t.Fatalf("Trailer().Size = %d, want 5", trailer.Size)
	}
	if trailer.Root == nil || *trailer.Root != (ObjectReference{Number: 1, Generation: 0}) {
		t.Fatalf("Trailer().Root = %+v, want 1 0 R", trailer.Root)
	}
	if trailer.Info != nil {
		t.Fatalf("Trailer().Info = %+v, want nil", trailer.Info)
	}
	if trailer.Encrypt != nil || trailer.Prev != nil || trailer.XRefStm != nil || len(trailer.ID) != 0 {
		t.Fatalf("unexpected optional trailer fields: %+v", trailer)
	}

	catalog, ok := doc.Catalog()
	if !ok {
		t.Fatal("Catalog() ok = false, want true")
	}
	if catalog.Object != (ObjectReference{Number: 1, Generation: 0}) {
		t.Fatalf("Catalog().Object = %+v, want 1 0 R", catalog.Object)
	}
	if catalog.Pages == nil || *catalog.Pages != (ObjectReference{Number: 2, Generation: 0}) {
		t.Fatalf("Catalog().Pages = %+v, want 2 0 R", catalog.Pages)
	}
	if catalog.Names != nil || catalog.Outlines != nil || catalog.PageLabels != nil || catalog.Metadata != nil {
		t.Fatalf("unexpected optional catalog refs: %+v", catalog)
	}
}

func TestPypdfCorpusTrailerInfoReference(t *testing.T) {
	resources := pypdfResources(t)
	doc, err := OpenFile(filepath.Join(resources, "hello-world.pdf"))
	if err != nil {
		t.Fatal(err)
	}

	trailer, ok := doc.Trailer()
	if !ok {
		t.Fatal("Trailer() ok = false, want true")
	}
	if trailer.Root == nil {
		t.Fatalf("Trailer().Root = nil, want catalog ref: %+v", trailer)
	}
	if trailer.Info == nil {
		t.Fatalf("Trailer().Info = nil, want info dictionary ref: %+v", trailer)
	}
	if trailer.Size <= 0 {
		t.Fatalf("Trailer().Size = %d, want positive", trailer.Size)
	}

	catalog, ok := doc.Catalog()
	if !ok {
		t.Fatal("Catalog() ok = false, want true")
	}
	if catalog.Object != *trailer.Root {
		t.Fatalf("Catalog().Object = %+v, want trailer root %+v", catalog.Object, trailer.Root)
	}
	if catalog.Pages == nil {
		t.Fatalf("Catalog().Pages = nil: %+v", catalog)
	}
}

func TestDocumentTrailerCompressedXrefStreamPDF(t *testing.T) {
	doc, err := OpenBytes(compressedXrefStreamPDF(t))
	if err != nil {
		t.Fatal(err)
	}

	trailer, ok := doc.Trailer()
	if !ok {
		t.Fatal("Trailer() ok = false, want true")
	}
	if trailer.Size != 6 {
		t.Fatalf("Trailer().Size = %d, want 6", trailer.Size)
	}
	if trailer.Root == nil || *trailer.Root != (ObjectReference{Number: 1, Generation: 0}) {
		t.Fatalf("Trailer().Root = %+v, want 1 0 R", trailer.Root)
	}
	if trailer.Info == nil || *trailer.Info != (ObjectReference{Number: 4, Generation: 0}) {
		t.Fatalf("Trailer().Info = %+v, want 4 0 R", trailer.Info)
	}
	if trailer.Prev != nil || trailer.XRefStm != nil || trailer.Encrypt != nil || len(trailer.ID) != 0 {
		t.Fatalf("unexpected optional trailer fields: %+v", trailer)
	}

	catalog, ok := doc.Catalog()
	if !ok {
		t.Fatal("Catalog() ok = false, want true")
	}
	if catalog.Object != *trailer.Root {
		t.Fatalf("Catalog().Object = %+v, want trailer root %+v", catalog.Object, trailer.Root)
	}
	if catalog.Pages == nil || *catalog.Pages != (ObjectReference{Number: 2, Generation: 0}) {
		t.Fatalf("Catalog().Pages = %+v, want 2 0 R", catalog.Pages)
	}

	xref := doc.Xref()
	if !xref.HasStream {
		t.Fatalf("Xref().HasStream = false, want true: %+v", xref)
	}
}

func TestDocumentXrefSummarySyntheticPDF(t *testing.T) {
	doc, err := OpenBytes(blankPDF([]PageSize{PageSizeLetter}))
	if err != nil {
		t.Fatal(err)
	}

	xref := doc.Xref()
	if !xref.HasTable {
		t.Fatalf("Xref().HasTable = false, want true: %+v", xref)
	}
	if xref.HasStream || xref.HasObjectStream || xref.HasHybridStream {
		t.Fatalf("unexpected xref stream flags: %+v", xref)
	}
	if xref.TableOffset <= 0 {
		t.Fatalf("Xref().TableOffset = %d, want positive", xref.TableOffset)
	}
	if xref.ObjectCount != 3 {
		t.Fatalf("Xref().ObjectCount = %d, want 3", xref.ObjectCount)
	}
	if len(xref.Objects) != 3 {
		t.Fatalf("len(Xref().Objects) = %d, want 3: %+v", len(xref.Objects), xref.Objects)
	}
	if xref.Objects[0].Number != 1 || xref.Objects[0].Generation != 0 || xref.Objects[0].Offset <= 0 || xref.Objects[0].Compressed {
		t.Fatalf("Xref().Objects[0] = %+v, want uncompressed object 1 0 with positive offset", xref.Objects[0])
	}
	if len(xref.StreamObjects) != 0 || len(xref.ObjectStreamObjects) != 0 {
		t.Fatalf("unexpected xref stream object lists: %+v", xref)
	}
}

func compressedXrefStreamPDF(t *testing.T) []byte {
	t.Helper()

	var out bytes.Buffer
	offsets := []int{0}
	out.WriteString("%PDF-1.5\n")
	writeObj := func(number int, body string) {
		for len(offsets) <= number {
			offsets = append(offsets, 0)
		}
		offsets[number] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", number, body)
	}
	writeObj(1, "<< /Type /Catalog /Pages 2 0 R >>")
	writeObj(2, "<< /Type /Pages /Kids [3 0 R] /Count 1 >>")
	writeObj(3, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>")
	writeObj(4, "<< /Producer (OxPDF test) >>")

	xrefOffset := out.Len()
	for len(offsets) <= 5 {
		offsets = append(offsets, 0)
	}
	offsets[5] = xrefOffset
	entries := xrefStreamEntries(offsets)
	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	if _, err := zw.Write(entries); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	fmt.Fprintf(&out, "5 0 obj\n<< /Type /XRef /Size 6 /Root 1 0 R /Info 4 0 R /W [1 4 2] /Index [0 6] /Filter /FlateDecode /Length %d >>\nstream\n", compressed.Len())
	out.Write(compressed.Bytes())
	fmt.Fprintf(&out, "\nendstream\nendobj\nstartxref\n%d\n%%%%EOF\n", xrefOffset)
	return out.Bytes()
}

func xrefStreamEntries(offsets []int) []byte {
	entries := make([]byte, 0, len(offsets)*7)
	for number, offset := range offsets {
		if number == 0 {
			entries = append(entries, 0, 0, 0, 0, 0, 0xff, 0xff)
			continue
		}
		entries = append(entries,
			1,
			byte(offset>>24),
			byte(offset>>16),
			byte(offset>>8),
			byte(offset),
			0,
			0,
		)
	}
	return entries
}

func TestNilDocumentTrailerAndCatalog(t *testing.T) {
	var doc *Document
	if trailer, ok := doc.Trailer(); ok || trailer.Size != 0 {
		t.Fatalf("nil Trailer() = %+v, %v; want empty false", trailer, ok)
	}
	if catalog, ok := doc.Catalog(); ok || catalog.Object.Number != 0 {
		t.Fatalf("nil Catalog() = %+v, %v; want empty false", catalog, ok)
	}
	if xref := doc.Xref(); xref.ObjectCount != 0 || xref.HasTable {
		t.Fatalf("nil Xref() = %+v, want empty", xref)
	}
}
