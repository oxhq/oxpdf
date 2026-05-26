package oxpdf

import (
	"path/filepath"
	"testing"
)

func TestPypdfCorpusNamedDestinations(t *testing.T) {
	resources := pypdfResources(t)
	doc, err := OpenFile(filepath.Join(resources, "outline-without-title.pdf"))
	if err != nil {
		t.Fatal(err)
	}

	destinations, err := doc.NamedDestinations()
	if err != nil {
		t.Fatal(err)
	}
	wantNames := []string{
		"Doc-Start",
		"page.1",
		"page.2",
		"page.3",
		"page.4",
		"section*.1",
		"section.1",
		"section.2",
		"section.3",
		"section.4",
		"section.5",
		"section.6",
		"section.7",
		"section.8",
		"section.9",
	}
	if len(destinations) != len(wantNames) {
		t.Fatalf("NamedDestinations() len = %d, want %d: %+v", len(destinations), len(wantNames), destinations)
	}
	for i, name := range wantNames {
		if destinations[i].Name != name {
			t.Fatalf("NamedDestinations()[%d].Name = %q, want %q", i, destinations[i].Name, name)
		}
	}
	assertNamedDestination(t, destinations, "Doc-Start", 0, "XYZ", 124.802, 716.092)
	assertNamedDestination(t, destinations, "page.1", 0, "XYZ", 123.802, 753.953)
	assertNamedDestination(t, destinations, "section.1", 1, "XYZ", 124.802, 716.092)
	assertNamedDestination(t, destinations, "section.5", 2, "XYZ", 124.802, 569.627)
	assertNamedDestination(t, destinations, "section.9", 3, "XYZ", 124.802, 514.86)
}

func TestPypdfCorpusNamedDestinationsEmpty(t *testing.T) {
	resources := pypdfResources(t)
	doc, err := OpenFile(filepath.Join(resources, "hello-world.pdf"))
	if err != nil {
		t.Fatal(err)
	}

	destinations, err := doc.NamedDestinations()
	if err != nil {
		t.Fatal(err)
	}
	if len(destinations) != 0 {
		t.Fatalf("NamedDestinations() = %+v, want empty", destinations)
	}
}

func assertNamedDestination(t *testing.T, destinations []NamedDestination, name string, page int, kind string, left, top float64) {
	t.Helper()
	for _, destination := range destinations {
		if destination.Name != name {
			continue
		}
		if destination.PageIndex == nil || *destination.PageIndex != page {
			t.Fatalf("%s PageIndex = %v, want %d", name, destination.PageIndex, page)
		}
		if destination.Kind != kind {
			t.Fatalf("%s Kind = %q, want %q", name, destination.Kind, kind)
		}
		if destination.Left == nil || *destination.Left != left {
			t.Fatalf("%s Left = %v, want %v", name, destination.Left, left)
		}
		if destination.Top == nil || *destination.Top != top {
			t.Fatalf("%s Top = %v, want %v", name, destination.Top, top)
		}
		if destination.Zoom != nil {
			t.Fatalf("%s Zoom = %v, want nil", name, destination.Zoom)
		}
		return
	}
	t.Fatalf("destination %q not found in %+v", name, destinations)
}
