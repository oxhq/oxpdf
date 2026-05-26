package oxpdf

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"
)

func TestSecurityReportsPublicKeyEncryptionMetadata(t *testing.T) {
	doc := &Document{input: []byte(`%PDF-1.7
1 0 obj
<< /Type /Catalog /Encrypt 2 0 R >>
endobj
2 0 obj
<< /Filter /Adobe.PubSec /SubFilter /adbe.pkcs7.s5 /V 4 /R 4 /Length 128 /EncryptMetadata false /StmF /DefaultCryptFilter /StrF /DefaultCryptFilter /EFF /DefaultCryptFilter /Recipients [<0011> <2233>] /CF << /DefaultCryptFilter << /CFM /AESV2 /Length 128 /AuthEvent /DocOpen >> >> >>
endobj
trailer
<< /Size 3 /Root 1 0 R /Encrypt 2 0 R >>
%%EOF
`)}

	security := doc.Security()
	encryption := security.Encryption
	if !security.Encrypted || !encryption.Present || !encryption.PublicKey {
		t.Fatalf("Security().Encryption = %+v, want public-key encryption boundary", encryption)
	}
	if encryption.Filter != "Adobe.PubSec" || encryption.SubFilter != "adbe.pkcs7.s5" || encryption.RecipientCount != 2 {
		t.Fatalf("public-key encryption metadata = %+v", encryption)
	}
	if !encryption.HasEncryptMetadata || encryption.EncryptMetadata {
		t.Fatalf("EncryptMetadata = %t/%t, want present false", encryption.HasEncryptMetadata, encryption.EncryptMetadata)
	}
	if encryption.StreamFilter != "DefaultCryptFilter" || encryption.StringFilter != "DefaultCryptFilter" || encryption.EmbeddedFileFilter != "DefaultCryptFilter" {
		t.Fatalf("crypt filter names = %+v", encryption)
	}
	if len(encryption.CryptFilters) != 1 {
		t.Fatalf("CryptFilters len = %d, want 1: %+v", len(encryption.CryptFilters), encryption.CryptFilters)
	}
	filter := encryption.CryptFilters[0]
	if filter.Name != "DefaultCryptFilter" || filter.CFM != "AESV2" || filter.Length != 128 || filter.AuthEvent != "DocOpen" {
		t.Fatalf("CryptFilters[0] = %+v", filter)
	}
}

func TestSecurityReportsSignatureByteRangeMetadata(t *testing.T) {
	doc := &Document{input: []byte(`%PDF-1.7
1 0 obj
<< /Type /Catalog /AcroForm << /SigFlags 3 /Fields [2 0 R] >> >>
endobj
2 0 obj
<< /FT /Sig /V 3 0 R >>
endobj
3 0 obj
<< /Type /Sig /Filter /Adobe.PPKLite /SubFilter /adbe.pkcs7.detached /ByteRange [0 10 20 30] /Contents <01020f> /M (D:20240526120000Z) >>
endobj
trailer
<< /Size 4 /Root 1 0 R >>
%%EOF
`)}

	security := doc.Security()
	signature := security.Signature
	if !security.Signed || !signature.Present {
		t.Fatalf("Security().Signature = %+v, want signature boundary", signature)
	}
	if signature.Filter != "Adobe.PPKLite" || signature.SubFilter != "adbe.pkcs7.detached" {
		t.Fatalf("signature filters = %+v", signature)
	}
	if signature.ByteRangeCount != 2 || signature.ByteRangeTotalRanges != 2 || signature.ByteRangeCoveredBytes != 40 {
		t.Fatalf("signature byte range metadata = %+v", signature)
	}
	if !signature.HasContentsByteLength || signature.ContentsByteLength != 3 {
		t.Fatalf("ContentsByteLength = %d/%t, want 3/present", signature.ContentsByteLength, signature.HasContentsByteLength)
	}
	if signature.ObjectNumber != 3 || signature.ObjectGeneration != 0 {
		t.Fatalf("signature object = %d %d, want 3 0", signature.ObjectNumber, signature.ObjectGeneration)
	}
}

