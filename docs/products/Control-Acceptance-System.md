# ERA Control — система контроля и приёмки

**Версия:** 1.1  
**Дата:** 30 июля 2026 г.  
**Статус:** Accepted  
**Канон:** [`ERA-Product-Acceptance-Standard.md`](ERA-Product-Acceptance-Standard.md) **v1.3**  
**Evidence:** [`Control-Evidence-Rules.md`](../Control-Evidence-Rules.md)  
**Индекс:** [`Control-Sprint-Index.md`](../Control-Sprint-Index.md)  
**Product Readiness (SSOT готовности):** [`Control-Product-Readiness-Matrix.md`](../Control-Product-Readiness-Matrix.md)  
**AC Matrix (SSOT BE):** [`Control-Implementation-Matrix.md`](../Control-Implementation-Matrix.md)  
**Аналоги:** [`Comms-Acceptance-System.md`](Comms-Acceptance-System.md) · [`Office-Acceptance-System.md`](Office-Acceptance-System.md)

---

## 1. Зачем

Обещание → код → proof → статус.  
**Готовность продукта** = [`Control-Product-Readiness-Matrix.md`](../Control-Product-Readiness-Matrix.md).  
**AC/BE** = Control-Implementation-Matrix.

**Правило:** «матрица готовности» → Readiness (все колонки). software-ga ≠ field PASS.

---

## 2. Слои

```
ERA-Platform-Vision / PRD-Perimeter / PRD-Resolve / Product-Line
        ↓
Production-GA-Spec · GA-Master-Execution-Plan · MVP-Sprint-1…4
        ↓
Control-Sprint-Index
        ↓
Control-Implementation-Matrix      ← AC / Scaffold BE
Control-Product-Readiness-Matrix   ← **один экран готовности**
ADR-Implementation-Matrix          ← ADR→код
        ↓
Control-Evidence-Rules · ci-gates · Pilot / WHQL / field gates
```

| Слой | Файл |
|------|------|
| **Product Readiness** | [`Control-Product-Readiness-Matrix.md`](../Control-Product-Readiness-Matrix.md) |
| AC / BE | [`Control-Implementation-Matrix.md`](../Control-Implementation-Matrix.md) |
| ADR | [`ADR-Implementation-Matrix.md`](../ADR-Implementation-Matrix.md) |
| Index | [`Control-Sprint-Index.md`](../Control-Sprint-Index.md) |

---

## 3. Легенда

Канон v1.3: Gate / BE / UI / Demo / Pilot; Product rollup = worst(слоёв).

---

## 4. Рабочий процесс

1. Backlog → `[~]` · AC Matrix 🟡  
2. Тесты + CI → `gate[x]`  
3. AC Matrix §3.2; обновить **Product-Readiness-Matrix**  
4. Index / Product-Line (§3.5)  
5. `.\scripts\check-acceptance-consistency.ps1`  

Signoff: `scaffold-gate-pass`, не product-green при открытом Pilot.

---

## 5. MVP-издания

Manage…Resolve — строки в **Control-Implementation-Matrix** + ADR; GA-гейт Product-Line §4.

---

## 6. Документы

- [`ERA-Product-Acceptance-Standard.md`](ERA-Product-Acceptance-Standard.md)
- [`Control-Product-Readiness-Matrix.md`](../Control-Product-Readiness-Matrix.md)
- [`Control-Implementation-Matrix.md`](../Control-Implementation-Matrix.md)
- [`Control-Sprint-Index.md`](../Control-Sprint-Index.md)
- [`Control-Evidence-Rules.md`](../Control-Evidence-Rules.md)
- [`Production-GA-Spec.md`](../Production-GA-Spec.md)
