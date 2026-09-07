-- Migration 108: „Derselbe Mensch" hat EINE Definition — der Namensindex rechnet in der
-- Normalform suchnorm, wie der LUSD-Schlüssel.
--
-- Bis hierher verglich unique_schueler_name_gebdatum Vor- und Nachname roh (Migration
-- 048: case-sensitiv, keine Normalform), der LUSD-Import rechnet seinen Schlüssel seit
-- 3848c9f6 in suchnorm (Umlaute, ß, Groß/Klein). Folge: Die Datenbank ließ „Anna Müller"
-- und „Anna Mueller" mit gleichem Geburtsdatum als zwei Zeilen zu (Handanlage), der
-- Import sah darin einen mehrdeutigen Schlüssel und fasste beide nicht an. Kein stiller
-- Schaden — die Mehrdeutig-Meldung fing es —, aber der Index versprach weniger, als der
-- Import annahm (Register B, 05.09.2026).
--
-- suchnorm ist IMMUTABLE (schema.sql) und darf im Index stehen. Teilindex wie bisher: nur
-- aktive Schüler mit Geburtsdatum und ohne LUSD-ID (mit ID ist die ID der Schlüssel,
-- uniq_schueler_lusd_id_active).
--
-- Falls CREATE INDEX hier scheitert, stehen in der Anlage bereits zwei Schreibvarianten
-- desselben Menschen. Sie finden mit:
--   SELECT suchnorm(vorname), suchnorm(nachname), geburtsdatum, array_agg(barcode_id)
--     FROM schueler WHERE deleted_at IS NULL AND geburtsdatum IS NOT NULL AND lusd_id IS NULL
--    GROUP BY 1, 2, 3 HAVING count(*) > 1;
-- und werden über „Schüler zusammenführen" (merge_students) zu einem — danach startet
-- der Dienst neu und die Migration läuft durch. Auf dem Test-Server (2.032 Schüler,
-- 07.09.2026) gab es null solcher Paare.

DROP INDEX IF EXISTS unique_schueler_name_gebdatum;
CREATE UNIQUE INDEX unique_schueler_name_gebdatum
    ON schueler (suchnorm(vorname), suchnorm(nachname), geburtsdatum)
    WHERE geburtsdatum IS NOT NULL AND deleted_at IS NULL AND lusd_id IS NULL;
