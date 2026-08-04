---
name: acceptance-closeout
description: >-
  Close a product task against ERA Acceptance Standard v1.2 — Matrix SSOT,
  gate vs AC, consistency script, no false-green. Use after finishing a feature,
  stage wave, AuthZ/license change, or before claiming Scaffold ✅ / Pilot-ready.
---

# Acceptance closeout (ERA One)

**Canon:** `docs/products/ERA-Product-Acceptance-Standard.md` (v1.2+)  
**Agent rule:** `.cursor/rules/task-acceptance.mdc`

## Steps (mandatory)

1. Identify product line (Control / Comms / Office / Shared).
2. Update **Implementation-Matrix** (SSOT of color):
   - Scaffold ✅ only if PRD wording + **negative path** + no Critical residual + not field-intent.
   - Else 🟡. Worst-component: any 🟡 sub-row → AC 🟡.
   - Pilot-ready separate; field AC max Scaffold 🟡.
3. Sync rollups in the **same change**: Sprint-Index, MVP/Program header, Gap-List, stage G3 prose, PRD status if colored.
4. Run:

```powershell
pwsh -NoProfile -File scripts/check-acceptance-consistency.ps1
```

Must exit 0. Fix banned `all ✅` / `ga (partner)` prose if FAIL.
5. editions `ga` only from yaml SSOT after Pilot-ready (Comms/Office never soft-ga).
6. Proof: `reports/*` log or CI artifact linked from Matrix/Index.

## Forbidden

- Closing Comms/Office via `MVP-Sprint-1-Spec.md` alone.
- Treating stage `gate[x]` as AC Scaffold ✅.
- Writing «Matrix all ✅» when any in-scope AC is 🟡.
