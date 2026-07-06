#!/usr/bin/env python3
"""Генерация HTML датащитов ERA One (RU) с оригинальным логотипом PNG.

Запуск:
  python scripts/generate_distributor_docs.py
  python scripts/html_to_pdf.py --all
"""
from __future__ import annotations

import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DIST = ROOT / "docs" / "distributor"
ASSETS = DIST / "assets"
DS = DIST / "datasheets"
H2H = DIST / "head-to-head"

LOGO_MAIN = "assets/era-one-logo-banner.png"
LOGO_REL = "../assets/era-one-logo-banner.png"

def footer(logo: str) -> str:
    return f"""
  <div class="ftr">
    <div class="ftr-contact">
      <div class="col"><b>Head Office</b>Geneva, Switzerland</div>
      <div class="col"><b>Engineering Center</b>Warsaw, Poland</div>
      <div class="col"><b>Contact</b>
        <a href="https://www.era-one.solutions">www.era-one.solutions</a><br>
        <a href="mailto:sales@era-one.solutions">sales@era-one.solutions</a><br>
        <a href="mailto:support@era-one.solutions">support@era-one.solutions</a>
      </div>
    </div>
    <div class="ftr-brand">
      <img class="logo-ftr" src="{logo}" alt="ERA One">
    </div>
  </div>
"""


def shell(
    title: str,
    body: str,
    logo: str,
    css_href: str,
    *,
    body_class: str = "",
) -> str:
    bc = f' class="{body_class}"' if body_class else ""
    return f"""<!doctype html>
<html lang="ru">
<head>
<meta charset="utf-8">
<title>{title}</title>
<link rel="stylesheet" href="{css_href}">
</head>
<body{bc}>
<div class="page page-stretch">
  <div class="hdr">
    <img class="logo-full" src="{logo}" alt="ERA One">
  </div>
{body}
{footer(logo)}
</div>
</body>
</html>
"""


def product_page(
    title: str,
    slug: str,
    subtitle: str,
    lead: str,
    sections: list[tuple[str, str]],
) -> str:
    parts = [
        f'  <div class="body">',
        f'    <h1>{title}<span class="sub">{subtitle}</span></h1>',
        f'    <p class="lead">{lead}</p>',
    ]
    for h, content in sections:
        parts.append(f"    <h2>{h}</h2>")
        parts.append(content)
    parts.append("  </div>")
    return shell(
        f"ERA One — {title}",
        "\n".join(parts),
        LOGO_REL,
        "../assets/datasheet-common.css",
    )


