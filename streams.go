package oxpdf

import (
	"bytes"
	"encoding/ascii85"
	"errors"
	"fmt"
	"regexp"
	"strconv"

	binaspdf "github.com/oxhq/binas/pkg/adapters/pdf"
	"github.com/oxhq/binas/pkg/core"
)

// ErrStreamIndexOutOfRange reports a zero-based stream index outside the
// document.
var ErrStreamIndexOutOfRange = errors.New("oxpdf: stream index out of range")

// Stream describes a PDF stream and the filter/editability metadata exposed by
// the backing parser.
type Stream struct {
	Index            int
	ObjectNumber     int
	ObjectGeneration int
	SpanStart        int64
	SpanEnd          int64
	EncodedLength    int
	DecodedLength    int
	HasDecodedLength bool
	Filter           string
	FilterChain      []string
	DecodeParms      string
	ImageXObject     bool
	FilterCapability string
	Editable         bool
	PassThrough      bool
	Target           bool
	Unsupported      string
}

// ImageXObject describes an extracted image XObject payload and the direct
// image metadata needed to interpret the bytes.
type ImageXObject struct {
	ObjectNumber     int
	ObjectGeneration int
	StreamIndex      int
	Width            int
	Height           int
	BitsPerComponent int
	ColorSpace       string
	Filter           string
	Extension        string
	PassThrough      bool
	EncodedLength    int
	DecodedLength    int
	HasDecodedLength bool
	FilterCapability string
	Content          []byte
}

// InlineImage describes an inline image embedded in a page content stream.
type InlineImage struct {
	StreamIndex      int
	Width            int
	Height           int
	BitsPerComponent int
	ColorSpace       string
	Filter           string
	FilterChain      []string
	Extension        string
	PassThrough      bool
	Content          []byte
}

// Streams lists detected PDF streams with filter and byte-boundary metadata.
func (d *Document) Streams() []Stream {
	if d == nil || d.tree == nil {
		return nil
	}
	nodes := d.tree.Query(core.Match{Kind: binaspdf.KindStream})
	streams := make([]Stream, 0, len(nodes))
	for i, node := range nodes {
		meta := node.Meta
		decodedLength, hasDecodedLength := intMeta(meta, "decoded_length")
		streams = append(streams, Stream{
			Index:            i,
			ObjectNumber:     intMetaDefault(meta, "object_number", 0),
			ObjectGeneration: intMetaDefault(meta, "object_generation", 0),
			SpanStart:        node.Span.Start,
			SpanEnd:          node.Span.End,
			EncodedLength:    intMetaDefault(meta, "encoded_length", int(node.Span.Len())),
			DecodedLength:    decodedLength,
			HasDecodedLength: hasDecodedLength,
			Filter:           stringMeta(meta, "filter"),
			FilterChain:      stringSliceMeta(meta, "filter_chain"),
			DecodeParms:      stringMeta(meta, "decode_parms"),
			ImageXObject:     boolMeta(meta, "image_xobject"),
			FilterCapability: stringMeta(meta, "filter_capability"),
			Editable:         boolMeta(meta, "filter_editable"),
			PassThrough:      boolMeta(meta, "filter_pass_through"),
			Target:           boolMeta(meta, "filter_target"),
			Unsupported:      stringMeta(meta, "unsupported"),
		})
	}
	return streams
}

// ImageXObjectStreams lists streams marked as image XObjects by the backing
// parser.
func (d *Document) ImageXObjectStreams() []Stream {
	streams := d.Streams()
	if len(streams) == 0 {
		return nil
	}
	images := make([]Stream, 0)
	for _, stream := range streams {
		if stream.ImageXObject {
			images = append(images, stream)
		}
	}
	return images
}

