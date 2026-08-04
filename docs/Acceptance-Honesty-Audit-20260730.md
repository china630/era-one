# Acceptance Honesty Audit — 30 июля 2026

**Канон:** [`ERA-Product-Acceptance-Standard.md`](products/ERA-Product-Acceptance-Standard.md)  
**Цель:** найти ложные `ga` и `[x]` без Pilot-ready; исправить docs/editions.

---

## 1. Вердикт

| Область | Ложный `ga` в editions? | Ложные `[x]` без Pilot-ready? | После фикса |
|---------|-------------------------|-------------------------------|-------------|
| **editions-comms / office / shared** | Нет — везде `mvp`/`roadmap` | — | OK |
| **editions-control Core/AI/Response** | `ga` = **software GA** (carve-out) | F-GA-5/8/15 были `[x]` при field block | DoD исправлен; Evidence carve-out |
| **Comms matrix** | Нет | CMM-H MM-H-8 заявлял Pilot-ready | Исправлено |
| **Office matrix / Index** | Нет | Waves `[x]` без колонки Pilot-ready; MVP-Spec «post-MVP roadmap only» vs Index `[x]` | Колонки + текст |
| **Control Production-GA-Spec** | — | F-GA-5/8/15 `[x]` vs `[blocked: field]` ниже | Таблица DoD разделена |

**Итог:** ложных `ga` у Comms/Office не было. Главный долг — Control DoD и смешение Scaffold/`[x]` с полем; частично CMM-H и Office post-MVP формулировки.

---

## 2. Editions snapshot (после аудита)

| Manifest | `ga` / `ga-option` | `mvp` | `roadmap` |
|----------|--------------------|-------|-----------|
| `editions-control.yaml` | Core, AI, Response · Vuln/Federated/National option | Manage…Resolve, Workbench… | — |
| `editions-comms.yaml` | — | Mail Server, Connect, Migration, Bridge, MM, Chat, Conference, AI | Mail Client |
| `editions-office.yaml` | — | Documents, Tables, Presentations, Projects, AI | — |
| `editions-shared.yaml` | — | Drive | Sign |

Comms/Office: **не повышать до `ga`** пока RT-09 / RT-O09 / partner field open.

---

## 3. Найденные нарушения и фиксы

### H-1 — Control F-GA-5 / F-GA-8 / F-GA-15

| Было | Стало |
|------|--------|
| DoD: `[x]` + «proof на пилоте» | Scaffold `[x]` · Pilot-ready **`[blocked: field]`** |
| Ниже в том же файле уже было `[blocked: field]` — противоречие | Противоречие снято |

Файлы: `Production-GA-Spec.md`, `Control-Sprint-Index.md`, `Control-Evidence-Rules.md` (software GA carve-out), comments в `editions-control.yaml` / `products.yaml`.

### H-2 — Comms CMM-H «matrix Pilot-ready»

| Было | Стало |
|------|--------|
| MM-H-8 `[x] gate + matrix` Pilot-ready | Gate PASS · Scaffold ✅ · Pilot-ready **[ ]** IceWarp |
| G3 «Matrix Pilot-ready AC-MM» | Scaffold ✅; Pilot-ready open |

Файл: `Comms-Stage-CMM-H-Spec.md`. Matrix сводка уже честная (`[ ] IceWarp`).

### H-3 — Office Index / Matrix / MVP-Spec

| Было | Стало |
|------|--------|
| Waves одним `[x]` | Scaffold \| Pilot-ready |
| Post-MVP «roadmap only» при Index `[x]` gates | Scaffold `[x]`, Pilot-ready open, editions `mvp` |
| Сводка изданий без Pilot-ready | Колонка Pilot-ready `[ ]` RT-O09 / field |

Файлы: `Office-Implementation-Matrix.md`, `Office-Sprint-Index.md`, `Office-MVP-Spec.md`.

### H-4 — ADR-0024 итог устарел

| Было | Стало |
|------|--------|
| «Comms/Office roadmap» | Comms/Office **mvp** scaffold; RT open |

Файл: `ADR-Implementation-Matrix.md`.

