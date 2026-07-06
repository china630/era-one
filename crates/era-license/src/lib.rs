//! ERA XDR — верификатор лицензий на стороне агента (ADR-0010, §10).
//!
//! Defense-in-depth: агент может локально проверить подпись лицензии встроенным
//! публичным ключом вендора. Формат токена единый с Go-реализацией
//! (`services/license`): `ERA1.<base64url(claims)>.<base64url(sig)>`.
//!
//! ВНИМАНИЕ: это дополнительный слой. Первичный гейт активации модулей —
//! серверный (control-plane). Клиентский бинарник теоретически патчится.

use base64::engine::general_purpose::URL_SAFE_NO_PAD;
use base64::Engine;
use ed25519_dalek::{Signature, VerifyingKey, SIGNATURE_LENGTH};
use serde::Deserialize;

const FORMAT: &str = "ERA1";
const CLAIMS_VERSION: i32 = 1;

/// Ошибки проверки лицензии.
#[derive(Debug)]
pub enum LicenseError {
    Format,
    Base64,
    Signature,
    Json,
    Version(i32),
    Key,
}

impl std::fmt::Display for LicenseError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            LicenseError::Format => write!(f, "неверный формат токена"),
            LicenseError::Base64 => write!(f, "ошибка base64"),
            LicenseError::Signature => write!(f, "подпись недействительна"),
            LicenseError::Json => write!(f, "ошибка разбора claims"),
            LicenseError::Version(v) => write!(f, "неподдерживаемая версия claims: {v}"),
            LicenseError::Key => write!(f, "некорректный публичный ключ"),
        }
    }
}

impl std::error::Error for LicenseError {}

/// Подписанная полезная нагрузка лицензии (зеркало Go `Claims`).
#[derive(Debug, Clone, Deserialize)]
pub struct Claims {
    pub v: i32,
    pub lid: String,
    pub cust: String,
    pub tenant: String,
    pub edition: String,
    #[serde(default)]
    pub modules: Vec<String>,
    pub max_nodes: i64,
    #[serde(default)]
    pub deployment: String,
    pub iat: i64,
    pub nbf: i64,
    pub exp: i64,
    pub grace_days: i64,
}

impl Claims {
    /// Включён ли модуль в лицензии.
    pub fn has_module(&self, m: &str) -> bool {
        self.modules.iter().any(|x| x == m)
    }

    /// Действительна ли лицензия по сроку на момент `now_unix`
    /// (упрощённо; полная оценка статусов — на control-plane).
    pub fn valid_at(&self, now_unix: i64) -> bool {
        now_unix >= self.nbf && now_unix <= self.exp + self.grace_days * 86_400
    }
}

/// Проверяет подпись токена публичным ключом (32 байта) и возвращает claims.
pub fn verify(token: &str, public_key: &[u8]) -> Result<Claims, LicenseError> {
    let key_arr: [u8; 32] = public_key.try_into().map_err(|_| LicenseError::Key)?;
    let vk = VerifyingKey::from_bytes(&key_arr).map_err(|_| LicenseError::Key)?;

    let parts: Vec<&str> = token.split('.').collect();
    if parts.len() != 3 || parts[0] != FORMAT {
        return Err(LicenseError::Format);
    }

    let payload = URL_SAFE_NO_PAD
        .decode(parts[1])
        .map_err(|_| LicenseError::Base64)?;
    let sig_bytes = URL_SAFE_NO_PAD
        .decode(parts[2])
        .map_err(|_| LicenseError::Base64)?;

    let sig_arr: [u8; SIGNATURE_LENGTH] =
        sig_bytes.as_slice().try_into().map_err(|_| LicenseError::Signature)?;
    let signature = Signature::from_bytes(&sig_arr);

    vk.verify_strict(&payload, &signature)
        .map_err(|_| LicenseError::Signature)?;

    let claims: Claims = serde_json::from_slice(&payload).map_err(|_| LicenseError::Json)?;
    if claims.v != CLAIMS_VERSION {
        return Err(LicenseError::Version(claims.v));
    }
    Ok(claims)
}
