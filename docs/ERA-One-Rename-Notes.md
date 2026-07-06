# ERA One — Rename Notes (L-06)

Документ описывает косметический переименование репозитория `era-xdr` → `era-one` **без** выполнения git rename в этом change-set.

## Зачем

- ADR-0014: продуктовое имя **ERA One** vs технический путь `era-xdr`.
- Helm chart, Docker images и дистрибутив уже используют префикс `era-one/`.

## Что менять при rename

| Область | Было | Станет |
|---|---|---|
| Корневая папка | `era-xdr` | `era-one` |
| Go module path | `era/...` | оставить `era/...` (внутренний модуль) или `era-one/...` по решению ADR |
| Docker compose project | `era-one-prod` | без изменений |
| Helm release | `era-one` | без изменений |
| CI paths | `d:\...\era-xdr` | обновить в runner config |

## Порядок (когда будет выполняться)

1. Заморозить мержи на 1 окно.
2. `git mv` не требуется для GitHub remote — достаточно rename в GitHub UI + `git remote set-url`.
3. Обновить локальные clone paths у команды.
4. Прогнать `scripts/ci-gates-stage10.ps1` и `scripts/helm-template-check.ps1`.
5. Обновить ссылки в `docs/`, `deploy/prod/README.md`, `.cursor/rules/` (пути в примерах).

## Что НЕ менять

- Kafka topics `xdr.*` — wire-контракт (ADR-0001).
- Proto package `era.v1`.
- Переменные `ERA_*` — стабильный операционный контракт.

## Проверка после rename

```powershell
docker compose -f deploy/docker-compose.prod.yml config --quiet
./scripts/helm-template-check.ps1
go test ./...
cargo test --workspace
```

## Статус

- [ ] Git rename — отложено (этот документ — guide only).
- [x] Helm/Docker image prefix `era-one` — уже в использовании.