### H-5 — Product-Line легенда GA

Уточнено: Core/AI/Response = software GA; не обещать field 10k без proof.  
Файл: `ERA-Product-Line.md`.

---

## 4. Открытые долги (не ложный `ga`, но не Pilot-ready)

| ID | Что | Статус |
|----|-----|--------|
| CTRL-P1 | F-GA-5 loadgen prod | `[blocked: field]` |
| CTRL-P2 | F-GA-8 coverage ≥90% | `[blocked: field]` |
| CTRL-P3 | F-GA-15 pilot signature | `[blocked: field]` |
| COMMS-P1 | RT-09 customer field | SKIP → Mail Server stays `mvp` |
| COMMS-P2 | MM IceWarp field | Pilot-ready `[ ]` |
| COMMS-P3 | Chat / Conference / Comms AI field | Pilot-ready `[ ]` |
| OFFICE-P1 | RT-O09 | Pilot-ready `[ ]` |
| DOC-P1 | `Comms-MVP-Spec.md` отсутствует (ссылки в Acceptance/Gap) | долг docs; Index/Matrix — SSOT волн |

---

## 5. Правила, которые подтверждены аудитом

1. Нет лога / field — нет Pilot-ready `[x]`.
2. Wave `[x]` в Sprint-Index = Scaffold gate, пока явно не написано field PASS.
3. Edition `ga` для Comms/Office — только после Pilot-ready + sign-off.
4. Control Core/AI/Response могут оставаться `ga` как software GA **только** с явным списком открытых Pilot-ready (carve-out Evidence).

---

## 6. Что не трогали

- Код сервисов / тесты (claims были в docs/editions).
- Понижение Core/AI/Response с `ga` → `mvp` (ломало бы sales SSOT и `products_test`); вместо этого — honesty carve-out + исправление F-GA.
- Закрытие RT-09 / RT-O09 (требует заказчика).

---

## 7. Следующий проход (опционально)

1. Восстановить или заменить ссылки на отсутствующий `Comms-MVP-Spec.md`.
2. В ADR-matrix для Manage/PAM/Perimeter добавить явные колонки Scaffold | Pilot-ready построчно (сейчас часто один ✅).
3. Field proof → обновить Pilot-ready и только потом edition `ga`.

---

## 8. Pass v1.2 стандарта (30 июля 2026, later)

**Канон:** [`ERA-Product-Acceptance-Standard.md`](products/ERA-Product-Acceptance-Standard.md) **v1.2**

В стандарт добавлены (чтобы ложное зелёное было **нарушением**, а не «недосмотром docs»):

| Правило | Где |
|---------|-----|
| Rollup SSOT = Matrix; запрет `all ✅` | §3.4 |
| Gate / AC / Pilot раздельно | §3.1 |
| Field-AC max Scaffold 🟡 | §3.2 |
| `ga` только из editions yaml | §3.3 |
| Consistency в одном PR | §3.5 |
| software-ga carve-out ограничен | §3.6 |
| G3/G4/G6 без false Matrix ✅ / product-green | §4 |
| Product Matrix Control = Control-Implementation-Matrix | §6 |
| CI helper | `scripts/check-acceptance-consistency.ps1` |

Consistency pass выполнен: Office/Comms шапки, Partner Bundle, PRD P3/Projects, F-GA-5/8/15 → 🟡, Control signoff переименован по смыслу в scaffold-gate.

---

## 9. Pass v1.3 — Product Readiness (30 июля 2026)

**Корень путаницы:** «матрица готовности» = Implementation-Matrix (BE).

**Фикс:** канон §3.4 — два SSOT; файлы:

- [`Office-Product-Readiness-Matrix.md`](Office-Product-Readiness-Matrix.md)
- [`Comms-Product-Readiness-Matrix.md`](Comms-Product-Readiness-Matrix.md)
- [`Control-Product-Readiness-Matrix.md`](Control-Product-Readiness-Matrix.md)

Агент: запрос готовности → Readiness (Gate/BE/UI/Demo/Pilot/Edition/Sell), не только AC.
