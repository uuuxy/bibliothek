-- Migration 118: Eine Ausweisnummer gehört genau einer Person — über Schüler und Kollegium
-- hinweg.
--
-- Ob jemand Schüler oder Lehrkraft ist, steht in den Stammdaten (LUSD bzw. Benutzerverwaltung),
-- nicht auf dem Ausweis. Die Theke sucht bei jeder Nummer unter beiden Tabellen, zuerst unter den
-- Schülern (internal/service/omnibox_service.go). Bis hierher galt die Eindeutigkeit nur je
-- Tabelle (uniq_schueler_barcode_active, benutzer_barcode_id_key), und jeder Schreibweg prüfte nur
-- seine eigene: Ein Schüler und eine Lehrkraft konnten dieselbe Nummer tragen, und der Scan lud
-- still den Schüler. Auf dem Server gab es am 15.09.2026 kein solches Doppel.
--
-- Ein UNIQUE-Index reicht nicht über zwei Tabellen, deshalb ein Trigger an beiden:
--  * Er prüft die jeweils andere Tabelle, wenn eine Nummer gesetzt oder geändert wird und wenn
--    ein Schüler aus dem Papierkorb zurückkommt. Gelöschte Schüler geben ihre Nummer frei, wie
--    beim Teilindex aus Migration 049.
--  * Vorher sperrt er die Nummer bis zum Ende der Transaktion. Ohne die Sperre sähe keine von zwei
--    gleichzeitigen Vergaben die noch nicht festgeschriebene Zeile der anderen. Die Prüfung setzt
--    READ COMMITTED voraus (jede Abfrage im Trigger sieht, was bis dahin festgeschrieben ist) —
--    die Stufe aller Schreibwege.
--  * Die Verletzung ist ein unique_violation namens uniq_ausweis_ueber_personen; die Schreibwege
--    übersetzen ihn in eine Auskunft.
--  * Ändert sich die Nummer nicht, prüft er nicht: Ein Doppel aus der Zeit davor blockiert weder
--    den Start noch eine Änderung an anderen Feldern. Die Migration meldet es als WARNING.

DO $$
DECLARE
    doppelt INT;
BEGIN
    SELECT count(*) INTO doppelt
      FROM schueler s JOIN benutzer b ON b.barcode_id = s.barcode_id
     WHERE s.deleted_at IS NULL;
    IF doppelt > 0 THEN
        RAISE WARNING 'Migration 118: % Ausweisnummer(n) tragen ein Schüler und eine Lehrkraft zugleich — die Theke lädt dort den Schüler', doppelt;
    END IF;
END $$;

CREATE OR REPLACE FUNCTION ausweis_eindeutig_ueber_personen()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.barcode_id IS NULL THEN
        RETURN NEW;
    END IF;
    IF TG_TABLE_NAME = 'schueler' THEN
        IF NEW.deleted_at IS NOT NULL THEN
            RETURN NEW;
        END IF;
        IF TG_OP = 'UPDATE' THEN
            IF NEW.barcode_id IS NOT DISTINCT FROM OLD.barcode_id AND OLD.deleted_at IS NULL THEN
                RETURN NEW;
            END IF;
        END IF;
        PERFORM pg_advisory_xact_lock(hashtext('ausweisnummer'), hashtext(NEW.barcode_id));
        IF EXISTS (SELECT 1 FROM benutzer WHERE barcode_id = NEW.barcode_id) THEN
            RAISE EXCEPTION 'Ausweisnummer % trägt bereits eine Lehrkraft', NEW.barcode_id
                USING ERRCODE = 'unique_violation', CONSTRAINT = 'uniq_ausweis_ueber_personen';
        END IF;
    ELSE
        IF TG_OP = 'UPDATE' THEN
            IF NEW.barcode_id IS NOT DISTINCT FROM OLD.barcode_id THEN
                RETURN NEW;
            END IF;
        END IF;
        PERFORM pg_advisory_xact_lock(hashtext('ausweisnummer'), hashtext(NEW.barcode_id));
        IF EXISTS (SELECT 1 FROM schueler WHERE barcode_id = NEW.barcode_id AND deleted_at IS NULL) THEN
            RAISE EXCEPTION 'Ausweisnummer % trägt bereits ein Schüler', NEW.barcode_id
                USING ERRCODE = 'unique_violation', CONSTRAINT = 'uniq_ausweis_ueber_personen';
        END IF;
    END IF;
    RETURN NEW;
END $$;

DROP TRIGGER IF EXISTS trg_schueler_ausweis_eindeutig ON schueler;
CREATE TRIGGER trg_schueler_ausweis_eindeutig
BEFORE INSERT OR UPDATE OF barcode_id, deleted_at ON schueler
FOR EACH ROW EXECUTE FUNCTION ausweis_eindeutig_ueber_personen();

DROP TRIGGER IF EXISTS trg_benutzer_ausweis_eindeutig ON benutzer;
CREATE TRIGGER trg_benutzer_ausweis_eindeutig
BEFORE INSERT OR UPDATE OF barcode_id ON benutzer
FOR EACH ROW EXECUTE FUNCTION ausweis_eindeutig_ueber_personen();