PRODUCTS: list[dict] = [
    {
        "slug": "ERA-Core",
        "title": "ERA Core (XDR)",
        "subtitle": "Фундамент платформы безопасности",
        "lead": (
            "Базовое издание XDR: один лёгкий агент на хостах, приём телеметрии, озеро данных "
            "в вашем ЦОД, детекция по правилам Sigma с привязкой к MITRE ATT&CK, учёт активов "
            "и управление кейсами в SOC-портале. Все данные остаются в контуре — без облака и phone-home. "
            "Это отправная точка любой поставки ERA One."
        ),
        "sections": [
            ("Назначение", """<p>Собрать полноценный цикл «событие → детекция → кейс → расследование» в изолированном периметре. Подходит банкам, госсектору и промышленности с требованием on-prem / air-gap.</p>
            <ul class="feats">
<li>Сквозной pipeline: агент → приём → очередь → хранилище → детекция → кейсы.</li>
<li>Единый формат событий; персональные данные маскируются на агенте до отправки.</li>
<li>Офлайн-лицензирование, без передачи телеметрии наружу.</li>
</ul>"""),
            ("Ключевые возможности", """<ul class="feats">
<li>Агент Windows / Linux / macOS — процессы, сеть, входы в систему.</li>
<li>Библиотека правил детекции с привязкой к MITRE ATT&CK; расширяемый корпус правил.</li>
<li>SOC Portal: кейсы, активы, события, детекции; роли analyst / admin / viewer.</li>
<li>Встроенная работа с индикаторами компрометации (IoC) и обогащение событий.</li>
<li>Неизменяемое хранение для расследований и регуляторной отчётности.</li>
</ul>"""),
            ("Для кого", "<p>Минимальная поставка для запуска SOC «с нуля» или замены иностранного XDR в закрытом контуре. Часто дополняется изданиями ERA AI и ERA Response.</p>"),
            ("Режимы развёртывания", """<p>Платформа поддерживает два профиля — без смены продукта, только настройки:</p>
<ul class="feats">
<li><b>Sovereign (air-gap):</b> полная изоляция; обновления правил, CVE и ИИ-паков приходят офлайн-пакетом на носителе.</li>
<li><b>Sovereign Hybrid:</b> данные, озеро и расследования остаются в контуре, а лицензии и обновления защиты подтягиваются автоматически из облака ERA по <b>исходящему</b> каналу.</li>
</ul>
<p>Гибридный режим включается осознанно и управляется политикой: сырьё, персональные данные и кейсы наружу не передаются. Идеально для банка по модели «данные дома — сопровождение как в облаке».</p>"""),
            ("Лицензирование", "<p>Базовая лицензия <b>ERA Core</b> — обязательна для любого bundle. Включает платформу телеметрии, детекции и SOC-портал.</p>"),
        ],
    },
    {
        "slug": "ERA-AI",
        "title": "ERA AI",
        "subtitle": "ИИ-аналитик SOC",
        "lead": (
            "Автоматизированное расследование инцидентов: цепочка связанных событий (storyline), "
            "вердикт, уровень уверенности и текстовое резюме для аналитика с привязкой к MITRE. "
            "Языковая модель работает <b>внутри контура</b> заказчика — без OpenAI, Copilot и иных облачных API. "
            "Сокращает время triage и снижает порог входа для SOC."
        ),
        "sections": [
            ("Назначение", "<p>Дать SOC «вторую пару глаз»: ИИ собирает контекст по хосту и инциденту, "
            "формулирует гипотезу и помогает аналитику принять решение. Данные не покидают периметр.</p>"),
            ("Ключевые возможности", """<ul class="feats">
<li>Расследование по хосту/инциденту: storyline, verdict, confidence, narrative.</li>
<li>Локальная LLM (Ollama / vLLM) — полный контроль над моделью и данными.</li>
<li>Привязка выводов к техникам MITRE ATT&CK.</li>
<li>Интеграция с кейсами и событиями ERA Core.</li>
</ul>"""),
            ("Для кого", "<p>Организации с дефицитом senior-аналитиков, банки с требованием on-prem AI, "
            "пилоты «ИИ в SOC» без экспорта данных.</p>"),
            ("Лицензирование", "<p>Отдельная лицензия <b>ERA AI</b>. Рекомендуемый старт: Core + AI + Response.</p>"),
        ],
    },
    {
        "slug": "ERA-Response",
        "title": "ERA Response (SOAR)",
        "subtitle": "Автоматизация реагирования",
        "lead": (
            "Закрывает цикл «обнаружили → отреагировали»: готовые сценарии изоляции хоста, "
            "блокировки IP, создания тикета в сервис-деске и журнал всех действий. "
            "Не требует отдельного SOAR-вендора — работает в том же контуре, что и XDR."
        ),
        "sections": [
            ("Назначение", "<p>Убрать ручное копирование команд между консолями при инциденте. "
            "Аналитик запускает playbook — платформа выполняет согласованные шаги и фиксирует результат.</p>"),
            ("Ключевые возможности", """<ul class="feats">
<li>Playbooks: изоляция хоста, блокировка IP, создание тикета.</li>
<li>Журнал действий реагирования для аудита.</li>
<li>Коннекторы: скрипт изоляции, webhook ITSM, интеграция с NGFW (напр. Palo Alto).</li>
<li>Запуск из кейса или по API — в рамках политик RBAC.</li>
</ul>"""),
            ("Для кого", "<p>SOC, которому нужен встроенный SOAR без отдельной лицензии XSOAR. Типовой апсейл к Core + AI.</p>"),
            ("Лицензирование", "<p>Отдельная лицензия <b>ERA Response</b>.</p>"),
        ],
    },
    {
        "slug": "ERA-Vuln",
        "title": "ERA Vuln",
        "subtitle": "Управление уязвимостями",
        "lead": (
            "Сканер уязвимостей в контуре: сверка установленного ПО с базами CVE, "
            "расписания сканов и проверка с учётными данными (credentialed). "
            "Результаты связываются с инвентарём активов ERA Core и ложатся в основу приоритизации риска (ERA Exposure)."
        ),
        "sections": [
            ("Назначение", "<p>Закрыть требования VM / CTEM в air-gap без облачных сканеров. "
            "Даёт единую картину «что уязвимо» по парку хостов с агентом ERA.</p>"),
            ("Ключевые возможности", """<ul class="feats">
<li>Сопоставление установленного ПО с базами CVE.</li>
<li>Расписания и политики сканирования.</li>
<li>Credentialed-скан для глубокой инвентаризации ПО.</li>
<li>Связь с учётом активов и отчётами для CISO.</li>
</ul>"""),
            ("Для кого", "<p>Заказчики с тендерным требованием VM, банки и промышленность с отдельным контуром уязвимостей.</p>"),
            ("Лицензирование", "<p>Опциональная лицензия <b>ERA Vuln</b>; часто добавляется к SecOps-bundle.</p>"),
        ],
    },
    {
        "slug": "ERA-Federated-National",
        "title": "ERA Federated / National",
        "subtitle": "Обмен IoC и федерация",
        "lead": (
            "Обмен индикаторами компрометации между подразделениями и организациями по стандартам "
            "STIX/TAXII — без выноса сырой телеметрии в зарубежное облако. "
            "Подходит госструктурам, холдингам с несколькими изолированными контурами и отраслевым CERT."
        ),
        "sections": [
            ("Назначение", "<p>Синхронизировать знания об угрозах между доверенными контурами, "
            "сохраняя суверенность каждого участника.</p>"),
            ("Ключевые возможности", """<ul class="feats">
<li><b>Federated:</b> обмен сигналами и моделями между зонами организации с защитой приватности.</li>
<li><b>National:</b> хаб IoC, приём и рассылка индикаторов по STIX/TAXII.</li>
<li>Включается только по лицензии — базовая поставка Core не требует этих модулей.</li>
</ul>"""),
            ("Для кого", "<p>Госсектор, банковские холдинги, критическая инфраструктура с несколькими ЦОД.</p>"),
            ("Лицензирование", "<p>Опциональные лицензии <b>ERA Federated</b> и <b>ERA National</b>.</p>"),
        ],
    },
    {
        "slug": "ERA-Workbench",
        "title": "ERA Workbench",
        "subtitle": "Единый timeline расследования",
        "lead": (
            "Один экран инцидента: события с хостов, учётных записей, сети и почтовых логов "
            "собраны в единую хронологию — без переключения между SIEM, EDR и отдельными консолями. "
            "Аналитик видит цепочку атаки целиком и быстрее принимает решение."
        ),
        "sections": [
            ("Назначение", "<p>Устранить разрыв между «много данных» и «удобное расследование». "
            "Workbench — рабочее место аналитика, а не ещё один отчёт.</p>"),
            ("Ключевые возможности", """<ul class="feats">
<li>Единый timeline по кейсу: endpoint, identity, network, email.</li>
<li>Связка алертов, событий и активов на одном экране SOC Portal.</li>
<li>Те же роли и политики доступа, что и для кейсов инцидентов.</li>
</ul>"""),
            ("Для кого", "<p>SOC-аналитики и threat hunters в организациях с несколькими источниками телеметрии.</p>"),
            ("Интеграция с платформой", "<p>Расширение <b>ERA Core</b> и SOC Portal; дополняет расследование <b>ERA AI</b>. "
            "Не требует отдельного агента — использует уже собранные данные платформы.</p>"),
        ],
    },
    {
        "slug": "ERA-Exposure",
        "title": "ERA Exposure",
        "subtitle": "Управление экспозицией и риском",
        "lead": (
            "Единый показатель риска для каждого актива: учитываются уязвимости, критичность хоста, "
            "активные детекции и сигналы конфигурации. Помогает CISO отвечать на вопрос «что чинить первым» — "
            "не по сырому CVSS, а по реальной экспозиции в вашем контуре."
        ),
        "sections": [
            ("Назначение", "<p>Снизить шум алертов и сфокусировать remediation на том, что действительно опасно "
            "для бизнеса. Особенно важно при парках от тысяч хостов.</p>"),
            ("Ключевые возможности", """<ul class="feats">
<li>Risk score актива на основе уязвимостей, детектов и критичности.</li>
<li>Отчёт «топ рисковых хостов» для руководства и VM-команды.</li>
<li>Связка с ERA Vuln и событиями безопасности ERA Core.</li>
</ul>"""),
            ("Для кого", "<p>CISO, команды VM/CTEM, банки с требованием измеримого снижения риска.</p>"),
            ("Интеграция с платформой", "<p>Работает поверх <b>ERA Core</b> и <b>ERA Vuln</b>: без сканера уязвимостей "
            "и телеметрии XDR картина экспозиции будет неполной.</p>"),
        ],
    },
    {
        "slug": "ERA-BYO-EDR",
        "title": "ERA BYO-EDR Hub",
        "subtitle": "Приём телеметрии сторонних EDR",
        "lead": (
            "На этапе миграции или в multi-vendor среде — приём событий от уже установленных endpoint-решений "
            "в единое озеро ERA и корреляция в Workbench. Не нужно «выбросить всё и начать с нуля»."
        ),
        "sections": [
            ("Назначение", "<p>Сохранить инвестиции в существующий EDR и постепенно консолидировать SOC "
            "вокруг ERA как единого «мозга» в контуре.</p>"),
            ("Ключевые возможности", """<ul class="feats">
<li>Адаптеры приёма: API, syslog, стандартизированный JSON.</li>
<li>Нормализация в единый формат событий платформы.</li>
<li>Обогащение индикаторами и отображение в Workbench.</li>
</ul>"""),
            ("Для кого", "<p>Холдинги, поэтапная миграция с иностранного EDR, гибридные контуры.</p>"),
            ("Лицензирование", "<p>Базовый адаптер — в составе <b>ERA Core</b>; дополнительные коннекторы — по опции.</p>"),
        ],
    },
    {
        "slug": "ERA-Manage",
        "title": "ERA Manage (UEM)",
        "subtitle": "IT-Ops и управление парком",
        "lead": (
            "Управление парком из одного контура: инвентаризация и CMDB, финансовый ITAM "
            "(контракты, лицензии ПО, TCO), развёртывание и патчи, контроль приложений и USB, "
            "BitLocker и защита серверов до выхода патча ОС (Virtual Patching). "
            "Один агент ERA — без отдельного тяжёлого UEM-клиента на каждую функцию."
        ),
        "sections": [
            ("Назначение", "<p>Заменить «зоопарк» ManageEngine/Ivanti в air-gap: IT и безопасность "
            "из одной платформы и одного агента.</p>"),
            ("Ключевые возможности", """<ul class="feats">
<li>ITAM/CMDB: железо, ПО, серийники, владельцы активов.</li>
<li>Финансовый ITAM: контракты, лицензии, стоимость владения.</li>
<li>Развёртывание и патчи через локальное зеркало в контуре.</li>
<li>Application Control и блокировка USB-устройств по политике.</li>
<li>Централизованное управление BitLocker и ключами восстановления.</li>
<li>Virtual Patching — снижение риска до установки патча ОС.</li>
</ul>"""),
            ("Для кого", "<p>Организации с тысячами рабочих мест, гос и банки с требованием UEM в закрытом контуре.</p>"),
            ("Лицензирование", "<p>Отдельная лицензия <b>ERA Manage</b>; часто в bundle ERA IT-Ops вместе с Service и Provision.</p>"),
        ],
    },
    {
        "slug": "ERA-Service",
        "title": "ERA Service (ITSM)",
        "subtitle": "Сервис-деск в контуре",
        "lead": (
            "Сервис-деск в вашем ЦОД: заявки и инциденты, портал самообслуживания, SLA "
            "и привязка к активам из CMDB. Тикет из SOAR ERA Response может сразу попасть "
            "в тот же сервис-деск — без интеграционных «костылей»."
        ),
        "sections": [
            ("Назначение", "<p>Связать SecOps и IT-Ops: пользователь, ИТ и безопасность работают в одной экосистеме.</p>"),
            ("Ключевые возможности", """<ul class="feats">
<li>Заявки и инциденты, портал самообслуживания.</li>
<li>SLA и эскалации.</li>
<li>Привязка тикетов к активам ERA Manage.</li>
<li>Расширяемая модель данных под зрелый ITSM.</li>
</ul>"""),
            ("Для кого", "<p>Заказчики без отдельного ITSM в контуре или с требованием единого локального вендора.</p>"),
            ("Лицензирование", "<p>Отдельная лицензия <b>ERA Service</b>.</p>"),
        ],
    },
    {
        "slug": "ERA-Provision",
        "title": "ERA Provision",
        "subtitle": "Развёртывание ОС на bare-metal",
        "lead": (
            "Развёртывание операционной системы на «голое железо»: сеть PXE, образы, "
            "автоматические сценарии установки и регистрация агента ERA сразу после инсталляции. "
            "Репозиторий образов — локально, в air-gap."
        ),
        "sections": [
            ("Назначение", "<p>Закрыть цикл «железо → ОС → агент → управление» без облачных imaging-сервисов.</p>"),
            ("Ключевые возможности", """<ul class="feats">
<li>Provisioning bare-metal (PXE и аналоги).</li>
<li>Локальное хранилище образов Windows / Linux.</li>
<li>Автоматическая регистрация агента после установки ОС.</li>
</ul>"""),
            ("Для кого", "<p>Дата-центры, филиалы, проекты развёртывания нового парка.</p>"),
            ("Лицензирование", "<p>Отдельная лицензия <b>ERA Provision</b>.</p>"),
        ],
    },
    {
        "slug": "ERA-PAM",
        "title": "ERA PAM",
        "subtitle": "Привилегированный доступ",
        "lead": (
            "Корпоративный сейф паролей и секретов администраторов, выдача доступа по регламенту, "
            "безопасный SSH/RDP-прокси и запись привилегированных сессий. "
            "Отдельное издание для контуров, где критичен контроль admin-доступа (аналог Password Manager Pro)."
        ),
        "sections": [
            ("Назначение", "<p>Исключить «общие пароли админов» и неучтённый доступ к серверам. "
            "Все действия привилегированных пользователей прозрачны для аудита.</p>"),
            ("Ключевые возможности", """<ul class="feats">
<li>Корпоративный vault паролей и ключей.</li>
<li>Выдача креденшелов по регламенту (checkout).</li>
<li>SSH/RDP через прокси с записью сессии.</li>
<li>Готовность к интеграции с HSM в требовательных контурах.</li>
</ul>"""),
            ("Для кого", "<p>Банки, госсектор, операторы КИИ с политикой привилегированного доступа.</p>"),
            ("Лицензирование", "<p>Отдельная лицензия <b>ERA PAM</b> — самостоятельное издание, не входит в базовый XDR-bundle.</p>"),
        ],
    },
    {
        "slug": "ERA-Observe",
        "title": "ERA Observe",
        "subtitle": "Мониторинг сети (agentless)",
        "lead": (
            "Мониторинг сети и инфраструктуры без агента на каждом устройстве: discovery, SNMP, NetFlow. "
            "Закрывает слепую зону «коммутатор упал — а XDR молчит». "
            "Можно начать с интеграции PRTG/Zabbix, стратегически — полноценный модуль Observe в контуре."
        ),
        "sections": [
            ("Назначение", "<p>Дать SOC и NOC общую картину: сбой сети и подозрительный процесс на хосте "
            "видны в одной платформе.</p>"),
            ("Ключевые возможности", """<ul class="feats">
<li>Опрос SNMP, обнаружение устройств, приём NetFlow.</li>
<li>Корреляция сетевых алертов с событиями endpoint.</li>
<li>Объединение MAC/IP/hostname с инвентарём ERA.</li>
</ul>"""),
            ("Для кого", "<p>NOC+SOC, промышленные сети, стратегия ERA Unified / Sovereign Stack.</p>"),
            ("Лицензирование", "<p>Отдельная лицензия <b>ERA Observe</b>; на старте возможна интеграция с уже установленным NMS.</p>"),
        ],
    },
]


