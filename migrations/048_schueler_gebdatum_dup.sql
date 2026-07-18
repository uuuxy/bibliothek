-- =============================================================================
-- Migration 048: Zwillings-Blockade bei fehlendem Geburtsdatum aufheben
-- =============================================================================
-- Der Duplikatsschutz für Schüler lief über einen UNIQUE-Index auf
--   (vorname, nachname, coalesce(geburtsdatum, '1900-01-01')).
-- Das coalesce stülpte JEDEM Schüler ohne Geburtsdatum dasselbe Ersatzdatum über.
-- Folge: Zwei namensgleiche Schüler OHNE Geburtsdatum (typisch beim Anlegen eines
-- Fünftklässlers, dessen Geburtsdatum noch nicht aus der LUSD vorliegt) kollidierten auf
-- (Leon, Müller, 1900-01-01) — der zweite ließ sich gar nicht anlegen (harter 23505-Fehler,
-- vom Handler als 500 ausgeliefert). Namensvettern ohne Geburtsdatum konnten so nicht
-- gemeinsam im System existieren.
--
-- Fix: partieller UNIQUE-Index, der nur greift, wenn das Geburtsdatum BEKANNT ist. Ein
-- fehlendes Datum ist kein Duplikat-Kriterium; bei beidseitig bekanntem, identischem Datum
-- bleibt der Schutz voll erhalten. Zusätzlich (wie uniq_schueler_lusd_id_active) nur unter
-- AKTIVEN Schülern: eine soft-gelöschte Namens-/Datumskombination blockiert die
-- Wiederanmeldung nicht mehr. Die anwendungsseitige Prüfung (pruefeSchuelerDuplikat) wurde
-- passend angezogen.
--
-- Gefahrlos auf Bestandsdaten: Der alte Index verbot bereits jede Dopplung auf
-- (vorname, nachname, geburtsdatum) für nicht-leere Geburtsdaten; der neue, engere Index
-- deckt nur eine Teilmenge davon ab und kann daher nicht an Bestandskonflikten scheitern.
-- =============================================================================

DROP INDEX IF EXISTS unique_schueler_name_gebdatum;

CREATE UNIQUE INDEX unique_schueler_name_gebdatum
    ON schueler (vorname, nachname, geburtsdatum)
    WHERE geburtsdatum IS NOT NULL AND deleted_at IS NULL;
