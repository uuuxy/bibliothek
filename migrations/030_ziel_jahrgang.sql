-- Migration: Rename nutzungsdauer_jahre to ziel_jahrgang
-- Diese Migration stellt die Fristenberechnung auf dynamische Ziel-Jahrgänge um.

ALTER TABLE buecher_titel RENAME COLUMN nutzungsdauer_jahre TO ziel_jahrgang;

-- Default auf 0 (kein fester Zieljahrgang)
ALTER TABLE buecher_titel ALTER COLUMN ziel_jahrgang SET DEFAULT 0;

-- Optional: Wenn man die bestehenden '1' Werte (welche bei Nutzungsdauer Standard waren) auf '0' bereinigen will:
UPDATE buecher_titel SET ziel_jahrgang = 0 WHERE ziel_jahrgang = 1;
