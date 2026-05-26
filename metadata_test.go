package oxpdf

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestXMPMetadataCommonFieldsSyntheticPDF(t *testing.T) {
	raw := `<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description
      xmlns:dc="http://purl.org/dc/elements/1.1/"
      xmlns:xmp="http://ns.adobe.com/xap/1.0/"
      xmlns:pdf="http://ns.adobe.com/pdf/1.3/"
      xmp:CreateDate="2022-04-09T15:22:43+02:00"
      xmp:CreatorTool="OxPDF fixture writer"
      pdf:Producer="OxPDF synthetic producer"
      pdf:Keywords="alpha, beta">
      <dc:title>
        <rdf:Alt>
          <rdf:li xml:lang="x-default">Synthetic Title</rdf:li>
        </rdf:Alt>
      </dc:title>
      <dc:creator>
        <rdf:Seq>
          <rdf:li>Alice Example</rdf:li>
          <rdf:li>Bob Example</rdf:li>
        </rdf:Seq>
      </dc:creator>
      <dc:description>
        <rdf:Alt>
          <rdf:li xml:lang="x-default">Synthetic description</rdf:li>
        </rdf:Alt>
      </dc:description>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`
	rawXML, ok, err := extractXMPMetadataXML(metadataSyntheticPDFObjects(
		[]byte("<< /Type /Catalog /Pages 2 0 R /Metadata 4 0 R >>"),
		[]byte("<< /Type /Pages /Kids [3 0 R] /Count 1 >>"),
		[]byte("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>"),
		[]byte(fmt.Sprintf("<< /Type /Metadata /Subtype /XML /Length %d >>\nstream\n%s\nendstream", len(raw), raw)),
	))
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("extractXMPMetadataXML() ok = false, want true")
	}
	xmp := parseXMPMetadata(rawXML)
	if !strings.Contains(xmp.RawXML, "<x:xmpmeta") {
		t.Fatalf("XMPMetadata().RawXML missing packet: %.80q", xmp.RawXML)
	}
	if xmp.Title != "Synthetic Title" {
		t.Fatalf("XMPMetadata().Title = %q", xmp.Title)
	}
	if !metadataSameStrings(xmp.Creator, []string{"Alice Example", "Bob Example"}) {
		t.Fatalf("XMPMetadata().Creator = %v", xmp.Creator)
	}
	if xmp.Description != "Synthetic description" {
		t.Fatalf("XMPMetadata().Description = %q", xmp.Description)
	}
	if xmp.CreateDate != "2022-04-09T13:22:43" {
		t.Fatalf("XMPMetadata().CreateDate = %q, want UTC-normalized value", xmp.CreateDate)
	}
	if xmp.CreatorTool != "OxPDF fixture writer" {
		t.Fatalf("XMPMetadata().CreatorTool = %q", xmp.CreatorTool)
	}
	if xmp.Producer != "OxPDF synthetic producer" {
		t.Fatalf("XMPMetadata().Producer = %q", xmp.Producer)
	}
	if xmp.Keywords != "alpha, beta" {
		t.Fatalf("XMPMetadata().Keywords = %q", xmp.Keywords)
	}
}

func TestParseXMPMetadataCommonFieldsByLocalName(t *testing.T) {
	raw := `<meta:xmpmeta xmlns:meta="adobe:ns:meta/">
  <r:RDF xmlns:r="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <r:Description xmlns:d="urn:test-dc" xmlns:x="urn:test-xmp" xmlns:p="urn:test-pdf">
      <d:title>Loose Title</d:title>
      <d:creator>Loose Creator</d:creator>
      <d:description>Loose Description</d:description>
      <x:CreateDate>2022-04-09T15:22:43</x:CreateDate>
      <x:CreatorTool>Loose Tool</x:CreatorTool>
      <p:Producer>Loose Producer</p:Producer>
      <p:Keywords>loose keywords</p:Keywords>
    </r:Description>
  </r:RDF>
</meta:xmpmeta>`

	xmp := parseXMPMetadata(raw)
	if xmp.RawXML != raw {
		t.Fatal("parseXMPMetadata().RawXML did not preserve input")
	}
	if xmp.Title != "Loose Title" ||
		!metadataSameStrings(xmp.Creator, []string{"Loose Creator"}) ||
		xmp.Description != "Loose Description" ||
		xmp.CreateDate != "2022-04-09T15:22:43" ||
		xmp.CreatorTool != "Loose Tool" ||
		xmp.Producer != "Loose Producer" ||
		xmp.Keywords != "loose keywords" {
		t.Fatalf("parseXMPMetadata() = %+v", xmp)
	}
}