// ImageXObjects extracts supported image XObject payloads with direct image
// metadata. DCTDecode images return their original encoded bytes. FlateDecode
// images return decoded raw sample bytes when no DecodeParms are present.
//
// Unsupported image filters, filter chains, indirect or complex image metadata,
// and unproven stream/object mappings fail closed with ErrUnsupported.
func (d *Document) ImageXObjects() ([]ImageXObject, error) {
	if d == nil || d.tree == nil {
		return nil, nil
	}
	streams := d.ImageXObjectStreams()
	if len(streams) == 0 {
		return nil, nil
	}
	objects := parseIndirectObjects(d.input)
	images := make([]ImageXObject, 0, len(streams))
	for _, stream := range streams {
		image, err := d.imageXObject(stream, objects)
		if err != nil {
			return nil, err
		}
		images = append(images, image)
	}
	return images, nil
}

// InlineImages extracts supported raw inline images from direct page content
// streams. This is intentionally separate from ImageXObjects: image XObject
// streams are never scanned as inline images.
//
// Only inline images with direct W/H/BPC/CS metadata and exact raw sample
// lengths are supported. Raw image data and ASCII85+Flate inline image filters
// are supported. Other inline image filters, unsupported content stream
// filters, contents arrays, and ambiguous image data fail closed with
// ErrUnsupported.
func (d *Document) InlineImages() ([]InlineImage, error) {
	if d == nil || d.tree == nil {
		return nil, nil
	}
	indexes, err := d.pageContentStreamIndexes()
	if err != nil {
		return nil, err
	}
	if len(indexes) == 0 {
		return nil, nil
	}
	images := make([]InlineImage, 0)
	for _, index := range indexes {
		decoded, err := d.DecodedStream(index)
		if err != nil {
			return nil, err
		}
		streamImages, err := inlineImagesInContentStream(index, decoded)
		if err != nil {
			return nil, err
		}
		images = append(images, streamImages...)
	}
	return images, nil
}

// DecodedStream returns decoded bytes for a stream by its Streams index.
//
// Only identity, ASCII85Decode, and FlateDecode streams without DecodeParms are
// currently decoded. Other filters fail closed with ErrUnsupported.
func (d *Document) DecodedStream(index int) ([]byte, error) {
	if d == nil || d.tree == nil {
		return nil, fmt.Errorf("%w: %d", ErrStreamIndexOutOfRange, index)
	}
	streams := d.Streams()
	if index < 0 || index >= len(streams) {
		return nil, fmt.Errorf("%w: %d", ErrStreamIndexOutOfRange, index)
	}
	stream := streams[index]
	encoded, err := d.encodedStreamBytes(stream)
	if err != nil {
		return nil, err
	}
	filters := streamFilters(stream)
	if len(filters) == 0 {
		return bytes.Clone(encoded), nil
	}
	if stream.DecodeParms != "" {
		return nil, unsupported(fmt.Sprintf("stream %d uses unsupported DecodeParms", index))
	}
	decoded := bytes.Clone(encoded)
	for _, filter := range filters {
		switch filter {
		case "ASCII85Decode":
			var err error
			decoded, err = ascii85Decode(decoded)
			if err != nil {
				return nil, err
			}
		case "FlateDecode":
			var err error
			decoded, err = flateDecode(decoded)
			if err != nil {
				return nil, err
			}
		default:
			return nil, unsupported(fmt.Sprintf("stream %d uses unsupported filter chain %v", index, filters))
		}
	}
	return decoded, nil
}

func ascii85Decode(input []byte) ([]byte, error) {
	encoded := bytes.TrimSpace(input)
	encoded = bytes.TrimPrefix(encoded, []byte("<~"))
	encoded = bytes.TrimSpace(encoded)
	encoded = bytes.TrimSuffix(encoded, []byte("~>"))
	encoded = bytes.TrimSpace(encoded)
	out := make([]byte, len(encoded)*4/5+4)
	n, _, err := ascii85.Decode(out, encoded, true)
	if err != nil {
		return nil, err
	}
	return out[:n], nil
}

