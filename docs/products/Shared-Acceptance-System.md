# ERA Shared Platform — система приёмки (ownership)

**Версия:** 1.0  
**Дата:** 30 июля 2026 г.  
**Статус:** Accepted  
**Канон:** [`ERA-Product-Acceptance-Standard.md`](ERA-Product-Acceptance-Standard.md) **v1.3** — Product Readiness у consumer (Drive UI → [`Office-Product-Readiness-Matrix.md`](../Office-Product-Readiness-Matrix.md))
**ADR:** [`0025-era-one-shared-platform.md`](../adr/0025-era-one-shared-platform.md)  
**Evidence:** [`Control-Evidence-Rules.md`](../Control-Evidence-Rules.md) (общие правила) · consumer Evidence при product AC

---

## 1. Зачем

Shared (`platform/*`) используется Control, Comms и Office. Без явного ownership легко:

- поставить `[x]` дважды в разных матрицах, или
- закрыть Scaffold в одной линейке и объявить Pilot-ready в другой без proof.

**Правило:** один компонент — один Scaffold owner; Pilot-ready принадлежит **продуктовому AC**, который продаёт/обещает сценарий.

---

## 2. Ownership table

| Пакет / сервис | Scaffold proof (где) | Product AC / Pilot-ready owner |
|----------------|----------------------|--------------------------------|
| `platform/identity` | ADR-Implementation-Matrix §0025 | Shared; Comms/Office login AC **ссылаются** |
| `platform/tenant` | ADR §0025 | Shared |
| `platform/licensegate` | ADR §0025 + edition tests | Shared; editions-*.yaml honesty |
| `platform/adminportal` | ADR §0024/0025 | Shared |
| `platform/drive` | **Office**-Implementation-Matrix (AC-O3…) | **Office** |
| `platform/docs-engine` | **Office** matrix (AC-O1, O2, …) | **Office** |
| `platform/workspace` | **Office** matrix | **Office** |
| `platform/signing` | Office / Shared unit | Office (docs signing) · Shared (keys) |
| Comms mail → CH / identity hooks | **Comms**-Implementation-Matrix | **Comms** |

Манифест пакетов: [`products.yaml`](../../products.yaml) → `shared_platform` · [`editions-shared.yaml`](../../editions-shared.yaml).

---

## 3. Как принимать изменение в shared

1. Определить owner по таблице §2.
2. Обновить **матрицу owner-продукта** (не все три).
3. Если затронут ADR-0025 контракт — строка в [`ADR-Implementation-Matrix.md`](../ADR-Implementation-Matrix.md).
4. Proof: package tests + лог; для Drive/Docs — ещё `run-office-stage-gate` при wave AC.
5. Edition status в `editions-shared.yaml` — только с Evidence (нет ложного `ga`).

---

## 4. Слои (минимальные)

```
ADR-0025 / editions-shared.yaml
        ↓
ADR-Implementation-Matrix §0025     ← Scaffold shared core
Office- / Comms-Implementation-Matrix ← product AC на shared deps
        ↓
Control-Evidence-Rules + consumer Evidence-Rules
```

Отдельный Sprint-Index для Shared **не** требуется: волны живут у consumer-продукта.

---

## 5. Документы

- [`ERA-Product-Acceptance-Standard.md`](ERA-Product-Acceptance-Standard.md)
- [`Control-Acceptance-System.md`](Control-Acceptance-System.md)
- [`Office-Acceptance-System.md`](Office-Acceptance-System.md)
- [`Comms-Acceptance-System.md`](Comms-Acceptance-System.md)
