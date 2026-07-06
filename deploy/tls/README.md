# ERA XDR — dev TLS/mTLS (GA-1)

Генерация локальных сертификатов (только dev/pilot, не production PKI):

```powershell
# из корня репо
powershell -ExecutionPolicy Bypass -File scripts/gen-dev-tls.ps1
```

Файлы:

| Файл | Назначение |
|---|---|
| `ca.crt` / `ca.key` | CA |
| `server.crt` / `server.key` | ingest-gateway |
| `agent.crt` / `agent.key` | era-agent client cert |

Gateway:

```powershell
$env:ERA_TLS_CERT="deploy/tls/server.crt"
$env:ERA_TLS_KEY="deploy/tls/server.key"
$env:ERA_TLS_CA="deploy/tls/ca.crt"
```

Agent:

```powershell
$env:ERA_TLS_CA="deploy/tls/ca.crt"
$env:ERA_TLS_CLIENT_CERT="deploy/tls/agent.crt"
$env:ERA_TLS_CLIENT_KEY="deploy/tls/agent.key"
$env:ERA_GATEWAY_ADDR="https://127.0.0.1:50051"
```

Production: офлайн PKI заказчика, см. GA-2 S6-9.
