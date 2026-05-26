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
	Script       string
}

// Attachment describes an embedded file payload.
type Attachment struct {
	Name         string
	ObjectNumber int
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

func javascriptActionsForInput(input []byte) ([]JavaScriptAction, error) {
	actions := make([]JavaScriptAction, 0)
	for _, object := range parseIndirectObjects(input) {
		dict, ok := objectDictionary(object.body)
		if !ok || !hasPDFNameEntry(dict, "S", "JavaScript") {
			continue
		}
		raw, ok := directNameValue(dict, "JS")
		if !ok {
			continue
		}
		script, ok := parsePDFTextValue(raw)
		if !ok {
			return nil, unsupported(fmt.Sprintf("JavaScript action %d uses unsupported /JS representation", object.number))
		}
		actions = append(actions, JavaScriptAction{
			ObjectNumber: object.number,
			Script:       script,
		})
	}
	return actions, nil
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
		streamRef, ok := filespecEmbeddedFileRef(dict)
		if !ok {
			continue
		}
		streamObject, ok := indirectObjectByNumber(objects, streamRef.number, streamRef.gen)
		if !ok {
			return nil, unsupported(fmt.Sprintf("embedded file stream %d %d R is missing", streamRef.number, streamRef.gen))
		}
		content, filter, err := embeddedFileStreamContent(streamObject)
		if err != nil {
			return nil, err
		}
		attachments = append(attachments, Attachment{
			Name:         filespecName(dict),
			ObjectNumber: streamObject.number,
			Filter:       filter,
			Size:         len(content),
			Content:      bytes.Clone(content),
		})
	}
	return attachments, nil
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
