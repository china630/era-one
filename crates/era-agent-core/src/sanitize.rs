//! PII-редакция на агенте ДО отправки (ADR-0009, S1-7).

use crate::envelope;
use crate::Envelope;
use hmac::{Hmac, Mac};
use sha2::Sha256;

type HmacSha256 = Hmac<Sha256>;

/// HMAC-SHA256(value, tenant_pseudonym_key) — детерминированная псевдонимизация.
pub fn pseudonymize(tenant_id: &str, value: &str, key: &str) -> String {
    let mut mac =
        HmacSha256::new_from_slice(key.as_bytes()).expect("HMAC key length");
    mac.update(tenant_id.as_bytes());
    mac.update(b"|");
    mac.update(value.as_bytes());
    format!("pseudo:{}", hex::encode(mac.finalize().into_bytes()))
}

/// Маскирует секреты и идентификаторы пользователя в command line.
pub fn mask_secrets(input: &str) -> String {
    let tokens: Vec<&str> = input.split_whitespace().collect();
    let mut out = Vec::with_capacity(tokens.len());
    let mut i = 0;
    while i < tokens.len() {
        let token = tokens[i];
        let lower = token.to_ascii_lowercase();
        if lower.contains("password=")
            || lower.contains("token=")
            || lower.contains("secret=")
            || lower.contains("apikey=")
        {
            if let Some(eq) = token.find('=') {
                out.push(format!("{}***REDACTED***", &token[..=eq]));
            } else {
                out.push("***REDACTED***".into());
            }
        } else if lower.starts_with("--user=") {
            out.push("--user=***REDACTED***".into());
        } else if lower == "--user" {
            out.push("--user".into());
            out.push("***REDACTED***".into());
            if i + 1 < tokens.len() {
                i += 1;
            }
        } else {
            out.push(token.to_string());
        }
        i += 1;
    }
    out.join(" ")
}

/// Применяет редакцию и выставляет pii_sanitized.
pub fn sanitize(mut env: Envelope, pseudonym_key: &str) -> Envelope {
    let tenant = env
        .source
        .as_ref()
        .map(|s| s.tenant_id.clone())
        .unwrap_or_default();

    if let Some(envelope::Payload::Process(p)) = env.payload.as_mut() {
        p.user = pseudonymize(&tenant, &p.user, pseudonym_key);
        p.command_line = mask_secrets(&p.command_line);
    }
    if let Some(envelope::Payload::Auth(a)) = env.payload.as_mut() {
        a.user = pseudonymize(&tenant, &a.user, pseudonym_key);
    }

    env.pii_sanitized = true;
    env
}

/// JSON-представление полей Process для golden-тестов.
pub fn process_fields_json(env: &Envelope) -> Option<String> {
    let envelope::Payload::Process(p) = env.payload.as_ref()? else {
        return None;
    };
    Some(format!(
        r#"{{"user":"{}","command_line":"{}","pii_sanitized":{}}}"#,
        p.user,
        p.command_line,
        env.pii_sanitized
    ))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::sample;

    #[test]
    fn pseudonymize_is_deterministic_within_tenant() {
        let key = "test-key";
        let a = pseudonymize("t1", "alice", key);
        let b = pseudonymize("t1", "alice", key);
        let c = pseudonymize("t2", "alice", key);
        assert_eq!(a, b);
        assert_ne!(a, c);
    }

    #[test]
    fn mask_secrets_redacts_values() {
        let masked = mask_secrets("login --password=hunter2 --user bob");
        assert!(masked.contains("password=***REDACTED***"));
        assert!(!masked.contains("hunter2"));
        assert!(!masked.contains("bob"));
    }

    #[test]
    fn sanitize_sets_flag() {
        let clean = sanitize(sample::process_envelope("cmd.exe"), "test-key");
        assert!(clean.pii_sanitized);
        let envelope::Payload::Process(p) = clean.payload.as_ref().unwrap() else {
            panic!("expected process");
        };
        assert!(!p.user.contains("alice"));
        assert!(!p.command_line.contains("SECRET123"));
        assert!(!p.command_line.contains("alice"));
    }
}
