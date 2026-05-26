package oxpdf

import (
	"bytes"
	"context"
	"crypto/x509"
	"strconv"
	"time"

	binaspdf "github.com/oxhq/binas/pkg/adapters/pdf"
)

// Security summarizes encryption and signature boundaries.
type Security struct {
	Encrypted  bool
	Signed     bool
	Encryption Encryption
	Signature  Signature
}

// SignatureTrustPolicy controls the optional certificate-chain status reported
// for document signatures. OxPDF never loads system roots for this check.
type SignatureTrustPolicy struct {
	Roots         []*x509.Certificate
	Intermediates []*x509.Certificate
	CurrentTime   time.Time
}

// SignatureByteRange describes one public byte range used for PDF signature
// digest calculation.
type SignatureByteRange struct {
	Offset int
	Length int
}

// ExternalSigningCallback signs a caller-provided digest with an external key.
//
// OxPDF never accepts private key material in this API. The callback receives
// digest bytes and byte-range metadata only; key lookup stays behind the
// caller's ExternalKeyID boundary.
type ExternalSigningCallback func(context.Context, ExternalSigningRequest) (ExternalSigningResponse, error)

// ExternalSignerOptions configures incremental re-signing with an external
// signer.
type ExternalSignerOptions struct {
	Name               string
	ExternalKeyID      string
	DigestAlgorithm    string
	SignatureContainer string
	SubFilter          string
	ReservedBytes      int
	Sign               ExternalSigningCallback
}

// ExternalSigningCallbackMetadata describes the external signing boundary that
// is safe to report in plans.
type ExternalSigningCallbackMetadata struct {
	Name               string
	ExternalKeyID      string
	DigestAlgorithm    string
	SignatureContainer string
	SubFilter          string
}

// ExternalSigningRequest is sent to the caller-provided external signer.
type ExternalSigningRequest struct {
	Digest             []byte
	DigestAlgorithm    string
	SignatureContainer string
	SubFilter          string
	ByteRanges         []SignatureByteRange
	Signature          Signature
}

// ExternalSigningResponse is returned by the caller-provided external signer.
type ExternalSigningResponse struct {
	Signature          []byte
	SignatureContainer string
	CertificateChain   [][]byte
}

// ExternalSigningPlan reports whether the current PDF shape can be re-signed
// through the narrow external-signer path.
type ExternalSigningPlan struct {
	Supported         bool
	UnsupportedReason string
	CallbackMetadata  ExternalSigningCallbackMetadata
	Signature         Signature
	ByteRanges        []SignatureByteRange
}

// SignatureReSigningVerification reports post-apply proof for incremental
// re-signing.
type SignatureReSigningVerification struct {
	IncrementalUpdate                  bool
	ReparseOK                          bool
	ByteRanges                         []SignatureByteRange
	ByteRangeDigestValidation          bool
	ByteRangeDigestValidationStatus    string
	CertificateTrustValidation         bool
	CertificateTrustValidationStatus   string
	CryptographicSignatureVerification bool
	CryptographicSignatureStatus       string
}

// Encryption summarizes an encryption dictionary when one is present.
type Encryption struct {
	Present            bool
	Filter             string
	SubFilter          string
	V                  int
	R                  int
	Length             int
	EncryptMetadata    bool
	HasEncryptMetadata bool
	StreamFilter       string
	StringFilter       string
	EmbeddedFileFilter string
	CryptFilters       []EncryptionCryptFilter
	PublicKey          bool
	RecipientCount     int
	DictionaryParsed   bool
	ObjectNumber       int
	ObjectGeneration   int
}

// EncryptionCryptFilter summarizes one encryption crypt filter dictionary.
type EncryptionCryptFilter struct {
	Name      string
	CFM       string
	AuthEvent string
	Length    int
}

// Signature summarizes signature markers without making trust claims.
type Signature struct {
	Present                          bool
	ByteRangeCount                   int
	ByteRangeTotalRanges             int
	ByteRangeCoveredBytes            int
	ByteRangeStatus                  string
	ContentsByteLength               int
	HasContentsByteLength            bool
	SubFilter                        string
	Filter                           string
	SigningTime                      string
	ObjectNumber                     int
	ObjectGeneration                 int
	SignatureContainer               string
	DigestAlgorithm                  string
	DigestAlgorithmStatus            string
	CertificateCount                 int
	SignerCertificateSubject         string
	SignerCertificateIssuer          string
	ByteRangeDigestValidation        bool
	ByteRangeDigestValidationStatus  string
	CryptographicValidation          bool
	CryptographicValidationStatus    string
	CertificateTrustValidation       bool
	CertificateTrustValidationStatus string
}

// Security returns read-only security boundary metadata.
func (d *Document) Security() Security {
	return d.securityWithOptions(binaspdf.SecurityMetadataOptions{})
}

// SecurityWithSignatureTrustPolicy returns read-only security boundary metadata
// using only the caller-provided signature trust policy. It reports validation
// status fields without claiming legal trust, revocation, timestamp, or viewer
// policy acceptance.
func (d *Document) SecurityWithSignatureTrustPolicy(policy SignatureTrustPolicy) Security {
	return d.securityWithOptions(binaspdf.SecurityMetadataOptions{
		SignatureTrust: binaspdf.SignatureTrustOptions{
			Roots:         policy.Roots,
			Intermediates: policy.Intermediates,
			CurrentTime:   policy.CurrentTime,
		},
	})
}

