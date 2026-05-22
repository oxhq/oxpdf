package oxpdf

import (
	"errors"
	"os"
	"path/filepath"
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

func TestFormAndAnnotationAPIsRemainExplicitlyUnsupported(t *testing.T) {
	doc, err := OpenBytes(blankPDF([]PageSize{PageSizeLetter}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := doc.Fill(map[string]string{"name": "Ada"}); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Fill() error = %v, want ErrUnsupported", err)
	}
	page, err := doc.Page(0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := page.Annotations(); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Annotations() error = %v, want ErrUnsupported", err)
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
