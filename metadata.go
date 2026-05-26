package oxpdf

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"sort"
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

// MetadataWrite describes a conservative document-level metadata update.
// Info replaces the document information dictionary with PDF text strings.
// XMPRawXML installs an uncompressed document-level XMP metadata stream.
type MetadataWrite struct {
	Info      map[string]string
	XMPRawXML string
}

// MetadataWriteVerification reports the reparse checks OxPDF requires before
// returning rewritten metadata bytes.
type MetadataWriteVerification struct {
	ReparseOK   bool
	InfoUpdated bool
	XMPUpdated  bool
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

// SetMetadata appends a verified incremental update for document Info and XMP
// metadata. It intentionally supports only simple table-xref PDFs where OxPDF
// can preserve the existing document graph and prove the output by reopening it.
func (d *Document) SetMetadata(update MetadataWrite) ([]byte, MetadataWriteVerification, error) {
	if d == nil {
		return nil, MetadataWriteVerification{}, unsupported("missing document")
	}
	normalized := MetadataWrite{
		Info:      cloneMetadataInfo(update.Info),
		XMPRawXML: strings.TrimSpace(update.XMPRawXML),
	}
	if len(normalized.Info) == 0 && normalized.XMPRawXML == "" {
		return nil, MetadataWriteVerification{}, unsupported("metadata write requires Info or XMP data")
	}
	for key := range normalized.Info {
		if !validInfoKey(key) {
			return nil, MetadataWriteVerification{}, unsupported(fmt.Sprintf("metadata Info key %q is not a safe PDF name", key))
		}
	}
	if normalized.XMPRawXML != "" {
		if err := validateXMPMetadataXML(normalized.XMPRawXML); err != nil {
			return nil, MetadataWriteVerification{}, err
		}
	}
	out, err := d.appendMetadataIncrementalUpdate(normalized)
	if err != nil {
		return nil, MetadataWriteVerification{}, err
	}
	reopened, err := OpenBytes(out)
	if err != nil {
		return nil, MetadataWriteVerification{}, unsupported(fmt.Sprintf("metadata write did not reparse: %v", err))
	}
	verification := MetadataWriteVerification{ReparseOK: true}
	if len(normalized.Info) > 0 {
		metadata := reopened.Metadata()
		for key, want := range normalized.Info {
			if metadata.Values[key] != want {
				return nil, MetadataWriteVerification{}, unsupported(fmt.Sprintf("metadata Info key %q was not updated after reparse", key))
			}
		}
		verification.InfoUpdated = true
	}
	if normalized.XMPRawXML != "" {
		xmp, ok, err := reopened.XMPMetadata()
		if err != nil {
			return nil, MetadataWriteVerification{}, err
		}
		if !ok || xmp.RawXML != normalized.XMPRawXML {
			return nil, MetadataWriteVerification{}, unsupported("XMP metadata was not updated after reparse")
		}
		verification.XMPUpdated = true
	}
	return out, verification, nil
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

func cloneMetadataInfo(info map[string]string) map[string]string {
	if len(info) == 0 {
		return nil
	}
	out := make(map[string]string, len(info))
	for key, value := range info {
		out[key] = value
	}
	return out
}

func validateXMPMetadataXML(raw string) error {
	if !strings.Contains(raw, "<x:xmpmeta") && !strings.Contains(raw, "<xmpmeta") {
		return unsupported("XMP metadata must contain an xmpmeta packet")
	}
	if bytes.Contains([]byte(raw), []byte("endstream")) || bytes.Contains([]byte(raw), []byte("endobj")) {
		return unsupported("XMP metadata contains an unsafe PDF stream marker")
	}
	decoder := xml.NewDecoder(strings.NewReader(raw))
	for {
		if _, err := decoder.Token(); err != nil {
			if err == io.EOF {
				return nil
			}
			return unsupported(fmt.Sprintf("XMP metadata XML is not well-formed: %v", err))
		}
	}
}

func validInfoKey(key string) bool {
	if key == "" {
		return false
	}
	for i := 0; i < len(key); i++ {
		c := key[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c == '.' {
			continue
		}
		return false
	}
	return true
}

func (d *Document) appendMetadataIncrementalUpdate(update MetadataWrite) ([]byte, error) {
	trailer, ok := parseTrailer(d.input)
	if !ok || trailer.Root == nil {
		return nil, unsupported("metadata write requires a direct trailer Root reference")
	}
	if trailer.Encrypt != nil {
		return nil, unsupported("metadata write refuses encrypted PDFs")
	}
	if trailer.XRefStm != nil {
		return nil, unsupported("metadata write supports only table-xref PDFs")
	}
	xref := d.Xref()
	if xref.HasStream || xref.HasHybridStream {
		return nil, unsupported("metadata write supports only table-xref PDFs")
	}
	prev, ok := lastStartxrefOffset(d.input)
	if !ok {
		return nil, unsupported("metadata write requires startxref")
	}
	objects := parseIndirectObjects(d.input)
	maxObject := trailer.Size - 1
	for _, object := range objects {
		if object.number > maxObject {
			maxObject = object.number
		}
	}
	nextObject := maxObject + 1
	rootRef := *trailer.Root
	infoRef := trailer.Info
	records := make([]metadataIncrementalObject, 0, 3)
	if len(update.Info) > 0 {
		infoRef = &ObjectReference{Number: nextObject, Generation: 0}
		records = append(records, metadataIncrementalObject{
			number: nextObject,
			body:   buildInfoDictionary(update.Info),
		})
		nextObject++
	}
	if update.XMPRawXML != "" {
		metadataRef := ObjectReference{Number: nextObject, Generation: 0}
		records = append(records, metadataIncrementalObject{
			number: nextObject,
			body:   buildXMPMetadataStream(update.XMPRawXML),
		})
		nextObject++
		rootObject, ok := findIndirectObject(d.input, rootRef.Number, rootRef.Generation)
		if !ok {
			return nil, unsupported("metadata write catalog object not found")
		}
		rootDict, ok := objectDictionary(rootObject.body)
		if !ok || !hasPDFNameEntry(rootDict, "Type", "Catalog") {
			return nil, unsupported("metadata write requires a readable catalog dictionary")
		}
		updatedRoot, err := setTopLevelDictionaryEntry(rootDict, "Metadata", fmt.Sprintf("%d %d R", metadataRef.Number, metadataRef.Generation))
		if err != nil {
			return nil, err
		}
		rootRef = ObjectReference{Number: nextObject, Generation: 0}
		records = append(records, metadataIncrementalObject{
			number: nextObject,
			body:   updatedRoot,
		})
		nextObject++
	}
	return appendIncrementalObjects(d.input, records, metadataTrailer{
		size: nextObject,
		root: rootRef,
		info: infoRef,
		prev: prev,
		id:   trailer.ID,
	}), nil
}

type metadataIncrementalObject struct {
	number int
	body   []byte
}

type metadataTrailer struct {
	size int
	root ObjectReference
	info *ObjectReference
	prev int
	id   []string
}

func buildInfoDictionary(info map[string]string) []byte {
	keys := make([]string, 0, len(info))
	for key := range info {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var out bytes.Buffer
	out.WriteString("<<")
	for _, key := range keys {
		fmt.Fprintf(&out, "\n/%s %s", key, encodePDFTextString(info[key]))
	}
	out.WriteString("\n>>")
	return out.Bytes()
}

func encodePDFTextString(value string) string {
	encoded := utf16.Encode([]rune(value))
	raw := make([]byte, 2, 2+len(encoded)*2)
	raw[0] = 0xfe
	raw[1] = 0xff
	for _, unit := range encoded {
		raw = append(raw, byte(unit>>8), byte(unit))
	}
	return "<" + strings.ToUpper(hex.EncodeToString(raw)) + ">"
}

func buildXMPMetadataStream(raw string) []byte {
	return []byte(fmt.Sprintf("<< /Type /Metadata /Subtype /XML /Length %d >>\nstream\n%sendstream", len([]byte(raw)), raw))
}

func appendIncrementalObjects(input []byte, records []metadataIncrementalObject, trailer metadataTrailer) []byte {
	out := bytes.Clone(input)
	if len(out) > 0 && out[len(out)-1] != '\n' {
		out = append(out, '\n')
	}
	offsets := make([]int, len(records))
	for i, record := range records {
		offsets[i] = len(out)
		out = fmt.Appendf(out, "%d 0 obj\n", record.number)
		out = append(out, record.body...)
		out = append(out, "\nendobj\n"...)
	}
	xrefAt := len(out)
	out = fmt.Appendf(out, "xref\n%d %d\n", records[0].number, len(records))
	for _, offset := range offsets {
		out = fmt.Appendf(out, "%010d 00000 n \n", offset)
	}
	out = fmt.Appendf(out, "trailer\n<< /Size %d /Root %d %d R", trailer.size, trailer.root.Number, trailer.root.Generation)
	if trailer.info != nil {
		out = fmt.Appendf(out, " /Info %d %d R", trailer.info.Number, trailer.info.Generation)
	}
	if len(trailer.id) > 0 {
		out = append(out, " /ID ["...)
		for i, id := range trailer.id {
			if i > 0 {
				out = append(out, ' ')
			}
			out = append(out, id...)
		}
		out = append(out, ']')
	}
	out = fmt.Appendf(out, " /Prev %d >>\nstartxref\n%d\n%%%%EOF\n", trailer.prev, xrefAt)
	return out
}

func setTopLevelDictionaryEntry(dict []byte, key, value string) ([]byte, error) {
	entryStart, entryEnd, found, err := topLevelDictionaryEntryRange(dict, key)
	if err != nil {
		return nil, err
	}
	out := bytes.Clone(dict)
	if found {
		out = append(append([]byte{}, dict[:entryStart]...), dict[entryEnd:]...)
	}
	closeAt := bytes.LastIndex(out, []byte(">>"))
	if closeAt == -1 {
		return nil, unsupported("metadata write could not update catalog dictionary")
	}
	prefix := bytes.TrimRight(out[:closeAt], "\x00\t\n\f\r ")
	suffix := out[closeAt:]
	updated := append([]byte{}, prefix...)
	updated = fmt.Appendf(updated, " /%s %s ", key, value)
	updated = append(updated, suffix...)
	return updated, nil
}

func topLevelDictionaryEntryRange(dict []byte, key string) (int, int, bool, error) {
	needle := []byte("/" + key)
	depth := 0
	for i := 0; i < len(dict); i++ {
		switch dict[i] {
		case '(':
			end, ok := scanLiteralEnd(dict, i)
			if !ok {
				return 0, 0, false, unsupported("metadata write found malformed literal string in catalog")
			}
			i = end
		case '<':
			if i+1 < len(dict) && dict[i+1] == '<' {
				depth++
				i++
				continue
			}
			end := bytes.IndexByte(dict[i+1:], '>')
			if end == -1 {
				return 0, 0, false, unsupported("metadata write found malformed hex string in catalog")
			}
			i += end + 1
		case '[':
			end, ok := scanArrayEnd(dict, i)
			if !ok {
				return 0, 0, false, unsupported("metadata write found malformed array in catalog")
			}
			i = end
		case '>':
			if i+1 < len(dict) && dict[i+1] == '>' {
				depth--
				i++
			}
		case '/':
			if depth == 1 && bytes.HasPrefix(dict[i:], needle) && isTokenBoundary(dict, i+len(needle)) {
				valueStart := skipPDFSpace(dict, i+len(needle))
				valueEnd, ok := scanPDFObjectEnd(dict, valueStart)
				if !ok {
					return 0, 0, false, unsupported("metadata write could not scan existing catalog entry")
				}
				return i, valueEnd, true, nil
			}
		}
	}
	return 0, 0, false, nil
}

func scanPDFObjectEnd(input []byte, start int) (int, bool) {
	start = skipPDFSpace(input, start)
	if start >= len(input) {
		return start, false
	}
	switch input[start] {
	case '(':
		end, ok := scanLiteralEnd(input, start)
		return end + 1, ok
	case '<':
		if start+1 < len(input) && input[start+1] == '<' {
			return scanDictionaryEnd(input, start)
		}
		end := bytes.IndexByte(input[start+1:], '>')
		if end == -1 {
			return 0, false
		}
		return start + end + 2, true
	case '[':
		end, ok := scanArrayEnd(input, start)
		return end + 1, ok
	case '/':
		return scanPDFTokenEnd(input, start+1), true
	default:
		firstEnd := scanPDFTokenEnd(input, start)
		secondStart := skipPDFSpace(input, firstEnd)
		secondEnd := scanPDFTokenEnd(input, secondStart)
		thirdStart := skipPDFSpace(input, secondEnd)
		thirdEnd := scanPDFTokenEnd(input, thirdStart)
		if isIntegerToken(input[start:firstEnd]) && isIntegerToken(input[secondStart:secondEnd]) && string(input[thirdStart:thirdEnd]) == "R" {
			return thirdEnd, true
		}
		return firstEnd, firstEnd > start
	}
}

func scanPDFTokenEnd(input []byte, start int) int {
	end := start
	for end < len(input) && !isPDFSpaceByte(input[end]) && !isPDFDelimiterByte(input[end]) {
		end++
	}
	return end
}

func isIntegerToken(input []byte) bool {
	if len(input) == 0 {
		return false
	}
	for _, b := range input {
		if b < '0' || b > '9' {
			return false
		}
	}
	return true
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