func (d *Document) securityWithOptions(options binaspdf.SecurityMetadataOptions) Security {
	if d == nil {
		return Security{}
	}
	metadata := binaspdf.SecurityMetadataForInputWithOptions(d.input, options)
	security := Security{
		Encrypted: metadata.Encrypted,
		Signed:    metadata.Signed,
		Signature: mapBinasSignatureMetadata(metadata.Signature),
	}
	if metadata.Encryption != nil {
		security.Encryption = Encryption{
			Present:            metadata.Encryption.Present,
			Filter:             metadata.Encryption.Filter,
			SubFilter:          metadata.Encryption.SubFilter,
			V:                  intValueFromPointer(metadata.Encryption.V),
			R:                  intValueFromPointer(metadata.Encryption.R),
			Length:             intValueFromPointer(metadata.Encryption.Length),
			EncryptMetadata:    boolValueFromPointer(metadata.Encryption.EncryptMetadata),
			HasEncryptMetadata: metadata.Encryption.EncryptMetadata != nil,
			StreamFilter:       metadata.Encryption.StreamFilter,
			StringFilter:       metadata.Encryption.StringFilter,
			EmbeddedFileFilter: metadata.Encryption.EmbeddedFileFilter,
			CryptFilters:       mapEncryptionCryptFilters(metadata.Encryption.CryptFilters),
			PublicKey:          metadata.Encryption.PublicKey,
			RecipientCount:     intValueFromPointer(metadata.Encryption.RecipientCount),
			DictionaryParsed:   metadata.Encryption.DictionaryParsed,
			ObjectNumber:       intValueFromPointer(metadata.Encryption.ObjectNumber),
			ObjectGeneration:   intValueFromPointer(metadata.Encryption.ObjectGeneration),
		}
	}
	if security.Encrypted && (!security.Encryption.DictionaryParsed || security.Encryption.V == 0 || security.Encryption.R == 0) {
		if fallback, ok := parseEncryptionDictionary(d.input); ok {
			security.Encryption = mergeEncryption(security.Encryption, fallback)
		}
	}
	return security
}

// PlanIncrementalReSigning validates whether input can be incrementally
// re-signed through an external signer without calling the signer.
func PlanIncrementalReSigning(input []byte, options ExternalSignerOptions) (ExternalSigningPlan, error) {
	plan, err := binaspdf.PlanIncrementalReSigning(input, binaspdf.SignatureSigningPlanOptions{
		Callback:         noopBinasExternalSigner,
		CallbackMetadata: mapExternalSignerMetadata(options),
		ReservedBytes:    options.ReservedBytes,
	})
	out := mapExternalSigningPlan(plan)
	if err != nil {
		return out, unsupported(err.Error())
	}
	return out, nil
}

// ApplyIncrementalReSigning appends a new signature dictionary update, sends
// the resulting digest to the caller-provided external signer, embeds the
// returned signature bytes, and verifies the new byte-range digest layer.
func ApplyIncrementalReSigning(ctx context.Context, input []byte, options ExternalSignerOptions) ([]byte, ExternalSigningPlan, SignatureReSigningVerification, error) {
	out, plan, verification, err := binaspdf.ApplyIncrementalReSigning(ctx, input, binaspdf.SignatureSigningPlanOptions{
		Callback:         mapExternalSigningCallback(options.Sign),
		CallbackMetadata: mapExternalSignerMetadata(options),
		ReservedBytes:    options.ReservedBytes,
	})
	mappedPlan := mapExternalSigningPlan(plan)
	mappedVerification := mapSignatureReSigningVerification(verification)
	if err != nil {
		return nil, mappedPlan, mappedVerification, unsupported(err.Error())
	}
	return out, mappedPlan, mappedVerification, nil
}

