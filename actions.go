package oxpdf

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// JavaScriptAction describes a document JavaScript action dictionary.
type JavaScriptAction struct {
	ObjectNumber int
	Name         string
	Source       string
	Script       string
}

// Attachment describes an embedded file payload.
type Attachment struct {
	Name         string
	ObjectNumber int
	Source       string
	Filter       string
	Size         int
	Content      []byte
}

// JavaScriptActions lists direct JavaScript action dictionaries.
func (d *Document) JavaScriptActions() ([]JavaScriptAction, error) {
	if d == nil {
		return nil, nil
	}
	return javascriptActionsForInput(d.input)
}

// JavaScriptNameTreeActions lists JavaScript actions reachable from the
// catalog /Names /JavaScript name tree.
func (d *Document) JavaScriptNameTreeActions() ([]JavaScriptAction, error) {
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
	javascriptRef := directReference(namesDict, "JavaScript")
	if javascriptRef == nil {
		return nil, nil
	}
	actions := make([]JavaScriptAction, 0)
	err := collectJavaScriptNameTreeActions(objects, *javascriptRef, map[objectRef]bool{}, &actions)
	if err != nil {
		return nil, err
	}
	return actions, nil
}

func javascriptActionsForInput(input []byte) ([]JavaScriptAction, error) {
	actions := make([]JavaScriptAction, 0)
	for _, object := range parseIndirectObjects(input) {
		action, ok, err := javascriptActionFromObject(object, "", "direct")
		if err != nil {
			return nil, err
		}
		if ok {
			actions = append(actions, action)
		}
	}
	return actions, nil
}

func collectJavaScriptNameTreeActions(objects []indirectObject, ref ObjectReference, seen map[objectRef]bool, out *[]JavaScriptAction) error {
	key := objectRef{number: ref.Number, gen: ref.Generation}
	if seen[key] {
		return unsupported(fmt.Sprintf("JavaScript name tree cycle at %d %d R", ref.Number, ref.Generation))
	}
	seen[key] = true
	object, ok := indirectObjectByNumber(objects, ref.Number, ref.Generation)
	if !ok {
		return unsupported(fmt.Sprintf("JavaScript name tree node %d %d R is missing", ref.Number, ref.Generation))
	}
	dict, ok := objectDictionary(object.body)
	if !ok {
		return unsupported(fmt.Sprintf("JavaScript name tree node %d is not a dictionary", object.number))
	}
	for _, kid := range directReferenceArray(dict, "Kids") {
		if err := collectJavaScriptNameTreeActions(objects, kid, seen, out); err != nil {
			return err
		}
	}
	namesRaw, ok := directArrayValue(dict, "Names")
	if !ok {
		return nil
	}
	pairs, err := parseNameTreeReferencePairs(namesRaw, "JavaScript")
	if err != nil {
		return err
	}
	for _, pair := range pairs {
		actionObject, ok := indirectObjectByNumber(objects, pair.ref.Number, pair.ref.Generation)
		if !ok {
			return unsupported(fmt.Sprintf("JavaScript action %q object %d %d R is missing", pair.name, pair.ref.Number, pair.ref.Generation))
		}
		action, ok, err := javascriptActionFromObject(actionObject, pair.name, "name-tree")
		if err != nil {
			return err
		}
		if !ok {
			return unsupported(fmt.Sprintf("JavaScript name tree entry %q does not reference a JavaScript action", pair.name))
		}
		*out = append(*out, action)
	}
	return nil
}

func javascriptActionFromObject(object indirectObject, name, source string) (JavaScriptAction, bool, error) {
	dict, ok := objectDictionary(object.body)
	if !ok || !hasPDFNameEntry(dict, "S", "JavaScript") {
		return JavaScriptAction{}, false, nil
	}
	raw, ok := directNameValue(dict, "JS")
	if !ok {
		return JavaScriptAction{}, false, nil
	}
	script, ok := parsePDFTextValue(raw)
	if !ok {
		return JavaScriptAction{}, false, unsupported(fmt.Sprintf("JavaScript action %d uses unsupported /JS representation", object.number))
	}
	return JavaScriptAction{
		ObjectNumber: object.number,
		Name:         name,
		Source:       source,
		Script:       script,
	}, true, nil
}

type nameTreeReferencePair struct {
	name string
	ref  ObjectReference
}

