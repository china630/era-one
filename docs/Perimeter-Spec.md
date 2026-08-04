# ERA Perimeter — Spec (MVP + Phase 2)

**Статус:** server MVP ✅ · Phase 2 code ✅ · field/pen-test ⏸  
**ADR:** [0031](adr/0031-era-resolve-and-perimeter-editions.md) · **PRD:** [PRD-ERA-Perimeter.md](products/PRD-ERA-Perimeter.md)

## Компоненты

| Сервис | Порт | Роль |
|--------|------|------|
| `services/waf` | `:8093` | Reverse-proxy + OWASP-style + CRS-lite body scan |
| `services/ngfw` | `:8094` | L3/L4 policy decision API (+ optional Linux host apply) |
| `services/dlp` | `:8095` | Session DLP (PAM) + content inspect API |

## WAF

- `GET /healthz`
- `GET /api/v1/waf/rules` — loaded rules
- `POST /api/v1/waf/reload` — reload pack from `ERA_WAF_RULES_PATH`
- Catch-all `/` — evaluate path/query/headers/**POST body** (limit `ERA_WAF_BODY_LIMIT`, default 1MiB) → 403 JSON or proxy to `ERA_WAF_UPSTREAM`
- Rule pack: OWASP-style + **CRS-lite** subset (`era-waf-crs-*`); not full ModSecurity CRS dump
- Env: `ERA_KAFKA_BROKERS` (optional block telemetry)
- License: `ModulePerimeter`

## NGFW

- `POST /api/v1/ngfw/evaluate` — `{src_ip,dst_ip,protocol,dst_port}` (+ `dry_run` / optional `host_apply`)
- `GET/POST /api/v1/ngfw/policies` — list / replace rules (file persist `ERA_NGFW_POLICY_PATH`)
- `POST /api/v1/ngfw/apply` — opt-in host applicator when `ERA_NGFW_APPLY=1` (Linux nftables/iptables); default = policy API only
- Windows / non-root: document-only dry-run; SOAR/Response consume evaluate
- Deny → optional Kafka network Envelope
- License: `ModulePerimeter`

## DLP

### Session (PAM shared)

Session start/command/end — keyword exfil alerts. Licensed via `pam` or `perimeter`.

### Content (Phase 2)

- `POST /api/v1/dlp/inspect` — `{path, mime, content}` → findings + `blocked`
- Rules: PII/PCI/secret/MIME/path-glob; session DLP unchanged

## Compose

```bash
docker compose -f deploy/docker-compose.prod.yml --profile perimeter up -d waf ngfw dlp
```

## Smoke

`scripts/run-perimeter-smoke.ps1`

## Explicit non-claims

Not Palo-class inline ASIC NGFW; not full OWASP CRS / eBPF datapath; GA marketing still needs pen-test/field soak.
