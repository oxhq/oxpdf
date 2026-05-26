package oxpdf

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPypdfCorpusReaderBasics(t *testing.T) {
	resources := pypdfResources(t)
	tests := []struct {
		name  string
		file  string
		pages int
	}{
		{name: "hello world", file: "hello-world.pdf", pages: 1},
		{name: "two pages", file: "two-different-pages.pdf", pages: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := OpenFile(filepath.Join(resources, tt.file))
			if err != nil {
				t.Fatal(err)
			}
			if got := doc.NumPages(); got != tt.pages {
				t.Fatalf("NumPages() = %d, want %d", got, tt.pages)
			}
			for i := 0; i < tt.pages; i++ {
				if _, err := doc.Page(i); err != nil {
					t.Fatalf("Page(%d) returned error: %v", i, err)
				}
			}
		})
	}
}

func TestPypdfCorpusEncryptedFailsClosed(t *testing.T) {
	resources := pypdfResources(t)
	_, err := OpenFile(filepath.Join(resources, "encrypted-file.pdf"))
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("OpenFile(encrypted-file.pdf) error = %v, want ErrUnsupported", err)
	}
}

func TestPypdfCorpusOpenEncryptedWithPassword(t *testing.T) {
	resources := pypdfResources(t)
	doc, err := OpenFile(filepath.Join(resources, "encryption", "r2-user-password.pdf"), WithPassword("asdfzxcv"))
	if err != nil {
		t.Fatal(err)
	}
	if got := doc.NumPages(); got != 1 {
		t.Fatalf("NumPages() = %d, want 1", got)
	}
	security := doc.Security()
	if !security.Encrypted || !security.Encryption.Present {
		t.Fatalf("Security() = %+v, want encrypted metadata", security)
	}
}

func TestPypdfCorpusOpenEncryptedWrongPasswordFailsClosed(t *testing.T) {
	resources := pypdfResources(t)
	_, err := OpenFile(filepath.Join(resources, "encryption", "r2-user-password.pdf"), WithPassword("wrong"))
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("OpenFile(encrypted wrong password) error = %v, want ErrUnsupported", err)
	}
	if err != nil && strings.Contains(err.Error(), "wrong") {
		t.Fatalf("wrong password leaked in error: %v", err)
	}
}