func parseNameTreeReferencePairs(input []byte, label string) ([]nameTreeReferencePair, error) {
	i := 0
	pairs := make([]nameTreeReferencePair, 0)
	for {
		i = skipPDFSpace(input, i)
		if i >= len(input) {
			return pairs, nil
		}
		if input[i] != '(' && input[i] != '<' {
			return nil, unsupported(fmt.Sprintf("%s name tree uses unsupported name representation", label))
		}
		nameEnd := i
		var ok bool
		if input[i] == '(' {
			nameEnd, ok = scanLiteralEnd(input, i)
			if !ok {
				return nil, unsupported(fmt.Sprintf("%s name tree has malformed literal name", label))
			}
			nameEnd++
		} else {
			endRel := bytes.IndexByte(input[i+1:], '>')
			if endRel == -1 {
				return nil, unsupported(fmt.Sprintf("%s name tree has malformed hex name", label))
			}
			nameEnd = i + 1 + endRel + 1
		}
		name, ok := parsePDFTextValue(input[i:nameEnd])
		if !ok {
			return nil, unsupported(fmt.Sprintf("%s name tree name is not readable", label))
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
			return nil, unsupported(fmt.Sprintf("%s name tree entry %q has incomplete reference", label, name))
		}
		ref, ok := parsePDFRef(input[valueStart:i])
		if !ok {
			return nil, unsupported(fmt.Sprintf("%s name tree entry %q is not a direct object reference", label, name))
		}
		pairs = append(pairs, nameTreeReferencePair{
			name: name,
			ref:  ObjectReference{Number: ref.number, Generation: ref.gen},
		})
	}
}

// Attachments lists embedded file payloads backed by direct Filespec objects.
func (d *Document) Attachments() ([]Attachment, error) {
	if d == nil {
		return nil, nil
	}
	objects := parseIndirectObjects(d.input)
	attachments := make([]Attachment, 0)
	for _, object := range objects {
		dict, ok := objectDictionary(object.body)
		if !ok || !bytes.Contains(dict, []byte("/EF")) {
			continue
		}
		attachment, ok, err := attachmentFromFilespec(objects, dict, "direct")
		if err != nil {
			return nil, err
		}
		if ok {
			attachments = append(attachments, attachment)
		}
	}
	return attachments, nil
}

// AttachmentNameTree lists embedded files reachable from the catalog
// /Names /EmbeddedFiles name tree.
func (d *Document) AttachmentNameTree() ([]Attachment, error) {
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
	embeddedFilesRef := directReference(namesDict, "EmbeddedFiles")
	if embeddedFilesRef == nil {
		return nil, nil
	}
	attachments := make([]Attachment, 0)
	err := collectAttachmentNameTree(objects, *embeddedFilesRef, map[objectRef]bool{}, &attachments)
	if err != nil {
		return nil, err
	}
	return attachments, nil
}

func collectAttachmentNameTree(objects []indirectObject, ref ObjectReference, seen map[objectRef]bool, out *[]Attachment) error {
	key := objectRef{number: ref.Number, gen: ref.Generation}
	if seen[key] {
		return unsupported(fmt.Sprintf("embedded file name tree cycle at %d %d R", ref.Number, ref.Generation))
	}
	seen[key] = true
	object, ok := indirectObjectByNumber(objects, ref.Number, ref.Generation)
	if !ok {
		return unsupported(fmt.Sprintf("embedded file name tree node %d %d R is missing", ref.Number, ref.Generation))
	}
	dict, ok := objectDictionary(object.body)
	if !ok {
		return unsupported(fmt.Sprintf("embedded file name tree node %d is not a dictionary", object.number))
	}
	for _, kid := range directReferenceArray(dict, "Kids") {
		if err := collectAttachmentNameTree(objects, kid, seen, out); err != nil {
			return err
		}
	}
	namesRaw, ok := directArrayValue(dict, "Names")
	if !ok {
		return nil
	}
	pairs, err := parseNameTreeReferencePairs(namesRaw, "EmbeddedFiles")
	if err != nil {
		return err
	}
	for _, pair := range pairs {
		filespecObject, ok := indirectObjectByNumber(objects, pair.ref.Number, pair.ref.Generation)
		if !ok {
			return unsupported(fmt.Sprintf("embedded file %q filespec %d %d R is missing", pair.name, pair.ref.Number, pair.ref.Generation))
		}
		filespecDict, ok := objectDictionary(filespecObject.body)
		if !ok {
			return unsupported(fmt.Sprintf("embedded file %q filespec object is not a dictionary", pair.name))
		}
		attachment, ok, err := attachmentFromFilespec(objects, filespecDict, "name-tree")
		if err != nil {
			return err
		}
		if !ok {
			return unsupported(fmt.Sprintf("embedded file name tree entry %q does not reference a filespec with an embedded file", pair.name))
		}
		if attachment.Name == "" {
			attachment.Name = pair.name
		}
		*out = append(*out, attachment)
	}
	return nil
}

func attachmentFromFilespec(objects []indirectObject, dict []byte, source string) (Attachment, bool, error) {
	streamRef, ok := filespecEmbeddedFileRef(dict)
	if !ok {
		return Attachment{}, false, nil
	}
	streamObject, ok := indirectObjectByNumber(objects, streamRef.number, streamRef.gen)
	if !ok {
		return Attachment{}, false, unsupported(fmt.Sprintf("embedded file stream %d %d R is missing", streamRef.number, streamRef.gen))
	}
	content, filter, err := embeddedFileStreamContent(streamObject)
	if err != nil {
		return Attachment{}, false, err
	}
	return Attachment{
		Name:         filespecName(dict),
		ObjectNumber: streamObject.number,
		Source:       source,
		Filter:       filter,
		Size:         len(content),
		Content:      bytes.Clone(content),
	}, true, nil
}

