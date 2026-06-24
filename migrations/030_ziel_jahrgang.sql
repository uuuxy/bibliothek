-- Migration: Rename nutzungsdauer_jahre to ziel_jahrgang
-- Diese Migration stellt die Fristenberechnung auf dynamische Ziel-Jahrgänge um.
-- Idempotent: Prüft ob die Umbenennung nötig ist, bevor sie ausgeführt wird.

DO $$
DECLARE
    has_alt  BOOLEAN := EXISTS (SELECT 1 FROM information_schema.columns
                                WHERE table_name = 'buecher_titel' AND column_name = 'nutzungsdauer_jahre');
    has_ziel BOOLEAN := EXISTS (SELECT 1 FROM information_schema.columns
                                WHERE table_name = 'buecher_titel' AND column_name = 'ziel_jahrgang');
BEGIN
    IF has_alt AND has_ziel THEN
        -- Beide Spalten existieren (z. B. Neuinstallation: schema.sql legt ziel_jahrgang an,
        -- Migration 029 ergänzt zusätzlich nutzungsdauer_jahre). Die Altspalte ist redundant
        -- und wird verworfen — Umbenennen würde sonst an "column already exists" scheitern.
        ALTER TABLE buecher_titel DROP COLUMN nutzungsdauer_jahre;
    ELSIF has_alt THEN
        -- Nur die Altspalte existiert: regulärer Rename-Pfad für bestehende Datenbanken.
        ALTER TABLE buecher_titel RENAME COLUMN nutzungsdauer_jahre TO ziel_jahrgang;
    ELSIF NOT has_ziel THEN
        -- Weder alte noch neue Spalte vorhanden: frisch anlegen.
        ALTER TABLE buecher_titel ADD COLUMN ziel_jahrgang INTEGER NOT NULL DEFAULT 0;
    END IF;
    -- Fall (NOT has_alt AND has_ziel): nichts zu tun, Zielzustand bereits erreicht.
END $$;

-- Default auf 0 (kein fester Zieljahrgang)
ALTER TABLE buecher_titel ALTER COLUMN ziel_jahrgang SET DEFAULT 0;

-- Bestehende '1'-Werte (Standard-Nutzungsdauer) auf 0 bereinigen
UPDATE buecher_titel SET ziel_jahrgang = 0 WHERE ziel_jahrgang = 1;
