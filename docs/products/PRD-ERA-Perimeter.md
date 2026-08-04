# PRD — ERA Perimeter (MVP)

**Статус:** MVP code  
**ADR:** [0031](../adr/0031-era-resolve-and-perimeter-editions.md)  
**Спека:** [Perimeter-Spec.md](../Perimeter-Spec.md)

## Цель

Edge add-on: WAF reverse-proxy, NGFW policy API, session DLP (shared with PAM) — всё в контуре, license `perimeter`.

## Scope MVP

| Компонент | Must |
|-----------|------|
| WAF | Real reverse-proxy; rule pack + golden block/allow; admin list/reload; block → Envelope; licensegate |
| NGFW | Evaluate + CRUD policies; deny telemetry; licensegate; **не** packet path |
| DLP | Session command alerts (as today); available under pam **or** perimeter |

## Out of scope

ModSecurity CRS full corpus, eBPF/nftables, content DLP, TLS termination productization.

## Acceptance

| ID | Criterion | Proof |
|----|-----------|-------|
| AC-P1 | WAF blocks SQLi/XSS goldens, proxies clean | `go test ./services/waf/...` |
| AC-P2 | NGFW evaluate deny/allow golden | `go test ./services/ngfw/...` |
| AC-P3 | ModulePerimeter gates APIs | unit + smoke |
| AC-P4 | `docker compose --profile perimeter` documented | Perimeter-Spec + smoke script |
