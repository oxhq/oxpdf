package oxpdf

import (
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

func findStreamByFilter(streams []Stream, filter string) (Stream, bool) {
	for _, stream := range streams {
		if stream.Filter == filter {
			return stream, true
		}
	}
	return Stream{}, false
}