func intValueFromPointer(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func boolValueFromPointer(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}

func mapBinasSignatureMetadata(signature binaspdf.SignatureMetadata) Signature {
	return Signature{
		Present:                          signature.Present,
		ByteRangeCount:                   signature.ByteRangeCount,
		ByteRangeTotalRanges:             signature.ByteRangeTotalRanges,
		ByteRangeCoveredBytes:            signature.ByteRangeCoveredBytes,
		ByteRangeStatus:                  signature.ByteRangeStatus,
		ContentsByteLength:               intValueFromPointer(signature.ContentsByteLength),
		HasContentsByteLength:            signature.ContentsByteLength != nil,
		SubFilter:                        signature.SubFilter,
		Filter:                           signature.Filter,
		SigningTime:                      signature.SigningTime,
		ObjectNumber:                     intValueFromPointer(signature.ObjectNumber),
		ObjectGeneration:                 intValueFromPointer(signature.ObjectGeneration),
		SignatureContainer:               signature.SignatureContainer,
		DigestAlgorithm:                  signature.DigestAlgorithm,
		DigestAlgorithmStatus:            signature.DigestAlgorithmStatus,
		CertificateCount:                 signature.CertificateCount,
		SignerCertificateSubject:         signature.SignerCertificateSubject,
		SignerCertificateIssuer:          signature.SignerCertificateIssuer,
		ByteRangeDigestValidation:        signature.ByteRangeDigestValidation,
		ByteRangeDigestValidationStatus:  signature.ByteRangeDigestValidationStatus,
		CryptographicValidation:          signature.CryptographicValidation,
		CryptographicValidationStatus:    signature.CryptographicValidationStatus,
		CertificateTrustValidation:       signature.CertificateTrustValidation,
		CertificateTrustValidationStatus: signature.CertificateTrustValidationStatus,
	}
}

func mapExternalSignerMetadata(options ExternalSignerOptions) binaspdf.SignatureSigningCallbackMetadata {
	return binaspdf.SignatureSigningCallbackMetadata{
		Name:               options.Name,
		ExternalKeyID:      options.ExternalKeyID,
		DigestAlgorithm:    options.DigestAlgorithm,
		SignatureContainer: options.SignatureContainer,
		SubFilter:          options.SubFilter,
	}
}

func mapExternalSigningPlan(plan binaspdf.SignatureSigningPlan) ExternalSigningPlan {
	return ExternalSigningPlan{
		Supported:         plan.Supported,
		UnsupportedReason: plan.UnsupportedReason,
		CallbackMetadata: ExternalSigningCallbackMetadata{
			Name:               plan.CallbackMetadata.Name,
			ExternalKeyID:      plan.CallbackMetadata.ExternalKeyID,
			DigestAlgorithm:    plan.CallbackMetadata.DigestAlgorithm,
			SignatureContainer: plan.CallbackMetadata.SignatureContainer,
			SubFilter:          plan.CallbackMetadata.SubFilter,
		},
		Signature:  mapBinasSignatureMetadata(plan.Signature),
		ByteRanges: mapBinasSignatureByteRanges(plan.ByteRanges),
	}
}

func mapBinasSignatureByteRanges(ranges []binaspdf.SignatureByteRange) []SignatureByteRange {
	out := make([]SignatureByteRange, 0, len(ranges))
	for _, r := range ranges {
		out = append(out, SignatureByteRange{Offset: r.Offset, Length: r.Length})
	}
	return out
}

func mapSignatureReSigningVerification(verification binaspdf.SignatureReSigningVerification) SignatureReSigningVerification {
	return SignatureReSigningVerification{
		IncrementalUpdate:                  verification.IncrementalUpdate,
		ReparseOK:                          verification.ReparseOK,
		ByteRanges:                         mapBinasSignatureByteRanges(verification.ByteRanges),
		ByteRangeDigestValidation:          verification.ByteRangeDigestValidation,
		ByteRangeDigestValidationStatus:    verification.ByteRangeDigestValidationStatus,
		CertificateTrustValidation:         verification.CertificateTrustValidation,
		CertificateTrustValidationStatus:   verification.CertificateTrustValidationStatus,
		CryptographicSignatureVerification: verification.CryptographicSignatureVerification,
		CryptographicSignatureStatus:       verification.CryptographicSignatureStatus,
	}
}

func mapExternalSigningCallback(callback ExternalSigningCallback) binaspdf.SignatureSigningCallback {
	if callback == nil {
		return nil
	}
	return func(ctx context.Context, request binaspdf.SignatureSigningRequest) (binaspdf.SignatureSigningResponse, error) {
		response, err := callback(ctx, ExternalSigningRequest{
			Digest:             bytes.Clone(request.Digest),
			DigestAlgorithm:    request.DigestAlgorithm,
			SignatureContainer: request.SignatureContainer,
			SubFilter:          request.SubFilter,
			ByteRanges:         mapBinasSignatureByteRanges(request.ByteRanges),
			Signature:          mapBinasSignatureMetadata(request.Signature),
		})
		if err != nil {
			return binaspdf.SignatureSigningResponse{}, err
		}
		return binaspdf.SignatureSigningResponse{
			Signature:          bytes.Clone(response.Signature),
			SignatureContainer: response.SignatureContainer,
			CertificateChain:   cloneByteSlices(response.CertificateChain),
		}, nil
	}
}

func noopBinasExternalSigner(context.Context, binaspdf.SignatureSigningRequest) (binaspdf.SignatureSigningResponse, error) {
	return binaspdf.SignatureSigningResponse{Signature: []byte{0}}, nil
}

func cloneByteSlices(values [][]byte) [][]byte {
	out := make([][]byte, 0, len(values))
	for _, value := range values {
		out = append(out, bytes.Clone(value))
	}
	return out
}

func mapEncryptionCryptFilters(filters []binaspdf.EncryptionCryptFilter) []EncryptionCryptFilter {
	out := make([]EncryptionCryptFilter, 0, len(filters))
	for _, filter := range filters {
		out = append(out, EncryptionCryptFilter{
			Name:      filter.Name,
			CFM:       filter.CFM,
			AuthEvent: filter.AuthEvent,
			Length:    intValueFromPointer(filter.Length),
		})
	}
	return out
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
