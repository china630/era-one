# Deepen D0 evidence — 2026-08-04

## Code

- `ERA_MAIL_CORE_ADDR=era-mail-core:8152` + `ERA_MAIL_ADMIN_HOST=0.0.0.0` / port 8152
- readyz: `ERA_MAIL_AUDIT_REQUIRE=1` + noop CH → 503 (`TestReadyzAuditRequiredWithoutCH`)
- ui/mail `/mail/api/drive/attachment-link` JWT forward + deny without Drive license

## Tests

```
go test -C services/comms/mail ./internal/api/ -run Readyz — PASS
go test -C ui/mail . -run Drive — PASS
cargo test --manifest-path services/comms/mail/core/Cargo.toml --lib — PASS
```

## Matrix

AC-C1/C5/C7 Scaffold ✅ lab; Pilot lab [x]; Pilot field [ ] RT-09.
