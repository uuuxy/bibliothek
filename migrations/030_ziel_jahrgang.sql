-- Migration: Rename nutzungsdauer_jahre to ziel_jahrgang
-- Diese Migration stellt die Fristenberechnung auf dynamische Ziel-Jahrgänge um.
-- Idempotent: Prüft ob die Umbenennung nötig ist, bevor sie ausgeführt wird.

DO $$
BEGIN
    -- Umbenennen, falls die alte Spalte noch existiert
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'buecher_titel' AND column_name = 'nutzungsdauer_jahre'
    ) THEN
        ALTER TABLE buecher_titel RENAME COLUMN nutzungsdauer_jahre TO ziel_jahrgang;
    END IF;

    -- Spalte hinzufügen, falls weder alte noch neue Spalte existiert
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'buecher_titel' AND column_name = 'ziel_jahrgang'
    ) THEN
        ALTER TABLE buecher_titel ADD COLUMN ziel_jahrgang INTEGER NOT NULL DEFAULT 0;
    END IF;
END $$;

-- Default auf 0 (kein fester Zieljahrgang)
ALTER TABLE buecher_titel ALTER COLUMN ziel_jahrgang SET DEFAULT 0;

-- Bestehende '1'-Werte (Standard-Nutzungsdauer) auf 0 bereinigen
UPDATE buecher_titel SET ziel_jahrgang = 0 WHERE ziel_jahrgang = 1;
