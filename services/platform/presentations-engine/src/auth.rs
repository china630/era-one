//! JWT principal extraction for Office engines (O-AUTH).

use axum::http::{header, HeaderMap, StatusCode};
use jsonwebtoken::{decode, Algorithm, DecodingKey, Validation};
use serde::{Deserialize, Serialize};

const DEV_JWT_SECRET: &str = "dev-only-change-in-prod";

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Claims {
    pub sub: String,
    pub tenant_id: String,
    #[serde(default)]
    pub email: String,
    /// Unix expiry — required; validated via `validate_exp`.
    pub exp: i64,
}

#[derive(Debug, Clone)]
pub struct Principal {
    pub tenant_id: String,
    pub user_id: String,
}

fn production_like() -> bool {
    env_truthy("ERA_LICENSE_STRICT")
        || env_truthy("ERA_PRODUCTION")
        || env_truthy("ERA_ENV_PRODUCTION")
        || std::env::var("ERA_ENV")
            .map(|v| v.eq_ignore_ascii_case("production"))
            .unwrap_or(false)
}

fn env_truthy(k: &str) -> bool {
    matches!(
        std::env::var(k).ok().as_deref(),
        Some("1") | Some("true") | Some("yes")
    )
}

/// JWT HMAC secret from env.
///
/// In production-like mode, returns empty if unset or lab default → 401.
pub fn jwt_secret_from_env() -> Vec<u8> {
    let raw = std::env::var("ERA_IDENTITY_JWT_SECRET").unwrap_or_default();
    if production_like() {
        if raw.is_empty() || raw == DEV_JWT_SECRET {
            eprintln!(
                "era-presentations-engine: ERA_IDENTITY_JWT_SECRET missing or default in production — JWT auth disabled (401)"
            );
            return Vec::new();
        }
        return raw.into_bytes();
    }
    if raw.is_empty() {
        DEV_JWT_SECRET.as_bytes().to_vec()
    } else {
        raw.into_bytes()
    }
}

/// Extract Bearer JWT principal. Returns 401 if missing/invalid/expired.
pub fn principal_from_headers(headers: &HeaderMap, secret: &[u8]) -> Result<Principal, StatusCode> {
    let auth = headers
        .get(header::AUTHORIZATION)
        .and_then(|v| v.to_str().ok())
        .ok_or(StatusCode::UNAUTHORIZED)?;
    let token = auth
        .strip_prefix("Bearer ")
        .or_else(|| auth.strip_prefix("bearer "))
        .map(str::trim)
        .filter(|s| !s.is_empty())
        .ok_or(StatusCode::UNAUTHORIZED)?;
    if secret.is_empty() {
        return Err(StatusCode::UNAUTHORIZED);
    }
    let mut validation = Validation::new(Algorithm::HS256);
    validation.validate_exp = true;
    validation.set_required_spec_claims(&["exp"]);
    let data = decode::<Claims>(
        token,
        &DecodingKey::from_secret(secret),
        &validation,
    )
    .map_err(|_| StatusCode::UNAUTHORIZED)?;
    if data.claims.tenant_id.is_empty() || data.claims.sub.is_empty() {
        return Err(StatusCode::UNAUTHORIZED);
    }
    Ok(Principal {
        tenant_id: data.claims.tenant_id,
        user_id: data.claims.sub,
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use jsonwebtoken::{encode, EncodingKey, Header};

    fn bearer(tok: &str) -> HeaderMap {
        let mut h = HeaderMap::new();
        h.insert(
            header::AUTHORIZATION,
            format!("Bearer {tok}").parse().unwrap(),
        );
        h
    }

    fn clear_prod_env() {
        std::env::remove_var("ERA_PRODUCTION");
        std::env::remove_var("ERA_LICENSE_STRICT");
        std::env::remove_var("ERA_ENV_PRODUCTION");
        std::env::remove_var("ERA_ENV");
        std::env::remove_var("ERA_IDENTITY_JWT_SECRET");
    }

    #[test]
    fn rejects_missing_auth() {
        let h = HeaderMap::new();
        assert!(principal_from_headers(&h, b"secret").is_err());
    }

    #[test]
    fn accepts_valid_jwt() {
        let claims = Claims {
            sub: "u1".into(),
            tenant_id: "t1".into(),
            email: "u@x".into(),
            exp: 4102444800,
        };
        let tok = encode(
            &Header::default(),
            &claims,
            &EncodingKey::from_secret(b"secret"),
        )
        .unwrap();
        let p = principal_from_headers(&bearer(&tok), b"secret").unwrap();
        assert_eq!(p.tenant_id, "t1");
        assert_eq!(p.user_id, "u1");
    }

    #[test]
    fn rejects_expired_jwt() {
        let claims = Claims {
            sub: "u1".into(),
            tenant_id: "t1".into(),
            email: "u@x".into(),
            exp: 1_600_000_000,
        };
        let tok = encode(
            &Header::default(),
            &claims,
            &EncodingKey::from_secret(b"secret"),
        )
        .unwrap();
        let err = principal_from_headers(&bearer(&tok), b"secret").unwrap_err();
        assert_eq!(err, StatusCode::UNAUTHORIZED);
    }

    #[test]
    fn jwt_secret_empty_in_production_when_default() {
        clear_prod_env();
        std::env::set_var("ERA_PRODUCTION", "1");
        std::env::set_var("ERA_IDENTITY_JWT_SECRET", DEV_JWT_SECRET);
        assert!(jwt_secret_from_env().is_empty());
        clear_prod_env();
    }
}
