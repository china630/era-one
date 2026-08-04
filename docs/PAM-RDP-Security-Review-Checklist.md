# PAM RDP — security-review checklist

**Статус:** Phase 2 **code ready** · GA claim still blocked on external review/pen-test  
**Связано:** [PAM-Spec.md](PAM-Spec.md) · ADR-0013 · `services/pam/internal/{broker,recording,sessionpolicy}`

## Current MVP + Phase 2 code

- [x] Local TCP listener → dial target `:3389`
- [x] Session id + custody audit `proxy.rdp.start` / `end` / `timeout`
- [x] Credential broker: server-side reveal; API returns `inject_token` + username only (no password)
- [x] Session idle/max timeout (`ERA_PAM_SESSION_IDLE`, `ERA_PAM_SESSION_MAX`)
- [x] Metadata recording (JSON artifact + SHA-256); not Guacamole video
- [x] TLS via platform `ERA_TLS_*` / `httpserver.Listen` (lab certs)

## Still requires security-review before GA claim

- [ ] Threat model signed; pen-test scope agreed
- [ ] Abuse cases: port forward abuse, lateral RDP via proxy (exercise + mitigations)
- [ ] HSM-backed vault master key for prod
- [ ] Graphical/session video recording (optional productization)

## Explicit non-claims

Do **not** market as Guacamole/Teleport replacement until remaining checklist is signed.
