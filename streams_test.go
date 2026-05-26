package oxpdf

import (
	"bytes"
	"compress/zlib"
	"errors"
	"path/filepath"
	"testing"
)

func TestPypdfCorpusStreamInventoryImagePassThrough(t *testing.T) {
	resources := pypdfResources(t)
	doc, err := OpenFile(filepath.Join(resources, "jpeg.pdf"))
	if err != nil {
		t.Fatal(err)
	}

	streams := doc.Streams()
	if len(streams) != 5 {
		t.Fatalf("Streams() len = %d, want 5", len(streams))
	}
	image, ok := findStreamByFilter(streams, "DCTDecode")
	if !ok {
		t.Fatalf("Streams() missing DCTDecode image stream: %+v", streams)
	}
	if !image.ImageXObject {
		t.Fatalf("DCT stream ImageXObject = false")
	}
	if image.EncodedLength != 41402 {
		t.Fatalf("DCT stream EncodedLength = %d, want 41402", image.EncodedLength)
	}
	if image.HasDecodedLength {
		t.Fatalf("DCT stream HasDecodedLength = true, want false")
	}
	if !sameStrings(image.FilterChain, []string{"DCTDecode"}) {
		t.Fatalf("DCT stream FilterChain = %v", image.FilterChain)
	}
	if image.FilterCapability != "pass_through_image" || image.Editable || !image.PassThrough || image.Target {
		t.Fatalf("DCT stream capability = %+v", image)
	}
}

func TestPypdfCorpusStreamInventoryEditableFilterChain(t *testing.T) {
	resources := pypdfResources(t)
	doc, err := OpenFile(filepath.Join(resources, "reportlab-inline-image.pdf"))
	if err != nil {
		t.Fatal(err)
	}

	streams := doc.Streams()
	if len(streams) != 1 {
		t.Fatalf("Streams() len = %d, want 1", len(streams))
	}
	stream := streams[0]
	if !sameStrings(stream.FilterChain, []string{"ASCII85Decode", "FlateDecode"}) {
		t.Fatalf("FilterChain = %v, want ASCII85Decode+FlateDecode", stream.FilterChain)
	}
	if stream.FilterCapability != "editable_reversible" || !stream.Editable || stream.PassThrough || !stream.Target {
		t.Fatalf("stream capability = %+v", stream)
	}
	if stream.EncodedLength <= 0 || stream.DecodedLength <= 0 || !stream.HasDecodedLength {
		t.Fatalf("stream lengths = %+v, want encoded and decoded lengths", stream)
	}
}

func TestNilDocumentStreams(t *testing.T) {
	var doc *Document
	if streams := doc.Streams(); streams != nil {
		t.Fatalf("nil Streams() = %+v, want nil", streams)
	}
	if streams := doc.ImageXObjectStreams(); streams != nil {
		t.Fatalf("nil ImageXObjectStreams() = %+v, want nil", streams)
	}
}

func TestDecodedStreamIdentity(t *testing.T) {
	content := "BT /F1 12 Tf 72 720 Td (Invoice 1234) Tj ET"
	doc, err := OpenBytes(textPDF("Invoice 1234"))
	if err != nil {
		t.Fatal(err)
	}

	decoded, err := doc.DecodedStream(0)
	if err != nil {
		t.Fatal(err)
	}
	if string(decoded) != content {
		t.Fatalf("DecodedStream(identity) = %q, want %q", decoded, content)
	}
	decoded[0] = 'X'
	again, err := doc.DecodedStream(0)
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != content {
		t.Fatalf("DecodedStream returned mutable backing bytes: %q", again)
	}
}

func TestDecodedStreamFlateDecode(t *testing.T) {
	content := []byte("BT /F1 12 Tf 72 720 Td (compressed stream) Tj ET")
	doc, err := OpenBytes(singleStreamPDF("/Filter /FlateDecode ", zlibCompress(t, content)))
	if err != nil {
		t.Fatal(err)
	}

	decoded, err := doc.DecodedStream(0)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decoded, content) {
		t.Fatalf("DecodedStream(FlateDecode) = %q, want %q", decoded, content)
	}
}

func TestDecodedStreamUnsupportedFilterFailsClosed(t *testing.T) {
	doc, err := OpenBytes(singleStreamPDF("/Filter /DCTDecode ", []byte("not really jpeg")))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := doc.DecodedStream(0); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("DecodedStream(DCTDecode) error = %v, want ErrUnsupported", err)
	}
}

func TestDecodedStreamIndexOutOfRange(t *testing.T) {
	doc, err := OpenBytes(textPDF("Invoice 1234"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := doc.DecodedStream(1); !errors.Is(err, ErrStreamIndexOutOfRange) {
		t.Fatalf("DecodedStream(out of range) error = %v, want ErrStreamIndexOutOfRange", err)
	}
	var nilDoc *Document
	if _, err := nilDoc.DecodedStream(0); !errors.Is(err, ErrStreamIndexOutOfRange) {
		t.Fatalf("nil DecodedStream error = %v, want ErrStreamIndexOutOfRange", err)
	}
}

func findStreamByFilter(streams []Stream, filter string) (Stream, bool) {
	for _, stream := range streams {
		if stream.Filter == filter {
			return stream, true
		}
	}
	return Stream{}, false
}

func singleStreamPDF(extraDict string, content []byte) []byte {
	return pdfObjects(
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << >> /Contents 4 0 R >>",
		"<< "+extraDict+"/Length "+itoa(len(content))+" >>\nstream\n"+string(content)+"\nendstream",
	)
}

func zlibCompress(t *testing.T, input []byte) []byte {
	t.Helper()
	var out bytes.Buffer
	writer := zlib.NewWriter(&out)
	if _, err := writer.Write(input); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}
