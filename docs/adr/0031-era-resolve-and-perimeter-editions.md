# ADR-0031 — ERA Perimeter и ERA Resolve

**Статус:** Accepted → Implemented (MVP)  
**Дата:** 2026-07-30  
**Связано:** ADR-0003 (donors), ADR-0005 (editions), ADR-0010 (license), ADR-0013 (PAM/dlp), ADR-0020 (Observe)

## Контекст

В линейке ERA Control не хватало явных изданий для (1) edge app/network policy и (2) DNS Detection & Response. Sprint-3 оставил scaffold `waf`/`ngfw`/`dlp`; DNS DDR отсутствовал. Сайт ошибочно описывал Resolve как ITSM.

## Решение

### ERA Perimeter (`license_module: perimeter`)

- **WAF** (`services/waf`): reverse-proxy + OWASP-style regex rule pack (идея Coraza, код свой).
- **NGFW** (`services/ngfw`): L3/L4 **policy decision API** (идея Cilium NetworkPolicy), **не** packet firewall / eBPF datapath.
- **DLP** (`services/dlp`): privileged **session** audit (общие с ERA PAM). Content-DLP (file/email) — Phase 2.

Позиционирование: дополняет NGFW заказчика; не замена Palo Alto-класса.

### ERA Resolve (`license_module: resolve`)

DNS Detection & Response (класс DNSSense), slogan: **ONE QUERY. ONE POLICY. ONE VERDICT.**

| Pillar | Donor pattern (идея) | Роль |
|--------|----------------------|------|
| **Guard** | DNSDome | Protective DNS + verdict API (`allow` / `nxdomain` / `sinkhole`) |
| **Trace** | DNSEye | DnsEvent Envelope → ingest / Kafka |
| **Atlas** | Cyber X-Ray | Offline domain TI packs (air-gap USB/bundle; Hybrid pack delivery — Phase 2) |

Сервис: `services/resolve`. Core NDR DNS-tunnel heuristics остаются в `detection-engine` — Resolve владеет enforcement + enrichment.

### Глоссарий

- **ERA Resolve** ≠ пакет `services/comms/.../resolve` (mail curator).
- Brand tagline «ONE PERIMETER» ≠ издание ERA Perimeter.

## Последствия

- `editions-control.yaml`: оба издания `status: mvp`, `license_module.exists: true`.
- Compose profiles: `perimeter`, `resolve`.
- GA-гейты: field lab DNS :53, pen-test WAF, commercial TI, content-DLP, inline NGFW — вне MVP.
