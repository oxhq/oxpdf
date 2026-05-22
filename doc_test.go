package oxpdf

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenBytesProfileAndValidateSyntheticPDF(t *testing.T) {
	input := blankPDF([]PageSize{PageSizeLetter})
	doc, err := OpenBytes(input)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := doc.Header(), "%PDF-1.7"; got != want {
		t.Fatalf("Header() = %q, want %q", got, want)
	}
	if got, want := doc.NumPages(), 1; got != want {
		t.Fatalf("NumPages() = %d, want %d", got, want)
	}
	if _, err := doc.Page(0); err != nil {
		t.Fatalf("Page(0) returned error: %v", err)
	}
	if _, err := doc.Page(1); !errors.Is(err, ErrPageIndexOutOfRange) {
		t.Fatalf("Page(1) error = %v, want ErrPageIndexOutOfRange", err)
	}
	profile := doc.Profile()
	if !profile.Editable {
		t.Fatalf("Profile().Editable = false, reasons = %v", profile.UnsupportedReasons)
	}
	verification, err := doc.Validate()
	if err != nil {
		t.Fatal(err)
	}
	if !verification.ReparseOK || verification.PageCount != 1 {
		t.Fatalf("Validate() = %+v", verification)
	}
}

func TestOpenFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "blank.pdf")
	if err := os.WriteFile(path, blankPDF([]PageSize{PageSizeA4}), 0o600); err != nil {
		t.Fatal(err)
	}
	doc, err := OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if doc.NumPages() != 1 {
		t.Fatalf("NumPages() = %d, want 1", doc.NumPages())
	}
}

func TestWithPasswordFailsClosed(t *testing.T) {
	_, err := OpenBytes(blankPDF([]PageSize{PageSizeLetter}), WithPassword("secret"))
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("OpenBytes() error = %v, want ErrUnsupported", err)
	}
}

func TestWriterBlankPageParseAfterWrite(t *testing.T) {
	w := NewWriter()
	w.AddBlankPage(PageSizeA4)
	out, err := w.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	doc, err := OpenBytes(out)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := doc.NumPages(), 1; got != want {
		t.Fatalf("written NumPages() = %d, want %d", got, want)
	}
}
