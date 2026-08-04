# ERA Office — Solo Documents full UI (embed web docs)

**Дата:** 3 августа 2026 г.  
**Статус:** [x]  
**Targets: Browser ❌ · Solo ✅ · Corporate ❌**  
**Связано:** [Office-Roadmap](Office-Roadmap.md) §3.1–3.2 · [Office-Stage-S3-Solo-Spec](Office-Stage-S3-Solo-Spec.md)

## Цель

Полный веб-редактор [`ui/docs/web`](../ui/docs/web) в Solo desktop через локальный loopback-bridge (без Drive / Postgres / Identity).

## Архитектура

| Компонент | Роль |
|-----------|------|
| `solo_bridge.rs` | axum на `127.0.0.1:0` — статика + `/api/v1/docs*` + WS apply DocOp |
| `assets/solo-docs-boot.js` | Fake JWT, hide Platform chrome, Open/Save через bridge + snapshot с `document` |
| `assets/solo-docs-skin.css` | Solo skin |
| `solo_docs_go` | Navigate WebView → `http://127.0.0.1:{port}/docs/solo` |

## Acceptance

| ID | Criterion | Evidence |
|----|-----------|----------|
| AC-FD-1 | Solo → Documents opens full web chrome | `solo_docs_go` + injected `index.html` |
| AC-FD-2 | create/get/snapshot without Drive | `solo_bridge::tests::create_get_roundtrip`, `snapshot_with_document_body` |
| AC-FD-3 | Open/Save `.erad` via native dialog | `/api/v1/solo/file/*` + `rfd` |
| AC-FD-4 | Import/Export docx endpoints | bridge export/import routes |
| AC-FD-5 | No `/login` redirect; nav/presence hidden | boot.js + skin.css |
| AC-FD-6 | Tests | `cargo test -p era-office-desktop --lib` PASS (25) |

## Run

```powershell
cargo run -p era-office-desktop
```

Choose **Solo** → app navigates to full Documents. **Tables** stays thin UI (Profile → Solo → Tables).

## Out of scope

Co-edit peers, Drive versions/images, Office AI, Tables full UI, MS Store.
