package oxpdf

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
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
	page, ok := p.pageObject()
	if !ok {
		return 0
	}
	return pageTreeRotation(p.doc.input, page)
}

func (p *Page) box(name string) Rectangle {
	if p == nil || p.doc == nil {
		return Rectangle{}
	}
	page, ok := p.pageObject()
	if !ok {
		return Rectangle{}
	}
	return pageTreeBox(p.doc.input, page, name)
}

func (p *Page) pageDictionary() ([]byte, bool) {
	page, ok := p.pageObject()
	if !ok {
		return nil, false
	}
	return page.dict, true
}

func (p *Page) pageObject() (pageObject, bool) {
	pages := parsePageObjects(p.doc.input)
	if p.index < 0 || p.index >= len(pages) {
		return pageObject{}, false
	}
	return pages[p.index], true
}

type pageObject struct {
	number int
	gen    int
	dict   []byte
}

func pageTreeBox(input []byte, page pageObject, name string) Rectangle {
	if rect, ok := inheritedPageBox(input, page, name); ok {
		return rect
	}
	if name != "MediaBox" && name != "CropBox" {
		if rect, ok := inheritedPageBox(input, page, "CropBox"); ok {
			return rect
		}
	}
	if name != "MediaBox" {
		if rect, ok := inheritedPageBox(input, page, "MediaBox"); ok {
			return rect
		}
	}
	return Rectangle{}
}

func inheritedPageBox(input []byte, page pageObject, name string) (Rectangle, bool) {
	for _, dict := range pageInheritanceChain(input, page) {
		if rect, ok := parsePageBox(dict, name); ok {
			return rect, true
		}
	}
	return Rectangle{}, false
}

func pageTreeRotation(input []byte, page pageObject) int {
	for _, dict := range pageInheritanceChain(input, page) {
		if raw, ok := directNameValue(dict, "Rotate"); ok {
			return parsePageRotation(input, raw)
		}
	}
	return 0
}

func pageInheritanceChain(input []byte, page pageObject) [][]byte {
	objects := parseIndirectObjects(input)
	chain := [][]byte{page.dict}
	seen := map[int]bool{page.number: true}
	current := page.dict
	for {
		ref := directReference(current, "Parent")
		if ref == nil || seen[ref.Number] {
			return chain
		}
		seen[ref.Number] = true
		parent, ok := indirectObjectByNumber(objects, ref.Number, ref.Generation)
		if !ok {
			return chain
		}
		dict, ok := objectDictionary(parent.body)
		if !ok {
			return chain
		}
		chain = append(chain, dict)
		current = dict
	}
}

func parsePageDictionaries(input []byte) [][]byte {
	pageObjects := parsePageObjects(input)
	pages := make([][]byte, 0, len(pageObjects))
	for _, page := range pageObjects {
		pages = append(pages, page.dict)
	}
	return pages
}

func parsePageObjects(input []byte) []pageObject {
	objects := parseIndirectObjects(input)
	if pages, ok := parsePageTreeObjects(input, objects); ok {
		return pages
	}
	return parsePageObjectsFromList(objects)
}

func parsePageTreeObjects(input []byte, objects []indirectObject) ([]pageObject, bool) {
	trailer, ok := parseTrailer(input)
	if !ok || trailer.Root == nil {
		return nil, false
	}
	catalog, ok := indirectObjectByNumber(objects, trailer.Root.Number, trailer.Root.Generation)
	if !ok {
		return nil, false
	}
	dict, ok := objectDictionary(catalog.body)
	if !ok || !hasPDFNameEntry(dict, "Type", "Catalog") {
		return nil, false
	}
	pagesRef := directReference(dict, "Pages")
	if pagesRef == nil {
		return nil, false
	}
	pages, ok := collectPageTreeObjects(objects, *pagesRef, map[objectRef]bool{})
	if !ok || len(pages) == 0 {
		return nil, false
	}
	return pages, true
}

