# ERA One — публичный сайт

Многостраничный статический сайт бренда **ERA One** и продуктовых семейств.
Vanilla HTML/CSS/JS — без внешних CDN/шрифтов/скриптов (air-gap friendly).

Канонический хост: **https://www.era-one.solutions**

## Сборка и деплой

```bash
./scripts/build-site.sh          # → dist/site/
# или
./scripts/build-site.ps1
```

Сборка: pricing SSOT → тесты калькулятора → UTF-8 gate → copy `site/` →
**`scripts/site_seo_enrich.py`** (robots, sitemap, prerender, schema) →
`site/test/check_seo_artifacts.py`.

CI: [`.github/workflows/site-deploy.yml`](../.github/workflows/site-deploy.yml)

- push в `main` (пути `site/**`, `scripts/build-site.*`, `scripts/site_seo_enrich.py`, …)
  → artifact → ветка **`site-prod`** → DigitalOcean App Platform (`region: fra`).
- Staging: push в `dev` / workflow_dispatch `staging`.

После выката на прод см. **Post-deploy: Google Search Console** ниже.

## Страницы

| Файл / URL | Назначение |
|---|---|
| `index.html` | Главная |
| `control.html` · `/control/` | ERA Control + datasheet (prerender) + калькулятор |
| `communications.html` · `/communications/` | ERA Communications |
| `office.html` · `/office/` | ERA Office |
| `editions/<slug>.html` | Модуль (статический prerender EN) |
| `edition.html?id=` | Редирект на `editions/<slug>.html` (совместимость) |
| `about.html` / `vision.html` / `contacts.html` | Company |
| `compare.html` | Head-to-head индекс |
| `downloads.html` | Trial-загрузки |
| `login.html` / `register.html` | noindex |
| `404.html` | error_document на DO |
| `robots.txt` / `sitemap.xml` | генерируются при сборке |

## SEO (сборка)

Enrich ([`scripts/site_seo_enrich.py`](../scripts/site_seo_enrich.py)) добавляет в `dist/site/`:

- `robots.txt`, `sitemap.xml`, `favicon.svg`, `assets/og-default.png`
- canonical, Open Graph, Twitter Card, JSON-LD (`Organization`, `WebSite`, `SoftwareApplication`)
- prerender EN-datasheet в family pages и `editions/*.html`
- `noindex` на сырых `datasheets/**` (дубли)
- stubs `/control/`, `/communications/`, `/office/` для ссылок без `.html`

Канон модулей: `/editions/<slug>.html` (см. `assets/products-catalog.js` → `moduleHref`).

## Навигация

- **Products** — мега-меню: три линейки, модули, Compare / Downloads.
- **Company** — About, Vision, Contacts, Compare, Downloads, Partners, Careers.
- Шапка/футер инжектятся `assets/site.js` (корневые пути `/assets/…`, `/index.html`).

## Datasheets

`site/datasheets/{lang}/` — ru / en / tr / ar. В UI смена языка → client `fetch`
(EN уже в HTML для краулеров). PDF — печатная версия datasheet.

## Логотип

В шапке/футере: `/assets/era-one-logo.svg`. OG: `assets/og-default.png`
(banner из distributor assets, если доступен при сборке).

## Локальный просмотр

```bash
./scripts/build-site.sh
cd dist/site && python -m http.server 8080
# http://localhost:8080
```

(корневые `/assets/…` требуют HTTP-сервер из корня dist, не `file://`).

## Post-deploy: Google Search Console

После зелёного **Site deploy** на `main`:

1. Smoke: `https://www.era-one.solutions/robots.txt`, `/sitemap.xml`,
   `/control.html` (View Source: H1 + prerendered body, не только `Loading…`),
   `/editions/era-core.html`, `/favicon.svg`, `/404` на несуществующий путь.
2. В DO Domains: предпочтительный **www**; apex → www (301).
   Служебный `*.ondigitalocean.app` не рекламировать (лучше редирект или noindex).
3. GSC: подтвердить property `https://www.era-one.solutions`
   (DNS TXT или файл `google*.html` положить в `site/` → уедет следующим деплоем).
4. GSC → Sitemaps → отправить `https://www.era-one.solutions/sitemap.xml`.
5. URL Inspection → **Request indexing** для:
   `/`, `/control.html`, `/communications.html`, `/office.html`,
   `/about.html`, `/contacts.html`, `/compare.html`,
   `/editions/era-core.html` (+ 1–2 ключевых edition).

CI не вызывает Google Indexing API — пункт 5 только вручную.
