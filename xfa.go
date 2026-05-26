package oxpdf

import (
	"fmt"

	binaspdf "github.com/oxhq/binas/pkg/adapters/pdf"
	"github.com/oxhq/binas/pkg/core"
)

// XFASelector filters XFA packets by label or semantic packet kind.
type XFASelector struct {
	PacketKind string
	Label      string
}

// XFAPacket describes an XFA packet embedded in an AcroForm.
type XFAPacket struct {
	Index            int
	Label            string
	Kind             string
	HasXMLProlog     bool
	RootElement      string
	UnsafeXML        bool
	XMLParseError    string
	ObjectNumber     int
	ObjectGeneration int
	IsStream         bool
	Filter           string
	DecodeParms      string
	HasDecodeError   bool
	DecodeError      string
	TextLength       int
	ByteLength       int
	Preview          string
}

// XFADatasetField describes one leaf value inside an XFA datasets packet.
type XFADatasetField struct {
	PacketIndex int
	Label       string
	Path        string
	Value       string
}

// XFATemplateDatasetMapping links a template field name to a dataset path.
type XFATemplateDatasetMapping struct {
	FieldName           string
	DatasetPath         string
	Value               string
	TemplatePacketIndex int
	DatasetPacketIndex  int
	Label               string
}

// XFASemantics reports whether XFA can be treated as static datasets.
type XFASemantics struct {
	Classification                string
	RequiresRendering             bool
	DatasetSemanticEditsSupported bool
	DynamicMarkers                []XFADynamicMarker
	Warnings                      []string
	RefusalReason                 string
}

// XFADynamicMarker identifies XFA syntax that requires renderer semantics.
type XFADynamicMarker struct {
	PacketIndex int
	Label       string
	PacketKind  string
	Path        string
	Reason      string
}

// XFADatasetFieldEditOptions configures an XFA dataset field edit.
type XFADatasetFieldEditOptions struct {
	Selector XFASelector
}

// XFADatasetFieldEditVerification reports verification for an XFA dataset edit.
type XFADatasetFieldEditVerification struct {
	ReparseOK      bool
	OldTextRemoved bool
	NewSelectable  bool
	PageUnchanged  bool
}

// XFAPackets lists XFA packets embedded in the document.
func (d *Document) XFAPackets(selector ...XFASelector) ([]XFAPacket, error) {
	if d == nil {
		return nil, nil
	}
	packets, err := binaspdf.ListXFAPacketsWithOptions(d.input, binaspdf.XFAPacketListOptions{
		Selector: mapXFASelector(firstXFASelector(selector)),
	})
	if err != nil {
		return nil, classifyParseError(err)
	}
	out := make([]XFAPacket, 0, len(packets))
	for _, packet := range packets {
		out = append(out, mapXFAPacket(packet))
	}
	return out, nil
}

// XFADatasetFields lists leaf values inside XFA datasets packets.
func (d *Document) XFADatasetFields(selector ...XFASelector) ([]XFADatasetField, error) {
	if d == nil {
		return nil, nil
	}
	fields, err := binaspdf.ListXFADatasetFieldsWithOptions(d.input, binaspdf.XFADatasetFieldListOptions{
		Selector: mapXFASelector(firstXFASelector(selector)),
	})
	if err != nil {
		return nil, classifyParseError(err)
	}
	out := make([]XFADatasetField, 0, len(fields))
	for _, field := range fields {
		out = append(out, mapXFADatasetField(field))
	}
	return out, nil
}

// XFATemplateDatasetMappings links static XFA template fields to dataset leaves.
func (d *Document) XFATemplateDatasetMappings() ([]XFATemplateDatasetMapping, error) {
	if d == nil {
		return nil, nil
	}
	mappings, err := binaspdf.ListXFATemplateDatasetMappings(d.input)
	if err != nil {
		return nil, classifyParseError(err)
	}
	out := make([]XFATemplateDatasetMapping, 0, len(mappings))
	for _, mapping := range mappings {
		out = append(out, XFATemplateDatasetMapping{
			FieldName:           mapping.FieldName,
			DatasetPath:         mapping.DatasetPath,
			Value:               mapping.Value,
			TemplatePacketIndex: mapping.TemplatePacketIndex,
			DatasetPacketIndex:  mapping.DatasetPacketIndex,
			Label:               mapping.Label,
		})
	}
	return out, nil
}

