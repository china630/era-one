//! TLS helpers for SMTP/IMAP STARTTLS (R2-C).

use anyhow::{anyhow, Result};
use rustls::pki_types::{CertificateDer, PrivateKeyDer};
use rustls::ServerConfig;
use std::fs;
use std::sync::Arc;
use tokio::io::{AsyncRead, AsyncWrite};
use tokio::net::TcpStream;
use tokio_rustls::{TlsAcceptor, server::TlsStream};

/// Returns TLS acceptor when ERA_MAIL_TLS_CERT and ERA_MAIL_TLS_KEY are set.
pub fn server_acceptor() -> Result<Option<TlsAcceptor>> {
    let _ = rustls::crypto::ring::default_provider().install_default();
    let cert_path = match std::env::var("ERA_MAIL_TLS_CERT") {
        Ok(p) if !p.is_empty() => p,
        _ => return Ok(None),
    };
    let key_path = std::env::var("ERA_MAIL_TLS_KEY").unwrap_or_default();
    if key_path.is_empty() {
        return Ok(None);
    }
    let cert_pem = fs::read(&cert_path)?;
    let key_pem = fs::read(&key_path)?;
    let certs: Vec<CertificateDer<'static>> =
        rustls_pemfile::certs(&mut &*cert_pem).collect::<Result<Vec<_>, _>>()?;
    let key = rustls_pemfile::private_key(&mut &*key_pem)?
        .ok_or_else(|| anyhow!("no private key in {}", key_path))?;
    let cfg = ServerConfig::builder()
        .with_no_client_auth()
        .with_single_cert(certs, PrivateKeyDer::from(key))?;
    Ok(Some(TlsAcceptor::from(Arc::new(cfg))))
}

pub async fn accept_tls(
    acceptor: &TlsAcceptor,
    stream: TcpStream,
) -> Result<TlsStream<TcpStream>> {
    acceptor.accept(stream).await.map_err(|e| anyhow!(e))
}

pub type MaybeTlsStream = tokio_util::either::Either<TcpStream, TlsStream<TcpStream>>;

pub async fn upgrade_if_tls(
    acceptor: &Option<TlsAcceptor>,
    stream: TcpStream,
    use_tls: bool,
) -> Result<MaybeTlsStream> {
    if use_tls {
        if let Some(acc) = acceptor {
            let tls = accept_tls(acc, stream).await?;
            return Ok(tokio_util::either::Either::Right(tls));
        }
    }
    Ok(tokio_util::either::Either::Left(stream))
}

pub trait AsyncStream: AsyncRead + AsyncWrite + Unpin + Send {}
impl<T: AsyncRead + AsyncWrite + Unpin + Send> AsyncStream for T {}
