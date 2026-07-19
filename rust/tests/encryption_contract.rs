use oxpdf::{StandardEncryptionOptions, StandardEncryptionRevision, open};

mod support;

#[test]
fn standard_password_encryption_and_decryption_reopen_through_binas() {
    let clear = support::fixture_pdf(&["TOP SECRET"], None);
    let document = open(&clear).expect("clear fixture opens");

    let encrypted = document
        .encrypt_standard(StandardEncryptionOptions {
            revision: StandardEncryptionRevision::R4AesV2,
            user_password: "user-password".into(),
            owner_password: "owner-password".into(),
            permissions: -4,
        })
        .expect("standard encryption succeeds");
    assert!(encrypted.verification.passed);
    assert!(encrypted.verification.encrypted_reparsed);

    let reopened_encrypted = open(&encrypted.bytes).expect("encrypted output reopens");
    assert!(
        reopened_encrypted
            .encryption_metadata()
            .expect("encryption metadata reads")
            .encrypted
    );

    let decrypted = reopened_encrypted
        .decrypt_to_plain("user-password")
        .expect("standard decryption succeeds");
    assert!(decrypted.verification.passed);
    assert!(decrypted.verification.reparsed);

    let reopened_plain = open(&decrypted.bytes).expect("decrypted output reopens");
    assert!(
        !reopened_plain
            .encryption_metadata()
            .expect("decrypted metadata reads")
            .encrypted
    );
    assert_eq!(
        reopened_plain
            .extract_text()
            .expect("decrypted text extracts")
            .spans[0]
            .text,
        "TOP SECRET"
    );
}
