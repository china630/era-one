# PRD — ERA Resolve (MVP)

**Статус:** MVP code  
**ADR:** [0031](../adr/0031-era-resolve-and-perimeter-editions.md)  
**Спека:** [Resolve-Spec.md](../Resolve-Spec.md)  
**Slogan:** ONE QUERY. ONE POLICY. ONE VERDICT.

## Цель

DNS Detection & Response (DNSSense-class): Guard / Trace / Atlas в `services/resolve`, license `resolve`.

## Scope MVP

| Pillar | Must |
|--------|------|
| Guard | Policy domain→allow/nxdomain/sinkhole; HTTP verdict API; DNS UDP/TCP lab listener |
| Trace | DnsEvent Envelope per query/verdict; ring buffer + optional Kafka/ingest |
| Atlas | Offline domain pack load + match → deny |

## Out of scope

DoH/DoT prod, agent DNS collector, live commercial TI, Guacamole-class UI, full DNSSense feature parity.

## Acceptance

| ID | Criterion | Proof |
|----|-----------|-------|
| AC-R1 | DNS query → NXDOMAIN/sinkhole per policy | golden + smoke |
| AC-R2 | Trace emits DnsEvent | unit |
| AC-R3 | Atlas pack hit → deny | golden pack |
| AC-R4 | ModuleResolve gates service | unit |
