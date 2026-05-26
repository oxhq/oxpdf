package oxpdf

import "fmt"

// OutlineItem describes a flat, read-only outline item.
type OutlineItem struct {
	Index           int
	Depth           int
	ParentIndex     int
	Title           string
	DestinationName string
	Status          string
	Blockers        []string
}

// OutlineItems lists top-level outline items in linked-list order.
func (d *Document) OutlineItems() ([]OutlineItem, error) {
	if d == nil {
		return nil, nil
	}
	catalog, ok := d.Catalog()
	if !ok || catalog.Outlines == nil {
		return nil, nil
	}
	objects := parseIndirectObjects(d.input)
	root, ok := indirectObjectByNumber(objects, catalog.Outlines.Number, catalog.Outlines.Generation)
	if !ok {
		return nil, unsupported("outline root object is missing")
	}
	rootDict, ok := objectDictionary(root.body)
	if !ok {
		return nil, unsupported("outline root object is not a dictionary")
	}
	first := directReference(rootDict, "First")
	if first == nil {
		return nil, nil
	}
	items, err := collectOutlineItems(objects, first, 0, -1, map[objectRef]bool{})
	if err != nil {
		return nil, err
	}
	return items, nil
}

func collectOutlineItems(objects []indirectObject, first *ObjectReference, depth, parentIndex int, seen map[objectRef]bool) ([]OutlineItem, error) {
	var items []OutlineItem
	if err := appendOutlineItems(objects, first, depth, parentIndex, seen, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func appendOutlineItems(objects []indirectObject, first *ObjectReference, depth, parentIndex int, seen map[objectRef]bool, items *[]OutlineItem) error {
	current := first
	for current != nil {
		key := objectRef{number: current.Number, gen: current.Generation}
		if seen[key] {
			return unsupported(fmt.Sprintf("outline cycle at %d %d R", current.Number, current.Generation))
		}
		seen[key] = true
		object, ok := indirectObjectByNumber(objects, current.Number, current.Generation)
		if !ok {
			return unsupported(fmt.Sprintf("outline item %d %d R is missing", current.Number, current.Generation))
		}
		dict, ok := objectDictionary(object.body)
		if !ok {
			return unsupported(fmt.Sprintf("outline item %d is not a dictionary", object.number))
		}
		item, err := parseOutlineItem(objects, dict, depth, parentIndex)
		if err != nil {
			return err
		}
		item.Index = len(*items)
		*items = append(*items, item)
		childFirst := directReference(dict, "First")
		if childFirst != nil {
			if err := appendOutlineItems(objects, childFirst, depth+1, item.Index, seen, items); err != nil {
				return err
			}
		}
		current = directReference(dict, "Next")
	}
	return nil
}

func parseOutlineItem(objects []indirectObject, dict []byte, depth, parentIndex int) (OutlineItem, error) {
	item := OutlineItem{Depth: depth, ParentIndex: parentIndex, Status: "supported"}
	title, err := outlineTitle(objects, dict)
	if err != nil {
		return OutlineItem{}, err
	}
	item.Title = title
	actionRef := directReference(dict, "A")
	if actionRef == nil {
		item.Status = "unsupported"
		item.Blockers = append(item.Blockers, "missing_action")
		return item, nil
	}
	destination, err := outlineActionDestination(objects, *actionRef)
	if err != nil {
		return OutlineItem{}, err
	}
	if destination == "" {
		item.Status = "unsupported"
		item.Blockers = append(item.Blockers, "unsupported_action")
		return item, nil
	}
	item.DestinationName = destination
	return item, nil
}

func outlineTitle(objects []indirectObject, dict []byte) (string, error) {
	raw, ok := directNameValue(dict, "Title")
	if !ok {
		return "", nil
	}
	if title, ok := parsePDFTextValue(raw); ok {
		return title, nil
	}
	ref, ok := parsePDFRef(raw)
	if !ok {
		return "", unsupported("outline title uses unsupported representation")
	}
	object, ok := indirectObjectByNumber(objects, ref.number, ref.gen)
	if !ok {
		return "", unsupported(fmt.Sprintf("outline title object %d %d R is missing", ref.number, ref.gen))
	}
	title, ok := parsePDFTextValue(object.body)
	if !ok {
		return "", unsupported(fmt.Sprintf("outline title object %d uses unsupported representation", ref.number))
	}
	return title, nil
}

func outlineActionDestination(objects []indirectObject, ref ObjectReference) (string, error) {
	object, ok := indirectObjectByNumber(objects, ref.Number, ref.Generation)
	if !ok {
		return "", unsupported(fmt.Sprintf("outline action object %d %d R is missing", ref.Number, ref.Generation))
	}
	dict, ok := objectDictionary(object.body)
	if !ok {
		return "", unsupported(fmt.Sprintf("outline action object %d is not a dictionary", object.number))
	}
	if !hasPDFNameEntry(dict, "S", "GoTo") {
		return "", nil
	}
	raw, ok := directNameValue(dict, "D")
	if !ok {
		return "", nil
	}
	destination, ok := parsePDFTextValue(raw)
	if !ok {
		return "", unsupported(fmt.Sprintf("outline action object %d has unsupported destination representation", object.number))
	}
	return destination, nil
}