type objectRef struct {
	number int
	gen    int
}

func objectDictionary(body []byte) ([]byte, bool) {
	dictStart := bytes.Index(body, []byte("<<"))
	if dictStart == -1 {
		return nil, false
	}
	dictEnd, ok := scanDictionaryEnd(body, dictStart)
	if !ok {
		return nil, false
	}
	return body[dictStart:dictEnd], true
}

func parsePDFTextValue(input []byte) (string, bool) {
	input = bytes.TrimSpace(input)
	switch {
	case len(input) > 0 && input[0] == '(':
		end, ok := scanLiteralEnd(input, 0)
		if !ok {
			return "", false
		}
		return decodePDFTextBytes(decodeLiteralBytes(input[1:end])), true
	case len(input) > 0 && input[0] == '<' && (len(input) == 1 || input[1] != '<'):
		end := bytes.IndexByte(input[1:], '>')
		if end == -1 {
			return "", false
		}
		raw, err := hex.DecodeString(strings.Map(dropPDFSpace, string(input[1:1+end])))
		if err != nil {
			return "", false
		}
		return decodePDFTextBytes(raw), true
	default:
		return "", false
	}
}

func parsePDFRef(input []byte) (objectRef, bool) {
	fields := bytes.Fields(input)
	if len(fields) < 3 || string(fields[2]) != "R" {
		return objectRef{}, false
	}
	number, err := strconv.Atoi(string(fields[0]))
	if err != nil {
		return objectRef{}, false
	}
	gen, err := strconv.Atoi(string(fields[1]))
	if err != nil {
		return objectRef{}, false
	}
	return objectRef{number: number, gen: gen}, true
}

func filespecEmbeddedFileRef(dict []byte) (objectRef, bool) {
	efRaw, ok := directNameValue(dict, "EF")
	if !ok {
		return objectRef{}, false
	}
	efStart := bytes.Index(efRaw, []byte("<<"))
	if efStart == -1 {
		return objectRef{}, false
	}
	efEnd, ok := scanDictionaryEnd(efRaw, efStart)
	if !ok {
		return objectRef{}, false
	}
	fRaw, ok := directNameValue(efRaw[efStart:efEnd], "F")
	if !ok {
		return objectRef{}, false
	}
	return parsePDFRef(fRaw)
}

func filespecName(dict []byte) string {
	for _, key := range []string{"UF", "F"} {
		raw, ok := directNameValue(dict, key)
		if !ok {
			continue
		}
		if value, ok := parsePDFTextValue(raw); ok {
			return value
		}
	}
	return ""
}

func indirectObjectByNumber(objects []indirectObject, number, gen int) (indirectObject, bool) {
	for _, object := range objects {
		if object.number == number && object.gen == gen {
			return object, true
		}
	}
	return indirectObject{}, false
}

func embeddedFileStreamContent(object indirectObject) ([]byte, string, error) {
	dict, ok := objectDictionary(object.body)
	if !ok || !hasPDFNameEntry(dict, "Type", "EmbeddedFile") {
		return nil, "", unsupported(fmt.Sprintf("object %d is not an embedded file stream", object.number))
	}
	stream, ok := streamBytesAfterDictionary(object.body, dict)
	if !ok {
		return nil, "", unsupported(fmt.Sprintf("embedded file object %d has no stream", object.number))
	}
	filter := streamFilterName(dict)
	switch filter {
	case "":
		return stream, "", nil
	case "FlateDecode":
		content, err := flateDecode(stream)
		if err != nil {
			return nil, "", err
		}
		return content, filter, nil
	default:
		return nil, "", unsupported(fmt.Sprintf("embedded file object %d uses unsupported filter %q", object.number, filter))
	}
}

func streamBytesAfterDictionary(body []byte, dict []byte) ([]byte, bool) {
	dictAt := bytes.Index(body, dict)
	if dictAt == -1 {
		return nil, false
	}
	streamAt := bytes.Index(body[dictAt+len(dict):], []byte("stream"))
	if streamAt == -1 {
		return nil, false
	}
	start := dictAt + len(dict) + streamAt + len("stream")
	if start < len(body) && body[start] == '\r' {
		start++
		if start < len(body) && body[start] == '\n' {
			start++
		}
	} else if start < len(body) && body[start] == '\n' {
		start++
	}
	endRel := bytes.Index(body[start:], []byte("endstream"))
	if endRel == -1 {
		return nil, false
	}
	return bytes.TrimRight(body[start:start+endRel], "\x00\t\n\f\r "), true
}

func streamFilterName(dict []byte) string {
	raw, ok := directNameValue(dict, "Filter")
	if !ok {
		return ""
	}
	match := regexp.MustCompile(`/([A-Za-z0-9]+)`).FindSubmatch(raw)
	if len(match) < 2 {
		return ""
	}
	return string(match[1])
}
