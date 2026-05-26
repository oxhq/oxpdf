package oxpdf

import (
	"bytes"
	"fmt"
	"sort"
)

// NamedDestination describes a named destination resolved from the document
// name tree.
type NamedDestination struct {
	Name      string
	PageIndex *int
	Kind      string
	Left      *float64
	Top       *float64
	Zoom      *float64
	Status    string
	Blockers  []string
}

// NamedDestinations lists direct named destinations from a readable /Dests name tree.
func (d *Document) NamedDestinations() ([]NamedDestination, error) {
	if d == nil {
		return nil, nil
	}
	catalog, ok := d.Catalog()
	if !ok || catalog.Names == nil {
		return nil, nil
	}
	objects := parseIndirectObjects(d.input)
	namesObject, ok := indirectObjectByNumber(objects, catalog.Names.Number, catalog.Names.Generation)
	if !ok {
		return nil, unsupported("catalog names object is missing")
	}
	namesDict, ok := objectDictionary(namesObject.body)
	if !ok {
		return nil, unsupported("catalog names object is not a dictionary")
	}
	destsRef := directReference(namesDict, "Dests")
	if destsRef == nil {
		return nil, nil
	}
	pageIndexes := pageObjectIndexMap(objects)
	var destinations []NamedDestination
	if err := collectNamedDestinations(objects, *destsRef, pageIndexes, map[objectRef]bool{}, &destinations); err != nil {
		return nil, err
	}
	sort.SliceStable(destinations, func(i, j int) bool {
		return destinations[i].Name < destinations[j].Name
	})
	return destinations, nil
}

func collectNamedDestinations(objects []indirectObject, ref ObjectReference, pageIndexes map[objectRef]int, seen map[objectRef]bool, out *[]NamedDestination) error {
	key := objectRef{number: ref.Number, gen: ref.Generation}
	if seen[key] {
		return unsupported(fmt.Sprintf("named destination tree cycle at %d %d R", ref.Number, ref.Generation))
	}
	seen[key] = true
	object, ok := indirectObjectByNumber(objects, ref.Number, ref.Generation)
	if !ok {
		return unsupported(fmt.Sprintf("named destination node %d %d R is missing", ref.Number, ref.Generation))
	}
	dict, ok := objectDictionary(object.body)
	if !ok {
		return unsupported(fmt.Sprintf("named destination node %d is not a dictionary", object.number))
	}
	for _, kid := range directReferenceArray(dict, "Kids") {
		if err := collectNamedDestinations(objects, kid, pageIndexes, seen, out); err != nil {
			return err
		}
	}
	namesRaw, ok := directArrayValue(dict, "Names")
	if !ok {
		return nil
	}
	pairs, err := parseDestinationNamePairs(namesRaw)
	if err != nil {
		return err
	}
	for _, pair := range pairs {
		destRef, ok := parsePDFRef(pair.destination)
		if !ok {
			return unsupported(fmt.Sprintf("named destination %q is not a direct object reference", pair.name))
		}
		destination, err := resolveNamedDestination(objects, pair.name, destRef, pageIndexes)
		if err != nil {
			return err
		}
		*out = append(*out, destination)
	}
	return nil
}

type destinationNamePair struct {
	name        string
	destination []byte
}

func parseDestinationNamePairs(input []byte) ([]destinationNamePair, error) {
	i := 0
	pairs := make([]destinationNamePair, 0)
	for {
		i = skipPDFSpace(input, i)
		if i >= len(input) {
			return pairs, nil
		}
		if input[i] != '(' && input[i] != '<' {
			return nil, unsupported("named destination name tree uses unsupported name representation")
		}
		nameEnd := i
		var ok bool
		if input[i] == '(' {
			nameEnd, ok = scanLiteralEnd(input, i)
			if !ok {
				return nil, unsupported("named destination name has malformed literal string")
			}
			nameEnd++
		} else {
			endRel := bytes.IndexByte(input[i+1:], '>')
			if endRel == -1 {
				return nil, unsupported("named destination name has malformed hex string")
			}
			nameEnd = i + 1 + endRel + 1
		}
		name, ok := parsePDFTextValue(input[i:nameEnd])
		if !ok {
			return nil, unsupported("named destination name is not readable")
		}
		i = skipPDFSpace(input, nameEnd)
		valueStart := i
		fields := 0
		for i < len(input) && fields < 3 {
			i = skipPDFSpace(input, i)
			if i >= len(input) {
				break
			}
			for i < len(input) && !isPDFSpaceByte(input[i]) && !isPDFDelimiterByte(input[i]) {
				i++
			}
			fields++
		}
		if fields < 3 {
			return nil, unsupported(fmt.Sprintf("named destination %q has incomplete reference", name))
		}
		pairs = append(pairs, destinationNamePair{name: name, destination: input[valueStart:i]})
	}
}

