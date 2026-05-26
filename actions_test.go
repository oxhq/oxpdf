package oxpdf

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestPypdfCorpusJavaScriptActions(t *testing.T) {
	resources := pypdfResources(t)
	doc, err := OpenFile(filepath.Join(resources, "issue-297.pdf"))
	if err != nil {
		t.Fatal(err)
	}

	actions, err := doc.JavaScriptActions()
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 {
		t.Fatalf("JavaScriptActions() len = %d, want 1: %+v", len(actions), actions)
	}
	if actions[0].ObjectNumber != 7 {
		t.Fatalf("JavaScriptActions()[0].ObjectNumber = %d, want 7", actions[0].ObjectNumber)
	}
	if !strings.Contains(actions[0].Script, "app.alert") || !strings.Contains(actions[0].Script, "Hello alert") {
		t.Fatalf("JavaScriptActions()[0].Script = %q", actions[0].Script)
	}

	plain, err := OpenFile(filepath.Join(resources, "hello-world.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	actions, err = plain.JavaScriptActions()
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 0 {
		t.Fatalf("plain JavaScriptActions() = %+v, want empty", actions)
	}
}

func TestJavaScriptNameTreeActionsRequireCatalogReachability(t *testing.T) {
	doc, err := OpenBytes(syntheticJavaScriptNameTreePDF())
	if err != nil {
		t.Fatal(err)
	}

	direct, err := doc.JavaScriptActions()
	if err != nil {
		t.Fatal(err)
	}
	if len(direct) != 2 {
		t.Fatalf("JavaScriptActions() len = %d, want blind scan of 2 direct actions: %+v", len(direct), direct)
	}

	reachable, err := doc.JavaScriptNameTreeActions()
	if err != nil {
		t.Fatal(err)
	}
	if len(reachable) != 1 {
		t.Fatalf("JavaScriptNameTreeActions() len = %d, want 1 reachable action: %+v", len(reachable), reachable)
	}
	action := reachable[0]
	if action.Name != "ReachableJS" || action.Source != "name-tree" || action.ObjectNumber != 7 {
		t.Fatalf("reachable JavaScript action metadata = %+v", action)
	}
	if !strings.Contains(action.Script, "reachable") {
		t.Fatalf("reachable JavaScript action script = %q", action.Script)
	}
}

func TestPypdfCorpusAttachments(t *testing.T) {
	resources := pypdfResources(t)
	doc, err := OpenFile(filepath.Join(resources, "attachment.pdf"))
	if err != nil {
		t.Fatal(err)
	}

	attachments, err := doc.Attachments()
	if err != nil {
		t.Fatal(err)
	}
	if len(attachments) != 1 {
		t.Fatalf("Attachments() len = %d, want 1: %+v", len(attachments), attachments)
	}
	attachment := attachments[0]
	if attachment.Name != "jpeg.pdf" {
		t.Fatalf("Attachment.Name = %q, want jpeg.pdf", attachment.Name)
	}
	if attachment.ObjectNumber != 3 || attachment.Filter != "FlateDecode" {
		t.Fatalf("Attachment metadata = %+v", attachment)
	}
	if attachment.Size != len(attachment.Content) || attachment.Size != 100898 {
		t.Fatalf("Attachment size/content = %d/%d, want 100898", attachment.Size, len(attachment.Content))
	}
	if !strings.HasPrefix(string(attachment.Content), "%PDF-") {
		t.Fatalf("Attachment content prefix = %.12q, want PDF header", attachment.Content)
	}

	plain, err := OpenFile(filepath.Join(resources, "hello-world.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	attachments, err = plain.Attachments()
	if err != nil {
		t.Fatal(err)
	}
	if len(attachments) != 0 {
		t.Fatalf("plain Attachments() = %+v, want empty", attachments)
	}
}

func TestAttachmentNameWithoutWhitespace(t *testing.T) {
	payload := []byte("Hello World!\n\nLorem ipsum")
	doc, err := OpenBytes(syntheticAttachmentPDF(t, "factur-x.xml", payload))
	if err != nil {
		t.Fatal(err)
	}

	attachments, err := doc.Attachments()
	if err != nil {
		t.Fatal(err)
	}
	if len(attachments) != 1 {
		t.Fatalf("Attachments() len = %d, want 1: %+v", len(attachments), attachments)
	}
	attachment := attachments[0]
	if attachment.Name != "factur-x.xml" {
		t.Fatalf("Attachment.Name = %q, want factur-x.xml", attachment.Name)
	}
	if attachment.ObjectNumber != 4 || attachment.Filter != "FlateDecode" {
		t.Fatalf("Attachment metadata = %+v", attachment)
	}
	if !bytes.Equal(attachment.Content, payload) || attachment.Size != len(payload) {
		t.Fatalf("Attachment content = %q size %d, want %q size %d", attachment.Content, attachment.Size, payload, len(payload))
	}
}

func TestAttachmentNameTreeRequiresCatalogReachability(t *testing.T) {
	payload := []byte("reachable attachment")
	doc, err := OpenBytes(syntheticAttachmentNameTreePDF(t, payload, []byte("orphan attachment")))
	if err != nil {
		t.Fatal(err)
	}

	direct, err := doc.Attachments()
	if err != nil {
		t.Fatal(err)
	}
	if len(direct) != 2 {
		t.Fatalf("Attachments() len = %d, want blind scan of 2 direct filespecs: %+v", len(direct), direct)
	}

	reachable, err := doc.AttachmentNameTree()
	if err != nil {
		t.Fatal(err)
	}
	if len(reachable) != 1 {
		t.Fatalf("AttachmentNameTree() len = %d, want 1 reachable attachment: %+v", len(reachable), reachable)
	}
	attachment := reachable[0]
	if attachment.Name != "reachable.txt" || attachment.Source != "name-tree" || attachment.ObjectNumber != 8 {
		t.Fatalf("reachable attachment metadata = %+v", attachment)
	}
	if !bytes.Equal(attachment.Content, payload) || attachment.Size != len(payload) {
		t.Fatalf("reachable attachment content = %q size %d, want %q size %d", attachment.Content, attachment.Size, payload, len(payload))
	}
}

func syntheticJavaScriptNameTreePDF() []byte {
	return syntheticPDFObjects(
		[]byte("<< /Type /Catalog /Pages 2 0 R /Names 4 0 R >>"),
		[]byte("<< /Type /Pages /Kids [3 0 R] /Count 1 >>"),
		[]byte("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>"),
		[]byte("<< /JavaScript 5 0 R >>"),
		[]byte("<< /Kids [6 0 R] >>"),
		[]byte("<< /Names [(ReachableJS) 7 0 R] >>"),
		[]byte("<< /S /JavaScript /JS (app.alert('reachable')) >>"),
		[]byte("<< /S /JavaScript /JS (app.alert('orphan')) >>"),
	)
}

func syntheticAttachmentNameTreePDF(t *testing.T, reachablePayload, orphanPayload []byte) []byte {
	t.Helper()
	reachableStream := compressedEmbeddedFileObject(t, reachablePayload)
	orphanStream := compressedEmbeddedFileObject(t, orphanPayload)
	return syntheticPDFObjects(
		[]byte("<< /Type /Catalog /Pages 2 0 R /Names 4 0 R >>"),
		[]byte("<< /Type /Pages /Kids [3 0 R] /Count 1 >>"),
		[]byte("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>"),
		[]byte("<< /EmbeddedFiles 5 0 R >>"),
		[]byte("<< /Kids [6 0 R] >>"),
		[]byte("<< /Names [(reachable.txt) 7 0 R] >>"),
		[]byte("<< /Type /Filespec /F (reachable.txt) /EF << /F 8 0 R >> >>"),
		reachableStream,
		[]byte("<< /Type /Filespec /F (orphan.txt) /EF << /F 10 0 R >> >>"),
		orphanStream,
	)
}

func syntheticAttachmentPDF(t *testing.T, name string, payload []byte) []byte {
	t.Helper()
	embeddedFile := compressedEmbeddedFileObject(t, payload)
	objects := [][]byte{
		[]byte("<< /Type /Catalog /Pages 2 0 R >>"),
		[]byte("<< /Type /Pages /Kids [3 0 R] /Count 1 >>"),
		[]byte("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Annots [5 0 R] >>"),
		embeddedFile,
		[]byte(fmt.Sprintf("<< /Type /Filespec /F(%s) /UF(%s) /EF<< /F 4 0 R /UF 4 0 R >> >>", name, name)),
		[]byte("<< /Type /Annot /Subtype /FileAttachment /Rect [0 0 10 10] /FS 5 0 R >>"),
	}
	return syntheticPDFObjects(objects...)
}

func compressedEmbeddedFileObject(t *testing.T, payload []byte) []byte {
	t.Helper()
	var compressed bytes.Buffer
	writer := zlib.NewWriter(&compressed)
	if _, err := writer.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return bytes.Join([][]byte{
		[]byte(fmt.Sprintf("<< /Type /EmbeddedFile /Length %d /Filter /FlateDecode >>\nstream\n", compressed.Len())),
		compressed.Bytes(),
		[]byte("\nendstream"),
	}, nil)
}

func syntheticPDFObjects(objects ...[]byte) []byte {
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
