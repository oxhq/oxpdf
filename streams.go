package oxpdf

import (
	"bytes"
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

// DecodedStream returns decoded bytes for a stream by its Streams index.
//
// Only identity streams and FlateDecode streams without DecodeParms are
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
	if len(filters) != 1 || filters[0] != "FlateDecode" {
		return nil, unsupported(fmt.Sprintf("stream %d uses unsupported filter chain %v", index, filters))
	}
	if stream.DecodeParms != "" {
		return nil, unsupported(fmt.Sprintf("stream %d uses unsupported DecodeParms", index))
	}
	decoded, err := flateDecode(encoded)
	if err != nil {
		return nil, err
	}
	return decoded, nil
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
