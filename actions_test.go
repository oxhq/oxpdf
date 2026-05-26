package oxpdf

import (
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