// XFASemantics classifies XFA without claiming renderer-grade dynamic support.
func (d *Document) XFASemantics() (XFASemantics, error) {
	if d == nil {
		return XFASemantics{}, nil
	}
	semantics, err := binaspdf.InspectXFASemantics(d.input)
	if err != nil {
		return XFASemantics{}, classifyParseError(err)
	}
	out := XFASemantics{
		Classification:                semantics.Classification,
		RequiresRendering:             semantics.RequiresRendering,
		DatasetSemanticEditsSupported: semantics.DatasetSemanticEditsSupported,
		Warnings:                      append([]string(nil), semantics.Warnings...),
		RefusalReason:                 semantics.RefusalReason,
	}
	for _, marker := range semantics.DynamicMarkers {
		out.DynamicMarkers = append(out.DynamicMarkers, XFADynamicMarker{
			PacketIndex: marker.PacketIndex,
			Label:       marker.Label,
			PacketKind:  marker.PacketKind,
			Path:        marker.Path,
			Reason:      marker.Reason,
		})
	}
	return out, nil
}

// SetXFADatasetField updates one static XFA dataset leaf and returns rewritten PDF bytes.
func (d *Document) SetXFADatasetField(path string, value string, opts ...XFADatasetFieldEditOptions) ([]byte, XFADatasetFieldEditVerification, error) {
	if d == nil {
		return nil, XFADatasetFieldEditVerification{}, unsupported("missing document")
	}
	options := binaspdf.XFADatasetFieldUpdateOptions{}
	if len(opts) > 0 {
		options.Selector = mapXFASelector(opts[0].Selector)
	}
	out, _, verification, err := binaspdf.ApplyXFADatasetFieldUpdateWithOptions(d.input, path, value, options)
	if err != nil {
		return nil, XFADatasetFieldEditVerification{}, unsupported(err.Error())
	}
	if !verification.ReparseOK || !verification.NewSelectable || !verification.PageUnchanged {
		return nil, XFADatasetFieldEditVerification{}, unsupported(fmt.Sprintf("XFA dataset field %q edit did not satisfy verification", path))
	}
	return out, mapXFADatasetVerification(verification), nil
}

func firstXFASelector(selectors []XFASelector) XFASelector {
	if len(selectors) == 0 {
		return XFASelector{}
	}
	return selectors[0]
}

func mapXFASelector(selector XFASelector) binaspdf.XFASelector {
	return binaspdf.XFASelector{
		PacketKind: selector.PacketKind,
		Label:      selector.Label,
	}
}

func mapXFAPacket(packet binaspdf.XFAPacketMetadata) XFAPacket {
	return XFAPacket{
		Index:            packet.Index,
		Label:            packet.Label,
		Kind:             packet.PacketKind,
		HasXMLProlog:     packet.HasXMLProlog,
		RootElement:      packet.RootElement,
		UnsafeXML:        packet.UnsafeXML,
		XMLParseError:    packet.XMLParseError,
		ObjectNumber:     intValueFromPointer(packet.ObjectNumber),
		ObjectGeneration: intValueFromPointer(packet.ObjectGeneration),
		IsStream:         packet.IsStream,
		Filter:           packet.Filter,
		DecodeParms:      packet.DecodeParms,
		HasDecodeError:   packet.HasDecodeError,
		DecodeError:      packet.DecodeError,
		TextLength:       packet.TextLength,
		ByteLength:       packet.ByteLength,
		Preview:          packet.Preview,
	}
}

func mapXFADatasetField(field binaspdf.XFADatasetField) XFADatasetField {
	return XFADatasetField{
		PacketIndex: field.PacketIndex,
		Label:       field.Label,
		Path:        field.Path,
		Value:       field.Value,
	}
}

func mapXFADatasetVerification(verification core.Verification) XFADatasetFieldEditVerification {
	return XFADatasetFieldEditVerification{
		ReparseOK:      verification.ReparseOK,
		OldTextRemoved: verification.OldTextRemoved,
		NewSelectable:  verification.NewSelectable,
		PageUnchanged:  verification.PageUnchanged,
	}
}
