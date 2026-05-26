package oxpdf

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"encoding/hex"
	"encoding/xml"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
)

// Metadata is the high-level document information dictionary exposed by OxPDF.
type Metadata struct {
	Header       string
	Title        string
	Author       string
	Subject      string
	Keywords     string
	Creator      string
	Producer     string
	CreationDate string
	ModDate      string
	Values       map[string]string
}

// XMPMetadata exposes read-only XMP packet data for common metadata workflows.
type XMPMetadata struct {
	RawXML      string
	Title       string
	Creator     []string
	Description string
	CreateDate  string
	CreatorTool string
	Producer    string
	Keywords    string
	TIFFArtist  []string
	ModifyDate  string
}

// XMPMetadata returns the document-level XMP metadata packet when present.
func (d *Document) XMPMetadata() (XMPMetadata, bool, error) {
	if d == nil {
		return XMPMetadata{}, false, nil
	}
	raw, ok, err := extractXMPMetadataXML(d.input)
	if err != nil {
		return XMPMetadata{}, false, err
	}
	if !ok {
		return XMPMetadata{}, false, nil
	}
	return parseXMPMetadata(raw), true, nil
}

func parseMetadata(input []byte) Metadata {
	values := parseInfoDictionary(input)
	return Metadata{
		Title:        values["Title"],
		Author:       values["Author"],
		Subject:      values["Subject"],
		Keywords:     values["Keywords"],
		Creator:      values["Creator"],
		Producer:     values["Producer"],
		CreationDate: values["CreationDate"],
		ModDate:      values["ModDate"],
		Values:       values,
	}
}

func extractXMPMetadataXML(input []byte) (string, bool, error) {
	refRe := regexp.MustCompile(`/Metadata\s+(\d+)\s+(\d+)\s+R\b`)
	refs := refRe.FindAllSubmatch(input, -1)
	if len(refs) == 0 {
		return "", false, nil
	}
	for i := len(refs) - 1; i >= 0; i-- {
		raw, ok, err := extractXMPMetadataXMLRef(input, refs[i])
		if err != nil || ok {
			return raw, ok, err
		}
	}
	if raw, ok, err := extractXMPMetadataXMLByType(input); err != nil || ok {
		return raw, ok, err
	}
	return "", false, nil
}

func extractXMPMetadataXMLRef(input []byte, ref [][]byte) (string, bool, error) {
	objectRe := regexp.MustCompile(regexp.QuoteMeta(string(ref[1])) + `\s+` + regexp.QuoteMeta(string(ref[2])) + `\s+obj\b`)
	objectAt := objectRe.FindIndex(input)
	if objectAt == nil {
		return "", false, nil
	}
	dictStartRel := bytes.Index(input[objectAt[1]:], []byte("<<"))
	if dictStartRel == -1 {
		return "", false, nil
	}
	return extractXMPMetadataXMLAt(input, objectAt[1]+dictStartRel)
}

func extractXMPMetadataXMLByType(input []byte) (string, bool, error) {
	for _, marker := range [][]byte{[]byte("/Type/Metadata"), []byte("/Type /Metadata")} {
		searchFrom := 0
		for {
			at := bytes.Index(input[searchFrom:], marker)
			if at == -1 {
				break
			}
			at += searchFrom
			objStart := bytes.LastIndex(input[:at], []byte(" obj"))
			if objStart == -1 {
				searchFrom = at + len(marker)
				continue
			}
			lineStart := bytes.LastIndexAny(input[:objStart], "\r\n")
			if lineStart == -1 {
				lineStart = 0
			} else {
				lineStart++
			}
			dictStart := bytes.Index(input[lineStart:], []byte("<<"))
			if dictStart == -1 {
				searchFrom = at + len(marker)
				continue
			}
			raw, ok, err := extractXMPMetadataXMLAt(input, lineStart+dictStart)
			if err != nil || ok {
				return raw, ok, err
			}
			searchFrom = at + len(marker)
		}
	}
	return "", false, nil
}

