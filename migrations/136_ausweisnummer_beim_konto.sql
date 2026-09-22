-- Migration 136: Ein Konto bekommt beim Anlegen eine Ausweisnummer — aus EINEM Generator.
--
-- Bis hierher entstand eine Ausweisnummer an drei Stellen („Neuer Leser", LUSD-Import,
-- Littera-Übernahme), ein Konto bekam keine: konto_hat_leserzeile (Migration 125) legte die
-- Leserzeile eines Kollegen ohne Ausweis an, und der Ausweisdruck lieferte dann ein kaputtes
-- Bild und eine leere Zeile auf der Karte. Entschieden am 16.09.2026, freigegeben am
-- 22.09.2026 (docs/OFFEN.md 5.16). Gemessen am Testserver am 21.09.2026: 8 von 41 Lesern
-- ohne Nummer, alle 8 Lehrkräfte.
--
-- Die Nummer vergab bisher repository.SequenceRepository.GetNextSequence in Go (höchste
-- A-Nummer + 1 unter einem Advisory-Lock). Der Trigger unten läuft in der Datenbank und kann
-- Go nicht rufen; eine zweite Vergabe daneben wäre der Fehler aus Migration 068 in neuer Form
-- (zwei Zähler, ein Nummernkreis). Deshalb zieht der Generator hierher, und Go ruft ihn:
--
--   ausweis_nummer_start()  die nächste freie laufende Nummer, unter einem Advisory-Lock,
--                           der bis zum Ende der Transaktion hält — ein LUSD-Lauf zieht
--                           einmal und zählt dann selbst weiter (api/lusd_apply.go).
--   ausweisnummer(n)        die gedruckte Form, gleich api.AusweisNummer: „A-" und
--                           mindestens fünf Ziffern.
--
-- Die Regeln sind die von GetNextSequence: numerisch statt lexikografisch (A-100000 >
-- A-99999), Fallback 10001, und eine Nummer mit mehr als 15 Ziffern (verrutschter Scan)
-- wird übergangen, statt jede Vergabe am bigint-Cast scheitern zu lassen. Gezählt wird die
-- TABELLE leser samt gelöschter Zeilen, wie bisher.
--
-- Nummer 136, nicht 135: 135 war am 22.09.2026 kurz auf dem Testserver eingetragen und ist
-- zurückgenommen (7981e347, scripts/repair_klasse_nach_135.sql). Trüge eine Datenbank sie
-- noch in schema_migrations, liefe eine neue 135 dort nie.

CREATE OR REPLACE FUNCTION ausweis_nummer_start()
RETURNS bigint LANGUAGE plpgsql AS $$
DECLARE
    letzte bigint;
BEGIN
    PERFORM pg_advisory_xact_lock(hashtextextended('leser.barcode_id.A-', 0));
    SELECT coalesce(max(substr(barcode_id, 3)::bigint), 0) INTO letzte
    FROM leser
    WHERE barcode_id LIKE 'A-%' AND substr(barcode_id, 3) ~ '^[0-9]{1,15}$';
    IF letzte > 0 THEN
        RETURN letzte + 1;
    END IF;
    RETURN 10001;
END $$;

CREATE OR REPLACE FUNCTION ausweisnummer(n bigint)
RETURNS text LANGUAGE sql IMMUTABLE AS $$
    SELECT 'A-' || CASE WHEN length(n::text) < 5 THEN lpad(n::text, 5, '0') ELSE n::text END
$$;

-- Wann ein Konto die Nummer bekommt: sobald es AKTIV ist — beim Anlegen durch die
-- Verwaltung oder bei der Freischaltung einer Zugangsanfrage. Eine offene Anfrage bekommt
-- keine: Eine vergebene Nummer wird nie recycelt, und wer eine hat, steht in der Leserdatei
-- (repository.loescheUnberuehrteLeserzeile). Die Zeile einer abgelehnten Anfrage bliebe
-- sonst als Waise stehen — der Fund aus docs/OFFEN.md 5.18.
--
-- Aufgeschoben bis zum Commit (CONSTRAINT TRIGGER … INITIALLY DEFERRED), weil die
-- Freischaltung in derselben Transaktion danach die Leserzeile schreibt:
-- repository.UpdateUser setzt barcode_id aus dem Formular, ein leeres Feld als NULL. Eine
-- sofort vergebene Nummer wäre dort wieder gelöscht worden. Beim Commit hat jeder Schreiber
-- der Transaktion gesprochen; fehlt die Nummer dann noch, kommt sie.
--
-- Ein aktives Konto, dessen Nummer die Verwaltung leert, behält die Lücke: Der Trigger
-- greift nur beim Anlegen und beim Wechsel von inaktiv auf aktiv, nicht bei jeder Änderung.
-- konto_hat_leserzeile (Migration 125) bleibt, wie es ist — es legt die Zeile an, die
-- Nummer kommt von hier.
CREATE OR REPLACE FUNCTION aktives_konto_hat_ausweis()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NOT NEW.aktiv OR NEW.leser_id IS NULL THEN
        RETURN NULL;
    END IF;
    IF TG_OP = 'UPDATE' AND OLD.aktiv THEN
        RETURN NULL;
    END IF;
    UPDATE leser
    SET barcode_id = ausweisnummer(ausweis_nummer_start()), aktualisiert_am = CURRENT_TIMESTAMP
    WHERE id = NEW.leser_id AND barcode_id IS NULL AND deleted_at IS NULL AND anonymized_at IS NULL;
    RETURN NULL;
END $$;

DROP TRIGGER IF EXISTS trg_aktives_konto_hat_ausweis ON benutzer;
CREATE CONSTRAINT TRIGGER trg_aktives_konto_hat_ausweis
AFTER INSERT OR UPDATE OF aktiv ON benutzer
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION aktives_konto_hat_ausweis();

-- Nachtrag für den Bestand: jeder aktive Leser ohne Nummer, in der Reihenfolge seiner
-- Anlage — ausgenommen die Leserzeile einer noch offenen Zugangsanfrage (siehe oben).
-- Einmal gezogen, fortlaufend gezählt — derselbe Weg wie ein LUSD-Lauf. Ohne Treffer
-- schreibt die Anweisung nichts; ein zweiter Lauf findet keinen Leser mehr.
WITH start AS (
    SELECT ausweis_nummer_start() AS n
),
ohne AS (
    SELECT id, row_number() OVER (ORDER BY erstellt_am, id) - 1 AS k
    FROM leser
    WHERE barcode_id IS NULL AND deleted_at IS NULL AND anonymized_at IS NULL
      AND NOT EXISTS (SELECT 1 FROM benutzer b WHERE b.leser_id = leser.id AND NOT b.aktiv)
)
UPDATE leser l
SET barcode_id = ausweisnummer(start.n + ohne.k), aktualisiert_am = CURRENT_TIMESTAMP
FROM ohne, start
WHERE l.id = ohne.id;
