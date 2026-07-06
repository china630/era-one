# SSO + SOC Portal (Wave GA-2 / pilot)

## Быстрый старт (lab)

```powershell
docker compose -f deploy/docker-compose.prod.yml --profile sso up -d
# Portal через proxy: http://localhost:8443/ui/portal/
```

Nginx проксирует на control-plane и добавляет:
- `X-ERA-Role` — analyst | admin | viewer (RBAC в control-plane)
- `X-ERA-Actor` — имя пользователя из IdP

Конфиг: [`deploy/nginx/soc-portal.conf`](../nginx/soc-portal.conf)

## Production (банк)

1. **IdP** (Keycloak / ADFS / LDAP+oauth2-proxy) — аутентификация пользователя.
2. **Reverse proxy** (nginx / Traefik) — после auth выставляет:
   - `X-ERA-Role` из группы AD/LDAP (SOC-Analyst → analyst)
   - `X-ERA-Actor` из email/samAccountName
3. **mTLS** на ingest (`ERA_TLS_*`) — отдельно от SSO UI.

Portal v2 читает роль из селектора (lab) или из заголовков, если proxy их пробрасывает.

## Проверка

```powershell
curl -H "X-ERA-Role: admin" -H "X-ERA-Actor: pilot@test" http://localhost:8090/api/v1/cases
```

Refs: S6-8, F-GA-7, Install-Guide-GA §6.