func extractXMPMetadataXMLAt(input []byte, dictStart int) (string, bool, error) {
	dictEnd, ok := scanDictionaryEnd(input, dictStart)
	if !ok {
		return "", false, nil
	}
	dict := input[dictStart:dictEnd]
	if !bytes.Contains(dict, []byte("/Type")) || !bytes.Contains(dict, []byte("/Metadata")) || !bytes.Contains(dict, []byte("/Subtype")) || !bytes.Contains(dict, []byte("/XML")) {
		return "", false, nil
	}
	streamAtRel := bytes.Index(input[dictEnd:], []byte("stream"))
	if streamAtRel == -1 {
		return "", false, nil
	}
	streamStart := dictEnd + streamAtRel + len("stream")
	if streamStart < len(input) && input[streamStart] == '\r' {
		streamStart++
		if streamStart < len(input) && input[streamStart] == '\n' {
			streamStart++
		}
	} else if streamStart < len(input) && input[streamStart] == '\n' {
		streamStart++
	}
	streamEndRel := bytes.Index(input[streamStart:], []byte("endstream"))
	if streamEndRel == -1 {
		return "", false, nil
	}
	stream := bytes.TrimRight(input[streamStart:streamStart+streamEndRel], "\x00\t\n\f\r ")
	if bytes.Contains(dict, []byte("/Filter")) {
		if !bytes.Contains(dict, []byte("/FlateDecode")) {
			return "", false, unsupported("XMP metadata stream filter is not supported")
		}
		decoded, err := flateDecode(stream)
		if err != nil {
			return "", false, err
		}
		stream = decoded
	}
	if !bytes.Contains(stream, []byte("<x:xmpmeta")) && !bytes.Contains(stream, []byte("<xmpmeta")) {
		return "", false, nil
	}
	return string(stream), true, nil
}

func flateDecode(input []byte) ([]byte, error) {
	if reader, err := zlib.NewReader(bytes.NewReader(input)); err == nil {
		defer reader.Close()
		return io.ReadAll(reader)
	}
	reader := flate.NewReader(bytes.NewReader(input))
	defer reader.Close()
	return io.ReadAll(reader)
}

func parseXMPMetadata(raw string) XMPMetadata {
	out := XMPMetadata{RawXML: raw}
	decoder := xml.NewDecoder(strings.NewReader(raw))
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		applyXMPAttributes(&out, start.Attr)
		switch start.Name.Local {
		case "Artist":
			if text, ok := readXMLElementText(decoder, start.Name); ok {
				out.TIFFArtist = append(out.TIFFArtist, text)
			}
		case "title":
			if values, ok := readXMLElementTextValues(decoder, start.Name); ok && len(values) > 0 {
				out.Title = values[0]
			}
		case "creator":
			if values, ok := readXMLElementTextValues(decoder, start.Name); ok {
				out.Creator = append(out.Creator, values...)
			}
		case "description":
			if values, ok := readXMLElementTextValues(decoder, start.Name); ok && len(values) > 0 {
				out.Description = values[0]
			}
		case "CreateDate":
			if text, ok := readXMLElementText(decoder, start.Name); ok {
				out.CreateDate = normalizeXMPDate(text)
			}
		case "CreatorTool":
			if text, ok := readXMLElementText(decoder, start.Name); ok {
				out.CreatorTool = text
			}
		case "Producer":
			if text, ok := readXMLElementText(decoder, start.Name); ok {
				out.Producer = text
			}
		case "Keywords":
			if text, ok := readXMLElementText(decoder, start.Name); ok {
				out.Keywords = text
			}
		case "ModifyDate":
			if text, ok := readXMLElementText(decoder, start.Name); ok {
				out.ModifyDate = normalizeXMPDate(text)
			}
		}
	}
	return out
}

func applyXMPAttributes(out *XMPMetadata, attrs []xml.Attr) {
	for _, attr := range attrs {
		value := strings.TrimSpace(attr.Value)
		if value == "" {
			continue
		}
		switch attr.Name.Local {
		case "title":
			if out.Title == "" {
				out.Title = value
			}
		case "creator":
			out.Creator = append(out.Creator, value)
		case "description":
			if out.Description == "" {
				out.Description = value
			}
		case "CreateDate":
			if out.CreateDate == "" {
				out.CreateDate = normalizeXMPDate(value)
			}
		case "CreatorTool":
			if out.CreatorTool == "" {
				out.CreatorTool = value
			}
		case "Producer":
			if out.Producer == "" {
				out.Producer = value
			}
		case "Keywords":
			if out.Keywords == "" {
				out.Keywords = value
			}
		case "ModifyDate":
			if out.ModifyDate == "" {
				out.ModifyDate = normalizeXMPDate(value)
			}
		}
	}
}

