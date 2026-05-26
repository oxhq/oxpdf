package oxpdf

import (
	"bytes"
	"reflect"
	"strconv"
)

// ObjectReference identifies an indirect PDF object.
type ObjectReference struct {
	Number     int
	Generation int
}

// Trailer exposes stable references from the last trailer dictionary.
type Trailer struct {
	Size    int
	Root    *ObjectReference
	Info    *ObjectReference
	Encrypt *ObjectReference
	Prev    *int
	XRefStm *int
	ID      []string
}

// Catalog exposes direct references from the document catalog dictionary.
type Catalog struct {
	Object     ObjectReference
	Pages      *ObjectReference
	Names      *ObjectReference
	Outlines   *ObjectReference
	PageLabels *ObjectReference
	Metadata   *ObjectReference
	AcroForm   *ObjectReference
}

// XrefObject describes an object entry reported by the cross-reference summary.
type XrefObject struct {
	Number             int
	Generation         int
	Offset             int
	Compressed         bool
	ObjectStreamNumber int
	ObjectStreamIndex  int
}

// Xref summarizes cross-reference structure detected by the backing parser.
type Xref struct {
	HasTable            bool
	TableOffset         int
	HasStream           bool
	HasHybridStream     bool
	HybridStreamOffset  int
	HasObjectStream     bool
	ObjectCount         int
	StreamCount         int
	ObjectStreamCount   int
	Objects             []XrefObject
	StreamObjects       []XrefObject
	ObjectStreamObjects []XrefObject
}

// Trailer returns the last trailer dictionary references when they are directly readable.
func (d *Document) Trailer() (Trailer, bool) {
	if d == nil {
		return Trailer{}, false
	}
	return parseTrailer(d.input)
}

// Catalog returns direct references from the catalog object referenced by the trailer.
func (d *Document) Catalog() (Catalog, bool) {
	if d == nil {
		return Catalog{}, false
	}
	trailer, ok := parseTrailer(d.input)
	if !ok || trailer.Root == nil {
		return Catalog{}, false
	}
	object, ok := findIndirectObject(d.input, trailer.Root.Number, trailer.Root.Generation)
	if !ok {
		return Catalog{}, false
	}
	dict, ok := objectDictionary(object.body)
	if !ok || !hasPDFNameEntry(dict, "Type", "Catalog") {
		return Catalog{}, false
	}
	return Catalog{
		Object:     *trailer.Root,
		Pages:      directReference(dict, "Pages"),
		Names:      directReference(dict, "Names"),
		Outlines:   directReference(dict, "Outlines"),
		PageLabels: directReference(dict, "PageLabels"),
		Metadata:   directReference(dict, "Metadata"),
		AcroForm:   directReference(dict, "AcroForm"),
	}, true
}

// Xref returns the detected cross-reference summary.
func (d *Document) Xref() Xref {
	if d == nil {
		return Xref{}
	}
	value, _ := d.root.Value.(map[string]any)
	xref := valueMap(value["xref"])
	return Xref{
		HasTable:            boolValue(xref["has_table"]),
		TableOffset:         intValue(xref["table_offset"]),
		HasStream:           boolValue(xref["has_stream"]),
		HasHybridStream:     boolValue(xref["has_hybrid_stream"]),
		HybridStreamOffset:  intValue(xref["hybrid_stream_offset"]),
		HasObjectStream:     boolValue(xref["has_object_stream"]),
		ObjectCount:         intValue(xref["object_count"]),
		StreamCount:         intValue(xref["stream_count"]),
		ObjectStreamCount:   intValue(xref["object_stream_count"]),
		Objects:             xrefObjects(xref["objects"]),
		StreamObjects:       xrefObjects(xref["stream_objects"]),
		ObjectStreamObjects: xrefObjects(xref["object_stream_objects"]),
	}
}

