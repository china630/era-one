# ERA PAM — Privileged Access Management (Stage 8)

Спецификация vault, checkout и session-proxy.

**Связано:** [ADR-0013](adr/0013-era-pam-edition.md) · лицензия `pam` ·
[PAM-RDP-Security-Review-Checklist.md](PAM-RDP-Security-Review-Checklist.md).

**Статус кода:** vault/checkout/SSH ✅ · RDP TCP relay MVP ✅ · HSM prod / graphical RDP ⏸ external.

## Компоненты

| Компонент | Путь | Порт |
|---|---|---|
| PAM API | `services/pam` | `:8130` |
| Session recording | `services/dlp` (переиспользуется) | `:8095` |
| Custody audit | `services/platform/custody` | hash-chain |
| SSH proxy | `internal/proxy.SSHProxy` | local TCP + command log |
| RDP proxy | `internal/proxy.RDPProxy` | local TCP binary relay → :3389 |

## Криптоинварианты

- AES-256-GCM at-rest; мастер-ключ через KMS-абстракцию (`software-sealed-dev` в dev)
- Seal/unseal Shamir (2-of-3); vault стартует **sealed**
- Zero-knowledge UI: списки секретов без plaintext
- Каждый доступ → custody hash-chain

## API

| Метод | Путь | Описание |
|---|---|---|
| GET | `/api/v1/vault/status` | sealed/unsealed |
| POST | `/api/v1/vault/unseal` | `{shares: [hex,...]}` |
| POST | `/api/v1/vault/seal` | admin only |
| GET/POST | `/api/v1/secrets` | static secrets (meta only in GET) |
| POST | `/api/v1/checkout` | запрос креденшела |
| POST | `/api/v1/checkout/{id}/approve` | approval |
| POST | `/api/v1/checkout/{id}/reveal` | one-shot password |
| POST | `/api/v1/proxy/ssh/start` | session + TCP proxy + credential inject meta |
| POST | `/api/v1/proxy/ssh/command` | command log + detection |
| POST | `/api/v1/proxy/rdp/start` | session + TCP proxy (`mode=tcp_relay`, `proxy_addr`) |
| GET | `/api/v1/custody/head` | chain head |

## Compose

```bash
docker compose -f deploy/docker-compose.prod.yml --profile pam up -d pam dlp
```

Kafka topic: `xdr.privileged`

## Тесты

- `go test ./services/pam/...` — shamir golden, vault, SSH/RDP proxy relay, custody
- `go test ./services/platform/custody/...`

## Гейты

| Гейт | Статус |
|---|---|
| SSH TCP proxy + command log | ✅ |
| RDP TCP proxy MVP (binary relay) | ✅ |
| Credential inject (server-side broker) + metadata recording | Phase 2 code ✅ — **code-ready**; Guacamole video / pen-test GA after security-review ⏸ |
| Крипто-аудит vault/HSM prod | [gate: external] — **code-ready** KMS abstraction; GA after HSM review ⏸ |
