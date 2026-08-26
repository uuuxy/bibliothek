-- =============================================================================
-- Migration 087: Klassen-Anzeigeform — „09G4", nicht „9G4"; „10G1", nicht „10g1"
-- =============================================================================
-- Anlass (26.08.2026, Testserver): Die Liste der Klassensätze zeigte „09G1, 09G2,
-- 09G3, 09G5, 9G4" und „10G2 … 10g1" nebeneinander. Das Vokabular (079) kennt jede
-- Klasse nur einmal — aber als ANZEIGEFORM speichert es die erste Schreibweise, die
-- jemals geschrieben wurde. Beim Klassensatz-Import kamen „9G4", „10g1", „07r1"
-- zuerst; seitdem wird jede spätere „09G4" still darauf umgebogen. Auf dem
-- Schulserver trifft es jede Klasse, die nur in Bücherlisten vorkommt (noch keine
-- Schüler aus der LUSD).
--
-- Entscheidung Peter: EINE feste Anzeigeform, wie die LUSD sie liefert — Jahrgang
-- zweistellig, Rest groß: „05F1", „09G4", „10G1". Nur für Namen, die wie eine
-- Klasse aussehen (Ziffern, Buchstaben, optional Ziffern). Sonderwerte wie
-- 'lehrer' (Handapparat-Entleiher), 'ABG' (Versetzungslauf) oder Kursnamen
-- bleiben, wie sie sind.
--
-- Die eine Tür: BEFORE INSERT OR UPDATE auf klassen. Alle vier Tabellen schreiben
-- über klasse_kanonisieren() dorthin; die FKs (ON UPDATE CASCADE) ziehen sie beim
-- Umschreiben des Bestands mit. klassen_normkey ist unter der Anzeigeform stabil
-- (klein, ohne führende Nullen) — der Unique-Index kollidiert deshalb nicht.
-- Idempotent: CREATE OR REPLACE, DROP TRIGGER IF EXISTS, UPDATE nur WHERE <>.
-- =============================================================================

CREATE OR REPLACE FUNCTION klassen_anzeigeform(text) RETURNS text
LANGUAGE sql IMMUTABLE AS $$
    -- Innere Leerzeichen („5 f 1") fallen wie in klassen_normkey weg.
    SELECT CASE
        WHEN replace(btrim($1), ' ', '') ~ '^[0-9]{1,2}[A-Za-z]+[0-9]*$'
            THEN lpad(substring(replace(btrim($1), ' ', '') from '^[0-9]+'), 2, '0')
                 || upper(substring(replace(btrim($1), ' ', '') from '^[0-9]+(.*)$'))
        ELSE btrim($1)
    END
$$;

CREATE OR REPLACE FUNCTION klassen_anzeigeform_trigger() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    NEW.name := klassen_anzeigeform(NEW.name);
    RETURN NEW;
END $$;

DROP TRIGGER IF EXISTS trg_klassen_anzeigeform ON klassen;
CREATE TRIGGER trg_klassen_anzeigeform
BEFORE INSERT OR UPDATE OF name ON klassen
FOR EACH ROW EXECUTE FUNCTION klassen_anzeigeform_trigger();

-- Bestand nachziehen: Der CASCADE der vier FKs schreibt Schüler, Bücherlisten,
-- Reservierungen und Lehrkraft-Zuordnungen mit um.
UPDATE klassen SET name = klassen_anzeigeform(name)
WHERE name <> klassen_anzeigeform(name);
