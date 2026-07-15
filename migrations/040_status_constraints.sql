-- =============================================================================
-- Migration 040: Wertemengen-Constraints für Status-Felder + grade_level-Untergrenze
-- =============================================================================
-- Schließt geschlossene Zustandsvokabulare auf DB-Ebene ab (Dimension „Lifecycle"):
--   * vormerkungen.status: 'wartend' → 'abholbereit' (erfüllte Vormerkungen werden
--     gelöscht, nicht umgesetzt — daher nur diese zwei Werte).
--   * buecher_exemplare.inventur_status: NULL (nicht in Inventur) / 'ausstehend' /
--     'erfasst'.
--   * buecher_titel.grade_level: 0–13. 0 = „noch nicht kategorisiert" (Sentinel,
--     siehe hintergrund_jobs), 5–13 = kooperative Gesamtschule inkl. Oberstufe.
--     Deckt sich mit der App-Validierung (endpunkte_buecher_schreiben/aktualisieren:
--     "gradeLevel muss zwischen 0 und 13 sein"). NULL bleibt erlaubt.
--
-- Bewusst NICHT hier: cover_status (inkonsistente Groß-/Kleinschreibung im Code,
-- braucht erst Bereinigung) und medientyp (offenes, per Formular frei eingebbares
-- Vokabular).
--
-- Wie 039: erst Verletzer auf einen sinnvollen Wert bringen, dann CHECK setzen
-- (idempotent per DO-Guard), damit die Migration auf Bestandsdaten nie fehlschlägt.
-- =============================================================================

-- 1. Verletzer bereinigen.
UPDATE vormerkungen      SET status = 'wartend'      WHERE status NOT IN ('wartend', 'abholbereit');
UPDATE buecher_exemplare SET inventur_status = NULL  WHERE inventur_status IS NOT NULL
                                                       AND inventur_status NOT IN ('ausstehend', 'erfasst');
UPDATE buecher_titel     SET grade_level = 0          WHERE grade_level < 0 OR grade_level > 13;

-- 2. Constraints ergänzen (idempotent).
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_vormerkung_status') THEN
        ALTER TABLE vormerkungen ADD CONSTRAINT chk_vormerkung_status
            CHECK (status IN ('wartend', 'abholbereit'));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_inventur_status') THEN
        ALTER TABLE buecher_exemplare ADD CONSTRAINT chk_inventur_status
            CHECK (inventur_status IS NULL OR inventur_status IN ('ausstehend', 'erfasst'));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_grade_level_bereich') THEN
        ALTER TABLE buecher_titel ADD CONSTRAINT chk_grade_level_bereich
            CHECK (grade_level IS NULL OR grade_level BETWEEN 0 AND 13);
    END IF;
END $$;