func TestSetMetadataWritesInfoAndXMPWithReparseProof(t *testing.T) {
	input := metadataSyntheticPDFObjects(
		[]byte("<< /Type /Catalog /Pages 2 0 R >>"),
		[]byte("<< /Type /Pages /Kids [3 0 R] /Count 1 >>"),
		[]byte("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>"),
	)
	doc, err := OpenBytes(input)
	if err != nil {
		t.Fatal(err)
	}
	rawXMP := `<x:xmpmeta xmlns:x="adobe:ns:meta/"><rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"><rdf:Description xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title><rdf:Alt><rdf:li xml:lang="x-default">Written XMP</rdf:li></rdf:Alt></dc:title></rdf:Description></rdf:RDF></x:xmpmeta>`

	out, verification, err := doc.SetMetadata(MetadataWrite{
		Info: map[string]string{
			"Title":  "Written Title",
			"Author": "Alice Example",
		},
		XMPRawXML: rawXMP,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !verification.ReparseOK || !verification.InfoUpdated || !verification.XMPUpdated {
		t.Fatalf("verification = %+v", verification)
	}
	reopened, err := OpenBytes(out)
	if err != nil {
		t.Fatal(err)
	}
	metadata := reopened.Metadata()
	if metadata.Title != "Written Title" || metadata.Author != "Alice Example" {
		t.Fatalf("Metadata() = %+v", metadata)
	}
	xmp, ok, err := reopened.XMPMetadata()
	if err != nil {
		t.Fatal(err)
	}
	if !ok || xmp.RawXML != rawXMP || xmp.Title != "Written XMP" {
		t.Fatalf("XMPMetadata() = %+v ok=%v", xmp, ok)
	}
}

func TestSetMetadataRejectsUnsafeInfoKeyWithoutChangingBytes(t *testing.T) {
	input := metadataSyntheticPDFObjects(
		[]byte("<< /Type /Catalog /Pages 2 0 R >>"),
		[]byte("<< /Type /Pages /Kids [3 0 R] /Count 1 >>"),
		[]byte("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>"),
	)
	doc, err := OpenBytes(input)
	if err != nil {
		t.Fatal(err)
	}

	out, verification, err := doc.SetMetadata(MetadataWrite{
		Info: map[string]string{"Bad Key": "value"},
	})
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("SetMetadata() error = %v, want ErrUnsupported", err)
	}
	if out != nil {
		t.Fatalf("SetMetadata() returned %d bytes on rejected write", len(out))
	}
	if verification.ReparseOK || verification.InfoUpdated || verification.XMPUpdated {
		t.Fatalf("verification = %+v, want zero value", verification)
	}
	if !bytes.Equal(doc.Bytes(), input) {
		t.Fatal("document bytes changed after rejected write")
	}
}

func TestSetMetadataRejectsUnsafeXMPStreamMarker(t *testing.T) {
	input := metadataSyntheticPDFObjects(
		[]byte("<< /Type /Catalog /Pages 2 0 R >>"),
		[]byte("<< /Type /Pages /Kids [3 0 R] /Count 1 >>"),
		[]byte("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>"),
	)
	doc, err := OpenBytes(input)
	if err != nil {
		t.Fatal(err)
	}

	out, verification, err := doc.SetMetadata(MetadataWrite{
		XMPRawXML: `<x:xmpmeta xmlns:x="adobe:ns:meta/">endstream</x:xmpmeta>`,
	})
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("SetMetadata() error = %v, want ErrUnsupported", err)
	}
	if !strings.Contains(err.Error(), "stream marker") {
		t.Fatalf("SetMetadata() error = %v, want stream marker rejection", err)
	}
	if out != nil {
		t.Fatalf("SetMetadata() returned %d bytes on rejected XMP", len(out))
	}
	if verification.ReparseOK || verification.InfoUpdated || verification.XMPUpdated {
		t.Fatalf("verification = %+v, want zero value", verification)
	}
}

func metadataSyntheticPDFObjects(objects ...[]byte) []byte {
	var body bytes.Buffer
	offsets := []int{0}
	for i, object := range objects {
		offsets = append(offsets, body.Len())
		fmt.Fprintf(&body, "%d 0 obj\n", i+1)
		body.Write(object)
		body.WriteString("\nendobj\n")
	}
	var out bytes.Buffer
	out.WriteString("%PDF-1.7\n")
	base := out.Len()
	out.Write(body.Bytes())
	xrefAt := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n", len(offsets))
	out.WriteString("0000000000 65535 f \n")
	for i := 1; i < len(offsets); i++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", base+offsets[i])
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets), xrefAt)
	return out.Bytes()
}

func metadataSameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