func collectPageTreeObjects(objects []indirectObject, ref ObjectReference, seen map[objectRef]bool) ([]pageObject, bool) {
	key := objectRef{number: ref.Number, gen: ref.Generation}
	if seen[key] {
		return nil, false
	}
	seen[key] = true
	object, ok := indirectObjectByNumber(objects, ref.Number, ref.Generation)
	if !ok {
		return nil, false
	}
	dict, ok := objectDictionary(object.body)
	if !ok {
		return nil, false
	}
	if hasPDFNameEntry(dict, "Type", "Page") {
		return []pageObject{{
			number: object.number,
			gen:    object.gen,
			dict:   dict,
		}}, true
	}
	kids := directReferenceArray(dict, "Kids")
	if len(kids) == 0 {
		return nil, false
	}
	var pages []pageObject
	for _, kid := range kids {
		kidPages, ok := collectPageTreeObjects(objects, kid, seen)
		if !ok {
			return nil, false
		}
		pages = append(pages, kidPages...)
	}
	return pages, true
}

func parsePageObjectsFromList(objects []indirectObject) []pageObject {
	var pages []pageObject
	for _, object := range objects {
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
			pages = append(pages, pageObject{
				number: object.number,
				gen:    object.gen,
				dict:   dict,
			})
		}
	}
	return pages
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

func parsePageRotation(input, raw []byte) int {
	raw = bytes.TrimSpace(raw)
	rotationRe := regexp.MustCompile(`^([+-]?\d+)(?:\s+(\d+)\s+R\b)?`)
	match := rotationRe.FindSubmatch(raw)
	if len(match) == 0 {
		return 0
	}
	rotation, err := strconv.Atoi(string(match[1]))
	if err != nil {
		return 0
	}
	if len(match) >= 3 && len(match[2]) > 0 {
		return resolveIndirectInteger(input, rotation, atoiBytes(match[2]))
	}
	return normalizeRotation(rotation)
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
	objects = append(objects, objectStreamIndirectObjects(objects)...)
	return objects
}

func objectStreamIndirectObjects(objects []indirectObject) []indirectObject {
	seen := make(map[objectRef]bool, len(objects))
	for _, object := range objects {
		seen[objectRef{number: object.number, gen: object.gen}] = true
	}
	var out []indirectObject
	for _, object := range objects {
		dict, ok := objectDictionary(object.body)
		if !ok || !hasPDFNameEntry(dict, "Type", "ObjStm") {
			continue
		}
		members := parseObjectStreamMembers(object, dict)
		for _, member := range members {
			key := objectRef{number: member.number, gen: member.gen}
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, member)
		}
	}
	return out
}

func parseObjectStreamMembers(object indirectObject, dict []byte) []indirectObject {
	n := directInteger(dict, "N")
	first := directInteger(dict, "First")
	if n == nil || *n < 0 || first == nil || *first < 0 {
		return nil
	}
	stream, ok := streamBytesAfterDictionary(object.body, dict)
	if !ok {
		return nil
	}
	decoded, ok := decodeObjectStreamBytes(dict, stream)
	if !ok || *first > len(decoded) {
		return nil
	}
	fields := strings.Fields(string(decoded[:*first]))
	if len(fields) < *n*2 {
		return nil
	}
	type memberHeader struct {
		number int
		offset int
	}
	headers := make([]memberHeader, 0, *n)
	for i := 0; i < *n; i++ {
		number, numberErr := strconv.Atoi(fields[i*2])
		offset, offsetErr := strconv.Atoi(fields[i*2+1])
		if numberErr != nil || offsetErr != nil || offset < 0 || *first+offset > len(decoded) {
			return nil
		}
		headers = append(headers, memberHeader{number: number, offset: offset})
	}
	members := make([]indirectObject, 0, len(headers))
	for i, header := range headers {
		start := *first + header.offset
		end := len(decoded)
		if i+1 < len(headers) {
			end = *first + headers[i+1].offset
		}
		if start > end {
			return nil
		}
		body := bytes.TrimSpace(decoded[start:end])
		if len(body) == 0 || bytes.Contains(body, []byte("stream")) {
			continue
		}
		members = append(members, indirectObject{
			number: header.number,
			gen:    0,
			body:   body,
		})
	}
	return members
}

func decodeObjectStreamBytes(dict, stream []byte) ([]byte, bool) {
	if decodeParms, ok := directNameValue(dict, "DecodeParms"); ok {
		trimmed := bytes.TrimSpace(decodeParms)
		if !bytes.HasPrefix(trimmed, []byte("null")) {
			return nil, false
		}
	}
	switch streamFilterName(dict) {
	case "":
		return bytes.Clone(stream), true
	case "FlateDecode":
		decoded, err := flateDecode(stream)
		if err != nil {
			return nil, false
		}
		return decoded, true
	default:
		return nil, false
	}
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
