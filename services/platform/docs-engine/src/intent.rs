//! HMAC signed EditLink intent (AC-O8) — matches `ui/mail` DocumentsClient.

use std::time::{SystemTime, UNIX_EPOCH};

use hmac::{Hmac, Mac};
use sha2::Sha256;

type HmacSha256 = Hmac<Sha256>;

/// Secret for Comms→Documents deep-link intents.
/// Prefer `ERA_DOCS_INTENT_SECRET`; fall back to JWT secret (same as mail client).
pub fn intent_secret_from_env() -> Vec<u8> {
    if let Ok(s) = std::env::var("ERA_DOCS_INTENT_SECRET") {
        if !s.is_empty() {
            return s.into_bytes();
        }
    }
    crate::auth::jwt_secret_from_env()
}

pub fn sign_intent(secret: &[u8], object_id: &str, exp: i64) -> String {
    let mut mac =
        HmacSha256::new_from_slice(secret).expect("HMAC accepts any key length");
    mac.update(object_id.as_bytes());
    mac.update(b"|");
    mac.update(exp.to_string().as_bytes());
    hex::encode(mac.finalize().into_bytes())
}

/// Verify `intent_exp` + `intent_sig` query params (same algorithm as mail.VerifyIntent).
pub fn verify_intent(secret: &[u8], object_id: &str, exp_str: &str, sig: &str) -> bool {
    let Ok(exp) = exp_str.parse::<i64>() else {
        return false;
    };
    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_secs() as i64)
        .unwrap_or(0);
    if now > exp {
        return false;
    }
    let want = sign_intent(secret, object_id, exp);
    // Constant-time compare via hmac::Mac would need raw digests; hex strings are fixed-size.
    if want.len() != sig.len() {
        return false;
    }
    let mut diff = 0u8;
    for (a, b) in want.bytes().zip(sig.bytes()) {
        diff |= a ^ b;
    }
    diff == 0
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn verify_roundtrip() {
        let secret = b"intent-secret";
        let exp = 4102444800i64;
        let sig = sign_intent(secret, "obj-123", exp);
        assert!(verify_intent(secret, "obj-123", &exp.to_string(), &sig));
        assert!(!verify_intent(secret, "other", &exp.to_string(), &sig));
        assert!(!verify_intent(secret, "obj-123", "1", &sig));
    }
}
