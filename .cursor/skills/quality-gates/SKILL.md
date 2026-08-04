---
name: quality-gates
description: >-
  Run ERA quality tooling — acceptance consistency, secrets scan, Go/Rust lint,
  vuln advisories, optional Playwright smoke and dep-graph export. Use before
  claiming a PR ready, after AuthZ/security edits, or when user asks for gates.
---

# Quality gates (ERA One)

## Fast path (local)

```powershell
pwsh -NoProfile -File scripts/run-quality-gates.ps1
```

Flags:

- `-SkipLint` / `-SkipSecrets` / `-SkipVuln` / `-SkipAcceptance`
- `-WithE2E` — Office Playwright smoke (needs toolchain; may start workspace)
- `-WithDepGraph` — write `reports/deps/`

## What each gate means

| Gate | Script / tool | Blocks |
|------|---------------|--------|
| Acceptance | `check-acceptance-consistency.ps1` | False-green docs / false `ga` |
| Secrets | gitleaks (CI) / local if installed | Key/PII leaks |
| Lint | golangci-lint + cargo clippy (scoped) | Obvious code smell |
| Vuln | govulncheck + cargo deny advisories | Known CVEs |
| E2E | `ui/office/e2e` Playwright | UI smoke regressions |
| Dep graph | `export-dep-graph.ps1` | Agent context for blast radius |

## Agent rules

- Prefer these scripts over inventing ad-hoc commands.
- Air-gap: do not phone home; scanners run offline/local.
- After FAIL: fix or demote Matrix Scaffold to 🟡 — do not greenwash.
