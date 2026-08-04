# ERA Comms — Lab IMAP Notes (L-1)

Service **`dovecot-lab`** in `deploy/docker-compose.comms.dev.yml`.

- Image build: `deploy/dockerfiles/Dockerfile.lab-imap` (ERA RFC3501 lab server)
- Host port: `1144` → container `143`
- Compose DNS: `dovecot-lab:143`
- Seed user: `lab1@mail.gov.az` (LOGIN accepts any password)
- Conf notes / optional Dovecot swap: `deploy/comms-lab-imap/`

Env for Migration live IMAP:

```
ERA_MIG_SOURCE_IMAP_HOST=dovecot-lab
ERA_MAIL_API_URL=http://era-mail-api:8150
```

Connect lab: register mailbox `address=imap://dovecot-lab:143` (or `127.0.0.1:1144` from host).

### Connect vault lab limit (Deepen D4 / AC-C6)

- Real IMAP FETCH when mailbox `address` is set **and** `password_ref` resolves (`vault://name` → `ERA_CONNECT_SECRET_<NAME>`).
- Without address: honest `mode=stub`, `items_ok=0` (no silent inflate).
- Lab vault = **env mapping only** (`ERA_CONNECT_SECRET_*`). Not TPM/HSM/customer keystore — field RT-10 / GAP-P1-02 remains open.
- Script: `scripts/run-comms-connect-lab.ps1` (`-UseCompose` for dovecot-lab). Unit: `go test -C services/comms/mail-connect ./internal/sync/`.

Smoke: `scripts/run-comms-lab-imap-smoke.ps1`
