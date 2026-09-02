-- =============================================================================
-- Migration 094: Umbenennung ohne Schüler-ID — Eintrittsdatum als zweiter Schlüssel,
--                Abgangszeitpunkt als Karenz-Uhr
-- =============================================================================
-- Anlass (02.09.2026): Der LUSD-Export der Schule trägt keine Schüler-ID und bekommt
-- so schnell keine. Die Zuordnung läuft über Name + Geburtsdatum (Migration 084). Was
-- dabei bricht, sind genau zwei seltene Fälle: eine Namensänderung in der LUSD
-- (Schreibkorrektur, Umlaut, Adoption) und ein korrigiertes Geburtsdatum. Beides ergab
-- bisher Abgänger + Neuanlage — und der Abgänger ohne Bücher wurde sofort anonymisiert,
-- womit jede spätere Reparatur unmöglich war.
--
-- Zwei Spalten, zwei Werkzeuge:
--
-- 1. schul_eintritt_am — das Eintrittsdatum an der aktuellen Schule, wie es der
--    Individuelle Bericht der LUSD liefert („Schueler_Eintritt_AktuelleSchule"). Es
--    übersteht Namensänderungen und Datumskorrekturen und ist zusammen mit dem
--    Geburtsdatum oder dem Namen praktisch eindeutig. Ein „Neuzugang" im Export, dessen
--    Eintrittsdatum Jahre zurückliegt, ist fast sicher ein umbenannter Bestandsschüler.
--    Bewusst NUR dieses Feld und nicht Geburtsort oder Geschlecht (Datensparsamkeit):
--    Es reicht für die Paarung, und es ist das am wenigsten personenbezogene der
--    Kandidaten. NULL, solange der Bericht die Spalte nicht enthält.
--
-- 2. abgaenger_seit — der Zeitpunkt, zu dem ein Import den Schüler zum Abgänger gemacht
--    hat. Bisher gab es nur abgaenger_jahr (ein Jahr) und aktualisiert_am (jede Änderung
--    schiebt es). Die Karenzzeit vor der Anonymisierung (Einstellung
--    abgaenger_karenz_tage) braucht einen Zeitpunkt, der beim Abgang gesetzt und beim
--    Rückkehren geräumt wird — sonst liefe die Frist bei jedem Klick neu.
--
-- Backfill: Bestehende Abgänger bekommen aktualisiert_am als besten vorhandenen
-- Näherungswert (dieselbe Regel, mit der der Anonymisierungs-Job bisher rechnete).
-- Idempotent: ADD COLUMN IF NOT EXISTS, Backfill nur WHERE … IS NULL.
-- =============================================================================

ALTER TABLE schueler ADD COLUMN IF NOT EXISTS schul_eintritt_am DATE;
ALTER TABLE schueler ADD COLUMN IF NOT EXISTS abgaenger_seit TIMESTAMP WITH TIME ZONE;

COMMENT ON COLUMN schueler.schul_eintritt_am IS
    'Eintritt an der aktuellen Schule laut LUSD (Schueler_Eintritt_AktuelleSchule). Zweiter Zuordnungsschlüssel des LUSD-Imports: übersteht Namensänderung und Datumskorrektur. NULL = Bericht ohne diese Spalte.';
COMMENT ON COLUMN schueler.abgaenger_seit IS
    'Zeitpunkt, zu dem ein LUSD-Import den Schüler zum Abgänger gemacht hat. Uhr der Karenzzeit vor der Anonymisierung (Einstellung abgaenger_karenz_tage); NULL bei aktiven Schülern und nach Rückkehr.';

UPDATE schueler
SET abgaenger_seit = aktualisiert_am
WHERE abgaenger_seit IS NULL
  AND ist_abgaenger = true;