func resolveNamedDestination(objects []indirectObject, name string, ref objectRef, pageIndexes map[objectRef]int) (NamedDestination, error) {
	object, ok := indirectObjectByNumber(objects, ref.number, ref.gen)
	if !ok {
		return NamedDestination{}, unsupported(fmt.Sprintf("named destination %q object %d %d R is missing", name, ref.number, ref.gen))
	}
	dict, ok := objectDictionary(object.body)
	if !ok {
		return NamedDestination{}, unsupported(fmt.Sprintf("named destination %q object is not a dictionary", name))
	}
	raw, ok := directArrayValue(dict, "D")
	if !ok {
		return NamedDestination{}, unsupported(fmt.Sprintf("named destination %q has no /D array", name))
	}
	return parseDestinationArray(name, raw, pageIndexes)
}

func parseDestinationArray(name string, input []byte, pageIndexes map[objectRef]int) (NamedDestination, error) {
	fields := bytes.Fields(input)
	if len(fields) < 5 || string(fields[2]) != "R" {
		return NamedDestination{}, unsupported(fmt.Sprintf("named destination %q has unsupported destination array", name))
	}
	pageRef, ok := parsePDFRef(bytes.Join(fields[:3], []byte(" ")))
	if !ok {
		return NamedDestination{}, unsupported(fmt.Sprintf("named destination %q has unsupported page reference", name))
	}
	pageIndex, hasPage := pageIndexes[pageRef]
	kind := ""
	if len(fields) >= 4 && len(fields[3]) > 1 && fields[3][0] == '/' {
		kind = string(fields[3][1:])
	}
	destination := NamedDestination{Name: name, Kind: kind, Status: "supported"}
	if hasPage {
		destination.PageIndex = intPtr(pageIndex)
	} else {
		destination.Status = "unsupported"
		destination.Blockers = append(destination.Blockers, "page_reference_not_found")
	}
	if kind != "XYZ" {
		destination.Status = "unsupported"
		destination.Blockers = append(destination.Blockers, "unsupported_destination_kind")
		return destination, nil
	}
	if len(fields) > 4 && string(fields[4]) != "null" {
		if value, ok := parsePDFNumber(fields[4]); ok {
			destination.Left = floatPtr(value)
		}
	}
	if len(fields) > 5 && string(fields[5]) != "null" {
		if value, ok := parsePDFNumber(fields[5]); ok {
			destination.Top = floatPtr(value)
		}
	}
	if len(fields) > 6 && string(fields[6]) != "null" {
		if value, ok := parsePDFNumber(fields[6]); ok {
			destination.Zoom = floatPtr(value)
		}
	}
	return destination, nil
}

func pageObjectIndexMap(objects []indirectObject) map[objectRef]int {
	out := make(map[objectRef]int)
	for _, object := range objects {
		dict, ok := objectDictionary(object.body)
		if !ok || !hasPDFNameEntry(dict, "Type", "Page") {
			continue
		}
		out[objectRef{number: object.number, gen: object.gen}] = len(out)
	}
	return out
}

func directReferenceArray(dict []byte, key string) []ObjectReference {
	raw, ok := directArrayValue(dict, key)
	if !ok {
		return nil
	}
	fields := bytes.Fields(raw)
	out := make([]ObjectReference, 0)
	for i := 0; i+2 < len(fields); i += 3 {
		ref, ok := parsePDFRef(bytes.Join(fields[i:i+3], []byte(" ")))
		if !ok {
			return nil
		}
		out = append(out, ObjectReference{Number: ref.number, Generation: ref.gen})
	}
	return out
}

func directArrayValue(dict []byte, key string) ([]byte, bool) {
	raw, ok := directNameValue(dict, key)
	if !ok {
		return nil, false
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '[' {
		return nil, false
	}
	end, ok := scanArrayEnd(raw, 0)
	if !ok {
		return nil, false
	}
	return raw[1:end], true
}

func scanArrayEnd(input []byte, start int) (int, bool) {
	depth := 0
	for i := start; i < len(input); i++ {
		switch input[i] {
		case '(':
			end, ok := scanLiteralEnd(input, i)
			if !ok {
				return 0, false
			}
			i = end
		case '<':
			if i+1 < len(input) && input[i+1] == '<' {
				i++
			} else if endRel := bytes.IndexByte(input[i+1:], '>'); endRel != -1 {
				i += endRel + 1
			} else {
				return 0, false
			}
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return i, true
			}
		}
	}
	return 0, false
}

func intPtr(value int) *int {
	return &value
}

func floatPtr(value float64) *float64 {
	return &value
}