func (d *Document) pageContentStreamIndexes() ([]int, error) {
	pages := parsePageObjects(d.input)
	if len(pages) == 0 {
		return nil, nil
	}
	streams := d.Streams()
	objects := parseIndirectObjects(d.input)
	indexByObject := make(map[ObjectReference]int, len(streams))
	for _, stream := range streams {
		number := stream.ObjectNumber
		generation := stream.ObjectGeneration
		if number == 0 {
			if object, ok := streamObjectBySpan(d.input, stream); ok {
				number = object.number
				generation = object.gen
			}
		}
		if number > 0 {
			indexByObject[ObjectReference{Number: number, Generation: generation}] = stream.Index
		}
	}
	indexes := make([]int, 0, len(pages))
	for _, page := range pages {
		ref := directReference(page.dict, "Contents")
		if ref == nil {
			if _, hasContents := directNameValue(page.dict, "Contents"); hasContents {
				return nil, unsupported("page contents use unsupported non-direct stream reference")
			}
			continue
		}
		object, ok := indirectObjectByNumber(objects, ref.Number, ref.Generation)
		if !ok || !bytes.Contains(object.body, []byte("stream")) {
			return nil, unsupported("page contents reference is not a proven stream")
		}
		index, ok := indexByObject[ObjectReference{Number: ref.Number, Generation: ref.Generation}]
		if !ok {
			return nil, unsupported("page content stream is not mapped to stream inventory")
		}
		indexes = append(indexes, index)
	}
	return indexes, nil
}

func inlineImagesInContentStream(streamIndex int, content []byte) ([]InlineImage, error) {
	var images []InlineImage
	for offset := 0; offset < len(content); {
		bi := findPDFOperator(content, []byte("BI"), offset)
		if bi == -1 {
			return images, nil
		}
		id := findPDFOperator(content, []byte("ID"), bi+2)
		if id == -1 {
			return nil, unsupported(fmt.Sprintf("inline image in stream %d has no ID operator", streamIndex))
		}
		dict := content[bi+2 : id]
		image, dataEnd, err := inlineImageFromDictionary(streamIndex, dict, content, id+2)
		if err != nil {
			return nil, err
		}
		images = append(images, image)
		offset = dataEnd
	}
	return images, nil
}

func inlineImageFromDictionary(streamIndex int, dict, content []byte, dataStart int) (InlineImage, int, error) {
	dataStart = skipOnePDFSpace(content, dataStart)
	width := inlineImageInteger(dict, "W", "Width")
	height := inlineImageInteger(dict, "H", "Height")
	bpc := inlineImageInteger(dict, "BPC", "BitsPerComponent")
	colorSpace := inlineImageName(dict, "CS", "ColorSpace")
	if width == nil || *width <= 0 || height == nil || *height <= 0 || bpc == nil || *bpc != 8 || colorSpace == "" {
		return InlineImage{}, 0, unsupported(fmt.Sprintf("inline image in stream %d lacks supported direct metadata", streamIndex))
	}
	filters, err := inlineImageFilters(dict)
	if err != nil {
		return InlineImage{}, 0, err
	}
	colorSpace = normalizeInlineColorSpace(colorSpace)
	components := inlineColorComponents(colorSpace)
	if components == 0 {
		return InlineImage{}, 0, unsupported(fmt.Sprintf("inline image in stream %d uses unsupported color space %s", streamIndex, colorSpace))
	}
	length := *width * *height * components
	dataEnd, eiStart, err := inlineImageDataBounds(streamIndex, content, dataStart, length, filters)
	if err != nil {
		return InlineImage{}, 0, err
	}
	if eiStart+2 > len(content) || !bytes.Equal(content[eiStart:eiStart+2], []byte("EI")) || !isTokenBoundary(content, eiStart+2) {
		return InlineImage{}, 0, unsupported(fmt.Sprintf("inline image in stream %d has no proven EI operator", streamIndex))
	}
	imageData, extension, passThrough, err := decodeInlineImageData(streamIndex, bytes.TrimSpace(bytes.Clone(content[dataStart:dataEnd])), filters)
	if err != nil {
		return InlineImage{}, 0, err
	}
	if len(imageData) != length {
		return InlineImage{}, 0, unsupported(fmt.Sprintf("inline image in stream %d decoded data length is unsupported", streamIndex))
	}
	return InlineImage{
		StreamIndex:      streamIndex,
		Width:            *width,
		Height:           *height,
		BitsPerComponent: *bpc,
		ColorSpace:       colorSpace,
		Filter:           firstFilter(filters),
		FilterChain:      append([]string(nil), filters...),
		Extension:        extension,
		PassThrough:      passThrough,
		Content:          imageData,
	}, eiStart + 2, nil
}

