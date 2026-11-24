-- =============================================================================
-- Migration 183: Die Namenssuche zählt Bindestrich, Apostroph und Leerzeichen nicht
-- =============================================================================
-- Die Lesersuche (Theke, Leserdatei, Leser zusammenführen) vergleicht Namen jetzt in der Form
-- translate(suchnorm(name), '-'' ', '') (repository.sqlNameKompakt): „alkahil" findet
-- Al-Kahil, „annalena" findet Anna-Lena. Die Trigramm-Indizes müssen genau diesen Ausdruck
-- tragen, sonst kippt die Suche in den Seq Scan (leserSuchBedingung). Die beiden Indizes auf
-- suchnorm(vorname) und suchnorm(nachname) aus Migration 054 nutzt danach keine Abfrage mehr;
-- die Vergleiche auf Gleichheit beim Anlegen stehen auf dem eigenen Namensindex (Migration 108).
DROP INDEX IF EXISTS idx_schueler_vorname_suchnorm_trgm;
DROP INDEX IF EXISTS idx_schueler_nachname_suchnorm_trgm;

CREATE INDEX IF NOT EXISTS idx_schueler_vorname_kompakt_trgm
    ON leser USING gin (translate(suchnorm(vorname), '-'' ', '') gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_schueler_nachname_kompakt_trgm
    ON leser USING gin (translate(suchnorm(nachname), '-'' ', '') gin_trgm_ops);
