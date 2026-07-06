# ERA XDR — MVP Sprint-2 Specification (Фаза 2)



**Версия:** 1.0  

**Дата:** 9 июня 2026 г.  

**Статус:** Закрыт  

**Цель:** расширить ERA Core до платформы XDR — детекция, AI-расследование, SOAR,

мульти-доменная телеметрия, операционный контур SOC.



---



## Backlog (S2-1 … S2-12)



| ID | Задача | Модуль | DoD | Статус |

|---|---|---|---|---|

| S2-1 | control-plane: политики, assets, license gate | services/control-plane | F2-7, F2-9 | [x] |

| S2-2 | detection-engine: Sigma + запись detections | services/detection-engine | F2-3, F2-10 | [x] |

| S2-3 | Кросс-доменная корреляция APT | detection-engine | F2-3 | [x] |

| S2-4 | ai-core: alert → storyline → verdict | services/ai-core | F2-1 | [x] |

| S2-5 | SOAR: 3 плейбука | services/soar | F2-4 | [x] |

| S2-6 | VM findings → Envelope → Kafka | services/vm | F2-5 | [x] |

| S2-7 | Коллекторы network/auth/file | crates/era-agent | F2-2 | [x] |

| S2-8 | Tamper protection агента | crates/era-agent | F2-8 | [x] |

| S2-9 | Case Management API + UI | control-plane + ui | F2-6 | [x] |

| S2-10 | Sigma corpus ≥500 + lint | data/sigma-corpus | F2-10 | [x] |

| S2-11 | Asset Inventory UI | ui/assets | F2-7 | [x] |

| S2-12 | E2E Phase-2 smoke | scripts/run-phase2-e2e.ps1 | F2-* | [x] |



См. [`Development-Plan.md`](Development-Plan.md) — Definition of Done Фазы 2.



---



## Запуск Phase-2 dev-стека



```powershell

docker compose -f deploy/docker-compose.dev.yml up -d

cd services/ingest-gateway;  go run ./cmd/ingest-gateway   # :50051

cd services/event-writer;    go run ./cmd/event-writer     # :8089

cd services/control-plane;   go run ./cmd/control-plane    # :8090

cd services/detection-engine; go run ./cmd/detection-engine

cd services/ai-core;         go run ./cmd/ai-core          # :8091

cd services/soar;              go run ./cmd/soar             # :8092

$env:ERA_KAFKA_BROKERS="localhost:9092"; cd services/vm; go run ./cmd/vm-engine



# Агент: multi-domain + tamper sim

$env:ERA_DOMAIN_STUB="1"; $env:ERA_TAMPER_SIM="1"

$env:ERA_GATEWAY_ADDR="http://127.0.0.1:50051"

cargo run -p era-agent

```



Smoke: `.\scripts\run-phase2-e2e.ps1`



UI: `http://127.0.0.1:8090/ui/cases/` · `http://127.0.0.1:8090/ui/assets/` · events `:8089/ui/`

