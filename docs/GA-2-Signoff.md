# ERA XDR — Wave GA-2 Sign-off

**Версия:** 0.1 (skeleton)  
**Дата:** 11 июня 2026 г.  
**Статус:** `[~]` в работе

## Exit criteria

| ID | Критерий | Статус | Доказательство |
|---|---|---|---|
| S6-1 | ITDR rules + identity graph | `[x]` | `go test ./internal/itdr/...` PASS |
| S6-2 | Internal TIP STIX ingest | `[x]` | `go test ./internal/tip/...` PASS |
| S6-3 | Hash-chain custody | `[x]` | `go test ./custody/...` PASS, event-writer hook |
| S6-4 | Compliance HTML export + CH stub | `[x]` | `go test ./internal/report/...` PASS |
| S6-5 | Tamper OS-level watchdog | `[x]` | `cargo test tamper::` PASS |
| S6-6 | NDR beaconing + DNS tunnel | `[x]` | `go test ./internal/ndr/...` PASS |
| S6-7 | Risk score + alert dedup | `[x]` | `go test ./internal/risk/...` PASS |
| S6-10 | Chaos smoke Kafka/CH | `[x]` | `scripts/run-chaos-smoke.ps1` |
| S6-18 | MITRE eval scenarios | `[x]` | `data/mitre-eval/scenarios.json` + golden test |

## Не в scope этой итерации

| ID | Задача | Статус |
|---|---|---|
| S6-8 | React UI + SSO | `[ ]` |
| S6-9 | Full mTLS mesh | `[ ]` |
| S6-11 … S6-17 | Soak, pen-test, HA runbooks | `[ ]` |
| S6-19 | Pilot #1 feedback | `[ ]` |

## Подписи

| Роль | Имя | Дата | Подпись |
|---|---|---|---|
| Engineering Lead | | | |
| SOC Lead | | | |
| Security / Compliance | | | |
| Customer Pilot #1 | | | |
