//! Конфигурация mail-core (env + defaults).

use std::env;

/// Параметры SMTP/IMAP listeners.
#[derive(Clone, Debug)]
pub struct Config {
    pub smtp_host: String,
    pub smtp_port: u16,
    pub imap_host: String,
    pub imap_port: u16,
    pub admin_host: String,
    pub admin_port: u16,
}

impl Default for Config {
    fn default() -> Self {
        Self {
            smtp_host: "0.0.0.0".into(),
            smtp_port: 2525,
            imap_host: "0.0.0.0".into(),
            imap_port: 1143,
            admin_host: "127.0.0.1".into(),
            admin_port: 8151,
        }
    }
}

impl Config {
    /// Читает конфиг из окружения.
    pub fn from_env() -> Self {
        let mut cfg = Self::default();
        if let Ok(v) = env::var("ERA_MAIL_SMTP_PORT") {
            if let Ok(p) = v.parse() {
                cfg.smtp_port = p;
            }
        }
        if let Ok(v) = env::var("ERA_MAIL_IMAP_PORT") {
            if let Ok(p) = v.parse() {
                cfg.imap_port = p;
            }
        }
        if let Ok(v) = env::var("ERA_MAIL_ADMIN_PORT") {
            if let Ok(p) = v.parse() {
                cfg.admin_port = p;
            }
        }
        if let Ok(v) = env::var("ERA_MAIL_ADMIN_HOST") {
            if !v.trim().is_empty() {
                cfg.admin_host = v;
            }
        }
        cfg
    }

    pub fn admin_addr(&self) -> String {
        format!("{}:{}", self.admin_host, self.admin_port)
    }

    pub fn smtp_addr(&self) -> String {
        format!("{}:{}", self.smtp_host, self.smtp_port)
    }

    pub fn imap_addr(&self) -> String {
        format!("{}:{}", self.imap_host, self.imap_port)
    }
}
