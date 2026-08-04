# Deepen D2 — Webmail OIDC / PKCE (lab) — 2026-08-04

## Verdict

AC-C2 Scaffold ✅ **lab**: staging RT-05 OIDC token path + BFF Bearer forward + browser PKCE in `ui/mail/web/app.js`.  
Pilot field / customer IdP — still open.

## RT-05 (staging machine-flow)

Evidence: `reports/comms-pilot-staging.log` / `reports/comms-pilot-staging-honesty-20260730.log`

```
RT-05 webmail OIDC machine-flow
RT-05 OIDC send+list PASS
```

Script: `scripts/run-comms-pilot-staging.ps1` (identity token → ui-mail BFF → mail-api send+list).

## Browser PKCE path (`ui/mail/web/app.js`)

1. `startLogin()` — S256 `code_challenge` / `code_challenge_method=S256`, store `pkce_verifier` in `sessionStorage`.
2. Redirect: `{ERA_IDENTITY_URL}/oauth2/authorize?client_id=era-webmail&redirect_uri=…/mail/callback&response_type=code&…`
3. Callback `?code=` → `exchangeCode()` POSTs `/oauth2/token` with `code_verifier` + `grant_type=authorization_code`.
4. Store `access_token`; subsequent API calls use `Authorization: Bearer …`.

## BFF Bearer forward

`ui/mail/oidc_forward_test.go` — `TestProxySendForwardsBearer`: JWT accepted by BFF and forwarded as `Authorization: Bearer` to mail-api (no silent X-ERA-only path).

## Static PKCE gate

```
go test -C ui/mail -run 'StaticAppJS|ProxySendForwardsBearer' -count=1
```

`TestStaticAppJSContainsPKCE` asserts served `app.js` contains `code_challenge_method` / `code_verifier` / S256.

## Matrix

| AC-C2 | Scaffold | Pilot lab | Pilot field |
|-------|----------|-----------|-------------|
| after D2 | ✅ lab | [x] | [ ] |