func TestSecurityWithSignatureTrustPolicyUsesOnlyExplicitRoots(t *testing.T) {
	rootDER, root, leafDER := signatureTrustTestCertificateChain(t)
	input := signatureTrustSignedPDFWithCertificates(t, [][]byte{leafDER, rootDER})
	doc := &Document{input: input}

	withoutPolicy := doc.Security().Signature
	if !withoutPolicy.ByteRangeDigestValidation || withoutPolicy.ByteRangeDigestValidationStatus != "valid" {
		t.Fatalf("default byte-range digest validation = %t/%q, want true/valid", withoutPolicy.ByteRangeDigestValidation, withoutPolicy.ByteRangeDigestValidationStatus)
	}
	if withoutPolicy.CertificateTrustValidation || withoutPolicy.CertificateTrustValidationStatus != "not_performed" {
		t.Fatalf("default certificate trust validation = %t/%q, want false/not_performed", withoutPolicy.CertificateTrustValidation, withoutPolicy.CertificateTrustValidationStatus)
	}

	withPolicy := doc.SecurityWithSignatureTrustPolicy(SignatureTrustPolicy{
		Roots:       []*x509.Certificate{root},
		CurrentTime: time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
	}).Signature
	if !withPolicy.ByteRangeDigestValidation || withPolicy.ByteRangeDigestValidationStatus != "valid" {
		t.Fatalf("policy byte-range digest validation = %t/%q, want true/valid", withPolicy.ByteRangeDigestValidation, withPolicy.ByteRangeDigestValidationStatus)
	}
	if !withPolicy.CertificateTrustValidation || withPolicy.CertificateTrustValidationStatus != "valid" {
		t.Fatalf("policy certificate trust validation = %t/%q, want true/valid", withPolicy.CertificateTrustValidation, withPolicy.CertificateTrustValidationStatus)
	}
	if withPolicy.CryptographicValidationStatus != "byte_range_digest_valid" {
		t.Fatalf("CryptographicValidationStatus = %q, want byte_range_digest_valid", withPolicy.CryptographicValidationStatus)
	}
	if !strings.Contains(withPolicy.SignerCertificateSubject, "OxPDF Test Signer") || !strings.Contains(withPolicy.SignerCertificateIssuer, "OxPDF Test Root") {
		t.Fatalf("signer certificate metadata = subject %q issuer %q", withPolicy.SignerCertificateSubject, withPolicy.SignerCertificateIssuer)
	}
}

type signatureTrustByteRange struct {
	offset int
	length int
}

func signatureTrustSignedPDFWithCertificates(t *testing.T, certificates [][]byte) []byte {
	t.Helper()

	zeroDigest := make([]byte, sha256.Size)
	placeholderCMS := append(signatureTrustMinimalDetachedCMS(zeroDigest, certificates), make([]byte, 8)...)
	input, ranges := signatureTrustPDFWithContentsPlaceholder(t, len(placeholderCMS))
	digest := signatureTrustSHA256DigestForRanges(input, ranges)
	cms := append(signatureTrustMinimalDetachedCMS(digest, certificates), make([]byte, 8)...)
	if len(cms) != len(placeholderCMS) {
		t.Fatalf("CMS length changed from %d to %d", len(placeholderCMS), len(cms))
	}
	return signatureTrustReplaceContentsHex(t, input, cms)
}

func signatureTrustPDFWithContentsPlaceholder(t *testing.T, contentsLen int) ([]byte, []signatureTrustByteRange) {
	t.Helper()

	placeholderHex := strings.Repeat("0", contentsLen*2)
	byteRangePlaceholder := "[0000000000 0000000000 0000000000 0000000000]"
	input := signatureTrustPDF(
		"<< /Type /Catalog /SigFlags 3 /AcroForm << /Fields [2 0 R] >> >>",
		"<< /FT /Sig /T (Approval) /V 3 0 R >>",
		fmt.Sprintf("<< /Type /Sig /Filter /Adobe.PPKLite /SubFilter /adbe.pkcs7.detached /ByteRange %s /Contents <%s> >>", byteRangePlaceholder, placeholderHex),
	)
	contentsStart := bytes.Index(input, []byte("<"+placeholderHex+">"))
	if contentsStart < 0 {
		t.Fatal("signature contents placeholder not found")
	}
	contentsEnd := contentsStart + 1 + len(placeholderHex) + 1
	ranges := []signatureTrustByteRange{
		{offset: 0, length: contentsStart},
		{offset: contentsEnd, length: len(input) - contentsEnd},
	}
	byteRange := fmt.Sprintf("[%010d %010d %010d %010d]", ranges[0].offset, ranges[0].length, ranges[1].offset, ranges[1].length)
	if len(byteRange) != len(byteRangePlaceholder) {
		t.Fatalf("ByteRange replacement length = %d, want %d", len(byteRange), len(byteRangePlaceholder))
	}
	return bytes.Replace(input, []byte(byteRangePlaceholder), []byte(byteRange), 1), ranges
}

func signatureTrustPDF(objects ...string) []byte {
	var out bytes.Buffer
	out.WriteString("%PDF-1.7\n")
	for i, object := range objects {
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i+1, object)
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\n%%EOF\n", len(objects)+1)
	return out.Bytes()
}

func signatureTrustReplaceContentsHex(t *testing.T, input []byte, contents []byte) []byte {
	t.Helper()

	placeholder := []byte("<" + strings.Repeat("0", len(contents)*2) + ">")
	replacement := []byte("<" + strings.ToUpper(hex.EncodeToString(contents)) + ">")
	if !bytes.Contains(input, placeholder) {
		t.Fatal("signature contents placeholder not found")
	}
	return bytes.Replace(input, placeholder, replacement, 1)
}

func signatureTrustSHA256DigestForRanges(input []byte, ranges []signatureTrustByteRange) []byte {
	h := sha256.New()
	for _, r := range ranges {
		_, _ = h.Write(input[r.offset : r.offset+r.length])
	}
	return h.Sum(nil)
}

