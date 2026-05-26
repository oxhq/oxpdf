package oxpdf

import (
	"bytes"
	"strconv"

	binaspdf "github.com/oxhq/binas/pkg/adapters/pdf"
)

// Security summarizes encryption and signature boundaries.
type Security struct {
	Encrypted  bool
	Signed     bool
	Encryption Encryption
	Signature  Signature
}

// Encryption summarizes an encryption dictionary when one is present.
type Encryption struct {
	Present          bool
	Filter           string
	SubFilter        string
	V                int
	R                int
	Length           int
	PublicKey        bool
	DictionaryParsed bool
	ObjectNumber     int
	ObjectGeneration int
}

// Signature summarizes signature markers without making trust claims.
type Signature struct {
	Present                          bool
	ByteRangeStatus                  string
	SubFilter                        string
	Filter                           string
	SigningTime                      string
	SignatureContainer               string
	DigestAlgorithm                  string
	DigestAlgorithmStatus            string
	ByteRangeDigestValidation        bool
	ByteRangeDigestValidationStatus  string
	CryptographicValidation          bool
	CryptographicValidationStatus    string
	CertificateTrustValidation       bool
	CertificateTrustValidationStatus string
}

// Security returns read-only security boundary metadata.
func (d *Document) Security() Security {
	if d == nil {
		return Security{}
	}
	metadata := binaspdf.SecurityMetadataForInput(d.input)
	security := Security{
		Encrypted: metadata.Encrypted,
		Signed:    metadata.Signed,
		Signature: Signature{
			Present:                          metadata.Signature.Present,
			ByteRangeStatus:                  metadata.Signature.ByteRangeStatus,
			SubFilter:                        metadata.Signature.SubFilter,
			Filter:                           metadata.Signature.Filter,
			SigningTime:                      metadata.Signature.SigningTime,
			SignatureContainer:               metadata.Signature.SignatureContainer,
			DigestAlgorithm:                  metadata.Signature.DigestAlgorithm,
			DigestAlgorithmStatus:            metadata.Signature.DigestAlgorithmStatus,
			ByteRangeDigestValidation:        metadata.Signature.ByteRangeDigestValidation,
			ByteRangeDigestValidationStatus:  metadata.Signature.ByteRangeDigestValidationStatus,
			CryptographicValidation:          metadata.Signature.CryptographicValidation,
			CryptographicValidationStatus:    metadata.Signature.CryptographicValidationStatus,
			CertificateTrustValidation:       metadata.Signature.CertificateTrustValidation,
			CertificateTrustValidationStatus: metadata.Signature.CertificateTrustValidationStatus,
		},
	}
	if metadata.Encryption != nil {
		security.Encryption = Encryption{
			Present:          metadata.Encryption.Present,
			Filter:           metadata.Encryption.Filter,
			SubFilter:        metadata.Encryption.SubFilter,
			V:                intValueFromPointer(metadata.Encryption.V),
			R:                intValueFromPointer(metadata.Encryption.R),
			Length:           intValueFromPointer(metadata.Encryption.Length),
			PublicKey:        metadata.Encryption.PublicKey,
			DictionaryParsed: metadata.Encryption.DictionaryParsed,
			ObjectNumber:     intValueFromPointer(metadata.Encryption.ObjectNumber),
			ObjectGeneration: intValueFromPointer(metadata.Encryption.ObjectGeneration),
		}
	}
	if security.Encrypted && (!security.Encryption.DictionaryParsed || security.Encryption.V == 0 || security.Encryption.R == 0) {
		if fallback, ok := parseEncryptionDictionary(d.input); ok {
			security.Encryption = mergeEncryption(security.Encryption, fallback)
		}
	}
	return security
}

