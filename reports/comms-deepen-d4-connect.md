# Deepen D4 — Mail Connect (lab) — 2026-08-04

## Code honesty

`services/comms/mail-connect/internal/sync/imap_sync.go`:

- Address set + `vault://` / env secret resolves → **live** IMAP FETCH (`mode=live`, real `items_ok`).
- No address → `mode=stub`, `items_ok=0`, explicit error (no silent ItemsOK=12).

Vault lab limit: `vault://name` → `ERA_CONNECT_SECRET_<NAME>` only. Documented in [`docs/Comms-Lab-IMAP-Notes.md`](../docs/Comms-Lab-IMAP-Notes.md). TPM/keystore = field (GAP-P1-02).

## Tests

```
go test -C services/comms/mail-connect ./internal/sync/ ./internal/autodiscover/ ./internal/api/ -count=1
```

Expect: `TestStartSync_RealIMAP`, `TestStartSync_StubWithoutAddress`, vault resolve PASS.

## Lab script

```powershell
.\scripts\run-comms-connect-lab.ps1 -UseCompose
# or against already-running API: .\scripts\run-comms-connect-lab.ps1
```

Prior PASS: `reports/comms-connect-lab.log` (RT-10 items_ok>0 with dovecot-lab).  
If compose/IMAP down in this session — unit path above is the Deepen D4 gate; re-run script when lab stack is up.

## Matrix

AC-C6 Scaffold ✅ · Pilot lab [x] · Pilot field [ ] RT-10.
