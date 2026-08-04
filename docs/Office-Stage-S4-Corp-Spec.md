# ERA Office — Stage S4 Corporate desktop shell (B-corp v1)

**Дата:** 29 июля 2026 г. / обновлено 3 августа 2026 г.  
**Статус:** [x]  
**Targets: Browser ❌ · Solo ✅ (shared shell) · Corporate ✅**  
**Связано:** [Office-Roadmap](Office-Roadmap.md) §0 S4 / §1.1–1.2 / §3.3 · [ADR-0026](adr/0026-sovereign-office-engine.md)

## Цель

Один Tauri shell с профилем **Corporate**: WebView грузит tenant Workspace URL + SSO (tenant `/login` в том же WebView). Engines локально **не** поднимаются. Deep link `era-office://open?…` / CLI args.

## Deliverable

| Путь | Роль |
|------|------|
| `apps/era-office-desktop/src-tauri/src/config.rs` | `Profile` + `server_url`, env overrides, `config.json` |
| `apps/era-office-desktop/src-tauri/src/corp.rs` | Workspace href + deep-link parse/resolve |
| `apps/era-office-desktop/ui/` | First-run profile chooser + Corporate URL setup + Solo editor |
| Commands | `config_get` / `config_set`, `corp_go`, `corp_open_deep_link`, `corp_parse_deep_link` |

## Acceptance criteria

| ID | Criterion | Evidence |
|----|-----------|----------|
| AC-S4-1 | Persist profile + `server_url` | `config::tests::roundtrip_file` |
| AC-S4-2 | Normalize `server_url` (https default) | `config::tests::normalize_*` |
| AC-S4-3 | Corporate → `/drive/` on tenant host | `corp::tests::workspace_default_drive` |
| AC-S4-4 | Deep link `era-office://open?path=` | `corp::tests::deep_link_path`, `args_scan` |
| AC-S4-5 | First-run / setup UI for Corporate URL | `ui/index.html` boot panel |
| AC-S4-6 | Package builds + tests | `cargo test -p era-office-desktop --lib`; `cargo check -p era-office-desktop` |

## Config

- File: `{config_dir}/era-office-desktop/config.json`
- Env: `ERA_OFFICE_PROFILE=solo|corporate`, `ERA_OFFICE_SERVER_URL=https://…`

## Deep links

```
era-office://open?path=/docs/{id}
era-office://open?url=https%3A%2F%2Ftenant%2F…
era-office://open?path=/drive/
```

Passed as process argv (OS protocol handler / CLI). Absolute `http(s)` args also accepted.

## Out of scope

Local engine re-host, hybrid LocalFs↔Drive sync, OS protocol installer packaging (Store/MDM), Solo Tables (S5).
