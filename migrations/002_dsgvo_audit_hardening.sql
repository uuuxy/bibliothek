-- ============================================================
-- Migration: DSGVO-Revision & Audit-Log-Härtung
-- Zweck: Append-Only-Sicherung, JSONB-Details, Schüler-Audit
-- Datum: 2026-05-30
-- ============================================================

-- 1. Audit-Log: details-Spalte für maschinenlesbare JSONB-Metadaten hinzufügen
--    (snapshot des gelöschten Datensatzes, Begründung, Betrag etc.)
ALTER TABLE audit_log
  ADD COLUMN IF NOT EXISTS details JSONB DEFAULT NULL;

-- 2. Audit-Log: akteur-Typ unterscheiden (SYSTEM = Cronjob, USER = Mitarbeiter)
ALTER TABLE audit_log
  ADD COLUMN IF NOT EXISTS akteur VARCHAR(10) NOT NULL DEFAULT 'USER'
    CHECK (akteur IN ('USER', 'SYSTEM'));

-- 3. Datensatz-ID als TEXT (war UUID NOT NULL – Cronjob braucht Freitext-IDs
--    für Batch-Aktionen wie "GDPR_BATCH_2026-05-30").
--    Wir fügen eine neue Textspalte hinzu und belassen datensatz_id für FK-Kompatibilität.
ALTER TABLE audit_log
  ADD COLUMN IF NOT EXISTS kontext TEXT DEFAULT NULL;

-- 4. Append-Only-Sicherung: Entzug der DELETE- und UPDATE-Rechte für den
--    Anwendungsbenutzer (bibliothek_app). Passen Sie den Rollennamen ggf. an.
--    Superuser-Rechte bleiben für Backup/Migration erhalten.
--    WICHTIG: Ausführen als Superuser (postgres) nach Anpassung des Rollennamens.
--
--    DO $$
--    BEGIN
--      IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'bibliothek_app') THEN
--        REVOKE UPDATE, DELETE ON TABLE audit_log FROM bibliothek_app;
--      END IF;
--    END $$;
--
--    Alternativ (PostgreSQL 15+): Row Security Policy
--    ALTER TABLE audit_log ENABLE ROW LEVEL SECURITY;
--    CREATE POLICY audit_log_insert_only ON audit_log
--      FOR INSERT WITH CHECK (true);
--    CREATE POLICY audit_log_select_only ON audit_log
--      FOR SELECT USING (true);

-- 5. Performance-Index für Cronjob-Abfragen (Abgänger-Löschung)
CREATE INDEX IF NOT EXISTS idx_schueler_abgaenger_dsgvo
  ON schueler (abgaenger_jahr, ist_abgaenger)
  WHERE ist_abgaenger = true;

-- 6. Performance-Index für Audit-Log-Abfragen nach Aktion/Tabelle
CREATE INDEX IF NOT EXISTS idx_audit_log_tabelle_aktion
  ON audit_log (tabelle, aktion, timestamp DESC);

-- 7. Schadensfaelle: storniert_am Spalte für Stornierungsaudit
ALTER TABLE schadensfaelle
  ADD COLUMN IF NOT EXISTS storniert_am TIMESTAMP WITH TIME ZONE DEFAULT NULL;
ALTER TABLE schadensfaelle
  ADD COLUMN IF NOT EXISTS storniert_von UUID REFERENCES benutzer(id) ON DELETE SET NULL DEFAULT NULL;
ALTER TABLE schadensfaelle
  ADD COLUMN IF NOT EXISTS stornierungsgrund TEXT DEFAULT NULL;
