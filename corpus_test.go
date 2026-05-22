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

func pypdfResources(t *testing.T) string {
	t.Helper()
	resources := filepath.Join("C:", "Users", "garae", "Documents", "pypdf", "resources")
	if _, err := os.Stat(resources); err != nil {
		t.Skipf("local pypdf resources not available: %v", err)
	}
	return resources
}
