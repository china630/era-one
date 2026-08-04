# Comms lab IMAP (`dovecot-lab`)

Compose service hostname **`dovecot-lab`** (L-1). Implementation: ERA `lab-imap`
(RFC3501 memory server) — same client surface as Dovecot for Migration/Connect lab,
air-gap friendly (no Hub pull of full mailserver images).

| Item | Value |
|------|--------|
| Host (compose) | `dovecot-lab:143` |
| Host (dev) | `127.0.0.1:1144` → container 143 |
| User | any (LOGIN always OK); seed mailbox `lab1@mail.gov.az` |
| Seed | 1 message in INBOX |

Optional Dovecot conf snippets in this directory are for operators who swap the
image to `dovecot/dovecot`; default compose uses `Dockerfile.lab-imap`.
