-- =============================================================================
-- Migration 049: barcode_id-Eindeutigkeit auf aktive Schüler beschränken
-- =============================================================================
-- Analog zu Migration 035 (lusd_id): barcode_id war per Spalten-UNIQUE über ALLE
-- Zeilen eindeutig — inklusive soft-gelöschter (deleted_at IS NOT NULL). Das
-- kollidierte mit dem Soft-Delete-Modell: Der Ausweis-Barcode eines gelöschten
-- Schülers blieb dauerhaft "verbrannt"; ein Neuzugang mit recyceltem Ausweis oder
-- die Wiederanmeldung desselben Schülers scheiterte an der UNIQUE-Verletzung.
--
-- Korrekte Invariante: höchstens EIN aktiver Schüler je barcode_id. Eine zuvor
-- gelöschte barcode_id darf bei Wiederanmeldung neu vergeben werden; die gelöschte
-- Historie bleibt erhalten.
-- =============================================================================

-- 1. Vorhandenen vollständigen UNIQUE-Constraint auf barcode_id robust entfernen
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
      AND att.attname = 'barcode_id';

    IF constraint_name IS NOT NULL THEN
        EXECUTE format('ALTER TABLE schueler DROP CONSTRAINT %I', constraint_name);
    END IF;
END $$;

-- 2. Partieller Unique-Index: Eindeutigkeit nur für aktive (nicht soft-gelöschte) Zeilen.
CREATE UNIQUE INDEX IF NOT EXISTS uniq_schueler_barcode_active
    ON schueler (barcode_id)
    WHERE deleted_at IS NULL;
