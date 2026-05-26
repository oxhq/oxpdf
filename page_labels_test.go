package oxpdf

import "testing"

func TestPageLabelsDefaultToOneBasedNumbers(t *testing.T) {
	doc, err := OpenBytes(blankPDF([]PageSize{PageSizeLetter, PageSizeLetter, PageSizeLetter}))
	if err != nil {
		t.Fatal(err)
	}

	labels, err := doc.PageLabels()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"1", "2", "3"}
	if !sameStrings(labels, want) {
		t.Fatalf("PageLabels() = %v, want %v", labels, want)
	}
}

func TestPageLabelsSyntheticNumberTree(t *testing.T) {
	doc, err := OpenBytes(syntheticPDFObjects(
		[]byte("<< /Type /Catalog /Pages 2 0 R /PageLabels 6 0 R >>"),
		[]byte("<< /Type /Pages /Kids [3 0 R 4 0 R 5 0 R] /Count 3 >>"),
		[]byte("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>"),
		[]byte("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>"),
		[]byte("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>"),
		[]byte("<< /Nums [0 << /S /r /P (front-) /St 2 >> 2 << /S /D /P (body-) /St 10 >>] >>"),
	))
	if err != nil {
		t.Fatal(err)
	}

	labels, err := doc.PageLabels()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"front-ii", "front-iii", "body-10"}
	if !sameStrings(labels, want) {
		t.Fatalf("PageLabels() = %v, want %v", labels, want)
	}
}
