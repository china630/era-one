# ERA Resolve — Spec (MVP + Phase 2)

**Статус:** code MVP ✅ · Phase 2 (DoH / Atlas packs / agent DnsEvent) ✅ · field DNS lab ⏸ · live commercial TI ⏸  
**ADR:** [0031](adr/0031-era-resolve-and-perimeter-editions.md) · **PRD:** [PRD-ERA-Resolve.md](products/PRD-ERA-Resolve.md)  
**Slogan:** ONE QUERY. ONE POLICY. ONE VERDICT.

## Pillars

| Pillar | Path | Role |
|--------|------|------|
| Guard | `internal/guard`, `internal/dnsx` | Policy + Atlas → allow / nxdomain / sinkhole |
| Trace | `internal/trace` | DnsEvent Envelope + recent ring |
| Atlas | `internal/atlas` | Offline domain TI packs (+ Update Service `atlas-pack`) |
| DoH | `internal/doh` | RFC 8484 `/dns-query` (lab `:8443`) |

## API (HTTP `:8134`)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/healthz` | health + pack meta + `doh_enabled` |
| POST | `/api/v1/resolve/verdict` | `{qname,qtype}` → verdict |
| GET/POST | `/api/v1/resolve/policy` | list / replace rules |
| GET/POST | `/api/v1/resolve/packs` | Atlas meta / load pack |
| POST | `/api/v1/resolve/packs/reload` | reload from `ERA_ATLAS_PACK_DIR` / USB path |
| GET/POST | `/api/v1/resolve/settings` | `{doh_enabled}` |
| GET | `/api/v1/resolve/trace` | recent Trace records |
| GET/POST | `/dns-query` | DoH (also on `:8443`) |
| GET | `/ui/` | lab UI (pack status + DoH toggle) |

## DNS / DoH

- `ERA_RESOLVE_DNS_ADDR` default `:5353` (compose may map `53:5353`)
- `ERA_RESOLVE_DNS_DISABLE=1` to skip UDP listener
- `ERA_RESOLVE_DOH_ADDR` default `:8443`; `ERA_RESOLVE_DOH_DISABLE=1` to skip
- TLS: `ERA_RESOLVE_DOH_CERT`/`KEY` or `ERA_TLS_CERT`/`KEY`; else lab plain HTTP
- Minimal RFC 1035 UDP + RFC 8484 DoH; same Guard policy (no recursion)

## Atlas delivery (air-gap)

1. Place pack JSON under `data/atlas-packs/` (or USB mount).
2. Update Service kind `atlas-pack` (`ERA_BUNDLE_KIND=atlas-pack`, `ERA_ATLAS_PACK_DIR`).
3. Resolve: `ERA_ATLAS_PACK_DIR` at start **or** `POST /api/v1/resolve/packs/reload` with `{dir|path}`.
4. Signed/unsigned: Update Service signs manifests; Resolve loads JSON content (unsigned USB path documented for air-gap).

## Agent DNS Trace

- `era-collectors::emit_dns_event` / `stub_sample_observation` → `DnsEvent` Envelope → ingest
- Emitter metadata: `Source.environment = "mode=stub"` (`DNS_EMITTER_MODE`) until live ETW/WHQL hook
- Query/answers redacted before `pii_sanitized=true` (ADR-0009)
- `era-agent-core::builder::dns_envelope`
- Windows ETW / Linux: best-effort stub (not WHQL); Core NDR tunnel stays in detection-engine

## Env

| Var | Meaning |
|-----|---------|
| `ERA_RESOLVE_POLICY_PATH` | JSON policy file |
| `ERA_RESOLVE_ATLAS_PATH` | Atlas pack JSON |
| `ERA_ATLAS_PACK_DIR` | Directory of packs (Update Service / USB) |
| `ERA_KAFKA_BROKERS` | optional Trace publish |
| `ERA_UI_DIR` | static UI root |

## Compose

```bash
docker compose -f deploy/docker-compose.prod.yml --profile resolve up -d resolve
```

## Smoke

`scripts/run-resolve-smoke.ps1`

## Non-claims

Not DNSSense full parity; not live commercial TI SaaS feed; DoH/Atlas delivery = code ✅; live TI = external/content.