# Порядок изданий — как в ERA-Product-Line.md §1
PRODUCT_ORDER = [
    "ERA-Core",
    "ERA-AI",
    "ERA-Response",
    "ERA-Vuln",
    "ERA-Federated-National",
    "ERA-Workbench",
    "ERA-Exposure",
    "ERA-BYO-EDR",
    "ERA-Manage",
    "ERA-Service",
    "ERA-Provision",
    "ERA-PAM",
    "ERA-Observe",
]

PRODUCTS_BY_SLUG = {p["slug"]: p for p in PRODUCTS}

CARD_OVERVIEW: dict[str, tuple[str, str]] = {
    "ERA-Core": ("ERA Core (XDR)", "Телеметрия, озеро данных, Sigma + MITRE, активы, кейсы, SOC-портал."),
    "ERA-AI": ("ERA AI", "ИИ-расследование: storyline, вердикт. LLM внутри контура."),
    "ERA-Response": ("ERA Response", "SOAR: изоляция хоста, блок IP, тикет ITSM."),
    "ERA-Vuln": ("ERA Vuln", "CVE-скан, расписания, credentialed-скан."),
    "ERA-Federated-National": ("ERA Federated / National", "STIX/TAXII, обмен IoC между контурами."),
    "ERA-Workbench": ("ERA Workbench", "Единый timeline: endpoint + identity + network + email."),
    "ERA-Exposure": ("ERA Exposure", "Локальный CREM: risk score актива в контуре."),
    "ERA-BYO-EDR": ("ERA BYO-EDR Hub", "Телеметрия сторонних EDR в единое озеро."),
    "ERA-Manage": ("ERA Manage", "UEM: ITAM, патчи, App/USB Control, BitLocker, Virtual Patching."),
    "ERA-Service": ("ERA Service", "ITSM-lite: сервис-деск, SLA, CMDB."),
    "ERA-Provision": ("ERA Provision", "PXE/imaging bare-metal, авто-регистрация агента."),
    "ERA-PAM": ("ERA PAM", "Vault, SSH/RDP-прокси, запись сессий."),
    "ERA-Observe": ("ERA Observe", "SNMP, NetFlow, discovery (agentless)."),
}


