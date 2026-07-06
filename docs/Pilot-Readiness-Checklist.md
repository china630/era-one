# ERA XDR — Pilot Readiness Checklist (Wave GA-1)



**Заказчик:** ____________________  

**Дата пилота:** ____________________  

**Издания:** Core + AI + Response  

**Подпись ERA:** ____________________ · **Подпись заказчика:** ____________________



> **Локальная прогонка (наша инфра):** `.\scripts\run-pilot-local.cmd` → `reports/pilot-local-*.log`
>
> **Field sizing / AC2 10k:** [`Field-Server-Sizing.md`](Field-Server-Sizing.md) · **Setup:** [`Field-Server-Setup.md`](Field-Server-Setup.md) · **Preflight:** `scripts/check-field-server.ps1`  
> **Матрица готовности:** [`Production-Readiness-Assessment.md`](Production-Readiness-Assessment.md)  
> Loadgen 10k и подпись заказчика — [gate: field] (см. Field-Server-Sizing).



## Инфраструктура



- [ ] `docker compose -f deploy/docker-compose.prod.yml up` — все сервices healthy

- [ ] Пароли CH/MinIO изменены с дефолтных

- [ ] Firewall: агенты → :50051, SOC → :8090–8092, :8089

- [ ] NTP/sealed clock для лицензии (ADR-0010)

- [ ] Backup volumes (ClickHouse + control-plane SQLite/Postgres)

- [ ] SSO (опция): `docker compose --profile sso up` → :8443 ([SSO-Setup-GA.md](SSO-Setup-GA.md))



## Агенты Win + Linux + macOS



- [ ] ≥2 Windows хоста с `ERA_PRODUCTION=1`, Sysmon active

- [ ] ≥2 Linux хоста с auditd execve, `ERA_PRODUCTION=1`

- [ ] ≥1 macOS 12+ с `ERA_PRODUCTION=1`, `ERA_MACOS_UNIFIED_JSONL` (NDJSON export unified log / ES sidecar)

- [ ] Asset coverage ≥90% (`GET /api/v1/assets`)

- [ ] PII golden-тест PASS на агенте (no cleartext PII in CH) — `run-pii-gate.ps1`



### macOS (export-модель GA)



```bash

export ERA_PRODUCTION=1

export ERA_MACOS_UNIFIED_JSONL=/var/log/era/unified.ndjson

export ERA_GATEWAY_ADDR=http://<server>:50051

export ERA_CONTROL_PLANE_URL=http://<server>:8090

export ERA_NODE_ID=MB-FIN-001

./era-agent

```



Sidecar (Endpoint Security / unified logging → NDJSON) настраивается у заказчика; native ESF in-agent — Platform phase.



## Detection & AI

> Governance и честные границы MVP — [`ADR-0022`](adr/0022-detection-content-governance.md),
> [`ADR-0023`](adr/0023-ai-investigation-explainability.md).

- [ ] Curated Sigma ≥100 правил loaded (`ERA_SIGMA_CORPUS`)

- [ ] ≥1 detection → **auto-case** (`ERA_DETECTION_AUTO_CASE=1`, high/critical → control-plane)

- [ ] AI investigate возвращает verdict + MITRE refs (on-prem LLM или heuristic fallback documented)

- [ ] **Зафиксировано для заказчика:** tamper = detect-and-alert (не EPP-grade prevent до WHQL)

- [ ] **Зафиксировано для заказчика:** AI verdict = triage accelerator, не автономный forensic verdict (custody chain — roadmap ADR-0023 Фаза 2)



## Response (SOAR)



- [ ] ≥1 playbook с **реальным коннектором** (не simulated-only): `ERA_SOAR_ISOLATE_SCRIPT`, `ERA_SOAR_TICKET_WEBHOOK`

- [ ] Playbook → case link (`ERA_CONTROL_PLANE_URL` + notes API)



## Нагрузка и стабильность



- [ ] Loadgen ≥10k ev/s 5 мин — PASS (`scripts/run-loadgen-prod.ps1` → `reports/loadgen-prod.log`)

- [ ] Soak 24h без OOM на gateway/event-writer

- [ ] Restart control-plane — cases/assets сохранены (F-GA-4)



> **Софтверные скрипты приёмки:** `run-pilot-local.ps1`, `run-ga-full.ps1`, `run-loadgen-prod.ps1`, `run-ga-e2e-prod.ps1` — готовы; чекбоксы закрываются **логом прогона на контуре**.



## Документация и процесс



- [ ] Install-Guide-GA передан заказчику

- [ ] Runbook инцидента (escalation, contacts)

- [ ] Лицензия Core+AI+Response выпущена и проверена offline



## Sign-off



| Критерий F-GA | PASS/FAIL | Доказательство |

|---|---|---|

| F-GA-1 Win capture | | |

| F-GA-2 Linux capture | | |

| F-GA-3 Prod deploy | | |

| F-GA-4 Persistent CP | | |

| F-GA-7 Case lifecycle | | |

| F-GA-10 AI investigate | | |

| F-GA-12 SOAR real connector | | |

| F-GA-14 Install guide | | |

| macOS export capture | | |



**Итог пилота:** ☐ GO для production rollout · ☐ NO-GO (см. remediation plan)



Remediation plan: _______________________________________________

