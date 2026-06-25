-- =============================================================================
-- Migration 035: lusd_id-Eindeutigkeit auf aktive Schüler beschränken
-- =============================================================================
-- Bisher war lusd_id per Spalten-UNIQUE über ALLE Zeilen eindeutig — inklusive
-- soft-gelöschter (deleted_at IS NOT NULL). Das kollidierte mit dem Soft-Delete-
-- Modell: Wurde ein Schüler gelöscht, der weiterhin auf der LUSD-Liste steht,
-- blockierte seine gelöschte Zeile die Neuanlage beim nächsten Import — der
-- Schüler blieb dauerhaft unsichtbar (LusdSyncPreview), bzw. der Import wäre an
-- der UNIQUE-Verletzung gescheitert, sobald der deleted_at-Filter greift.
--
-- Korrekte Invariante: höchstens EIN aktiver Schüler je lusd_id. Eine zuvor
-- gelöschte lusd_id darf bei Wiederanmeldung als frischer aktiver Datensatz
-- neu angelegt werden; die gelöschte Historie bleibt erhalten.
-- =============================================================================

-- 1. Den vorhandenen vollständigen UNIQUE-Constraint auf lusd_id robust entfernen
--    (Name per Katalog-Lookup, nicht geraten — überlebt abweichende Benennung).
DO $$
DECLARE
    constraint_name text;
BEGIN
    SELECT con.conname INTO constraint_name
    FROM pg_constraint con
    JOIN pg_attribute att
      ON att.attrelid = con.conrelid
     AND att.attnum = ANY (con.conkey)
    WHERE con.conrelid = 'schueler'::regclass
      AND con.contype = 'u'
      AND array_length(con.conkey, 1) = 1
      AND att.attname = 'lusd_id';

    IF constraint_name IS NOT NULL THEN
        EXECUTE format('ALTER TABLE schueler DROP CONSTRAINT %I', constraint_name);
    END IF;
END $$;

-- 2. Partieller Unique-Index: Eindeutigkeit nur für aktive, gesetzte lusd_id.
CREATE UNIQUE INDEX IF NOT EXISTS uniq_schueler_lusd_id_active
    ON schueler (lusd_id)
    WHERE deleted_at IS NULL AND lusd_id IS NOT NULL;
