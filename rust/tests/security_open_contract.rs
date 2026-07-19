use oxpdf::{
    PublicKeyEncryptionMethod, PublicKeyEncryptionOptions, StandardEncryptionOptions,
    StandardEncryptionRevision, open, open_with_password, open_with_public_key,
};
use rcgen::{CertificateParams, DnType, KeyPair, PKCS_RSA_SHA256};
use rsa::{
    RsaPrivateKey,
    pkcs8::{EncodePrivateKey, LineEnding},
    rand_core::OsRng,
};

mod support;

#[test]
fn opens_standard_encrypted_input_and_refuses_a_wrong_password() {
    let clear = support::fixture_pdf(&["TOP SECRET"], None);
    let encrypted = open(&clear)
        .expect("clear fixture opens")
        .encrypt_standard(StandardEncryptionOptions {
            revision: StandardEncryptionRevision::R4AesV2,
            user_password: "user-password".into(),
            owner_password: "owner-password".into(),
            permissions: -4,
        })
        .expect("standard encryption succeeds");

    let opened = open_with_password(&encrypted.bytes, "user-password")
        .expect("correct password opens encrypted input");
    assert_eq!(
        opened.extract_text().expect("opened text extracts").spans[0].text,
        "TOP SECRET"
    );
    assert!(open_with_password(&encrypted.bytes, "wrong-password").is_err());
}

#[test]
fn opens_public_key_input_and_refuses_an_unrelated_private_key() {
    let (certificate, private_key) = recipient_credentials();
    let encrypted = open(&support::fixture_pdf(&["RECIPIENT ONLY"], None))
        .expect("clear fixture opens")
        .encrypt_public_key(PublicKeyEncryptionOptions {
            method: PublicKeyEncryptionMethod::AesV3,
            recipient_certificates_der: vec![certificate.clone()],
            permissions: -1028,
        })
        .expect("public-key encryption succeeds");

    let opened = open_with_public_key(&encrypted.bytes, &certificate, &private_key)
        .expect("authorized recipient opens encrypted input");
    assert_eq!(
        opened.extract_text().expect("opened text extracts").spans[0].text,
        "RECIPIENT ONLY"
    );

    let (_, unrelated_private_key) = recipient_credentials();
    assert!(open_with_public_key(&encrypted.bytes, &certificate, &unrelated_private_key).is_err());
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