func TestPypdfCorpusSecurityMetadata(t *testing.T) {
	resources := pypdfResources(t)
	plainBytes, err := os.ReadFile(filepath.Join(resources, "hello-world.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	plain, err := OpenBytes(plainBytes)
	if err != nil {
		t.Fatal(err)
	}
	if security := plain.Security(); security.Encrypted || security.Signed || security.Encryption.Present || security.Signature.Present {
		t.Fatalf("plain Security() = %+v", security)
	}

	encryptedBytes, err := os.ReadFile(filepath.Join(resources, "encryption", "r2-user-password.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	encrypted := (&Document{input: encryptedBytes}).Security()
	if !encrypted.Encrypted || !encrypted.Encryption.Present {
		t.Fatalf("encrypted Security() = %+v", encrypted)
	}
	if encrypted.Encryption.Filter != "Standard" || encrypted.Encryption.V != 1 || encrypted.Encryption.R != 2 || encrypted.Encryption.Length != 40 {
		t.Fatalf("encrypted Encryption = %+v", encrypted.Encryption)
	}
	if encrypted.Signed || encrypted.Signature.Present {
		t.Fatalf("encrypted Signature = %+v", encrypted.Signature)
	}
}

func TestPypdfCorpusMetadata(t *testing.T) {
	resources := pypdfResources(t)
	doc, err := OpenFile(filepath.Join(resources, "metadata.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	metadata := doc.Metadata()
	if metadata.Title != "The Title" {
		t.Fatalf("Metadata().Title = %q", metadata.Title)
	}
	if metadata.Author != "Martin Thoma" {
		t.Fatalf("Metadata().Author = %q", metadata.Author)
	}
	if metadata.Subject != "The Subject" {
		t.Fatalf("Metadata().Subject = %q", metadata.Subject)
	}
	if metadata.Keywords != "Some Keywords, other keywords; more keywords" {
		t.Fatalf("Metadata().Keywords = %q", metadata.Keywords)
	}
	if metadata.Creator != "pdflatex, or other tool" {
		t.Fatalf("Metadata().Creator = %q", metadata.Creator)
	}
	if metadata.Producer != "Latex with hyperref, or other system" {
		t.Fatalf("Metadata().Producer = %q", metadata.Producer)
	}
	if metadata.CreationDate != "D:20220415093243+02'00'" {
		t.Fatalf("Metadata().CreationDate = %q", metadata.CreationDate)
	}
	if metadata.ModDate != "D:20220415093243+02'00'" {
		t.Fatalf("Metadata().ModDate = %q", metadata.ModDate)
	}
	if metadata.Values["Trapped"] != "/False" {
		t.Fatalf("Metadata().Values[Trapped] = %q", metadata.Values["Trapped"])
	}
}

func TestPypdfCorpusMissingInfoHasEmptyMetadata(t *testing.T) {
	resources := pypdfResources(t)
	doc, err := OpenFile(filepath.Join(resources, "missing_info.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	metadata := doc.Metadata()
	if len(metadata.Values) != 0 {
		t.Fatalf("Metadata().Values len = %d, want 0: %#v", len(metadata.Values), metadata.Values)
	}
}

func TestPypdfCorpusXMPMetadata(t *testing.T) {
	resources := pypdfResources(t)
	doc, err := OpenFile(filepath.Join(resources, "commented-xmp.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	xmp, ok, err := doc.XMPMetadata()
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("XMPMetadata() ok = false, want true")
	}
	if !strings.Contains(xmp.RawXML, "<x:xmpmeta") {
		t.Fatalf("XMPMetadata().RawXML missing xmpmeta: %.80q", xmp.RawXML)
	}
	if !sameStrings(xmp.TIFFArtist, []string{"me"}) {
		t.Fatalf("XMPMetadata().TIFFArtist = %v, want [me]", xmp.TIFFArtist)
	}

	issue, err := OpenFile(filepath.Join(resources, "issue-914-xmp-data.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	issueXMP, ok, err := issue.XMPMetadata()
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("issue XMPMetadata() ok = false, want true")
	}
	if issueXMP.ModifyDate != "2022-04-09T15:22:43" {
		t.Fatalf("XMPMetadata().ModifyDate = %q, want UTC-normalized value", issueXMP.ModifyDate)
	}
}

func TestPypdfCorpusMissingXMPMetadata(t *testing.T) {
	resources := pypdfResources(t)
	doc, err := OpenFile(filepath.Join(resources, "metadata.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	xmp, ok, err := doc.XMPMetadata()
	if err != nil {
		t.Fatal(err)
	}
	if ok || xmp.RawXML != "" {
		t.Fatalf("XMPMetadata() = %+v, %v, want empty false", xmp, ok)
	}
}

func TestPypdfCorpusPageBoxesFallbackToMediaBox(t *testing.T) {
	resources := pypdfResources(t)
	doc, err := OpenFile(filepath.Join(resources, "box.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	page, err := doc.Page(0)
	if err != nil {
		t.Fatal(err)
	}
	want := Rectangle{Left: 0, Bottom: 0, Right: 60, Top: 60}
	assertRect(t, "MediaBox", page.MediaBox(), want)
	assertRect(t, "CropBox", page.CropBox(), want)
	assertRect(t, "BleedBox", page.BleedBox(), want)
	assertRect(t, "TrimBox", page.TrimBox(), want)
	assertRect(t, "ArtBox", page.ArtBox(), want)
	if got := page.Rotation(); got != 0 {
		t.Fatalf("Rotation() = %d, want 0", got)
	}
}

func TestPypdfCorpusIndirectRotation(t *testing.T) {
	resources := pypdfResources(t)
	doc, err := OpenFile(filepath.Join(resources, "indirect-rotation.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	if got := doc.NumPages(); got != 5 {
		t.Fatalf("NumPages() = %d, want 5", got)
	}
	want := Rectangle{Left: 0, Bottom: 0, Right: 612, Top: 792}
	for i := 0; i < doc.NumPages(); i++ {
		page, err := doc.Page(i)
		if err != nil {
			t.Fatal(err)
		}
		assertRect(t, "MediaBox", page.MediaBox(), want)
		if got := page.Rotation(); got != 0 {
			t.Fatalf("Page(%d).Rotation() = %d, want 0", i, got)
		}
	}
}

func TestPypdfCorpusFormAndAnnotationProfileBoundaries(t *testing.T) {
	resources := pypdfResources(t)
	tests := []struct {
		file                string
		fillable            bool
		formFields          int
		formFillable        int
		annotationsPresent  bool
		annotationCount     int
		editableAnnotations int
		blockedAnnotations  int
		annotationsMakeEdit bool
	}{
		{
			file:                "libreoffice-form.pdf",
			fillable:            true,
			formFields:          8,
			formFillable:        8,
			annotationsPresent:  true,
			annotationCount:     9,
			blockedAnnotations:  9,
			annotationsMakeEdit: false,
		},
		{
			file:                "commented.pdf",
			fillable:            false,
			annotationsPresent:  true,
			annotationCount:     6,
			editableAnnotations: 3,
			blockedAnnotations:  3,
			annotationsMakeEdit: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			doc, err := OpenFile(filepath.Join(resources, tt.file))
			if err != nil {
				t.Fatal(err)
			}
			profile := doc.Profile()
			if profile.Fillable != tt.fillable {
				t.Fatalf("Profile().Fillable = %v, want %v", profile.Fillable, tt.fillable)
			}
			if profile.Forms.FieldCount != tt.formFields {
				t.Fatalf("Profile().Forms.FieldCount = %d, want %d", profile.Forms.FieldCount, tt.formFields)
			}
			if profile.Forms.FillableCount != tt.formFillable {
				t.Fatalf("Profile().Forms.FillableCount = %d, want %d", profile.Forms.FillableCount, tt.formFillable)
			}
			if profile.Annotations.Present != tt.annotationsPresent {
				t.Fatalf("Profile().Annotations.Present = %v, want %v", profile.Annotations.Present, tt.annotationsPresent)
			}
			if profile.Annotations.Count != tt.annotationCount {
				t.Fatalf("Profile().Annotations.Count = %d, want %d", profile.Annotations.Count, tt.annotationCount)
			}
			if profile.Annotations.EditableCount != tt.editableAnnotations {
				t.Fatalf("Profile().Annotations.EditableCount = %d, want %d", profile.Annotations.EditableCount, tt.editableAnnotations)
			}
			if profile.Annotations.BlockerCount != tt.blockedAnnotations {
				t.Fatalf("Profile().Annotations.BlockerCount = %d, want %d", profile.Annotations.BlockerCount, tt.blockedAnnotations)
			}
			if (profile.Annotations.EditableCount > 0) != tt.annotationsMakeEdit {
				t.Fatalf("Profile().Annotations editability mismatch: %+v", profile.Annotations)
			}
		})
	}
}

func TestFormMutationMissingFieldFailsClosed(t *testing.T) {
	doc, err := OpenBytes(blankPDF([]PageSize{PageSizeLetter}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := doc.Fill(map[string]string{"name": "Ada"}); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Fill() error = %v, want ErrUnsupported", err)
	}
	if _, err := doc.SetCheckbox("agree", true); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("SetCheckbox() error = %v, want ErrUnsupported", err)
	}
}

func TestPypdfCorpusFieldsAndTextFields(t *testing.T) {
	resources := pypdfResources(t)
	doc, err := OpenFile(filepath.Join(resources, "libreoffice-form.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	fields, err := doc.Fields()
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 8 {
		t.Fatalf("Fields() len = %d, want 8", len(fields))
	}
	assertField(t, fields[0], Field{Name: "First Name", Type: "Tx", Value: "Alice", Status: "supported"})
	assertField(t, fields[1], Field{Name: "Last Name", Type: "Tx", Value: "", Status: "supported"})
	assertField(t, fields[2], Field{Name: "female", Type: "Btn", Value: "Off", Status: "supported"})
	assertField(t, fields[7], Field{Name: "Nationality", Type: "Ch", Value: "", Status: "supported"})

	textFields, err := doc.TextFields()
	if err != nil {
		t.Fatal(err)
	}
	if len(textFields) != 4 {
		t.Fatalf("TextFields() len = %d, want 4", len(textFields))
	}
	gotNames := make([]string, 0, len(textFields))
	for _, field := range textFields {
		gotNames = append(gotNames, field.Name)
		if field.Type != "Tx" {
			t.Fatalf("TextFields() returned non-text field: %+v", field)
		}
	}
	wantNames := []string{"First Name", "Last Name", "Birthday", "First Name_2"}
	for i := range wantNames {
		if gotNames[i] != wantNames[i] {
			t.Fatalf("TextFields()[%d].Name = %q, want %q", i, gotNames[i], wantNames[i])
		}
	}
}

func assertField(t *testing.T, got Field, want Field) {
	t.Helper()
	if got.Name != want.Name || got.Type != want.Type || got.Value != want.Value || got.Status != want.Status {
		t.Fatalf("field = %+v, want at least %+v", got, want)
	}
}

func TestPypdfCorpusFillTextFieldsAndCheckboxes(t *testing.T) {
	resources := pypdfResources(t)
	doc, err := OpenFile(filepath.Join(resources, "libreoffice-form.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	out, err := doc.Fill(map[string]string{
		"Last Name": "Lovelace",
		"Birthday":  "1815-12-10",
	})
	if err != nil {
		t.Fatal(err)
	}
	filled, err := OpenBytes(out)
	if err != nil {
		t.Fatal(err)
	}
	assertFieldValue(t, filled, "Last Name", "Lovelace")
	assertFieldValue(t, filled, "Birthday", "1815-12-10")

	out, err = filled.SetCheckbox("gdpr", true)
	if err != nil {
		t.Fatal(err)
	}
	checked, err := OpenBytes(out)
	if err != nil {
		t.Fatal(err)
	}
	assertFieldValue(t, checked, "gdpr", "Yes")

	out, err = checked.SetCheckbox("gdpr", false)
	if err != nil {
		t.Fatal(err)
	}
	unchecked, err := OpenBytes(out)
	if err != nil {
		t.Fatal(err)
	}
	assertFieldValue(t, unchecked, "gdpr", "Off")
}

func TestPypdfCorpusButtonChoiceAndCheckboxSemantics(t *testing.T) {
	resources := pypdfResources(t)
	doc, err := OpenFile(filepath.Join(resources, "libreoffice-form.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := doc.SetCheckbox("female", true); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("SetCheckbox(radio) error = %v, want ErrUnsupported", err)
	}

	for _, state := range []string{"1", "2", "Off"} {
		out, err := doc.SetButtonChoice("female", state)
		if err != nil {
			t.Fatalf("SetButtonChoice(%q) returned error: %v", state, err)
		}
		updated, err := OpenBytes(out)
		if err != nil {
			t.Fatal(err)
		}
		assertFieldValue(t, updated, "female", state)
	}
	if _, err := doc.SetButtonChoice("female", "Yes"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("SetButtonChoice(invalid state) error = %v, want ErrUnsupported", err)
	}
}

func TestPypdfCorpusPdflatexFormUnicodeNameBoundary(t *testing.T) {
	resources := pypdfResources(t)
	doc, err := OpenFile(filepath.Join(resources, "pdflatex-forms.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	fields, err := doc.Fields()
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 3 {
		t.Fatalf("Fields() len = %d, want 3", len(fields))
	}
	if fields[0].Name == "Name" || fields[1].Name == "Check" {
		t.Fatalf("expected current backing field names to preserve undecoded PDF text: %+v", fields[:2])
	}
	if _, err := doc.Fill(map[string]string{"Name": "Ada"}); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Fill(decoded name) error = %v, want ErrUnsupported", err)
	}
	if _, err := doc.SetCheckbox("Check", true); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("SetCheckbox(decoded name) error = %v, want ErrUnsupported", err)
	}
	if _, err := doc.SetButtonChoice("Submit", "Yes"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("SetButtonChoice(pushbutton) error = %v, want ErrUnsupported", err)
	}
}

func assertFieldValue(t *testing.T, doc *Document, name string, want string) {
	t.Helper()
	fields, err := doc.Fields()
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range fields {
		if field.Name == name {
			if field.Value != want {
				t.Fatalf("field %q value = %q, want %q", name, field.Value, want)
			}
			return
		}
	}
	t.Fatalf("field %q not found", name)
}

func TestPypdfCorpusPageAnnotations(t *testing.T) {
	resources := pypdfResources(t)
	doc, err := OpenFile(filepath.Join(resources, "commented.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	page, err := doc.Page(0)
	if err != nil {
		t.Fatal(err)
	}
	annotations, err := page.Annotations()
	if err != nil {
		t.Fatal(err)
	}
	if len(annotations) != 6 {
		t.Fatalf("Annotations() len = %d, want 6", len(annotations))
	}
	assertAnnotation(t, annotations[0], Annotation{
		Kind:     "Text",
		Contents: "Note in second paragraph",
		Title:    "moose",
		Status:   "approximate_supported",
		Rect:     []float64{270.75, 596.25, 294.75, 620.25},
	})
	assertAnnotation(t, annotations[1], Annotation{
		Kind:     "Popup",
		Status:   "unsupported",
		Blockers: []string{"unsupported_subtype"},
	})
	assertAnnotation(t, annotations[2], Annotation{
		Kind:   "Highlight",
		Title:  "moose",
		Status: "approximate_supported",
		Rect:   []float64{176, 557, 203, 568},
	})
	if annotations[4].Kind != "Text" || annotations[4].Title != "moose" {
		t.Fatalf("annotation 4 = %+v", annotations[4])
	}
	if annotations[4].Contents == "" || annotations[4].Contents == "note over \"kinds\"" {
		t.Fatalf("annotation 4 contents were not decoded fully: %q", annotations[4].Contents)
	}
}

func TestPypdfCorpusSetAnnotationContents(t *testing.T) {
	resources := pypdfResources(t)
	doc, err := OpenFile(filepath.Join(resources, "commented.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	out, verification, err := doc.SetAnnotationContents(0, "updated note")
	if err != nil {
		t.Fatal(err)
	}
	if !verification.ReparseOK || !verification.ContentsUpdated || !verification.PageUnchanged || verification.AppearanceRegenerated {
		t.Fatalf("verification = %+v", verification)
	}
	updated, err := OpenBytes(out)
	if err != nil {
		t.Fatal(err)
	}
	page, err := updated.Page(0)
	if err != nil {
		t.Fatal(err)
	}
	annotations, err := page.Annotations()
	if err != nil {
		t.Fatal(err)
	}
	if annotations[0].Contents != "updated note" {
		t.Fatalf("annotation 0 contents = %q", annotations[0].Contents)
	}
}

func TestPypdfCorpusSetAnnotationContentsRegeneratesSupportedAppearance(t *testing.T) {
	resources := pypdfResources(t)
	doc, err := OpenFile(filepath.Join(resources, "commented.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	for _, index := range []int{0, 2, 4} {
		out, verification, err := doc.SetAnnotationContents(index, "updated", AnnotationContentsEditOptions{RegenerateAppearance: true})
		if err != nil {
			t.Fatalf("SetAnnotationContents(%d) returned error: %v", index, err)
		}
		if !verification.AppearanceRegenerated {
			t.Fatalf("SetAnnotationContents(%d) verification = %+v, want regenerated appearance", index, verification)
		}
		updated, err := OpenBytes(out)
		if err != nil {
			t.Fatal(err)
		}
		page, err := updated.Page(0)
		if err != nil {
			t.Fatal(err)
		}
		annotations, err := page.Annotations()
		if err != nil {
			t.Fatal(err)
		}
		if annotations[index].Contents != "updated" {
			t.Fatalf("annotation %d contents = %q", index, annotations[index].Contents)
		}
	}
}

func TestPypdfCorpusSetAnnotationContentsRejectsUnsupportedPopup(t *testing.T) {
	resources := pypdfResources(t)
	doc, err := OpenFile(filepath.Join(resources, "commented.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	for _, index := range []int{1, 3, 5} {
		if _, _, err := doc.SetAnnotationContents(index, "updated", AnnotationContentsEditOptions{RegenerateAppearance: true}); !errors.Is(err, ErrUnsupported) {
			t.Fatalf("SetAnnotationContents(%d) error = %v, want ErrUnsupported", index, err)
		}
	}
}

func assertAnnotation(t *testing.T, got Annotation, want Annotation) {
	t.Helper()
	if got.Kind != want.Kind || got.Contents != want.Contents || got.Title != want.Title || got.Status != want.Status {
		t.Fatalf("annotation = %+v, want at least %+v", got, want)
	}
	if len(want.Blockers) > 0 && !sameStrings(got.Blockers, want.Blockers) {
		t.Fatalf("annotation blockers = %v, want %v", got.Blockers, want.Blockers)
	}
	if len(want.Rect) > 0 && !sameFloat64s(got.Rect, want.Rect) {
		t.Fatalf("annotation rect = %v, want %v", got.Rect, want.Rect)
	}
}

func sameStrings(a, b []string) bool {
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

func sameFloat64s(a, b []float64) bool {
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

func assertRect(t *testing.T, name string, got Rectangle, want Rectangle) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %+v, want %+v", name, got, want)
	}
}

func pypdfResources(t *testing.T) string {
	t.Helper()
	resources := `C:\Users\garae\Documents\pypdf\resources`
	if _, err := os.Stat(resources); err != nil {
		t.Skipf("local pypdf resources not available: %v", err)
	}
	return resources
}
