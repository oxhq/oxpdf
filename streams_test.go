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

func TestPypdfCorpusInlineImagesAreNotImageXObjectStreams(t *testing.T) {
	resources := pypdfResources(t)
	doc, err := OpenFile(filepath.Join(resources, "reportlab-inline-image.pdf"))
	if err != nil {
		t.Fatal(err)
	}

	if xobjects := doc.ImageXObjectStreams(); len(xobjects) != 0 {
		t.Fatalf("ImageXObjectStreams() len = %d, want 0 for inline-image fixture", len(xobjects))
	}
	images, err := doc.InlineImages()
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 1 {
		t.Fatalf("InlineImages() len = %d, want 1", len(images))
	}
	got := images[0]
	if got.Width != 16 || got.Height != 16 || got.BitsPerComponent != 8 || got.ColorSpace != "DeviceRGB" {
		t.Fatalf("inline image metadata = %+v, want 16x16 DeviceRGB bpc 8", got)
	}
	if !sameStrings(got.FilterChain, []string{"ASCII85Decode", "FlateDecode"}) || got.Extension != "raw" || got.PassThrough {
		t.Fatalf("inline image filter metadata = %+v, want ASCII85+Flate raw decoded bytes", got)
	}
	if len(got.Content) != 16*16*3 {
		t.Fatalf("inline image content len = %d, want %d", len(got.Content), 16*16*3)
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

func TestImageXObjectsExtractsDCTPassThroughBytesAndMetadata(t *testing.T) {
	image := []byte{0xff, 0xd8, 0xff, 0xdb, 0x00, 0x43, 0xff, 0xd9}
	doc, err := OpenBytes(imageXObjectPDF("/Filter /DCTDecode ", image))
	if err != nil {
		t.Fatal(err)
	}

	images, err := doc.ImageXObjects()
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 1 {
		t.Fatalf("ImageXObjects() len = %d, want 1", len(images))
	}
	got := images[0]
	if got.ObjectNumber != 5 || got.StreamIndex != 1 {
		t.Fatalf("image object/index = %d/%d, want 5/1", got.ObjectNumber, got.StreamIndex)
	}
	if got.Width != 2 || got.Height != 1 || got.BitsPerComponent != 8 || got.ColorSpace != "DeviceRGB" {
		t.Fatalf("image metadata = %+v, want 2x1 DeviceRGB bpc 8", got)
	}
	if got.Filter != "DCTDecode" || got.Extension != "jpg" || !got.PassThrough {
		t.Fatalf("image filter metadata = %+v, want DCTDecode jpg pass-through", got)
	}
	if !bytes.Equal(got.Content, image) {
		t.Fatalf("image content = %v, want %v", got.Content, image)
	}
	got.Content[0] = 0
	again, err := doc.ImageXObjects()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(again[0].Content, image) {
		t.Fatalf("ImageXObjects returned mutable backing bytes: %v", again[0].Content)
	}
}

func TestImageXObjectsExtractsFlateRawBytesAndMetadata(t *testing.T) {
	raw := []byte{0x10, 0x20, 0x30}
	doc, err := OpenBytes(imageXObjectPDF("/Filter /FlateDecode ", zlibCompress(t, raw)))
	if err != nil {
		t.Fatal(err)
	}

	images, err := doc.ImageXObjects()
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 1 {
		t.Fatalf("ImageXObjects() len = %d, want 1", len(images))
	}
	got := images[0]
	if !bytes.Equal(got.Content, raw) {
		t.Fatalf("Flate image content = %v, want %v", got.Content, raw)
	}
	if got.Filter != "FlateDecode" || got.Extension != "raw" || got.PassThrough {
		t.Fatalf("Flate image filter metadata = %+v, want raw decoded bytes", got)
	}
}

func TestImageXObjectsUnsupportedFilterFailsClosed(t *testing.T) {
	doc, err := OpenBytes(imageXObjectPDF("/Filter /JPXDecode ", []byte("jp2 bytes")))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := doc.ImageXObjects(); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("ImageXObjects(JPXDecode) error = %v, want ErrUnsupported", err)
	}
}

func TestInlineImagesExtractsSyntheticContentStreamImage(t *testing.T) {
	raw := []byte{0x01, 0x02, 0x03}
	doc, err := OpenBytes(inlineImagePDF("BI /W 1 /H 1 /CS /RGB /BPC 8 ID " + string(raw) + " EI"))
	if err != nil {
		t.Fatal(err)
	}

	images, err := doc.InlineImages()
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 1 {
		t.Fatalf("InlineImages() len = %d, want 1", len(images))
	}
	got := images[0]
	if got.StreamIndex != 0 || got.Width != 1 || got.Height != 1 || got.BitsPerComponent != 8 || got.ColorSpace != "DeviceRGB" {
		t.Fatalf("inline image metadata = %+v, want stream 0 1x1 DeviceRGB bpc 8", got)
	}
	if got.Filter != "" || len(got.FilterChain) != 0 || got.Extension != "raw" || got.PassThrough {
		t.Fatalf("inline image filter metadata = %+v, want raw unfiltered image", got)
	}
	if !bytes.Equal(got.Content, raw) {
		t.Fatalf("inline image content = %v, want %v", got.Content, raw)
	}
	got.Content[0] = 0
	again, err := doc.InlineImages()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(again[0].Content, raw) {
		t.Fatalf("InlineImages returned mutable backing bytes: %v", again[0].Content)
	}
}

func TestInlineImagesAreSeparateFromImageXObjectStreams(t *testing.T) {
	image := []byte{0xff, 0xd8, 0xff, 0xd9}
	doc, err := OpenBytes(imageXObjectPDF("/Filter /DCTDecode ", image))
	if err != nil {
		t.Fatal(err)
	}

	inline, err := doc.InlineImages()
	if err != nil {
		t.Fatal(err)
	}
	if len(inline) != 0 {
		t.Fatalf("InlineImages() len = %d, want 0 for image XObject-only PDF", len(inline))
	}
	xobjects, err := doc.ImageXObjects()
	if err != nil {
		t.Fatal(err)
	}
	if len(xobjects) != 1 {
		t.Fatalf("ImageXObjects() len = %d, want 1", len(xobjects))
	}
}

func TestInlineImagesUnsupportedImageFilterFailsClosed(t *testing.T) {
	doc, err := OpenBytes(inlineImagePDF("BI /W 1 /H 1 /CS /RGB /BPC 8 /F /DCT ID not-jpeg EI"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := doc.InlineImages(); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("InlineImages(filtered inline image) error = %v, want ErrUnsupported", err)
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

func inlineImagePDF(content string) []byte {
	return singleStreamPDF("", []byte(content))
}

func imageXObjectPDF(extraDict string, content []byte) []byte {
	return pdfObjects(
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /XObject << /Im1 5 0 R >> >> /Contents 4 0 R >>",
		"<< /Length 12 >>\nstream\nq /Im1 Do Q\nendstream",
		"<< /Type /XObject /Subtype /Image /Width 2 /Height 1 /ColorSpace /DeviceRGB /BitsPerComponent 8 "+extraDict+"/Length "+itoa(len(content))+" >>\nstream\n"+string(content)+"\nendstream",
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
