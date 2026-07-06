# ERA One — сайт (статический прототип)

Многостраничный статический сайт бренда **ERA One** и продуктовых семейств.
Собран без сборщика (vanilla HTML/CSS/JS) — работает из файловой системы и в
air-gap-контуре, без внешних CDN/шрифтов/скриптов.

## Страницы

| Файл | Назначение |
|---|---|
| `index.html` | Главная (hero, 3 продукта, «Why», About/Vision-тизеры) |
| `control.html` | ERA Control — издания + калькулятор |
| `communications.html` | ERA Communications (roadmap) — издания + калькулятор |
| `office.html` | ERA Office (roadmap) — издания + калькулятор |
| `about.html` | О компании |
| `vision.html` | Видение |
| `contacts.html` | Контакты + форма (прототип, не отправляет) |
| `compare.html` | Head-to-head по семействам (Control / Communications / Office) |
| `downloads.html` | Trial-загрузки (регистрация с корп. email) |
| `login.html` | Вход в единый admin-портал (прототип) |
| `legacy-portal.html` | Прежний портал-калькулятор (ADR-0021), сохранён |

## Навигация

- **Products** — мега-меню: три линейки, модули, ссылки **Compare** и **Downloads** per family.
- **Company** — About, Vision, Contacts, **Compare** (вкладки по семействам), Downloads, Partners, Careers.
- **Log in** — кнопка в шапке ведёт на `login.html`.
- Статусы **GA / Roadmap на сайте не показываются** (управляются в манифестах).

## Страницы продуктов и модулей

Контент datasheet **не открывается как отдельный сайт** — он встраивается в оболочку ERA One
(шапка, футер, меню, i18n):

| URL | Что показывает |
|---|---|
| `control.html` | ERA Control — datasheet + **калькулятор (modal)** + H2H |
| `communications.html` | ERA Communications — datasheet + **калькулятор (modal)** + H2H |
| `office.html` | ERA Office — datasheet + **калькулятор (modal)** + H2H |
| `edition.html?id=era-core` | Страница модуля (ERA Core) — datasheet модуля + PDF |

- **Download PDF** открывает печатную версию из `site/datasheets/` (A4, «Сохранить как PDF»).
- Каталог модулей и slug → datasheet: `assets/products-catalog.js`.
- Загрузка контента: `assets/datasheet-view.js` (fetch + inject `.body` из HTML).

### Зеркало datasheets

Структура `site/datasheets/{lang}/` — **ru** (канон) и **en** (перевод; TR/AR пока fallback на EN).

```powershell
$src = "docs/distributor/datasheets"
$ru  = "site/datasheets/ru"
$en  = "site/datasheets/en"
Copy-Item "$src/*.html" $ru -Force
# EN — отдельные файлы в site/datasheets/en/ (или regenerate)
Get-ChildItem $ru -Filter *.html | ForEach-Object {
  (Get-Content $_.FullName -Raw) -replace 'href="../assets/', 'href="../assets/' |
    Set-Content $_.FullName -NoNewline
}
```

Сайт подгружает `datasheets/{язык}/{файл}.html` с fallback: выбранный → EN → RU.
При смене языка в шапке контент перезагружается (`era-lang-changed`).

### Head-to-head (Compare)

- **Company → Compare** — индекс `compare.html`
- **ERA Control** — блок «Head-to-head» со ссылкой на сравнения
- Контент: `site/compare/{lang}/ERA-vs-*.html` (зеркало `docs/distributor/head-to-head/`)

## Общие ассеты

- `assets/site.css` — единая дизайн-система.
- `assets/i18n-data.js` — словарь локализации (EN/RU/TR/AR, EN — фолбэк).
- `assets/site.js` — общая шапка/футер (инъекция), переключатель языка,
  поиск, рендер изданий и калькуляторов из каталога `PRODUCTS`.

Шапка и футер не дублируются в разметке: каждая страница содержит
`<header id="site-header">` и `<footer id="site-footer">`, которые заполняет
`site.js`. Активный пункт меню — по атрибуту `data-page` у `<body>`.

## Слоганы

- **ERA One** — `ONE ECOSYSTEM. ONE PERIMETER. ONE VENDOR.`
- **ERA Control** — `ONE AGENT. ONE PLATFORM. ONE CONTROL.`
- **ERA Communications** — `ONE IDENTITY. ONE PLATFORM. ONE CONVERSATION.`
- **ERA Office** — `ONE WORKSPACE. ONE PLATFORM. ONE TEAM.`

## Логотип

Страницы ссылаются на `assets/era-one-logo.png`. Пока файла нет, показывается
векторный плейсхолдер `assets/era-one-logo.svg` (авто-подмена через `onerror`).
Положите настоящий PNG (для шапки — версию **без** зашитого слогана) по пути
`assets/era-one-logo.png` — он подхватится автоматически.

## Калькуляторы

У каждого продукта — свой калькулятор (`data-calc` в разметке, логика в
`site.js`). Ставки в `RATES` — **демо-значения, не оферта**; финальная цена
фиксируется в КП. Реальные ставки подключим из `ERA-Pricing` / SSOT позже.

## Локальный просмотр

```bash
cd site
python -m http.server 8080   # http://localhost:8080
```
