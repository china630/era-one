# ERA XDR — MVP Sprint-4 Specification (Фаза 4)

**Версия:** 1.0  
**Дата:** 9 июня 2026 г.  
**Статус:** Закрыт  
**Цель:** национальный STIX/TAXII hub, обмен знаниями без PII, регуляторная отчётность.

---

## Backlog (S4-1 … S4-8)

| ID | Задача | Модуль | DoD | Статус |
|---|---|---|---|---|
| S4-1 | STIX/TAXII hub publish/poll | services/national-hub | F4-1 | [x] |
| S4-2 | PII audit перед export | national-hub/sanitize | F4-2 | [x] |
| S4-3 | National IOC -> detection delta | detection-engine/tip | F4-3 | [x] |
| S4-4 | National license gate default off | platform/licensegate | F4-4 | [x] |
| S4-5 | AZ/CB regulatory report | services/compliance | F4-5 | [x] |
| S4-6 | DP noisy aggregates | national-hub/dp | F4-3 | [x] |
| S4-7 | PQC-readiness license | services/license/pqc | Scope | [x] |
| S4-8 | E2E Phase-4 smoke | scripts/run-phase4-e2e.ps1 | F4-* | [x] |

---

## Запуск

```powershell
cd services/national-hub
$env:ERA_NATIONAL_DEV="1"
go run ./cmd/national-hub    # :8099 TAXII

cd services/compliance
$env:ERA_NATIONAL_DEV="1"
go run ./cmd/compliance      # :8100 reports

# detection-engine подхватывает data/national-iocs/patterns.json
$env:ERA_NATIONAL_IOCS="../../data/national-iocs/patterns.json"
cd services/detection-engine; go run ./cmd/detection-engine
```

Smoke: `.\scripts\run-phase4-e2e.ps1`

Лицензия: `ERA_LICENSE_MODULES=national` или `ERA_NATIONAL_DEV=1`.
