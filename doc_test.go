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

func TestWithPasswordIsPassedToBackingAPI(t *testing.T) {
	doc, err := OpenBytes(blankPDF([]PageSize{PageSizeLetter}), WithPassword("secret"))
	if err != nil {
		t.Fatalf("OpenBytes() with password on unencrypted PDF returned error: %v", err)
	}
	if doc.NumPages() != 1 {
		t.Fatalf("NumPages() = %d, want 1", doc.NumPages())
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

func TestWriterInsertBlankPageAndWriteFile(t *testing.T) {
	w := NewWriter()
	w.AddBlankPage(PageSizeLetter)
	if err := w.InsertBlankPage(0, PageSizeA4); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "inserted.pdf")
	if err := w.WriteFile(path); err != nil {
		t.Fatal(err)
	}
	doc, err := OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := doc.NumPages(), 2; got != want {
		t.Fatalf("written NumPages() = %d, want %d", got, want)
	}
	page, err := doc.Page(0)
	if err != nil {
		t.Fatal(err)
	}
	want := Rectangle{Left: 0, Bottom: 0, Right: PageSizeA4.Width, Top: PageSizeA4.Height}
	if got := page.MediaBox(); got != want {
		t.Fatalf("inserted page MediaBox() = %+v, want %+v", got, want)
	}
}

func TestWriterInsertBlankPageRejectsOutOfRangeIndex(t *testing.T) {
	w := NewWriter()
	if err := w.InsertBlankPage(1, PageSizeLetter); !errors.Is(err, ErrPageIndexOutOfRange) {
		t.Fatalf("InsertBlankPage() error = %v, want ErrPageIndexOutOfRange", err)
	}
}

func TestWriterBlankPageEdgeCases(t *testing.T) {
	w := NewWriter()
	if err := w.InsertBlankPage(0, PageSize{Width: -1, Height: 0}); err != nil {
		t.Fatal(err)
	}
	if err := w.InsertBlankPage(1, PageSizeA4); err != nil {
		t.Fatal(err)
	}
	out, err := w.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	doc, err := OpenBytes(out)
	if err != nil {
		t.Fatal(err)
	}
	if got := doc.NumPages(); got != 2 {
		t.Fatalf("NumPages() = %d, want 2", got)
	}
	page, err := doc.Page(0)
	if err != nil {
		t.Fatal(err)
	}
	want := Rectangle{Left: 0, Bottom: 0, Right: PageSizeLetter.Width, Top: PageSizeLetter.Height}
	if got := page.MediaBox(); got != want {
		t.Fatalf("defaulted page MediaBox() = %+v, want %+v", got, want)
	}
}

func TestWriterWriteFilePropagatesEmptyWriterError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.pdf")
	err := NewWriter().WriteFile(path)
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("WriteFile() error = %v, want ErrUnsupported", err)
	}
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("output file stat error = %v, want not exists", statErr)
	}
}

func TestParsePageRange(t *testing.T) {
	tests := []struct {
		spec      string
		pageCount int
		want      []int
	}{
		{spec: "", pageCount: 3, want: []int{0, 1, 2}},
		{spec: "all", pageCount: 3, want: []int{0, 1, 2}},
		{spec: "1,3-4", pageCount: 5, want: []int{0, 2, 3}},
		{spec: "2-2,2,1", pageCount: 3, want: []int{1, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.spec, func(t *testing.T) {
			got, err := ParsePageRange(tt.spec, tt.pageCount)
			if err != nil {
				t.Fatal(err)
			}
			if !sameInts(got, tt.want) {
				t.Fatalf("ParsePageRange(%q) = %v, want %v", tt.spec, got, tt.want)
			}
		})
	}
}

func TestParsePageRangeRejectsInvalidSpecs(t *testing.T) {
	for _, spec := range []string{"0", "6", "4-2", "1,,2", "x", "-1"} {
		t.Run(spec, func(t *testing.T) {
			if _, err := ParsePageRange(spec, 5); err == nil {
				t.Fatalf("ParsePageRange(%q) returned nil error", spec)
			}
		})
	}
}

func sameInts(a, b []int) bool {
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
