use oxpdf::{PublicKeyEncryptionMethod, PublicKeyEncryptionOptions, open};
use rcgen::{CertificateParams, DnType, KeyPair, PKCS_RSA_SHA256};
use rsa::{
    RsaPrivateKey,
    pkcs8::{EncodePrivateKey, LineEnding},
    rand_core::OsRng,
};

mod support;

#[test]
fn public_key_encryption_and_decryption_reopen_through_binas() {
    let (certificate, private_key) = recipient_credentials();
    let document =
        open(&support::fixture_pdf(&["RECIPIENT ONLY"], None)).expect("clear fixture opens");

    let encrypted = document
        .encrypt_public_key(PublicKeyEncryptionOptions {
            method: PublicKeyEncryptionMethod::AesV3,
            recipient_certificates_der: vec![certificate.clone()],
            permissions: -1028,
        })
        .expect("public-key encryption succeeds");
    assert!(encrypted.verification.passed);
    assert!(encrypted.verification.encrypted_reparsed);

    let reopened_encrypted = open(&encrypted.bytes).expect("encrypted output reopens");
    assert_eq!(
        reopened_encrypted
            .encryption_metadata()
            .expect("encryption metadata reads")
            .filter
            .as_deref(),
        Some("Adobe.PubSec")
    );

    let decrypted = reopened_encrypted
        .decrypt_public_key(&certificate, &private_key)
        .expect("authorized recipient decrypts");
    assert!(decrypted.verification.passed);
    assert!(decrypted.verification.reparsed);
    assert_eq!(decrypted.permissions, -1028);

    let reopened_plain = open(&decrypted.bytes).expect("decrypted output reopens");
    assert_eq!(
        reopened_plain
            .extract_text()
            .expect("decrypted text extracts")
            .spans[0]
            .text,
        "RECIPIENT ONLY"
    );
}

fn recipient_credentials() -> (Vec<u8>, Vec<u8>) {
    let private = RsaPrivateKey::new(&mut OsRng, 2048).expect("test RSA key generates");
    let private_key = private
        .to_pkcs8_der()
        .expect("test key encodes")
        .as_bytes()
        .to_vec();
    let pem = private
        .to_pkcs8_pem(LineEnding::LF)
        .expect("test key PEM encodes");
    let key = KeyPair::from_pem_and_sign_algo(&pem, &PKCS_RSA_SHA256)
        .expect("test key becomes certificate key");
    let mut params = CertificateParams::default();
    params
        .distinguished_name
        .push(DnType::CommonName, "OxPDF test recipient");
    let certificate = params.self_signed(&key).expect("test certificate signs");
    (certificate.der().to_vec(), private_key)
}
