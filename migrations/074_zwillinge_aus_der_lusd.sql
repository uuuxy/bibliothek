-- 074: Echte Zwillinge aus der LUSD dürfen existieren (Befund F8).
--
-- Der Duplikatsschutz (vorname, nachname, geburtsdatum) sollte versehentliche
-- Doppel-Anlagen verhindern. Er traf aber auch den seltenen ECHTEN Fall: zwei
-- Schüler mit gleichem Namen und Geburtstag (bei ~1.000 Schülern möglich).
-- Der zweite ließ sich schlicht nicht anlegen — auch nicht vom LUSD-Import,
-- obwohl die LUSD beide über ihre personengebundene ID sauber unterscheidet.
--
-- Neue Regel: Der harte Index gilt nur noch für HANDEINGABEN (lusd_id IS NULL) —
-- dort ist ein Namens-Datums-Doppel fast sicher ein Versehen. LUSD-verbürgte
-- Zeilen sind über uniq_schueler_lusd_id_active bereits eindeutig; wer über die
-- LUSD kommt, ist eine echte Person. Die anwendungsseitige, verständliche
-- Prüfung beim Handanlegen (pruefeSchuelerDuplikat) bleibt unverändert und
-- warnt weiterhin über alle Zeilen hinweg.
DROP INDEX IF EXISTS unique_schueler_name_gebdatum;
CREATE UNIQUE INDEX unique_schueler_name_gebdatum
    ON schueler (vorname, nachname, geburtsdatum)
    WHERE geburtsdatum IS NOT NULL AND deleted_at IS NULL AND lusd_id IS NULL;
