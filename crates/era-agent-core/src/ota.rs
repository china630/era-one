//! OTA-скелет: подписанные артефакты из локального зеркала (ADR-0019 §4).

use anyhow::{bail, Context, Result};
use base64::engine::general_purpose::URL_SAFE_NO_PAD;
use base64::Engine;
use ed25519_dalek::{Signature, Verifier, VerifyingKey, SIGNATURE_LENGTH};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::fs;
use std::path::{Path, PathBuf};

pub const OTA_FORMAT: &str = "ERAAOT1";

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct OtaClaims {
    pub v: i32,
    pub name: String,
    pub version: String,
    pub sha256: String,
    pub iat: i64,
}

/// Подписанный токен манифеста (wire: ERAAOT1.payload.sig).
pub fn verify_manifest_token(token: &str, public_key: &[u8; 32]) -> Result<OtaClaims> {
    let parts: Vec<&str> = token.split('.').collect();
    if parts.len() != 3 || parts[0] != OTA_FORMAT {
        bail!("ota: bad format");
    }
    let payload = URL_SAFE_NO_PAD
        .decode(parts[1])
        .context("ota: decode payload")?;
    let sig_bytes = URL_SAFE_NO_PAD
        .decode(parts[2])
        .context("ota: decode sig")?;
    let sig_arr: [u8; SIGNATURE_LENGTH] = sig_bytes
        .as_slice()
        .try_into()
        .map_err(|_| anyhow::anyhow!("ota: bad sig len"))?;
    let vk = VerifyingKey::from_bytes(public_key).context("ota: bad key")?;
    vk.verify(&payload, &Signature::from_bytes(&sig_arr))
        .map_err(|_| anyhow::anyhow!("ota: invalid signature"))?;
    let claims: OtaClaims = serde_json::from_slice(&payload).context("ota: json")?;
    Ok(claims)
}

pub fn hash_file(path: &Path) -> Result<String> {
    let data = fs::read(path).with_context(|| format!("read {}", path.display()))?;
    Ok(hex::encode(Sha256::digest(data)))
}

/// Content-addressed кэш: копирует артефакт в `cache_dir/hash`.
pub fn install_to_cache(artifact: &Path, cache_dir: &Path, expected_sha256: &str) -> Result<PathBuf> {
    let got = hash_file(artifact)?;
    if got != expected_sha256 {
        bail!("ota: hash mismatch want={expected_sha256} got={got}");
    }
    fs::create_dir_all(cache_dir).context("ota: mkdir cache")?;
    let dest = cache_dir.join(&got);
    if !dest.exists() {
        fs::copy(artifact, &dest).context("ota: cache copy")?;
    }
    Ok(dest)
}

/// Читает манифест `.ota.token` из зеркала и верифицирует.
pub fn pull_from_mirror(mirror_dir: &Path, public_key: &[u8; 32]) -> Result<OtaClaims> {
    let token_path = mirror_dir.join("plugin.ota.token");
    let token = fs::read_to_string(&token_path)
        .with_context(|| format!("read {}", token_path.display()))?;
    verify_manifest_token(token.trim(), public_key)
}

#[cfg(test)]
mod tests {
    use super::*;
    use ed25519_dalek::{Signer, SigningKey};
    use std::io::Write;
    use tempfile::tempdir;

    fn sign_claims(claims: &OtaClaims, sk: &SigningKey) -> String {
        let payload = serde_json::to_vec(claims).unwrap();
        let sig = sk.sign(&payload);
        format!(
            "{}.{}.{}",
            OTA_FORMAT,
            URL_SAFE_NO_PAD.encode(&payload),
            URL_SAFE_NO_PAD.encode(sig.to_bytes())
        )
    }

    #[test]
    fn verify_rejects_tampered() {
        let sk = SigningKey::from_bytes(&[7u8; 32]);
        let pk = sk.verifying_key().to_bytes();
        let claims = OtaClaims {
            v: 1,
            name: "inventory".into(),
            version: "0.1.0".into(),
            sha256: "abc".into(),
            iat: 1,
        };
        let token = sign_claims(&claims, &sk);
        let mut bad = token.clone();
        bad.push('x');
        assert!(verify_manifest_token(&bad, &pk).is_err());
        assert!(verify_manifest_token(&token, &pk).is_ok());
    }

    #[test]
    fn install_checks_hash() {
        let dir = tempdir().unwrap();
        let artifact = dir.path().join("bin");
        let mut f = fs::File::create(&artifact).unwrap();
        writeln!(f, "plugin").unwrap();
        let hash = hash_file(&artifact).unwrap();
        let cache = dir.path().join("cache");
        install_to_cache(&artifact, &cache, &hash).unwrap();
        assert!(cache.join(&hash).exists());
    }
}
