package oxpdf

import (
	"bytes"
	"errors"
	"fmt"

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
	filters := stream.FilterChain
	if len(filters) == 0 && stream.Filter != "" {
		filters = []string{stream.Filter}
	}
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
