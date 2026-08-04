-- Bootstrap demo tenant for Office pilot (Office-only or Office+Comms stand).
-- Idempotent: safe to run multiple times.

INSERT INTO era_platform.tenants (id, name, slug, status)
VALUES ('t-demo', 'Demo Tenant', 't-demo', 'active')
ON CONFLICT (id) DO NOTHING;

INSERT INTO era_platform.tenant_domains (id, tenant_id, fqdn, is_primary)
VALUES ('td-demo-primary', 't-demo', 'demo.office.local', true)
ON CONFLICT (id) DO NOTHING;

-- Comms schema (no-op if era_comms not created yet).
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name = 'era_comms') THEN
    INSERT INTO era_comms.tenants (id, name, slug)
    VALUES ('t-demo', 'Demo Tenant', 't-demo')
    ON CONFLICT (id) DO NOTHING;
  END IF;
END $$;