def product_filename(index: int, slug: str) -> str:
    return f"{index:02d}-{slug}"


def main_overview() -> str:
    cards = [CARD_OVERVIEW[s] for s in PRODUCT_ORDER]

    def grid(items):
        rows = []
        for nm, ds in items:
            rows.append(
                f'      <div class="card"><div class="nm">{nm}</div>'
                f'<div class="ds">{ds}</div></div>'
            )
        return '<div class="grid">\n' + "\n".join(rows) + "\n    </div>"

    page1 = f"""  <div class="body">
    <h1>ERA One — единая платформа безопасности и IT-Ops</h1>
    <p class="lead">Суверенная платформа, объединяющая XDR и управление ИТ-инфраструктурой в одном
      лёгком агенте и одной консоли. Работает полностью в контуре заказчика —
      <b>on-premise / air-gap</b>, без зарубежного облака. Один агент, одна платформа,
      один центр управления вместо «зоопарка» из 5–6 систем и подписок.</p>

    <h2>Издания платформы</h2>
    {grid(cards)}
  </div>"""

    page2 = f"""  <div class="body">
    <h2>Инженерное наследие</h2>
    <div class="heritage">
      <div class="origin"><div class="o-nm">Promisec</div><div class="o-tag">Infrastructure Intelligence</div><div class="o-ds">Непрерывный аудит конфигураций и контроль комплаенса — основа управления конечными точками.</div></div>
      <div class="origin"><div class="o-nm">Sirin Labs</div><div class="o-tag">Secure-by-Design</div><div class="o-ds">Принцип изоляции угроз, защиты каналов и безопасности на уровне ядра архитектуры.</div></div>
      <div class="origin"><div class="o-nm">Syrinx Systems</div><div class="o-tag">High-Performance Telemetry</div><div class="o-ds">Сверхлёгкий сбор телеметрии и сетевой мониторинг без нагрузки на сеть.</div></div>
    </div>

    <h2>Архитектура платформы</h2>
    <div class="flow">
      <span class="node k">ERA One</span><span class="arr">&rsaquo;</span>
      <span class="node">Издания (XDR / Manage / Service / PAM …)</span><span class="arr">&rsaquo;</span>
      <span class="node">Сервер управления + Data Lake</span><span class="arr">&rsaquo;</span>
      <span class="node">Локальный репозиторий (MinIO)</span><span class="arr">&rsaquo;</span>
      <span class="node k">One Agent</span><span class="arr">&rsaquo;</span>
      <span class="node">Windows / Linux / macOS / серверы</span><span class="arr">&rsaquo;</span>
      <span class="node">Веб-консоль</span>
    </div>

    <h2>Модель развёртывания</h2>
    <p class="lead" style="margin-bottom:6px">Одна платформа — два профиля развёртывания. Данные, озеро,
      ИИ и расследования <b>всегда остаются в вашем контуре</b>; наружу — только лицензии,
      обновления защиты и служебная телеметрия сопровождения (и только если вы это разрешили).</p>
    <table class="cmp">
      <tr><th>Критерий</th><th>ERA Sovereign (air-gap)</th><th>ERA Sovereign Hybrid</th></tr>
      <tr><td>Данные, озеро, ИИ, кейсы</td><td class="win">В контуре заказчика</td><td class="win">В контуре заказчика</td></tr>
      <tr><td>Обновления правил / CVE / ИИ-паков</td><td>Офлайн-пакетом (носитель)</td><td class="win">Автоматически из облака ERA</td></tr>
      <tr><td>Лицензии и сопровождение</td><td>Офлайн-ключ</td><td>Онлайн-подписка + офлайн-резерв</td></tr>
      <tr><td>Связь наружу</td><td>Нет (полная изоляция)</td><td>Только исходящая, по политике</td></tr>
      <tr><td>Кому</td><td>Госсектор, КИИ, строгий air-gap</td><td>Банк: «данные дома, сопровождение — как в облаке»</td></tr>
    </table>
    <p class="note-int">Гибридный режим включается осознанно и настраивается: сырьё, персональные
      данные и материалы расследований не покидают периметр ни в одном режиме.</p>

    <h2>Преимущества для enterprise</h2>
    <div class="benefits">
      <div>Один лёгкий агент (CPU &lt; 2%, RAM &lt; 150 МБ) и единая консоль.</div>
      <div>UEM и XDR — один организм без интеграционных разрывов.</div>
      <div>On-premise и air-gap: данные не покидают контур.</div>
      <div>Непрерывные обновления защиты — офлайн-пакетом или из облака ERA (гибрид).</div>
      <div>Офлайн-лицензия Ed25519; PII-редакция на агенте.</div>
      <div>Масштаб до десятков тысяч узлов в кластере.</div>
      <div>Модульные лицензии — платите за используемое.</div>
      <div>AD/LDAP, REST API, SIEM, STIX/TAXII, интеграция NGFW.</div>
    </div>
  </div>"""

    return f"""<!doctype html>
<html lang="ru">
<head>
<meta charset="utf-8">
<title>ERA One — Data Sheet</title>
<link rel="stylesheet" href="assets/datasheet-common.css">
</head>
<body>
<div class="page">
  <div class="hdr">
    <img class="logo-full" src="{LOGO_MAIN}" alt="ERA One">
  </div>
{page1}
</div>
<div class="page page-stretch">
{page2}
{footer(LOGO_MAIN)}
</div>
</body>
</html>
"""


def h2h_page(competitor: str, slug: str, intro: str, rows: list[tuple]) -> str:
    thead = f"""<tr><th>Функция / слой</th><th>{competitor}</th><th>ERA One (полная линейка)</th></tr>"""
    tbody = []
    for func, them, us in rows:
        tbody.append(f"<tr><td>{func}</td><td>{them}</td><td>{us}</td></tr>")
    table = f'<table class="cmp">{thead}{"".join(tbody)}</table>'

    body = f"""  <div class="body">
    <h1>ERA One vs {competitor}</h1>
    <p class="lead">{intro}</p>
    {table}
    <div class="summary">
      <b>Итог:</b> ERA One выигрывает в <b>суверенном on-prem / air-gap</b>, едином агенте
      SecOps+IT-Ops и встроенном SOAR/AI без облака. С моделью <b>Sovereign Hybrid</b> мы
      закрываем и операционный аргумент конкурентов: данные и расследования остаются у
      заказчика, а обновления защиты, лицензии и сопровождение доставляются из облака ERA —
      без выноса сырья и PII. {competitor} сильнее в зрелости отдельных ниш и глобальной
      экосистеме; ERA — локальная альтернатива «зоопарку подписок» с современной эксплуатацией.
    </div>
  </div>"""
    return shell(f"ERA One vs {competitor}", body, LOGO_REL, "../assets/datasheet-common.css")


