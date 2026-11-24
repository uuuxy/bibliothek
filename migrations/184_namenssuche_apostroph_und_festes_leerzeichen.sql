-- =============================================================================
-- Migration 184: Die Namenssuche zählt auch ´, ` und das geschützte Leerzeichen nicht
-- =============================================================================
-- Auf deutschen Tastaturen steht ´ oder ` oft für den Apostroph („D´Angelo"), und aus Importen
-- kommt das geschützte Leerzeichen (U+00A0). Die Lesersuche vergleicht Namen jetzt ohne sie
-- (repository.sqlNamenTrenner), wie die Suchen im Browser (utils/suchnorm.js). Die Indizes
-- tragen genau den neuen Ausdruck, sonst kippt die Suche in den Seq Scan.
DROP INDEX IF EXISTS idx_schueler_vorname_kompakt_trgm;
DROP INDEX IF EXISTS idx_schueler_nachname_kompakt_trgm;

CREATE INDEX IF NOT EXISTS idx_schueler_vorname_kompakt_trgm
    ON leser USING gin (translate(suchnorm(vorname), E'-'' ´`\u00A0', '') gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_schueler_nachname_kompakt_trgm
    ON leser USING gin (translate(suchnorm(nachname), E'-'' ´`\u00A0', '') gin_trgm_ops);
