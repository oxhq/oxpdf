package oxpdf

import (
	"path/filepath"
	"strings"
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
		if items[i].Depth != 0 {
			t.Fatalf("OutlineItems()[%d].Depth = %d, want 0", i, items[i].Depth)
		}
		if items[i].ParentIndex != -1 {
			t.Fatalf("OutlineItems()[%d].ParentIndex = %d, want -1", i, items[i].ParentIndex)
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

func TestOutlineItemsSyntheticNestedHierarchy(t *testing.T) {
	doc, err := OpenBytes(syntheticPDFObjects(
		[]byte("<< /Type /Catalog /Pages 2 0 R /Outlines 4 0 R >>"),
		[]byte("<< /Type /Pages /Kids [3 0 R] /Count 1 >>"),
		[]byte("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>"),
		[]byte("<< /Type /Outlines /First 5 0 R /Last 9 0 R /Count 5 >>"),
		[]byte("<< /Title (Top A) /A 10 0 R /First 6 0 R /Last 8 0 R /Next 9 0 R /Parent 4 0 R >>"),
		[]byte("<< /Title (Child A.1) /A 11 0 R /First 7 0 R /Last 7 0 R /Next 8 0 R /Parent 5 0 R >>"),
		[]byte("<< /A 12 0 R /Parent 6 0 R >>"),
		[]byte("<< /Title (Child A.2) /A 13 0 R /Parent 5 0 R >>"),
		[]byte("<< /Title (Top B) /A 14 0 R /Parent 4 0 R >>"),
		[]byte("<< /S /GoTo /D (top-a) >>"),
		[]byte("<< /S /GoTo /D (child-a-1) >>"),
		[]byte("<< /S /GoTo /D (grandchild-missing-title) >>"),
		[]byte("<< /S /GoTo /D (child-a-2) >>"),
		[]byte("<< /S /GoTo /D (top-b) >>"),
	))
	if err != nil {
		t.Fatal(err)
	}

	items, err := doc.OutlineItems()
	if err != nil {
		t.Fatal(err)
	}
	want := []OutlineItem{
		{Index: 0, Depth: 0, ParentIndex: -1, Title: "Top A", DestinationName: "top-a", Status: "supported"},
		{Index: 1, Depth: 1, ParentIndex: 0, Title: "Child A.1", DestinationName: "child-a-1", Status: "supported"},
		{Index: 2, Depth: 2, ParentIndex: 1, Title: "", DestinationName: "grandchild-missing-title", Status: "supported"},
		{Index: 3, Depth: 1, ParentIndex: 0, Title: "Child A.2", DestinationName: "child-a-2", Status: "supported"},
		{Index: 4, Depth: 0, ParentIndex: -1, Title: "Top B", DestinationName: "top-b", Status: "supported"},
	}
	if len(items) != len(want) {
		t.Fatalf("OutlineItems() len = %d, want %d: %+v", len(items), len(want), items)
	}
	for i := range want {
		if items[i].Index != want[i].Index ||
			items[i].Depth != want[i].Depth ||
			items[i].ParentIndex != want[i].ParentIndex ||
			items[i].Title != want[i].Title ||
			items[i].DestinationName != want[i].DestinationName ||
			items[i].Status != want[i].Status ||
			len(items[i].Blockers) != 0 {
			t.Fatalf("OutlineItems()[%d] = %+v, want %+v", i, items[i], want[i])
		}
	}
}

func TestOutlineItemsSyntheticCycleFailsClosed(t *testing.T) {
	doc, err := OpenBytes(syntheticPDFObjects(
		[]byte("<< /Type /Catalog /Pages 2 0 R /Outlines 4 0 R >>"),
		[]byte("<< /Type /Pages /Kids [3 0 R] /Count 1 >>"),
		[]byte("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>"),
		[]byte("<< /Type /Outlines /First 5 0 R /Last 5 0 R /Count 2 >>"),
		[]byte("<< /Title (Loop) /A 6 0 R /First 5 0 R /Parent 4 0 R >>"),
		[]byte("<< /S /GoTo /D (loop) >>"),
	))
	if err != nil {
		t.Fatal(err)
	}

	_, err = doc.OutlineItems()
	if err == nil || !strings.Contains(err.Error(), "outline cycle") {
		t.Fatalf("OutlineItems() error = %v, want outline cycle", err)
	}
}
