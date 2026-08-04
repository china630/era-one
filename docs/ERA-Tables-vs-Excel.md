# ERA Tables vs Microsoft Excel — honest MVP limits

**Дата:** 30 июля 2026 г.  
**Издание:** ERA Tables (`.erat`) · Wave **O-T**  
**PRD:** [`PRD-Office-P2.md`](products/PRD-Office-P2.md)

## Что есть в MVP (Gov Eval)

| Capability | Limit |
|------------|-------|
| Grid | ~256 columns × 1024 rows (single sheet) |
| Formulas | `SUM`, `AVERAGE`, `MIN`, `MAX` + A1 / ranges |
| Co-edit | 2+ clients, cell-level OpLog + WS fan-out |
| xlsx I/O | Single sheet, no macros |
| Storage | ERA Drive only |
| License | `office-tables` |

## Чего нет (намеренно)

- VBA / macros  
- Pivot tables / Power Query  
- 400+ Excel functions  
- Charts, conditional formatting, named styles fidelity  
- Multi-sheet workbooks  

Позиционирование: «типовые отчёты и расчёты в контуре», не полный Excel.
