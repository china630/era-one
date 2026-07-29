//! Golden: SMTP deliver → store → IMAP fetch baseline (AC-C1).

use era_mail_core::{MailBackend, MailStore};

#[test]
fn deliver_and_list_golden() {
    let store = MailStore::new();
    let raw = "From: alice@mail.gov.az\r\nTo: bob@mail.gov.az\r\nSubject: golden\r\n\r\nHello ERA";
    let id = store.deliver("bob@mail.gov.az", raw).unwrap();

    let msgs = store.list_mailbox("bob@mail.gov.az").unwrap();
    assert_eq!(msgs.len(), 1);
    assert_eq!(msgs[0].id, id);

    let body = String::from_utf8_lossy(&msgs[0].raw);
    assert!(body.contains("Hello ERA"));
    assert!(body.contains("Subject: golden"));
}

#[test]
fn invalid_mailbox_rejected() {
    let store = MailStore::new();
    assert!(store.deliver("not-an-email", "body").is_err());
}