func inlineImageDataBounds(streamIndex int, content []byte, dataStart, rawLength int, filters []string) (int, int, error) {
	if len(filters) > 0 {
		eiStart := findPDFOperator(content, []byte("EI"), dataStart)
		if eiStart == -1 {
			return 0, 0, unsupported(fmt.Sprintf("inline image in stream %d has no proven EI operator", streamIndex))
		}
		return eiStart, eiStart, nil
	}
	if dataStart < 0 || dataStart+rawLength >= len(content) {
		return 0, 0, unsupported(fmt.Sprintf("inline image in stream %d has truncated raw data", streamIndex))
	}
	dataEnd := dataStart + rawLength
	if !isPDFSpaceByte(content[dataEnd]) {
		return 0, 0, unsupported(fmt.Sprintf("inline image in stream %d has ambiguous raw data length", streamIndex))
	}
	return dataEnd, skipPDFSpace(content, dataEnd), nil
}

func findPDFOperator(input, operator []byte, start int) int {
	for start < len(input) {
		at := bytes.Index(input[start:], operator)
		if at == -1 {
			return -1
		}
		at += start
		if (at == 0 || isPDFSpaceByte(input[at-1])) && isTokenBoundary(input, at+len(operator)) {
			return at
		}
		start = at + len(operator)
	}
	return -1
}

func inlineImageInteger(dict []byte, shortKey, longKey string) *int {
	if value := directInteger(dict, shortKey); value != nil {
		return value
	}
	return directInteger(dict, longKey)
}

func inlineImageName(dict []byte, shortKey, longKey string) string {
	if value := directPDFNameValue(dict, shortKey); value != "" {
		return value
	}
	return directPDFNameValue(dict, longKey)
}

