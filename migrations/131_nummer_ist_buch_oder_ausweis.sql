-- Migration 131: Eine Nummer ist entweder ein Buch oder ein Ausweis — nie beides.
--
-- Die Theke löst einen Scan ohne Vorsilbe zuerst als Buch auf (resolveOhnePraefix,
-- ErkenneScan). Trägt ein Leser die Nummer eines Exemplars, lädt sein Ausweis das Buch,
-- und niemand sieht warum. Bis hierher prüfte das allein die Littera-Übernahme, für ihren
-- eigenen Lauf (schreiber_personen.go: „Ausweisnummer ist schon der Barcode eines Buchs").
-- Wer von Hand eine Ausweisnummer eintrug oder ein Buch umetikettierte, wurde nicht
-- gebremst (OFFEN.md 5.15, Durchgang vom 15.09.2026).
--
-- Die Regel hängt an beiden Tabellen, nach dem Muster von Migration 118 (Ausweis über zwei
-- Personentabellen): ein Wächter, ein CONSTRAINT-Name, den der Code auswertet.
--
-- OHNE den Advisory-Lock je Nummer, den 118 hatte: Ein Transaktions-Lock je Zeile
-- belegt die Sperrtabelle, und ein Massen-Import füllt sie — nachgemessen am 22.09.2026:
-- 65.000 Exemplare in einer Transaktion → „out of shared memory" (max_locks_per_transaction
-- 64 × max_connections 100). Die Littera-Übernahme schreibt zwar 200 Titel je Transaktion,
-- ein Sammelimport aber alles in einer. BLINDHEIT: Vergeben zwei Arbeitsplätze im selben
-- Augenblick dieselbe neue Nummer, einer als Ausweis und einer als Buch, sehen beide
-- Prüfungen die andere Zeile noch nicht (READ COMMITTED), und beide gehen durch. Der Fall
-- braucht zwei Menschen, die gleichzeitig dieselbe erfundene Nummer tippen; er fällt beim
-- nächsten Scan auf und ist mit einer Änderung der Nummer behoben.
--
-- Was zählt:
--   - Bei Lesern nur aktive Zeilen (deleted_at IS NULL) — wie uniq_schueler_barcode_active.
--     Ein Leser im Papierkorb ist an der Theke unsichtbar; seine Nummer darf ein Buch
--     tragen. Kommt er zurück (deleted_at → NULL), stößt er an — der Wächter läuft deshalb
--     auch bei einer Änderung von deleted_at.
--   - Bei Exemplaren jede Zeile, auch ausgesonderte: Sie tragen ihren Barcode weiter, und
--     die Tresen-Auskunft findet sie darüber. Erst das endgültige Löschen gibt die Nummer
--     frei.
--
-- Bestehende Zeilen werden nicht geprüft: Am 15.09.2026 gab es auf dem Server keine
-- Überschneidung; der Wächter gilt für jede neue Vergabe. Er schreibt keine Daten und
-- lässt sich mit DROP TRIGGER zurücknehmen.

CREATE OR REPLACE FUNCTION nummer_ist_buch_oder_ausweis()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.barcode_id IS NULL THEN
        RETURN NEW;
    END IF;
    IF TG_TABLE_NAME = 'leser' THEN
        IF NEW.deleted_at IS NOT NULL THEN
            RETURN NEW;
        END IF;
        IF TG_OP = 'UPDATE' THEN
            IF NEW.barcode_id IS NOT DISTINCT FROM OLD.barcode_id AND OLD.deleted_at IS NULL THEN
                RETURN NEW;
            END IF;
        END IF;
        IF EXISTS (SELECT 1 FROM buecher_exemplare WHERE barcode_id = NEW.barcode_id) THEN
            RAISE EXCEPTION 'Nummer % ist schon der Barcode eines Buchs', NEW.barcode_id
                USING ERRCODE = 'unique_violation', CONSTRAINT = 'uniq_nummer_ueber_buch_und_ausweis';
        END IF;
    ELSE
        IF TG_OP = 'UPDATE' THEN
            IF NEW.barcode_id IS NOT DISTINCT FROM OLD.barcode_id THEN
                RETURN NEW;
            END IF;
        END IF;
        IF EXISTS (SELECT 1 FROM leser WHERE barcode_id = NEW.barcode_id AND deleted_at IS NULL) THEN
            RAISE EXCEPTION 'Nummer % ist schon der Ausweis eines Lesers', NEW.barcode_id
                USING ERRCODE = 'unique_violation', CONSTRAINT = 'uniq_nummer_ueber_buch_und_ausweis';
        END IF;
    END IF;
    RETURN NEW;
END $$;

DROP TRIGGER IF EXISTS trg_leser_nummer_ist_kein_buch ON leser;
CREATE TRIGGER trg_leser_nummer_ist_kein_buch
BEFORE INSERT OR UPDATE OF barcode_id, deleted_at ON leser
FOR EACH ROW EXECUTE FUNCTION nummer_ist_buch_oder_ausweis();

DROP TRIGGER IF EXISTS trg_exemplar_nummer_ist_kein_ausweis ON buecher_exemplare;
CREATE TRIGGER trg_exemplar_nummer_ist_kein_ausweis
BEFORE INSERT OR UPDATE OF barcode_id ON buecher_exemplare
FOR EACH ROW EXECUTE FUNCTION nummer_ist_buch_oder_ausweis();
