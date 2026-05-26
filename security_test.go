package oxpdf

import "testing"

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
