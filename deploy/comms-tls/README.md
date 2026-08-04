# Comms lab TLS

Run `pwsh deploy/comms-tls/gen-dev-certs.ps1` before:

```powershell
docker compose -f deploy/docker-compose.comms.yml -f deploy/docker-compose.comms.prod.yml up -d --wait
```

Sets `ERA_MAIL_TLS=1` (Autodiscover SSL on) and mounts certs into mail-core for STARTTLS/IMAP TLS.
