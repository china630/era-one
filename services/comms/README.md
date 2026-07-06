# ERA Communications — service placeholder



Продуктовое семейство **ERA Communications** (roadmap). Backend MVP — по [`PRD-Comms-MVP.md`](../../docs/products/PRD-Comms-MVP.md).



## Документы



- Vision: [`docs/products/ERA-Communications-Vision.md`](../../docs/products/ERA-Communications-Vision.md)

- ADR: [`docs/adr/0027-era-communications-architecture.md`](../../docs/adr/0027-era-communications-architecture.md)

- Издания: [`editions-comms.yaml`](../../editions-comms.yaml)

- Deploy: [`deploy/profiles/comms.yaml`](../../deploy/profiles/comms.yaml)



## Планируемая структура



```

services/comms/

├── mail/           # Rust core + Go API

├── mail-connect/   # ERA Mail Connect (BFF → IMAP/JMAP)

├── calendar/

├── chat/

├── vcs/            # LiveKit adapter

└── ai/             # ERA Comms AI

```



Shared platform (ADR-0025): identity, tenant, Drive API, workspace (`app.customer.local/mail`).



**Standalone** — не требует ERA Core.


