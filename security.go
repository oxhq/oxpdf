package oxpdf

import binaspdf "github.com/oxhq/binas/pkg/adapters/pdf"

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
	return security
}

func intValueFromPointer(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