func directPDFNameValue(dict []byte, key string) string {
	raw, ok := directNameValue(dict, key)
	if !ok {
		return ""
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '/' {
		return ""
	}
	end := 1
	for end < len(raw) && !isPDFSpaceByte(raw[end]) && !isPDFDelimiterByte(raw[end]) {
		end++
	}
	return string(raw[1:end])
}

func inlineImageFilters(dict []byte) ([]string, error) {
	raw, ok := directNameValue(dict, "F")
	if !ok {
		raw, ok = directNameValue(dict, "Filter")
	}
	if !ok {
		return nil, nil
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil, nil
	}
	if raw[0] == '/' {
		name := directPDFNameValue([]byte("/F "+string(raw)), "F")
		if name == "" {
			return nil, unsupported("inline image uses unsupported image filter")
		}
		return []string{normalizeInlineFilter(name)}, nil
	}
	if raw[0] != '[' {
		return nil, unsupported("inline image uses unsupported image filter")
	}
	closeAt := bytes.IndexByte(raw, ']')
	if closeAt == -1 {
		return nil, unsupported("inline image uses unsupported image filter array")
	}
	fields := bytes.Fields(raw[1:closeAt])
	filters := make([]string, 0, len(fields))
	for _, field := range fields {
		if len(field) == 0 || field[0] != '/' {
			return nil, unsupported("inline image uses unsupported image filter array")
		}
		filters = append(filters, normalizeInlineFilter(string(field[1:])))
	}
	return filters, nil
}

func normalizeInlineFilter(filter string) string {
	switch filter {
	case "A85":
		return "ASCII85Decode"
	case "Fl":
		return "FlateDecode"
	default:
		return filter
	}
}

func decodeInlineImageData(streamIndex int, data []byte, filters []string) ([]byte, string, bool, error) {
	if len(filters) == 0 {
		return data, "raw", false, nil
	}
	if !sameStringSlices(filters, []string{"ASCII85Decode", "FlateDecode"}) {
		return nil, "", false, unsupported(fmt.Sprintf("inline image in stream %d uses unsupported image filter chain %v", streamIndex, filters))
	}
	decoded, err := ascii85Decode(data)
	if err != nil {
		return nil, "", false, err
	}
	decoded, err = flateDecode(decoded)
	if err != nil {
		return nil, "", false, err
	}
	return decoded, "raw", false, nil
}

func sameStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func firstFilter(filters []string) string {
	if len(filters) == 0 {
		return ""
	}
	return filters[0]
}

func normalizeInlineColorSpace(colorSpace string) string {
	switch colorSpace {
	case "G":
		return "DeviceGray"
	case "RGB":
		return "DeviceRGB"
	case "CMYK":
		return "DeviceCMYK"
	default:
		return colorSpace
	}
}

func inlineColorComponents(colorSpace string) int {
	switch colorSpace {
	case "DeviceGray":
		return 1
	case "DeviceRGB":
		return 3
	case "DeviceCMYK":
		return 4
	default:
		return 0
	}
}

func skipOnePDFSpace(input []byte, pos int) int {
	if pos < len(input) && isPDFSpaceByte(input[pos]) {
		return pos + 1
	}
	return pos
}

func (d *Document) imageXObject(stream Stream, objects []indirectObject) (ImageXObject, error) {
	object, ok := streamObject(stream, objects)
	if !ok {
		object, ok = streamObjectBySpan(d.input, stream)
	}
	if !ok {
		return ImageXObject{}, unsupported(fmt.Sprintf("image stream %d has no proven indirect object", stream.Index))
	}
	if stream.ObjectNumber == 0 {
		stream.ObjectNumber = object.number
		stream.ObjectGeneration = object.gen
	}
	dict, ok := objectDictionary(object.body)
	if !ok {
		return ImageXObject{}, unsupported(fmt.Sprintf("image stream %d has no direct image dictionary", stream.Index))
	}
	meta, err := directImageXObjectMetadata(dict, stream)
	if err != nil {
		return ImageXObject{}, err
	}
	content, extension, passThrough, err := d.imageXObjectContent(stream)
	if err != nil {
		return ImageXObject{}, err
	}
	meta.Content = content
	meta.Extension = extension
	meta.PassThrough = passThrough
	return meta, nil
}

func streamObject(stream Stream, objects []indirectObject) (indirectObject, bool) {
	if stream.ObjectNumber <= 0 {
		return indirectObject{}, false
	}
	for _, object := range objects {
		if object.number == stream.ObjectNumber && object.gen == stream.ObjectGeneration {
			return object, true
		}
	}
	return indirectObject{}, false
}

func streamObjectBySpan(input []byte, stream Stream) (indirectObject, bool) {
	headerRe := regexp.MustCompile(`(?m)(\d+)\s+(\d+)\s+obj\b`)
	matches := headerRe.FindAllSubmatchIndex(input, -1)
	for _, match := range matches {
		bodyStart := match[1]
		endRel := bytes.Index(input[bodyStart:], []byte("endobj"))
		if endRel == -1 {
			continue
		}
		bodyEnd := bodyStart + endRel
		if stream.SpanStart < int64(bodyStart) || stream.SpanEnd > int64(bodyEnd) {
			continue
		}
		number, _ := strconv.Atoi(string(input[match[2]:match[3]]))
		gen, _ := strconv.Atoi(string(input[match[4]:match[5]]))
		return indirectObject{
			number: number,
			gen:    gen,
			body:   input[bodyStart:bodyEnd],
		}, true
	}
	return indirectObject{}, false
}

func directImageXObjectMetadata(dict []byte, stream Stream) (ImageXObject, error) {
	if !hasPDFNameEntry(dict, "Type", "XObject") || !hasPDFNameEntry(dict, "Subtype", "Image") {
		return ImageXObject{}, unsupported(fmt.Sprintf("stream %d is not a direct image XObject", stream.Index))
	}
	width := directInteger(dict, "Width")
	height := directInteger(dict, "Height")
	bitsPerComponent := directInteger(dict, "BitsPerComponent")
	colorSpace := topLevelPDFName(dict, "ColorSpace")
	if width == nil || *width <= 0 || height == nil || *height <= 0 || bitsPerComponent == nil || *bitsPerComponent <= 0 || colorSpace == "" {
		return ImageXObject{}, unsupported(fmt.Sprintf("image stream %d lacks direct image metadata", stream.Index))
	}
	return ImageXObject{
		ObjectNumber:     stream.ObjectNumber,
		ObjectGeneration: stream.ObjectGeneration,
		StreamIndex:      stream.Index,
		Width:            *width,
		Height:           *height,
		BitsPerComponent: *bitsPerComponent,
		ColorSpace:       colorSpace,
		Filter:           stream.Filter,
		EncodedLength:    stream.EncodedLength,
		DecodedLength:    stream.DecodedLength,
		HasDecodedLength: stream.HasDecodedLength,
		FilterCapability: stream.FilterCapability,
	}, nil
}

func (d *Document) imageXObjectContent(stream Stream) ([]byte, string, bool, error) {
	filters := streamFilters(stream)
	if len(filters) != 1 {
		return nil, "", false, unsupported(fmt.Sprintf("image stream %d uses unsupported filter chain %v", stream.Index, filters))
	}
	if stream.DecodeParms != "" && stream.DecodeParms != "null" {
		return nil, "", false, unsupported(fmt.Sprintf("image stream %d uses unsupported DecodeParms", stream.Index))
	}
	switch filters[0] {
	case "DCTDecode":
		encoded, err := d.encodedStreamBytes(stream)
		if err != nil {
			return nil, "", false, err
		}
		return bytes.Clone(encoded), "jpg", true, nil
	case "FlateDecode":
		decoded, err := d.DecodedStream(stream.Index)
		if err != nil {
			return nil, "", false, err
		}
		return decoded, "raw", false, nil
	default:
		return nil, "", false, unsupported(fmt.Sprintf("image stream %d uses unsupported image filter %s", stream.Index, filters[0]))
	}
}

func streamFilters(stream Stream) []string {
	if len(stream.FilterChain) > 0 {
		return stream.FilterChain
	}
	if stream.Filter != "" {
		return []string{stream.Filter}
	}
	return nil
}

func (d *Document) encodedStreamBytes(stream Stream) ([]byte, error) {
	if stream.SpanStart < 0 || stream.SpanEnd < stream.SpanStart || stream.SpanEnd > int64(len(d.input)) {
		return nil, unsupported(fmt.Sprintf("stream %d has invalid byte boundaries", stream.Index))
	}
	encoded := d.input[stream.SpanStart:stream.SpanEnd]
	if stream.EncodedLength >= 0 && len(encoded) != stream.EncodedLength {
		return nil, unsupported(fmt.Sprintf("stream %d byte boundaries do not match encoded length", stream.Index))
	}
	return encoded, nil
}

func intMetaDefault(meta map[string]any, key string, fallback int) int {
	if value, ok := intMeta(meta, key); ok {
		return value
	}
	return fallback
}

func intMeta(meta map[string]any, key string) (int, bool) {
	if meta == nil {
		return 0, false
	}
	switch value := meta[key].(type) {
	case int:
		return value, true
	case int64:
		return int(value), true
	case float64:
		return int(value), true
	default:
		return 0, false
	}
}

func boolMeta(meta map[string]any, key string) bool {
	if meta == nil {
		return false
	}
	value, _ := meta[key].(bool)
	return value
}

func stringMeta(meta map[string]any, key string) string {
	if meta == nil {
		return ""
	}
	value, _ := meta[key].(string)
	return value
}

func stringSliceMeta(meta map[string]any, key string) []string {
	if meta == nil {
		return nil
	}
	switch value := meta[key].(type) {
	case []string:
		return append([]string(nil), value...)
	case []any:
		out := make([]string, 0, len(value))
		for _, item := range value {
			text, ok := item.(string)
			if !ok {
				return nil
			}
			out = append(out, text)
		}
		return out
	default:
		return nil
	}
}
