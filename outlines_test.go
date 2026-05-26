package oxpdf

import (
	"path/filepath"
	"testing"
)

func TestPypdfCorpusFlatOutlineItems(t *testing.T) {
	resources := pypdfResources(t)
	doc, err := OpenFile(filepath.Join(resources, "outline-without-title.pdf"))
	if err != nil {
		t.Fatal(err)
	}

	items, err := doc.OutlineItems()
	if err != nil {
		t.Fatal(err)
	}
	wantTitles := []string{"", "Bar", "Baz", "Foo", "Bar", "Baz", "Foo", "Bar", "Baz"}
	wantDests := []string{"section.1", "section.2", "section.3", "section.4", "section.5", "section.6", "section.7", "section.8", "section.9"}
	if len(items) != len(wantTitles) {
		t.Fatalf("OutlineItems() len = %d, want %d: %+v", len(items), len(wantTitles), items)
	}
	for i := range wantTitles {
		if items[i].Index != i {
			t.Fatalf("OutlineItems()[%d].Index = %d, want %d", i, items[i].Index, i)
		}
		if items[i].Title != wantTitles[i] {
			t.Fatalf("OutlineItems()[%d].Title = %q, want %q", i, items[i].Title, wantTitles[i])
		}
		if items[i].DestinationName != wantDests[i] {
			t.Fatalf("OutlineItems()[%d].DestinationName = %q, want %q", i, items[i].DestinationName, wantDests[i])
		}
		if items[i].Status != "supported" || len(items[i].Blockers) != 0 {
			t.Fatalf("OutlineItems()[%d] status = %+v", i, items[i])
		}
	}
}

func TestPypdfCorpusFlatOutlineItemsEmpty(t *testing.T) {
	resources := pypdfResources(t)
	doc, err := OpenFile(filepath.Join(resources, "hello-world.pdf"))
	if err != nil {
		t.Fatal(err)
	}

	items, err := doc.OutlineItems()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("OutlineItems() = %+v, want empty", items)
	}
}