H2H_DATA = {
    "ERA-vs-TrendMicro": (
        "Trend Micro (Vision One + Apex One)",
        "Сравнение полной линейки ERA One с экосистемой Trend Micro. "
        "Фокус на том, что заказчик может развернуть в своём контуре — и на нашем ответе про гибрид.",
        [
            ("Модель поставки", "Cloud-first SaaS (data lake в облаке TM)", '<span class="win">On-prem / air-gap + Sovereign Hybrid</span>'),
            ("Гибридная модель", "Cloud обязателен для полноты", '<span class="win">Данные дома, обновления/ops из облака ERA</span>'),
            ("Обновления защиты", "Непрерывно из облака", '<span class="win">Из облака ERA (гибрид) или офлайн-пакет (air-gap)</span>'),
            ("Managed / MSSP", "Vision One MDR (облако)", "ERA Managed View — пульт партнёра без доступа к сырью"),
            ("Endpoint XDR", "Apex One + Vision One correlation", "ERA Core + Workbench"),
            ("AI-расследование", "TrendAI Companion (облако)", '<span class="win">ERA AI — LLM в контуре</span>'),
            ("SOAR / response", "Ограничено, MDR как сервис", '<span class="win">ERA Response встроен</span>'),
            ("VM / exposure", "CREM/ASRM (облачная аналитика)", "ERA Vuln + ERA Exposure"),
            ("Серверы / virtual patch", "Deep Security, сильная линейка", "ERA Manage + Virtual Patching"),
            ("Email / cloud CNAPP", "Сильные нативные модули", '<span class="gap">Collectors / логи, не полный gateway</span>'),
            ("NDR / сеть", "Deep Discovery", "ERA Observe + NDR"),
            ("Sandbox", "DDAN", '<span class="gap">Интеграция, своего нет</span>'),
            ("Нац. обмен IoC", "Глобальный threat cloud", '<span class="win">ERA National STIX/TAXII в контуре</span>'),
            ("UEM / ITSM / PAM", "Отдельные продукты", '<span class="win">Manage + Service + PAM в одной платформе</span>'),
            ("Сторонний EDR", "Интеграция (SentinelOne и др.)", "ERA BYO-EDR Hub"),
        ],
    ),
    "ERA-vs-ManageEngine": (
        "ManageEngine (Endpoint Central + PMP + PAM360)",
        "ManageEngine — типичный «зоопарк» модулей с отдельными подписками. ERA One — один агент и модульные серверные издания в контуре, с опцией Sovereign Hybrid.",
        [
            ("Модель", "Несколько агентов/продуктов, облачные опции", '<span class="win">Один агент + модульные издания</span>'),
            ("Развёртывание", "Cloud / on-prem вразнобой по продуктам", '<span class="win">Единый air-gap или Sovereign Hybrid</span>'),
            ("Обновления / сопровождение", "Облачные сервисы ME", '<span class="win">Облако ERA (гибрид) или офлайн, единый канал</span>'),
            ("UEM / инвентарь", "Endpoint Central — зрелый", "ERA Manage"),
            ("Патчи / deploy", "Patch Manager Plus", "ERA Manage"),
            ("App / USB control", "Отдельные лицензии", "ERA Manage"),
            ("BitLocker", "Endpoint Central", "ERA Manage"),
            ("PAM", "PAM360 — отдельный продукт", "ERA PAM"),
            ("XDR / SOC", '<span class="gap">Слабо / партнёры</span>', '<span class="win">ERA Core + AI + Response</span>'),
            ("Детекция MITRE", "Ограничено", '<span class="win">Sigma + curated + MITRE</span>'),
            ("Air-gap", "Частично", '<span class="win">Штатно, без phone-home</span>'),
            ("VM", "Vulnerability Manager", "ERA Vuln"),
            ("ITSM", "ServiceDesk Plus — отдельно", "ERA Service"),
            ("OS Provisioning", "Ограничено", "ERA Provision"),
        ],
    ),
    "ERA-vs-Ivanti": (
        "Ivanti (Neurons / UEM / Security)",
        "Ivanti объединяет endpoint security и IT-Ops в облачно-ориентированной модели (Neurons). ERA — суверенный аналог unified stack для закрытого контура, с опцией Sovereign Hybrid.",
        [
            ("Модель", "Cloud Neurons + on-prem опции", '<span class="win">Sovereign on-prem first + Hybrid</span>'),
            ("Гибрид / облако", "Neurons в облаке — источник функций", '<span class="win">Данные дома, ops из облака ERA (opt-in)</span>'),
            ("Обновления защиты", "Из облака Ivanti", '<span class="win">Облако ERA (гибрид) или офлайн (air-gap)</span>'),
            ("UEM", "Зрелый unified endpoint", "ERA Manage + Provision"),
            ("Device Control USB", "Есть", "ERA Manage"),
            ("Financial ITAM", "Есть", "ERA Manage"),
            ("ITSM", "Service Manager", "ERA Service"),
            ("XDR / EDR", "Ivanti Security / партнёры", '<span class="win">Нативный ERA Core + AI</span>'),
            ("SOAR", "Ограничено", '<span class="win">ERA Response</span>'),
            ("MDM mobile", "Сильно", '<span class="gap">Вне scope (ADR-0016)</span>'),
            ("VPN / ZTNA", "Ivanti Connect Secure", '<span class="gap">Интеграция only</span>'),
            ("PAM", "Частично в портфеле", "ERA PAM"),
            ("Сеть / Observe", "Партнёры", "ERA Observe"),
            ("Air-gap гос", "Ограничено", '<span class="win">Киллер-сценарий ERA</span>'),
        ],
    ),
}


def status_badge(kind: str, label: str) -> str:
    return f'<span class="badge {kind}">{label}</span>'


EDITION_STATUS: dict[str, tuple[str, str]] = {
    "ERA-Core": ("ga", "GA"),
    "ERA-AI": ("ga", "GA"),
    "ERA-Response": ("ga", "GA"),
    "ERA-Vuln": ("opt", "GA-опция"),
    "ERA-Federated-National": ("opt", "GA-опция"),
    "ERA-Workbench": ("road", "Roadmap"),
    "ERA-Exposure": ("road", "Roadmap"),
    "ERA-BYO-EDR": ("road", "Roadmap"),
    "ERA-Manage": ("road", "Roadmap"),
    "ERA-Service": ("road", "Roadmap"),
    "ERA-Provision": ("road", "Roadmap"),
    "ERA-PAM": ("road", "Roadmap"),
    "ERA-Observe": ("road", "Roadmap"),
}

EDITION_TYPE: dict[str, str] = {
    "ERA-Core": "Фундамент",
    "ERA-AI": "Апсейл",
    "ERA-Response": "Апсейл",
    "ERA-Vuln": "Опция",
    "ERA-Federated-National": "Опция гос/холдинг",
    "ERA-Workbench": "Усиление Core",
    "ERA-Exposure": "Усиление Vuln/Core",
    "ERA-BYO-EDR": "Миграция / гибрид",
    "ERA-Manage": "IT-Ops",
    "ERA-Service": "IT-Ops",
    "ERA-Provision": "IT-Ops",
    "ERA-PAM": "PAM",
    "ERA-Observe": "Сеть",
}

