// АВТОГЕНЕРАЦИЯ из pricing-data.yaml + pricing-comms/office-data.yaml
// Пересборка: python scripts/build_portal.py
window.ERA_PRICING = {
  "schema_version": "1.0",
  "edition": "2026-07",
  "currency": "EUR",
  "vat_included": false,
  "regions": {
    "eu": {
      "label": "EU / Global",
      "multiplier": 1.0
    },
    "cis": {
      "label": "СНГ",
      "multiplier": 0.5
    }
  },
  "server_multiplier": 3,
  "modules": {
    "core": {
      "title": "ERA Core",
      "desc": "XDR-база: телеметрия, детекция (Sigma+MITRE), кейсы, SOC-портал, Workbench",
      "unit": "endpoint",
      "eu_year": 12,
      "availability": "ga",
      "required": true
    },
    "control-ai": {
      "title": "ERA Control AI",
      "desc": "ИИ-аналитик SOC, расследование, LLM в контуре",
      "unit": "endpoint",
      "eu_year": 8,
      "availability": "ga",
      "flat_alt": [
        {
          "up_to": 1000,
          "eu_year": 18000
        },
        {
          "up_to": 5000,
          "eu_year": 45000
        },
        {
          "up_to": 15000,
          "eu_year": 90000
        },
        {
          "up_to": null,
          "eu_year": null
        }
      ]
    },
    "response": {
      "title": "ERA Response",
      "desc": "SOAR: изоляция хоста, блок IP, тикет",
      "unit": "endpoint",
      "eu_year": 4,
      "availability": "ga"
    },
    "vuln": {
      "title": "ERA Vuln",
      "desc": "Сканер уязвимостей (CVE), credentialed-скан",
      "unit": "endpoint",
      "eu_year": 4,
      "availability": "ga"
    },
    "exposure": {
      "title": "ERA Exposure",
      "desc": "Risk score актива (требует Core+Vuln)",
      "unit": "endpoint",
      "eu_year": 4,
      "availability": "project",
      "requires": [
        "core",
        "vuln"
      ]
    },
    "manage": {
      "title": "ERA Manage",
      "desc": "UEM: CMDB/ITAM, deploy/patch, BitLocker + App/USB Control + консольные пользователи (включены)",
      "unit": "endpoint",
      "eu_year": 12,
      "availability": "project"
    },
    "observe": {
      "title": "ERA Observe",
      "desc": "Мониторинг сети (SNMP/NetFlow/discovery)",
      "unit": "device",
      "eu_year": 6,
      "availability": "project"
    },
    "byo_edr": {
      "title": "ERA BYO-EDR Hub",
      "desc": "Приём телеметрии стороннего EDR",
      "unit": "source",
      "eu_year": 4,
      "availability": "project"
    },
    "service": {
      "title": "ERA Service",
      "desc": "ITSM: сервис-деск, SLA, портал (пользователи портала — бесплатно)",
      "unit": "technician",
      "eu_year": 900,
      "availability": "project"
    },
    "provision": {
      "title": "ERA Provision",
      "desc": "Развёртывание ОС (PXE/imaging)",
      "unit": "node",
      "eu_year": 8,
      "availability": "project"
    },
    "pam": {
      "title": "ERA PAM",
      "desc": "Сейф паролей, checkout, SSH-запись сессий",
      "unit": "admin",
      "eu_year": 50,
      "availability": "project",
      "addon": [
        {
          "key": "pam_target",
          "title": "ERA PAM — управляемая цель",
          "unit": "target",
          "eu_year": 30
        }
      ]
    },
    "federated": {
      "title": "ERA Federated",
      "desc": "Обмен IoC внутри организации",
      "unit": "site",
      "eu_year": 24000,
      "availability": "ga"
    },
    "national": {
      "title": "ERA National",
      "desc": "Межорг. хаб STIX/TAXII",
      "unit": "hub",
      "eu_year": 36000,
      "availability": "ga"
    }
  },
  "volume_discounts": [
    {
      "min": 1,
      "max": 250,
      "discount": 0.0
    },
    {
      "min": 251,
      "max": 1000,
      "discount": 0.1
    },
    {
      "min": 1001,
      "max": 5000,
      "discount": 0.2
    },
    {
      "min": 5001,
      "max": 10000,
      "discount": 0.3
    },
    {
      "min": 10001,
      "max": 25000,
      "discount": 0.4
    },
    {
      "min": 25001,
      "max": null,
      "discount": null
    }
  ],
  "term_discounts": [
    {
      "key": "1y",
      "label": "1 год",
      "discount": 0.0
    },
    {
      "key": "3y_annual",
      "label": "3 года, ежегодно",
      "discount": 0.1
    },
    {
      "key": "3y_prepaid",
      "label": "3 года, предоплата",
      "discount": 0.2
    },
    {
      "key": "5y_prepaid",
      "label": "5 лет, предоплата",
      "discount": 0.25
    }
  ],
  "bundles": [
    {
      "key": "secops",
      "title": "Старт SecOps",
      "modules": [
        "core",
        "control-ai",
        "response"
      ],
      "discount": 0.2
    },
    {
      "key": "secops_v",
      "title": "SecOps + уязвимости",
      "modules": [
        "core",
        "control-ai",
        "response",
        "vuln"
      ],
      "discount": 0.22
    },
    {
      "key": "itops",
      "title": "ERA IT-Ops",
      "modules": [
        "core",
        "manage",
        "service",
        "provision"
      ],
      "discount": 0.25
    },
    {
      "key": "unified",
      "title": "ERA Unified",
      "modules": [
        "core",
        "control-ai",
        "response",
        "manage",
        "service",
        "observe"
      ],
      "discount": 0.3
    },
    {
      "key": "full",
      "title": "Full",
      "modules": [
        "core",
        "control-ai",
        "response",
        "vuln",
        "manage",
        "service",
        "provision",
        "pam",
        "observe"
      ],
      "discount": 0.35
    }
  ],
  "perpetual": {
    "multiplier_of_annual": 3,
    "maintenance_rate": 0.2
  },
  "calc_formula": "total = sum(module_reg) * (1 - volume) * (1 - term); для Control AI берётся min(per-endpoint, flat)",
  "disclaimer": "Индикативный расчёт для региона СНГ; не публичная оферта. Итоговая цена — в КП.",
  "product_lines": {
    "communications": {
      "schema_version": "1.0",
      "product": "era-communications",
      "parent_pricing": "docs/distributor/pricing-data.yaml",
      "currency": "EUR",
      "unit": "user",
      "vat_included": false,
      "modules": {
        "comms-mail-server": {
          "title": "ERA Mail Server",
          "desc": "Почта + календарь (SMTP/IMAP, CalDAV, EWS subset), Autodiscover",
          "eu_year": 10,
          "availability": "roadmap",
          "included": [
            "comms-mail-client"
          ]
        },
        "comms-mail-connect": {
          "title": "ERA Mail Connect",
          "desc": "Migration: webmail + BFF к внешнему IMAP/JMAP/EWS (без native backend)",
          "eu_year": 4,
          "availability": "roadmap",
          "tier": "migration"
        },
        "comms-conference": {
          "title": "ERA Conference",
          "desc": "Видеоконференции on-prem (LiveKit)",
          "eu_year": 6,
          "availability": "roadmap"
        },
        "comms-chat": {
          "title": "ERA Chat",
          "desc": "Корпоративный мессенджер",
          "eu_year": 6,
          "availability": "roadmap"
        },
        "comms-ai": {
          "title": "ERA Comms AI",
          "desc": "Air-Gap LLM: аудит почты, саммари",
          "eu_year": 8,
          "availability": "roadmap"
        }
      },
      "bundles": [
        {
          "key": "comms-full",
          "title": "ERA Communications Full Suite",
          "modules": [
            "comms-mail-server",
            "comms-conference",
            "comms-chat",
            "comms-ai"
          ],
          "discount": 0.21,
          "note": "EU à la carte 30 → ~23.7; CIS ~€9.5/user ≈ 19 AZN (тизер)"
        },
        {
          "key": "comms-mail-only",
          "title": "ERA Mail (Server + Client)",
          "modules": [
            "comms-mail-server"
          ],
          "discount": 0.0
        }
      ],
      "disclaimer": "Roadmap; индикатив. Не публичная оферта."
    },
    "office": {
      "schema_version": "1.0",
      "product": "era-office",
      "parent_pricing": "docs/distributor/pricing-data.yaml",
      "currency": "EUR",
      "unit": "user",
      "vat_included": false,
      "modules": {
        "platform-drive": {
          "title": "ERA Drive",
          "desc": "Файлы, sync, ACL, версии; API для Mail и Office",
          "eu_year": 4,
          "availability": "roadmap",
          "shared_edition": "era-drive",
          "note": "Отдельная лицензия; upsell к ERA Mail"
        },
        "office-documents": {
          "title": "ERA Documents",
          "desc": "Текстовые документы, co-editing on-prem (.era-doc, docx I/O)",
          "eu_year": 8,
          "availability": "roadmap",
          "requires": [
            "platform-drive"
          ]
        },
        "office-tables": {
          "title": "ERA Tables",
          "desc": "Таблицы, формулы, co-editing (.era-sheet, xlsx I/O)",
          "eu_year": 6,
          "availability": "roadmap",
          "requires": [
            "platform-drive"
          ]
        },
        "office-presentations": {
          "title": "ERA Presentations",
          "desc": "Презентации on-prem (.era-deck)",
          "eu_year": 5,
          "availability": "roadmap",
          "requires": [
            "platform-drive"
          ]
        },
        "office-projects": {
          "title": "ERA Projects",
          "desc": "Управление проектами (post-MVP)",
          "eu_year": 4,
          "availability": "roadmap",
          "requires": [
            "platform-drive"
          ]
        },
        "office-ai": {
          "title": "ERA Office AI",
          "desc": "Air-Gap LLM: саммари, assist по документам",
          "eu_year": 6,
          "availability": "roadmap"
        }
      },
      "bundles": [
        {
          "key": "office-mvp",
          "title": "ERA Office MVP",
          "modules": [
            "platform-drive",
            "office-documents"
          ],
          "discount": 0.1,
          "note": "P0–P1 pilot: Drive + Documents; à la carte 12 → 10.8 EU"
        },
        {
          "key": "office-suite",
          "title": "ERA Office Suite",
          "modules": [
            "platform-drive",
            "office-documents",
            "office-tables",
            "office-presentations"
          ],
          "discount": 0.25,
          "note": "Drive включён; à la carte 23 → 17.25 EU; CIS ~8.6"
        },
        {
          "key": "office-suite-ai",
          "title": "ERA Office Suite + AI",
          "modules": [
            "platform-drive",
            "office-documents",
            "office-tables",
            "office-presentations",
            "office-ai"
          ],
          "discount": 0.28,
          "note": "à la carte 29 → ~20.9 EU"
        }
      ],
      "disclaimer": "Roadmap; индикатив. Не публичная оферта."
    }
  }
};
