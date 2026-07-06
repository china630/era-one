# ERA XDR — Master GA Execution Plan

**Цель:** последовательная реализация **Wave GA-1 → GA-2 → GA-3** до готового продукта (все издания ADR-0005).

**Трекинг:** [`Production-GA-Spec.md`](Production-GA-Spec.md) · [`Development-Plan.md`](Development-Plan.md)

---

## Порядок волн

| Волна | Exit | Ключевые артефакты |
|---|---|---|
| **GA-1** | F-GA-1…15 (кроме F-GA-5/7 partial) | Win+Linux capture, prod compose, mTLS, Ollama, SOAR connectors, curated Sigma, cases |
| **GA-1.1** | S8-1…3 PASS | macOS capture, React+SSO, loadgen 10k proof на prod |
| **GA-2** | S6-20 sign-off | ITDR, TIP STIX, custody, tamper prod, NDR depth, risk score, chaos, MITRE eval |
| **GA-3** | S7-18 sign-off | VM cred, federated SQLite, Helm, PQC license, perimeter prod, edition matrix |

---

## Команды приёмки

```powershell
# Полный прогон (после docker compose prod up)
powershell -ExecutionPolicy Bypass -File scripts/run-ga-full.ps1

# По волнам
powershell -File scripts/run-ga1-smoke.ps1
powershell -File scripts/run-chaos-smoke.ps1
helm template era-xdr deploy/helm/era-xdr/
```

---

## Текущий статус

- **GA-1 / GA-1.1 / GA-2 / GA-3 (софт):** закрыто — см. `Production-GA-Spec.md`
- **Приёмка:** пилот, loadgen proof, soak, pen-test — чек-листы

---

## Вне кода (параллельный трек)

- ISO 27001 программа вендора — [`Market-Positioning-AZ.md`](Market-Positioning-AZ.md) §5
- Pen-test / pilot SCSS — [`Pilot-Readiness-Checklist.md`](Pilot-Readiness-Checklist.md)
- Sandbox — интеграция WildFire, не свой модуль — Market Positioning §6
