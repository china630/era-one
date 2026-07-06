# ERA XDR — Wave GA-3 Sign-off

**Версия:** 0.1 (skeleton)  
**Дата:** 11 июня 2026 г.  
**Статус:** `[~]` в работе

## Exit criteria (partial)

| ID | Критерий | Издание | Статус | Доказательство |
|---|---|---|---|---|
| S7-1 | VM credentialed SSH scan stub | ERA Vuln | `[x]` | `go test ./internal/scanner/...` PASS |
| S7-3 | Federated hub SQLite persistence | ERA Federated | `[x]` | `go test ./internal/hub/...` PASS |
| S7-9 | PQC hybrid license verify | Security | `[x]` | `go test -run PQC` PASS |
| S7-10 | Helm chart (kafka+CH+CP) | Deploy | `[x]` | `deploy/helm/era-xdr/` |

## Не в scope этой итерации

| ID | Задача | Статус |
|---|---|---|
| S7-2 | VM scheduler + reports | `[ ]` |
| S7-4 … S7-8 | National, Perimeter, CTEM, BAS | `[ ]` |
| S7-11 … S7-17 | Multi-tenant, scale, upgrade automation | `[ ]` |
| S7-18 | Full edition catalog sign-off | `[ ]` |

## Edition matrix (ADR-0005)

| Издание | GA-3 ready | Лицензия |
|---|---|---|
| ERA Core | GA-1 | always |
| ERA Control AI | GA-1 | `control-ai` |
| ERA Response | GA-1 | `response` |
| ERA Vuln | partial (S7-1) | `vm` |
| ERA Federated | partial (S7-3) | `federated` |
| ERA National | — | `national` |

## Подписи

| Роль | Имя | Дата | Подпись |
|---|---|---|---|
| Product Owner | | | |
| Platform Engineering | | | |
| Legal / DP (Federated) | | | |
