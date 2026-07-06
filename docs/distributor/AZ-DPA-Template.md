# AZ — DPA Template (Sovereign Hybrid)

**Шаблон соглашения об обработке данных** для дистрибьютора/заказчика в Азербайджане.  
Не юридический совет — согласовать с местным counsel перед подписанием.

Связано: [ADR-0018](../adr/0018-hybrid-connected-operating-model.md) · [AZ Data Flow](AZ-Data-Flow.md)

---

## AZ (Azərbaycan)

**Mövzü:** ERA One Sovereign Hybrid — müştəri konturunda data plane, vendor control plane.

**Tərəflər:** Müştəri (Data Controller) · Vendor / Distribyutor (Processor, yalnız metadata)

**Data plane (müştəri DC):** xam hadisələr, ClickHouse/MinIO, LLM, case-lər, PII — **heç vaxt** vendora çıxmır.

**Control plane (opt-in `connected`):** yalnız lease, CRL, imzalanmış kontent paketləri, Health səviyyə A (agent sayı, versiya, coverage).

**Öhdəliklər:**
- Vendor PII və xam telemetriya qəbul etmir (Managed View RBAC).
- Şifrələmə: TLS 1.2+ / mTLS Relay↔Portal.
- Saxlama: health metadata ≤ 90 gün (konfiqurasiya).
- Subprocessor: yalnız ERA Cloud Portal + Update Service (vendor infrastruktur).

**İmza:** _________________ Tarix: _________

---

## RU (Русский)

**Предмет:** ERA One Sovereign Hybrid — data plane в ЦОД заказчика, control plane вендора.

**Стороны:** Заказчик (оператор ПДн) · Вендор/дистрибьютор (обработчик метаданных)

**Data plane (контур AZ):** сырые события, lake, кейсы, PII — **никогда** не передаются вендору.

**Control plane (opt-in `connected`):** lease, CRL, подписанные бандлы контента, health уровня A.

**Обязательства:**
- Вендор не получает PII и сырую телеметрию (RBAC Managed View).
- Шифрование: TLS 1.2+ / mTLS Relay↔Portal.
- Retention метаданных health: до 90 дней (настраивается).
- Субобработчики: только ERA Cloud Portal и Update Service.

**Подпись:** _________________ Дата: _________

---

## EN (English)

**Subject:** ERA One Sovereign Hybrid — customer-hosted data plane, vendor control plane.

**Parties:** Customer (data controller) · Vendor/distributor (metadata processor)

**Data plane (customer DC):** raw events, lake, cases, PII — **never** transmitted to vendor.

**Control plane (opt-in `connected`):** lease, CRL, signed content bundles, Health level A only.

**Obligations:**
- Vendor shall not receive PII or raw telemetry (Managed View RBAC).
- Encryption: TLS 1.2+ / mTLS Relay↔Portal.
- Health metadata retention: up to 90 days (configurable).
- Subprocessors: ERA Cloud Portal and Update Service only.

**Signature:** _________________ Date: _________