func readXMLElementTextValues(decoder *xml.Decoder, name xml.Name) ([]string, bool) {
	var values []string
	var direct strings.Builder
	for {
		token, err := decoder.Token()
		if err != nil {
			return nil, false
		}
		switch t := token.(type) {
		case xml.CharData:
			direct.Write([]byte(t))
		case xml.StartElement:
			switch t.Name.Local {
			case "Alt", "Bag", "Seq":
				continue
			case "li":
				if text, ok := readXMLElementText(decoder, t.Name); ok && text != "" {
					values = append(values, text)
				}
			default:
				if err := decoder.Skip(); err != nil {
					return nil, false
				}
			}
		case xml.EndElement:
			if t.Name.Local == name.Local {
				if text := strings.TrimSpace(direct.String()); text != "" {
					values = append([]string{text}, values...)
				}
				return values, true
			}
		}
	}
}

func readXMLElementText(decoder *xml.Decoder, name xml.Name) (string, bool) {
	var text strings.Builder
	for {
		token, err := decoder.Token()
		if err != nil {
			return "", false
		}
		switch t := token.(type) {
		case xml.CharData:
			text.Write([]byte(t))
		case xml.EndElement:
			if t.Name.Local == name.Local {
				return strings.TrimSpace(text.String()), true
			}
		case xml.StartElement:
			if err := decoder.Skip(); err != nil {
				return "", false
			}
		}
	}
}

func normalizeXMPDate(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			if _, offset := parsed.Zone(); offset != 0 || strings.HasSuffix(value, "Z") {
				return parsed.UTC().Format("2006-01-02T15:04:05")
			}
			return parsed.Format("2006-01-02T15:04:05")
		}
	}
	return value
}

func parseInfoDictionary(input []byte) map[string]string {
	refRe := regexp.MustCompile(`/Info\s+(\d+)\s+(\d+)\s+R\b`)
	refs := refRe.FindAllSubmatch(input, -1)
	if len(refs) == 0 {
		return map[string]string{}
	}
	ref := refs[len(refs)-1]
	objectRe := regexp.MustCompile(regexp.QuoteMeta(string(ref[1])) + `\s+` + regexp.QuoteMeta(string(ref[2])) + `\s+obj\b`)
	objectAt := objectRe.FindIndex(input)
	if objectAt == nil {
		return map[string]string{}
	}
	dictStartRel := bytes.Index(input[objectAt[1]:], []byte("<<"))
	if dictStartRel == -1 {
		return map[string]string{}
	}
	dictStart := objectAt[1] + dictStartRel
	dictEnd, ok := scanDictionaryEnd(input, dictStart)
	if !ok {
		return map[string]string{}
	}
	return parseInfoPairs(input[dictStart+2 : dictEnd-2])
}

func parseInfoPairs(dict []byte) map[string]string {
	values := map[string]string{}
	for i := 0; i < len(dict); {
		i = skipPDFSpace(dict, i)
		if i >= len(dict) {
			return values
		}
		if dict[i] != '/' {
			i++
			continue
		}
		keyStart := i + 1
		i = keyStart
		for i < len(dict) && !isPDFSpaceByte(dict[i]) && !isPDFDelimiterByte(dict[i]) {
			i++
		}
		key := string(dict[keyStart:i])
		i = skipPDFSpace(dict, i)
		if key == "" || i >= len(dict) {
			continue
		}
		value, next, ok := parseInfoValue(dict, i)
		if ok {
			values[key] = value
		}
		i = next
	}
	return values
}

