# Honesty → Green evidence (2026-07-30)

Waves G0–G6 lab closed; G7 partner ⏸. Unit suite PASS (sample):

- `services/comms/internal/httpauth`
- mail: api, repo, audit, internalapi, caladapter, ews, carddav, activesync, autodiscover
- ui/mail (incl. `TestProxySendForwardsBearer`)
- mail-core `smtp_tls_e2e` (552)
- calendar caldav/ical

## Closed (Scaffold ✅)

| Wave | Highlights |
|------|------------|
| G0 | AuthZ REST/internal/connect/mig/MM/chat/AI; explicit `mode=` |
| G1 | `ERA_MAIL_STORE=memory` opt-in; PG default; policy→SMTP 552; `ERA_MAIL_AUDIT_REQUIRE`; restart PASS (`comms-restart-persist-20260730-022601.log`) |
| G2 | OIDC JWT path + BFF Bearer forward; Drive deny AC-C5 |
| G3 | Gov lab checklist + units PASS (field Outlook/iOS open) |
| G4/G5 | Upsell + Chat/Conf/AI Scaffold ✅ |
| G6 | Staging RT-01…08 PASS (`comms-pilot-staging.log` 2026-07-30); AuthZ stack |
| G7 | [`comms-rt09-skip.md`](comms-rt09-skip.md) — no `ga` |

## Compose honesty knobs

`ERA_MAIL_AUDIT_REQUIRE=1`, `ERA_INTERNAL_TOKEN`, `ERA_IDENTITY_DEV=1`, `ERA_MAIL_DEV=1` (lab only).

## Still partner-gated

- RT-09 customer SignOff
- Edition `ga`
- IceWarp MM / Exchange Bridge field
- Outlook/iOS field Pilot-ready

See [`docs/Comms-Honesty-Green-Spec.md`](../docs/Comms-Honesty-Green-Spec.md) · Matrix.