func signatureTrustMinimalDetachedCMS(digest []byte, certificates [][]byte) []byte {
	messageDigestAttr := signatureTrustDERSeq(
		signatureTrustDEROID(0x2a, 0x86, 0x48, 0x86, 0xf7, 0x0d, 0x01, 0x09, 0x04),
		signatureTrustDERSet(signatureTrustDEROctetString(digest)),
	)
	signerInfo := signatureTrustDERSeq(
		signatureTrustDERInteger(1),
		signatureTrustCMSSignerIdentifier(certificates),
		signatureTrustDERAlgorithmIdentifier(signatureTrustDEROID(0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x01)),
		signatureTrustDERConstructed(0, messageDigestAttr),
		signatureTrustDERAlgorithmIdentifier(signatureTrustDEROID(0x2a, 0x86, 0x48, 0x86, 0xf7, 0x0d, 0x01, 0x01, 0x01)),
		signatureTrustDEROctetString([]byte{0}),
	)
	signedDataParts := [][]byte{
		signatureTrustDERInteger(1),
		signatureTrustDERSet(signatureTrustDERAlgorithmIdentifier(signatureTrustDEROID(0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x01))),
		signatureTrustDERSeq(signatureTrustDEROID(0x2a, 0x86, 0x48, 0x86, 0xf7, 0x0d, 0x01, 0x07, 0x01)),
	}
	if len(certificates) > 0 {
		signedDataParts = append(signedDataParts, signatureTrustDERConstructed(0, bytes.Join(certificates, nil)))
	}
	signedDataParts = append(signedDataParts, signatureTrustDERSet(signerInfo))
	return signatureTrustDERSeq(
		signatureTrustDEROID(0x2a, 0x86, 0x48, 0x86, 0xf7, 0x0d, 0x01, 0x07, 0x02),
		signatureTrustDERConstructed(0, signatureTrustDERSeq(signedDataParts...)),
	)
}

func signatureTrustCMSSignerIdentifier(certificates [][]byte) []byte {
	if len(certificates) == 0 {
		return signatureTrustDERSeq(signatureTrustDERSeq(), signatureTrustDERInteger(1))
	}
	cert, err := x509.ParseCertificate(certificates[0])
	if err != nil {
		return signatureTrustDERSeq(signatureTrustDERSeq(), signatureTrustDERInteger(1))
	}
	return signatureTrustDERSeq(cert.RawIssuer, signatureTrustDERIntegerBig(cert.SerialNumber))
}

func signatureTrustTestCertificateChain(t *testing.T) ([]byte, *x509.Certificate, []byte) {
	t.Helper()

	rootKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	leafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	notBefore := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	notAfter := time.Date(2036, 1, 1, 0, 0, 0, 0, time.UTC)
	rootTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "OxPDF Test Root"},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	rootDER, err := x509.CreateCertificate(rand.Reader, rootTemplate, rootTemplate, &rootKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	root, err := x509.ParseCertificate(rootDER)
	if err != nil {
		t.Fatal(err)
	}
	leafTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "OxPDF Test Signer"},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, root, &leafKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	return rootDER, root, leafDER
}

func signatureTrustDERAlgorithmIdentifier(oid []byte) []byte {
	return signatureTrustDERSeq(oid, []byte{0x05, 0x00})
}

func signatureTrustDERSeq(parts ...[]byte) []byte {
	return signatureTrustDERTLV(0x30, bytes.Join(parts, nil))
}

func signatureTrustDERSet(parts ...[]byte) []byte {
	return signatureTrustDERTLV(0x31, bytes.Join(parts, nil))
}

func signatureTrustDERInteger(value byte) []byte {
	return signatureTrustDERTLV(0x02, []byte{value})
}

func signatureTrustDERIntegerBig(value *big.Int) []byte {
	if value == nil {
		return signatureTrustDERInteger(0)
	}
	encoded := value.Bytes()
	if len(encoded) == 0 {
		encoded = []byte{0}
	}
	if encoded[0]&0x80 != 0 {
		encoded = append([]byte{0}, encoded...)
	}
	return signatureTrustDERTLV(0x02, encoded)
}

func signatureTrustDEROID(body ...byte) []byte {
	return signatureTrustDERTLV(0x06, body)
}

func signatureTrustDEROctetString(value []byte) []byte {
	return signatureTrustDERTLV(0x04, value)
}

func signatureTrustDERConstructed(tag byte, value []byte) []byte {
	return signatureTrustDERTLV(0xa0+tag, value)
}

func signatureTrustDERTLV(tag byte, value []byte) []byte {
	out := []byte{tag}
	if len(value) < 0x80 {
		out = append(out, byte(len(value)))
	} else if len(value) <= 0xff {
		out = append(out, 0x81, byte(len(value)))
	} else {
		out = append(out, 0x82, byte(len(value)>>8), byte(len(value)))
	}
	return append(out, value...)
}