func intValueFromPointer(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func parseEncryptionDictionary(input []byte) (Encryption, bool) {
	trailer, ok := parseTrailer(input)
	if !ok || trailer.Encrypt == nil {
		return Encryption{}, false
	}
	object, ok := findIndirectObject(input, trailer.Encrypt.Number, trailer.Encrypt.Generation)
	if !ok {
		return Encryption{}, false
	}
	dict, ok := objectDictionary(object.body)
	if !ok {
		return Encryption{}, false
	}
	encryption := Encryption{
		Present:          true,
		DictionaryParsed: true,
		ObjectNumber:     trailer.Encrypt.Number,
		ObjectGeneration: trailer.Encrypt.Generation,
		Filter:           topLevelPDFName(dict, "Filter"),
		SubFilter:        topLevelPDFName(dict, "SubFilter"),
		V:                topLevelInteger(dict, "V"),
		R:                topLevelInteger(dict, "R"),
		Length:           topLevelInteger(dict, "Length"),
	}
	if encryption.Length == 0 {
		encryption.Length = cryptFilterLength(dict)
	}
	encryption.PublicKey = encryption.SubFilter != ""
	return encryption, encryption.Filter != "" || encryption.V > 0 || encryption.R > 0 || encryption.Length > 0
}

func mergeEncryption(current, fallback Encryption) Encryption {
	if !current.Present {
		current.Present = fallback.Present
	}
	if current.Filter == "" {
		current.Filter = fallback.Filter
	}
	if current.SubFilter == "" {
		current.SubFilter = fallback.SubFilter
	}
	if current.V == 0 {
		current.V = fallback.V
	}
	if current.R == 0 {
		current.R = fallback.R
	}
	if current.Length == 0 {
		current.Length = fallback.Length
	}
	if !current.PublicKey {
		current.PublicKey = fallback.PublicKey
	}
	if !current.DictionaryParsed {
		current.DictionaryParsed = fallback.DictionaryParsed
	}
	if current.ObjectNumber == 0 {
		current.ObjectNumber = fallback.ObjectNumber
		current.ObjectGeneration = fallback.ObjectGeneration
	}
	return current
}

func directPDFName(dict []byte, key string) string {
	raw, ok := topLevelNameValue(dict, key)
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

func topLevelPDFName(dict []byte, key string) string {
	return directPDFName(dict, key)
}

func topLevelInteger(dict []byte, key string) int {
	raw, ok := topLevelNameValue(dict, key)
	if !ok {
		return 0
	}
	raw = bytes.TrimSpace(raw)
	end := 0
	for end < len(raw) && raw[end] >= '0' && raw[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0
	}
	value, err := strconv.Atoi(string(raw[:end]))
	if err != nil {
		return 0
	}
	return value
}

func topLevelNameValue(dict []byte, key string) ([]byte, bool) {
	needle := []byte("/" + key)
	depth := 0
	for i := 0; i < len(dict); i++ {
		switch dict[i] {
		case '(':
			end, ok := scanLiteralEnd(dict, i)
			if !ok {
				return nil, false
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
				return nil, false
			}
			i += end + 1
		case '>':
			if i+1 < len(dict) && dict[i+1] == '>' {
				depth--
				i++
			}
		case '/':
			if depth == 1 && bytes.HasPrefix(dict[i:], needle) && isTokenBoundary(dict, i+len(needle)) {
				valueStart := skipPDFSpace(dict, i+len(needle))
				return dict[valueStart:], true
			}
		}
	}
	return nil, false
}

func cryptFilterLength(dict []byte) int {
	cfRaw, ok := directNameValue(dict, "CF")
	if !ok {
		return 0
	}
	cfStart := bytes.Index(cfRaw, []byte("<<"))
	if cfStart == -1 {
		return 0
	}
	cfEnd, ok := scanDictionaryEnd(cfRaw, cfStart)
	if !ok {
		return 0
	}
	length := trailerInteger(cfRaw[cfStart:cfEnd], "Length")
	if length > 0 {
		return length * 8
	}
	return 0
}