func parseTrailer(input []byte) (Trailer, bool) {
	dict, ok := lastTrailerDictionary(input)
	if !ok {
		return Trailer{}, false
	}
	trailer := Trailer{
		Size:    trailerInteger(dict, "Size"),
		Root:    directReference(dict, "Root"),
		Info:    directReference(dict, "Info"),
		Encrypt: directReference(dict, "Encrypt"),
		Prev:    directInteger(dict, "Prev"),
		XRefStm: directInteger(dict, "XRefStm"),
		ID:      directIDArray(dict),
	}
	return trailer, trailer.Size > 0 || trailer.Root != nil || trailer.Info != nil
}

func lastTrailerDictionary(input []byte) ([]byte, bool) {
	for end := len(input); end > 0; {
		at := bytes.LastIndex(input[:end], []byte("trailer"))
		if at == -1 {
			return nil, false
		}
		dictStartRel := bytes.Index(input[at+len("trailer"):], []byte("<<"))
		if dictStartRel == -1 {
			end = at
			continue
		}
		dictStart := at + len("trailer") + dictStartRel
		dictEnd, ok := scanDictionaryEnd(input, dictStart)
		if ok {
			return input[dictStart:dictEnd], true
		}
		end = at
	}
	return nil, false
}

func trailerInteger(dict []byte, key string) int {
	value := directInteger(dict, key)
	if value == nil {
		return 0
	}
	return *value
}

func directInteger(dict []byte, key string) *int {
	raw, ok := directNameValue(dict, key)
	if !ok {
		return nil
	}
	raw = bytes.TrimSpace(raw)
	end := 0
	for end < len(raw) && raw[end] >= '0' && raw[end] <= '9' {
		end++
	}
	if end == 0 {
		return nil
	}
	value, err := strconv.Atoi(string(raw[:end]))
	if err != nil {
		return nil
	}
	return &value
}

func directReference(dict []byte, key string) *ObjectReference {
	raw, ok := directNameValue(dict, key)
	if !ok {
		return nil
	}
	ref, ok := parsePDFRef(raw)
	if !ok {
		return nil
	}
	return &ObjectReference{Number: ref.number, Generation: ref.gen}
}

func findIndirectObject(input []byte, number, gen int) (indirectObject, bool) {
	for _, object := range parseIndirectObjects(input) {
		if object.number == number && object.gen == gen {
			return object, true
		}
	}
	return indirectObject{}, false
}

func directIDArray(dict []byte) []string {
	raw, ok := directNameValue(dict, "ID")
	if !ok {
		return nil
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '[' {
		return nil
	}
	end := bytes.IndexByte(raw, ']')
	if end == -1 {
		return nil
	}
	fields := bytes.Fields(raw[1:end])
	ids := make([]string, 0, len(fields))
	for _, field := range fields {
		if len(field) > 0 && field[0] == '<' && field[len(field)-1] == '>' {
			ids = append(ids, string(field))
		}
	}
	return ids
}

func xrefObjects(value any) []XrefObject {
	rv := reflect.ValueOf(value)
	if !rv.IsValid() || rv.Kind() != reflect.Slice {
		return nil
	}
	out := make([]XrefObject, 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		item := rv.Index(i)
		if item.Kind() == reflect.Pointer {
			if item.IsNil() {
				continue
			}
			item = item.Elem()
		}
		if item.Kind() != reflect.Struct {
			continue
		}
		out = append(out, XrefObject{
			Number:             reflectedInt(item, "Number"),
			Generation:         reflectedInt(item, "Generation"),
			Offset:             reflectedInt(item, "Offset"),
			Compressed:         reflectedBool(item, "Compressed"),
			ObjectStreamNumber: reflectedInt(item, "ObjectStreamNumber"),
			ObjectStreamIndex:  reflectedInt(item, "ObjectStreamIndex"),
		})
	}
	return out
}

func reflectedInt(value reflect.Value, name string) int {
	field := value.FieldByName(name)
	if !field.IsValid() {
		return 0
	}
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(field.Int())
	default:
		return 0
	}
}

func reflectedBool(value reflect.Value, name string) bool {
	field := value.FieldByName(name)
	if !field.IsValid() || field.Kind() != reflect.Bool {
		return false
	}
	return field.Bool()
}