ROADMAP_ROWS = [
    ("Workbench", "ERA Core", "3–5 нед", "—"),
    ("Exposure", "ERA Exposure", "4–6 нед", "—"),
    ("BYO-EDR adapters", "era-collectors", "2–4 нед / адаптер", "—"),
    ("CMDB + финансовый ITAM", "ERA Manage", "2–4 нед", "—"),
    ("Application Control", "ERA Manage", "2–4 нед", "подпись драйвера + security-review"),
    ("Device Control (USB)", "ERA Manage", "2–4 нед", "подпись драйвера + security-review"),
    ("Virtual Patching", "ERA Manage", "3–5 нед", "подпись драйвера + security-review"),
    ("BitLocker mgmt", "ERA Manage", "1–2 нед", "хранение ключей"),
    ("Развёртывание ПО / патчи", "ERA Manage", "3–6 нед", "пилот rollout"),
    ("ITSM-lite", "ERA Service", "3–6 нед", "—"),
    ("OS Provisioning", "ERA Provision", "3–6 нед", "пилот rollout"),
    ("PAM", "ERA PAM", "4–8 нед", "крипто-аудит (vault/HSM)"),
    ("Network monitoring", "ERA Observe", "4–8 нед (или PRTG 1–2 нед)", "—"),
    ("Sovereign Hybrid MVP (Hybrid-0)", "ERA Core + Portal", "4–8 нед", "ops (Portal) + DPA/схема потоков AZ"),
]

BUNDLE_ROWS = [
    ("Старт SecOps", "ERA Core + ERA AI + ERA Response", "Greenfield SOC, банк, гос"),
    ("+ Уязвимости", "+ ERA Vuln", "Требование VM/CTEM"),
    ("ERA IT-Ops", "ERA Core + Manage + Service + Provision", "Замена legacy-UEM в контуре"),
    ("ERA Unified / Sovereign Stack", "XDR (Core+AI+Response) + Manage + Service + Observe", "Единый локальный вендор IT + Security"),
    ("+ PAM", "+ ERA PAM", "Корпоративный сейф паролей админов"),
]

ME_ROWS = [
    ("Vulnerability Management", "ERA Vuln", "GA-опция"),
    ("Базовый UEM", "ERA Manage", "Roadmap"),
    ("Application Control", "ERA Manage", "Roadmap"),
    ("BitLocker", "ERA Manage", "Roadmap"),
    ("Доп. SOC-аналитика", "ERA AI + Response", "GA"),
    ("Password Manager Pro (PAM)", "ERA PAM", "Roadmap"),
]

# Модель развёртывания (ADR-0018). Статусы: Sovereign — GA; Hybrid — Roadmap; Cloud — вне scope.
DEPLOY_MODE_ROWS = [
    ("Данные / озеро / ИИ / кейсы", "В контуре заказчика", "В контуре заказчика", "У заказчика или private cloud"),
    ("Связь с вендором", "Нет (air-gap)", "Outbound-only через Relay", "Полная (SaaS)"),
    ("Лицензия", "Offline Ed25519", "Offline + lease", "Subscription + metering"),
    ("Обновления (Sigma/CVE/AI)", "Offline-пакет (носитель)", "Pull из ERA Update Service", "Управляется вендором"),
    ("Health / сопровождение", "Нет", "По policy (уровни A/B/C)", "Полное у вендора"),
    ("Целевой клиент", "Госсектор, КИИ, air-gap", "Банк: данные дома, ops у вендора", "Mid-market, филиалы"),
    ("Статус ERA", "GA (текущий фокус)", "Roadmap (Hybrid-0)", "Вне scope — по спросу"),
]

# Именованные компоненты Sovereign Hybrid (ADR-0018 §1.1 / §1.1.1).
HYBRID_COMPONENT_ROWS = [
    ("ERA Cloud Portal", "Вендор — ядро-сервис", "Лицензии/контракты, выпуск lease, CRL, приём health; зонтик control plane"),
    ("ERA Update Service", "Вендор — отдельный сервис", "Конвейер подписи + доставка контента (Sigma/CVE/коннекторы/AI-паки); работает и в air-gap (носитель)"),
    ("ERA Hybrid Relay", "Контур клиента — модуль control-plane", "Единственный outbound-only канал: lease/CRL/updates + health/opt-in TI; egress allowlist + audit"),
    ("ERA Managed View", "Вендор — модуль Portal (RBAC)", "Мульти-клиентский пульт для MSSP/партнёра: health, лицензии, версии — без доступа к сырью и кейсам"),
]


def product_line_page() -> str:
    edition_rows = []
    for i, slug in enumerate(PRODUCT_ORDER, start=1):
        nm, ds = CARD_OVERVIEW[slug]
        sk, sl = EDITION_STATUS[slug]
        edition_rows.append(
            f"<tr><td class=\"num\">{i:02d}</td>"
            f"<td><b>{nm}</b>{status_badge(sk, sl)}</td>"
            f"<td>{ds}</td><td>{EDITION_TYPE[slug]}</td></tr>"
        )
    editions_table = (
        '<table class="cmp"><tr>'
        "<th>№</th><th>Издание</th><th>Что даёт</th><th>Тип</th></tr>"
        + "".join(edition_rows)
        + "</table>"
    )

    bundle_rows = "".join(
        f"<tr><td><b>{b}</b></td><td>{c}</td><td>{k}</td></tr>"
        for b, c, k in BUNDLE_ROWS
    )
    bundles_table = (
        '<table class="cmp"><tr><th>Bundle</th><th>Состав</th><th>Кому</th></tr>'
        + bundle_rows
        + "</table>"
    )

    roadmap_rows = "".join(
        f"<tr><td>{a}</td><td>{e}</td><td>{c}</td><td>{g}</td></tr>"
        for a, e, c, g in ROADMAP_ROWS
    )
    roadmap_table = (
        '<table class="cmp"><tr>'
        "<th>Возможность</th><th>Издание</th><th>Код (AI-assisted)</th><th>Гейты</th></tr>"
        + roadmap_rows
        + "</table>"
    )

    me_rows = "".join(
        f"<tr><td>{p}</td><td><b>{a}</b></td><td>{st}</td></tr>"
        for p, a, st in ME_ROWS
    )
    me_table = (
        '<table class="cmp"><tr><th>Позиция запроса</th><th>Наш ответ</th><th>Статус</th></tr>'
        + me_rows
        + "</table>"
    )

    deploy_rows = "".join(
        f"<tr><td>{c}</td><td>{s}</td><td>{h}</td><td>{cl}</td></tr>"
        for c, s, h, cl in DEPLOY_MODE_ROWS
    )
    deploy_table = (
        '<table class="cmp"><tr>'
        "<th>Критерий</th><th>Sovereign (air-gap)</th><th>Sovereign Hybrid</th><th>Cloud (SaaS)</th></tr>"
        + deploy_rows
        + "</table>"
    )

    comp_rows = "".join(
        f"<tr><td><b>{n}</b></td><td>{w}</td><td>{r}</td></tr>"
        for n, w, r in HYBRID_COMPONENT_ROWS
    )
    comp_table = (
        '<table class="cmp"><tr><th>Компонент</th><th>Где / деплой</th><th>Роль</th></tr>'
        + comp_rows
        + "</table>"
    )

    body = f"""  <div class="body">
    <h1>ERA One — продуктовая линейка<span class="sub">Внутренний справочник presales · v1.5 · 1 июля 2026</span></h1>
    <p class="meta">Аудитория: отдел продаж, пресейл, дистрибьюторы. Клиентские датащиты — без статусов GA/Roadmap.</p>
    <p class="lead">Единая суверенная платформа: один лёгкий агент и серверные издания по лицензии.
      <b>On-prem / air-gap</b>, без зарубежного облака. Ниже — честная карта готовности изданий.</p>

    <div class="legend">
      {status_badge("ga", "GA")} в продукте сегодня &nbsp;
      {status_badge("opt", "GA-опция")} опция лицензии &nbsp;
      {status_badge("road", "Roadmap")} в дорожной карте (см. §4)
    </div>

    <h2>1. Карта изданий</h2>
    {editions_table}
    <p class="note-int"><b>Терминология:</b> ERA Core = база XDR; ERA Manage = IT-Ops. Разные издания, включаются по лицензии.</p>

    <h2>2. Готовые bundle</h2>
    {bundles_table}

    <h2>3. Ориентир по срокам (внутренний)</h2>
    {roadmap_table}
    <p class="note-int">Вне продукта (ADR-0016): MDM/Mobile UEM и VPN/ZTNA — зона интеграции, не наше издание.</p>

    <h2>4. Запрос «в стиле ManageEngine»</h2>
    {me_table}
    <p class="lead" style="margin-top:8px">Питч: единый агент и модули по лицензии — SecOps + IT-Ops из одной консоли в контуре заказчика.</p>

    <h2>5. Модель развёртывания и ответ на «а где ваш cloud?»</h2>
    <p class="lead" style="margin-bottom:6px">Возражение рынка: «без облака/гибрида не возьмут». Ответ — <b>Sovereign Hybrid</b>
      (ADR-0018): отделяем <b>данные</b> (всегда в контуре) от <b>эксплуатации</b> (обновления/лицензии/сопровождение — из облака ERA).
      Мы <b>не</b> строим SaaS-клон Cortex Cloud как ближайший шаг.</p>
    {deploy_table}
    <p class="note-int"><b>Инвариант:</b> сырьё, PII и кейсы не покидают контур ни в одном режиме. Наружу — только
      метаданные, обновления, лицензии и (opt-in) обезличенные индикаторы.</p>

    <h3>Именованные компоненты (ADR-0018 §1.1)</h3>
    {comp_table}
    <p class="note-int"><b>Гранулярность:</b> Update Service — отдельный сервис (свой конвейер подписи, dual-use pull+offline);
      Managed View — модуль Portal с RBAC (выносится при white-label / мультитенантности партнёров).
      <b>BYO-EDR ≠ hybrid:</b> BYO-EDR — чужой EDR в наш lake (ADR-0017); hybrid — канал к вендору ERA.</p>
    <p class="note-int"><b>AZ:</b> режим «hybrid minimal» (lease + updates + CRL, health A, TI off) + DPA + схема потоков;
      Portal region AZ/EU или self-hosted для госа.</p>

    <p class="note-int">Связанные материалы: ERA-One-DataSheet.pdf · datasheets/01–13 · head-to-head/ · ADR-0011, 0012, 0013, 0016, 0017, 0018</p>
  </div>"""

    return shell(
        "ERA One — Product Line (internal)",
        body,
        LOGO_MAIN,
        "assets/datasheet-common.css",
    )


