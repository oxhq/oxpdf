package oxpdf

import (
	"bytes"
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"
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
