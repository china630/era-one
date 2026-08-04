# ERA Mail Server

Rust SMTP/IMAP core + Go HTTP API (Autodiscover, policy, ClickHouse audit).

Refs: ADR-0027, CM1-2..CM1-5.

## Компоненты

| Компонент | Порт (dev) | Описание |
|-----------|------------|----------|
| `era-mail-core` | 2525 SMTP, 1143 IMAP | In-memory mail store (MVP) |
| `mail-api` | 8150 HTTP | Autodiscover, policy API, status |

## Env

| Переменная | Default | Описание |
|------------|---------|----------|
| `ERA_MAIL_DEV` | — | `1` — bypass license check (dev) |
| `ERA_MAIL_HTTP_ADDR` | `:8150` | HTTP listen |
| `ERA_MAIL_SMTP_PORT` | `2525` | SMTP |
| `ERA_MAIL_IMAP_PORT` | `1143` | IMAP |
| `ERA_CH_ADDR` | — | ClickHouse native (audit) |

## Тесты

```powershell
go test ./services/comms/mail/...
cargo test -p era-mail-core
```

Golden Autodiscover update: `$env:ERA_UPDATE_GOLDEN = "1"; go test ./internal/autodiscover/...`