def hybrid_flow() -> str:
    return """<div class="flow">
      <span class="node">Агенты</span><span class="arr">&rsaquo;</span>
      <span class="node k">Озеро · ИИ · Кейсы — в контуре заказчика</span><span class="arr">&rsaquo;</span>
      <span class="node">Relay (только исходящий)</span><span class="arr">&rsaquo;</span>
      <span class="node k">Облако ERA: лицензии · обновления · сопровождение</span>
    </div>"""


def hybrid_client_page() -> str:
    body = f"""  <div class="body">
    <h1>ERA Sovereign Hybrid<span class="sub">Данные дома — сопровождение как в облаке</span></h1>
    <p class="lead">Современная операционная модель без потери суверенности: платформа, озеро данных,
      ИИ и материалы расследований <b>остаются в вашем контуре</b>, а обновления защиты, лицензии и
      служебная телеметрия сопровождения доставляются из облака ERA по <b>исходящему</b> каналу.
      Ответ на вопрос «нам нужен on-prem, но без облака мы отстаём в эксплуатации».</p>

    <h2>Как это работает</h2>
    {hybrid_flow()}
    <p class="note-int">Гибридный режим включается осознанно и настраивается политикой. Полностью
      изолированный режим (air-gap) остаётся доступным без каких-либо исходящих соединений.</p>

    <h2>Что остаётся в контуре, что уходит наружу</h2>
    <table class="cmp">
      <tr><th>Всегда в вашем контуре</th><th>Наружу (только по вашей политике)</th></tr>
      <tr><td>Сырые события и телеметрия</td><td>Лицензии и продление подписки</td></tr>
      <tr><td>Озеро данных (ClickHouse / MinIO)</td><td>Обновления правил, CVE, ИИ-паков</td></tr>
      <tr><td>ИИ-инференс (LLM в контуре)</td><td>Список отзыва (CRL)</td></tr>
      <tr><td>Кейсы, расследования, персональные данные</td><td>Служебная телеметрия сопровождения (по уровням)</td></tr>
      <tr><td class="win">Никогда не покидают периметр</td><td>Обезличенные индикаторы — только если вы разрешили</td></tr>
    </table>

    <h2>Компоненты решения</h2>
    <table class="cmp">
      <tr><th>Компонент</th><th>Где</th><th>Зачем</th></tr>
      <tr><td><b>ERA Cloud Portal</b></td><td>Облако ERA</td><td>Лицензии и контракты, продление подписки, список отзыва, приём статуса здоровья</td></tr>
      <tr><td><b>ERA Update Service</b></td><td>Облако ERA</td><td>Доставка подписанного контента: правила, CVE, коннекторы, ИИ-паки (работает и офлайн-пакетом)</td></tr>
      <tr><td><b>ERA Hybrid Relay</b></td><td>Ваш контур</td><td>Единственный исходящий канал наружу с журналированием и списком разрешённых адресов</td></tr>
      <tr><td><b>ERA Managed View</b></td><td>Облако ERA</td><td>Пульт вашего интегратора / MSSP: здоровье, лицензии, версии — <b>без доступа к вашим данным</b></td></tr>
    </table>

    <h2>Два профиля развёртывания</h2>
    <table class="cmp">
      <tr><th>Критерий</th><th>ERA Sovereign (air-gap)</th><th>ERA Sovereign Hybrid</th></tr>
      <tr><td>Данные / озеро / ИИ / кейсы</td><td class="win">В контуре</td><td class="win">В контуре</td></tr>
      <tr><td>Обновления защиты</td><td>Офлайн-пакетом</td><td class="win">Автоматически из облака ERA</td></tr>
      <tr><td>Связь наружу</td><td>Нет</td><td>Только исходящая, по политике</td></tr>
      <tr><td>Кому</td><td>Госсектор, КИИ, строгий air-gap</td><td>Банк, крупный бизнес: суверенность + удобная эксплуатация</td></tr>
    </table>

    <h2>Для службы безопасности и закупок</h2>
    <div class="benefits">
      <div>Прозрачная схема потоков: понятно, что и куда уходит.</div>
      <div>Персональные данные и материалы расследований наружу не передаются.</div>
      <div>Исходящий канал — один, с журналом и списком разрешённых адресов.</div>
      <div>Возможен региональный или размещённый в вашем контуре Portal.</div>
      <div>Соглашение об обработке данных (DPA) и схема потоков — в комплекте.</div>
      <div>Всегда можно перейти в полностью изолированный режим.</div>
    </div>
  </div>"""
    return shell(
        "ERA One — Sovereign Hybrid",
        body,
        LOGO_REL,
        "../assets/datasheet-common.css",
    )


