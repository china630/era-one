# ERA PAM — Privileged Access Management (Stage 8)

Спецификация vault, checkout и session-proxy.

**Связано:** [ADR-0013](adr/0013-era-pam-edition.md) · лицензия `pam`.

## Компоненты

| Компонент | Путь | Порт |
|---|---|---|
| PAM API | `services/pam` | `:8130` |
| Session recording | `services/dlp` (переиспользуется) | `:8095` |
| Custody audit | `services/platform/custody` | hash-chain |

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
| POST | `/api/v1/proxy/ssh/start` | session + credential inject |
| POST | `/api/v1/proxy/ssh/command` | command log + detection |
| GET | `/api/v1/custody/head` | chain head |

## Compose

```bash
docker compose -f deploy/docker-compose.prod.yml --profile pam up -d pam dlp
```

Kafka topic: `xdr.privileged`

## Тесты

- `go test ./services/pam/...` — shamir golden, vault at-rest, no-secret-leak, custody
- `go test ./services/platform/custody/...`

## Гейты

| Гейт | Статус |
|---|---|
| Крипто-аудит vault/HSM | [gate: external] |
| Security-review RDP-прокси | [gate: external] |

Код этапа: SSH command recording + simulated inject; боевой RDP за гейтом.