func parseInfoValue(input []byte, start int) (string, int, bool) {
	switch input[start] {
	case '(':
		end, ok := scanLiteralEnd(input, start)
		if !ok {
			return "", start + 1, false
		}
		return decodePDFTextBytes(decodeLiteralBytes(input[start+1 : end])), end + 1, true
	case '<':
		if start+1 < len(input) && input[start+1] == '<' {
			return "", start + 2, false
		}
		end := bytes.IndexByte(input[start+1:], '>')
		if end == -1 {
			return "", len(input), false
		}
		raw, err := hex.DecodeString(strings.Map(dropPDFSpace, string(input[start+1:start+1+end])))
		if err != nil {
			return "", start + 1 + end + 1, false
		}
		return decodePDFTextBytes(raw), start + 1 + end + 1, true
	case '/':
		end := start + 1
		for end < len(input) && !isPDFSpaceByte(input[end]) && !isPDFDelimiterByte(input[end]) {
			end++
		}
		return string(input[start:end]), end, true
	default:
		end := start
		for end < len(input) && !isPDFSpaceByte(input[end]) && !isPDFDelimiterByte(input[end]) {
			end++
		}
		return string(input[start:end]), end, end > start
	}
}

func scanDictionaryEnd(input []byte, start int) (int, bool) {
	depth := 0
	for i := start; i < len(input)-1; i++ {
		switch input[i] {
		case '(':
			end, ok := scanLiteralEnd(input, i)
			if !ok {
				return 0, false
			}
			i = end
		case '<':
			if input[i+1] == '<' {
				depth++
				i++
			} else {
				end := bytes.IndexByte(input[i+1:], '>')
				if end == -1 {
					return 0, false
				}
				i += end + 1
			}
		case '>':
			if input[i+1] == '>' {
				depth--
				i++
				if depth == 0 {
					return i + 1, true
				}
			}
		}
	}
	return 0, false
}

func scanLiteralEnd(input []byte, start int) (int, bool) {
	depth := 1
	escaped := false
	for i := start + 1; i < len(input); i++ {
		if escaped {
			escaped = false
			continue
		}
		switch input[i] {
		case '\\':
			escaped = true
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i, true
			}
		}
	}
	return 0, false
}

func decodeLiteralBytes(input []byte) []byte {
	out := make([]byte, 0, len(input))
	for i := 0; i < len(input); i++ {
		if input[i] != '\\' || i+1 >= len(input) {
			out = append(out, input[i])
			continue
		}
		i++
		switch input[i] {
		case 'n':
			out = append(out, '\n')
		case 'r':
			out = append(out, '\r')
		case 't':
			out = append(out, '\t')
		case 'b':
			out = append(out, '\b')
		case 'f':
			out = append(out, '\f')
		case '(', ')', '\\':
			out = append(out, input[i])
		case '\r', '\n':
			if input[i] == '\r' && i+1 < len(input) && input[i+1] == '\n' {
				i++
			}
		default:
			if input[i] >= '0' && input[i] <= '7' {
				start := i
				for i+1 < len(input) && i+1 < start+3 && input[i+1] >= '0' && input[i+1] <= '7' {
					i++
				}
				if value, err := strconv.ParseUint(string(input[start:i+1]), 8, 8); err == nil {
					out = append(out, byte(value))
					continue
				}
			}
			out = append(out, input[i])
		}
	}
	return out
}

func decodePDFTextBytes(input []byte) string {
	if len(input) >= 2 && input[0] == 0xfe && input[1] == 0xff {
		u16 := make([]uint16, 0, (len(input)-2)/2)
		for i := 2; i+1 < len(input); i += 2 {
			u16 = append(u16, uint16(input[i])<<8|uint16(input[i+1]))
		}
		return string(utf16.Decode(u16))
	}
	return string(input)
}

func skipPDFSpace(input []byte, start int) int {
	for start < len(input) && isPDFSpaceByte(input[start]) {
		start++
	}
	return start
}

func isPDFSpaceByte(b byte) bool {
	switch b {
	case 0, '\t', '\n', '\f', '\r', ' ':
		return true
	default:
		return false
	}
}

func isPDFDelimiterByte(b byte) bool {
	switch b {
	case '(', ')', '<', '>', '[', ']', '{', '}', '/', '%':
		return true
	default:
		return false
	}
}

func dropPDFSpace(r rune) rune {
	switch r {
	case '\x00', '\t', '\n', '\f', '\r', ' ':
		return -1
	default:
		return r
	}
}
