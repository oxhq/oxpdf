package oxpdf

import (
	"bytes"
	"regexp"
	"strconv"
)

// Rectangle describes a PDF rectangle in points.
type Rectangle struct {
	Left   float64
	Bottom float64
	Right  float64
	Top    float64
}

// Width returns the rectangle width.
func (r Rectangle) Width() float64 {
	return r.Right - r.Left
}

// Height returns the rectangle height.
func (r Rectangle) Height() float64 {
	return r.Top - r.Bottom
}

// MediaBox returns the page media box.
func (p *Page) MediaBox() Rectangle {
	return p.box("MediaBox")
}

// CropBox returns the page crop box, falling back to MediaBox when absent.
func (p *Page) CropBox() Rectangle {
	return p.box("CropBox")
}

// BleedBox returns the page bleed box, falling back to CropBox/MediaBox when absent.
func (p *Page) BleedBox() Rectangle {
	return p.box("BleedBox")
}

// TrimBox returns the page trim box, falling back to CropBox/MediaBox when absent.
func (p *Page) TrimBox() Rectangle {
	return p.box("TrimBox")
}

// ArtBox returns the page art box, falling back to CropBox/MediaBox when absent.
func (p *Page) ArtBox() Rectangle {
	return p.box("ArtBox")
}

// Rotation returns the page rotation in degrees. Missing rotation resolves to 0.
func (p *Page) Rotation() int {
	if p == nil || p.doc == nil {
		return 0
	}
	dict, ok := p.pageDictionary()
	if !ok {
		return 0
	}
	return parsePageRotation(p.doc.input, dict)
}

func (p *Page) box(name string) Rectangle {
	if p == nil || p.doc == nil {
		return Rectangle{}
	}
	dict, ok := p.pageDictionary()
	if !ok {
		return Rectangle{}
	}
	if rect, ok := parsePageBox(dict, name); ok {
		return rect
	}
	if name != "MediaBox" && name != "CropBox" {
		if rect, ok := parsePageBox(dict, "CropBox"); ok {
			return rect
		}
	}
	if rect, ok := parsePageBox(dict, "MediaBox"); ok {
		return rect
	}
	return Rectangle{}
}

func (p *Page) pageDictionary() ([]byte, bool) {
	pages := parsePageDictionaries(p.doc.input)
	if p.index < 0 || p.index >= len(pages) {
		return nil, false
	}
	return pages[p.index], true
}

func parsePageDictionaries(input []byte) [][]byte {
	var pages [][]byte
	for _, object := range parseIndirectObjects(input) {
		dictStart := bytes.Index(object.body, []byte("<<"))
		if dictStart == -1 {
			continue
		}
		dictEnd, ok := scanDictionaryEnd(object.body, dictStart)
		if !ok {
			continue
		}
		dict := object.body[dictStart:dictEnd]
		if hasPDFNameEntry(dict, "Type", "Page") {
			pages = append(pages, dict)
		}
	}
	return pages
}

type indirectObject struct {
	number int
	gen    int
	body   []byte
}

func parseIndirectObjects(input []byte) []indirectObject {
	headerRe := regexp.MustCompile(`(?m)(\d+)\s+(\d+)\s+obj\b`)
	matches := headerRe.FindAllSubmatchIndex(input, -1)
	objects := make([]indirectObject, 0, len(matches))
	for _, match := range matches {
		bodyStart := match[1]
		endRel := bytes.Index(input[bodyStart:], []byte("endobj"))
		if endRel == -1 {
			continue
		}
		number, _ := strconv.Atoi(string(input[match[2]:match[3]]))
		gen, _ := strconv.Atoi(string(input[match[4]:match[5]]))
		objects = append(objects, indirectObject{
			number: number,
			gen:    gen,
			body:   input[bodyStart : bodyStart+endRel],
		})
	}
	return objects
}

func parsePageBox(dict []byte, name string) (Rectangle, bool) {
	value, ok := directNameValue(dict, name)
	if !ok || len(value) == 0 || value[0] != '[' {
		return Rectangle{}, false
	}
	closeAt := bytes.IndexByte(value, ']')
	if closeAt == -1 {
		return Rectangle{}, false
	}
	fields := bytes.Fields(value[1:closeAt])
	if len(fields) < 4 {
		return Rectangle{}, false
	}
	left, ok1 := parsePDFNumber(fields[0])
	bottom, ok2 := parsePDFNumber(fields[1])
	right, ok3 := parsePDFNumber(fields[2])
	top, ok4 := parsePDFNumber(fields[3])
	if !ok1 || !ok2 || !ok3 || !ok4 {
		return Rectangle{}, false
	}
	return Rectangle{Left: left, Bottom: bottom, Right: right, Top: top}, true
}

func parsePageRotation(input, dict []byte) int {
	value, ok := directNameValue(dict, "Rotate")
	if !ok {
		return 0
	}
	fields := bytes.Fields(value)
	if len(fields) == 0 {
		return 0
	}
	if rotation, err := strconv.Atoi(string(fields[0])); err == nil {
		if len(fields) >= 3 && string(fields[2]) == "R" {
			return resolveIndirectInteger(input, rotation, atoiBytes(fields[1]))
		}
		return normalizeRotation(rotation)
	}
	return 0
}

func resolveIndirectInteger(input []byte, number, gen int) int {
	for _, object := range parseIndirectObjects(input) {
		if object.number != number || object.gen != gen {
			continue
		}
		fields := bytes.Fields(object.body)
		if len(fields) == 0 {
			return 0
		}
		value, err := strconv.Atoi(string(fields[0]))
		if err != nil {
			return 0
		}
		return normalizeRotation(value)
	}
	return 0
}

func normalizeRotation(rotation int) int {
	rotation %= 360
	if rotation < 0 {
		rotation += 360
	}
	return rotation
}

func directNameValue(dict []byte, key string) ([]byte, bool) {
	needle := []byte("/" + key)
	for start := 0; start < len(dict); {
		at := bytes.Index(dict[start:], needle)
		if at == -1 {
			return nil, false
		}
		at += start
		end := at + len(needle)
		if isTokenBoundary(dict, end) {
			valueStart := skipPDFSpace(dict, end)
			return dict[valueStart:], true
		}
		start = end
	}
	return nil, false
}

func hasPDFNameEntry(dict []byte, key, value string) bool {
	raw, ok := directNameValue(dict, key)
	if !ok {
		return false
	}
	want := []byte("/" + value)
	return bytes.HasPrefix(raw, want) && isTokenBoundary(raw, len(want))
}

func parsePDFNumber(input []byte) (float64, bool) {
	value, err := strconv.ParseFloat(string(input), 64)
	return value, err == nil
}

func atoiBytes(input []byte) int {
	value, _ := strconv.Atoi(string(input))
	return value
}

func isTokenBoundary(input []byte, pos int) bool {
	return pos >= len(input) || isPDFSpaceByte(input[pos]) || isPDFDelimiterByte(input[pos])
}