def hybrid_internal_page() -> str:
    deploy_rows = "".join(
        f"<tr><td>{c}</td><td>{s}</td><td>{h}</td><td>{cl}</td></tr>"
        for c, s, h, cl in DEPLOY_MODE_ROWS
    )
    deploy_table = (
        '<table class="cmp"><tr>'
        "<th>Критерий</th><th>Sovereign (air-gap)</th><th>Sovereign Hybrid</th><th>Cloud (SaaS)</th></tr>"
        + deploy_rows
        + "</table>"
    )
    comp_rows = "".join(
        f"<tr><td><b>{n}</b></td><td>{w}</td><td>{r}</td></tr>"
        for n, w, r in HYBRID_COMPONENT_ROWS
    )
    comp_table = (
        '<table class="cmp"><tr><th>Компонент</th><th>Где / деплой</th><th>Роль</th></tr>'
        + comp_rows
        + "</table>"
    )
    optin_table = (
        '<table class="cmp"><tr><th>Уровень opt-in</th><th>Что наружу</th><th>Кому</th></tr>'
        "<tr><td><b>hybrid-base</b> (минимум)</td><td>lease + updates + CRL</td><td>почти все connected</td></tr>"
        "<tr><td>+ <b>health</b></td><td>эксплуатационные метрики (A/B/C)</td><td>наше / партнёрское сопровождение</td></tr>"
        "<tr><td>+ <b>TI-share</b></td><td>обезличенные IoC / metadata / FP</td><td>зрелые SOC</td></tr>"
        "</table>"
    )
    lease_table = (
        '<table class="cmp"><tr><th>Параметр</th><th>Старт</th><th>Где задаётся</th></tr>'
        "<tr><td>lease_period_days</td><td>30</td><td>лицензия / Portal / контракт</td></tr>"
        "<tr><td>lease_renewal_interval</td><td>24 ч</td><td>tenant policy</td></tr>"
        "<tr><td>grace_days</td><td>30</td><td>лицензия</td></tr>"
        "<tr><td>offline_max_days</td><td>90 → деградация</td><td>tenant policy</td></tr>"
        "<tr><td>degradation_mode</td><td>no_new_nodes + no_updates (детект работает)</td><td>политика</td></tr>"
        "</table>"
    )

    body = f"""  <div class="body">
    <h1>ERA Sovereign Hybrid — операционная модель<span class="sub">Внутренний справочник presales · v1.0 · 1 июля 2026</span></h1>
    <p class="meta">Аудитория: пресейл, дистрибьюторы, ИБ-архитекторы. Клиентский one-pager — <b>ERA-Sovereign-Hybrid.pdf</b> (без статусов/ADR).</p>
    <p class="lead">Ответ на протест «без cloud/hybrid не возьмут»: отделяем <b>data plane</b> (всегда в контуре)
      от <b>control plane</b> (обновления/лицензии/сопровождение — из облака ERA). <b>Не</b> строим SaaS-клон
      Cortex Cloud как ближайший шаг. Полная детализация — <b>ADR-0018</b>.</p>

    <div class="legend">
      {status_badge("ga", "GA")} Sovereign сегодня &nbsp;
      {status_badge("road", "Roadmap")} Sovereign Hybrid (MVP Hybrid-0) &nbsp;
      {status_badge("opt", "вне scope")} Cloud SaaS (по спросу)
    </div>

    <h2>1. Профили развёртывания</h2>
    {deploy_table}
    <p class="note-int"><b>Инвариант:</b> сырьё, PII и кейсы не покидают контур ни в одном режиме. Наружу —
      только метаданные, обновления, лицензии и (opt-in) обезличенные индикаторы.</p>

    <h2>2. Именованные компоненты (ADR-0018 §1.1)</h2>
    {comp_table}
    <p class="note-int"><b>Гранулярность (§1.1.1):</b> Update Service — отдельный сервис (свой конвейер подписи, dual-use pull+offline);
      Managed View — модуль Portal с RBAC (выносится при white-label / мультитенантности партнёров).</p>

    <h2>3. Уровни opt-in (что разрешено наружу)</h2>
    {optin_table}
    <p class="note-int">Health: A Minimal (default) · B Operational (MSSP) · C Support (break-glass, TTL). Сырьё/PII/кейсы — запрещены на любом уровне.</p>

    <h2>4. Лицензирование: lease поверх ADR-0010</h2>
    <p class="lead" style="margin-bottom:6px">Offline Ed25519 остаётся ядром проверки; lease — верхний слой для connected. Всё в настройках, без хардкода.</p>
    {lease_table}

    <h2>5. AZ и BYO-EDR</h2>
    <p class="lead" style="margin-top:0"><b>AZ:</b> режим «hybrid minimal» (lease + updates + CRL, health A, TI off) + DPA + схема потоков; Portal region AZ/EU или self-hosted для госа.
      <br><b>BYO-EDR ≠ hybrid:</b> BYO-EDR — чужой EDR в наш lake (ADR-0017); hybrid — канал к вендору ERA. Не смешивать в пресейле.</p>

    <p class="note-int">Связанные материалы: ERA-Sovereign-Hybrid.pdf (клиентский) · ERA-Product-Line.pdf §5 · ADR-0018 · ADR-0010 · ADR-0009 · ADR-0017 · editions.yaml</p>
  </div>"""
    return shell(
        "ERA One — Sovereign Hybrid (internal)",
        body,
        LOGO_MAIN,
        "assets/datasheet-common.css",
    )


def ensure_logo_banner() -> None:
    crop = ROOT / "scripts" / "crop_logo_banner.py"
    subprocess.run([sys.executable, str(crop)], check=True)


def main() -> None:
    ensure_logo_banner()
    DS.mkdir(parents=True, exist_ok=True)
    H2H.mkdir(parents=True, exist_ok=True)

    # Удалить старые датащиты без номера
    for pattern in ("ERA-*.html", "ERA-*.pdf"):
        for old in DS.glob(pattern):
            old.unlink()

    (DIST / "ERA-One-DataSheet.html").write_text(main_overview(), encoding="utf-8")
    print("OK:", DIST / "ERA-One-DataSheet.html")

    pl_path = DIST / "ERA-Product-Line.html"
    pl_path.write_text(product_line_page(), encoding="utf-8")
    print("OK:", pl_path)

    hyb_int = DIST / "ERA-Sovereign-Hybrid-Internal.html"
    hyb_int.write_text(hybrid_internal_page(), encoding="utf-8")
    print("OK:", hyb_int)

    for i, slug in enumerate(PRODUCT_ORDER, start=1):
        p = PRODUCTS_BY_SLUG[slug]
        html = product_page(
            p["title"], p["slug"],
            p["subtitle"], p["lead"], p["sections"],
        )
        base = product_filename(i, slug)
        path = DS / f"{base}.html"
        path.write_text(html, encoding="utf-8")
        print("OK:", path)

    hyb_client = DS / "14-ERA-Sovereign-Hybrid.html"
    hyb_client.write_text(hybrid_client_page(), encoding="utf-8")
    print("OK:", hyb_client)

    for slug, (comp, intro, rows) in H2H_DATA.items():
        html = h2h_page(comp, slug, intro, rows)
        path = H2H / f"{slug}.html"
        path.write_text(html, encoding="utf-8")
        print("OK:", path)


if __name__ == "__main__":
    main()
