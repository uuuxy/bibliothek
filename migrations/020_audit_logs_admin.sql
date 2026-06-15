CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id UUID REFERENCES benutzer(id) ON DELETE SET NULL,
    aktion VARCHAR(255) NOT NULL,
    details JSONB NOT NULL DEFAULT '{}',
    ip_adresse VARCHAR(45),
    zeitstempel TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO schema_migrations (version) VALUES ('020_audit_logs_admin.sql') ON CONFLICT DO NOTHING;
