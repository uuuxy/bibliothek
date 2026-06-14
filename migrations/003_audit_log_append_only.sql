-- ============================================================
-- Migration: 003_audit_log_append_only.sql
-- Zweck: Harter Append-Only Schutz via Trigger
-- Datum: 2026-06-05
-- ============================================================

-- Erstellt eine Funktion, die jeden UPDATE oder DELETE Versuch blockiert
CREATE OR REPLACE FUNCTION prevent_audit_log_modification()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'Das Audit-Log ist Append-Only. Änderungen oder Löschungen sind aus DSGVO/Revisions-Gründen strengstens untersagt.';
END;
$$ LANGUAGE plpgsql;

-- Bindet die Funktion als Trigger an die Tabelle audit_log
DROP TRIGGER IF EXISTS audit_log_append_only_trigger ON audit_log;
CREATE TRIGGER audit_log_append_only_trigger
BEFORE UPDATE OR DELETE ON audit_log
FOR EACH ROW EXECUTE FUNCTION prevent_audit_log_modification();
