# Deepen D7 — Mail Moderation (lab) — 2026-08-04

**Status:** `[x] lab` · IceWarp field **⏸** · **not `ga`**.

## Admin UI

Surface: `GET /ui/` on `mail-moderation` adminapi (mm.admin AuthZ).

Deepen: documents **force-release** and **escalate** lab path:

```http
POST /v1/moderation/holds/{id}?action=approve   # force-release
POST /v1/moderation/holds/{id}?action=escalate&comment=L2
```

Requires Bearer JWT with role `mm.admin` (prod) or `ERA_MM_DEV=1` / `ERA_MAIL_DEV=1` (lab).

## Proof

- AuthZ: `TestForceReleaseRequiresMMAdmin` (`internal/adminapi`)
- Engine escalate: `internal/engine` unit (no deliver after escalate without L2)
- UI lists holds + curl lab snippets

## IceWarp

Milter / IceWarp partner path remains **paused** until DF/partner. Native SMTP/milter lab only.
