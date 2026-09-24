-- =============================================================================
-- Migration 145: Ein aktives Konto ist nie ohne Ausweisnummer — Leeren zieht eine neue
-- =============================================================================
-- Entschieden am 24.09.2026 (docs/OFFEN.md 5.16). Migration 136 vergibt die Nummer beim
-- Anlegen und bei der Freischaltung eines Kontos; eine Nummer, die die Verwaltung danach
-- leert, blieb leer (entschieden am 16.09.2026, als der Hinweis am Feld noch „Leer lassen"
-- sagte). Der Ausweisdruck lieferte dann eine leere Zeile. Künftig zieht Leeren eine neue
-- Nummer aus demselben Generator (ausweis_nummer_start, Migration 136).
--
-- Drei Wege führten zu einem aktiven Konto ohne Nummer, alle drei am Verhalten nachgestellt
-- (api/sequence_pg_test.go, api/leser_schul_email_nachtragen_pg_test.go,
-- api/leser_zusammenfuehren_kollegium_pg_test.go):
--
--   1. Der PATCH der Akte leert die Spalte (api/student_update.go).
--   2. Die Benutzerverwaltung schreibt ein leeres Feld als NULL (repository.UpdateUser).
--   3. Das Zusammenführen hängt ein aktives Konto an eine Zeile ohne Nummer
--      (repository.ZusammenfuehrenSchueler: Das Ziel behält seine Nummer, auch keine).
--
-- Die Regel steht deshalb hier und nicht in einer der Türen:
--
--   - 1 und 2 fängt der neue Trigger an leser, sobald eine vorhandene Nummer verschwindet.
--     Er läuft VOR dem Schreiben: Der Generator sieht die Zeile noch mit der alten Nummer
--     und zählt sie mit, sie kommt also nicht wieder. Ein Trigger nach dem Schreiben sähe
--     sie nicht mehr und gäbe sie, wenn sie die höchste war, gleich noch einmal aus.
--     NULL über NULL lässt er liegen: Das ist kein Leeren, die Zeile hat noch keine
--     Nummer. So schreibt die Freischaltung einer Zugangsanfrage (repository.UpdateUser,
--     leeres Feld); die Nummer kommt wie bisher beim Commit vom aufgeschobenen Trigger.
--   - 3 fängt der aufgeschobene Trigger aus 136, der dafür auch beim Wechsel von
--     benutzer.leser_id feuert. Seine Rückkehr bei „war schon aktiv" entfällt: Sie hielt
--     eine geleerte Nummer leer, und das war die alte Entscheidung. Ob eine Nummer fehlt,
--     entscheidet das WHERE; ohne Treffer zieht er nichts.
--
-- Ohne Nummer bleiben: ein Leser ohne Konto oder mit inaktivem Konto (offene
-- Zugangsanfrage, Waisen-Regel aus Migration 136), ein gelöschter und ein anonymisierter.
--
-- Der Name sortiert vor trg_leser_nummer_ist_kein_buch (BEFORE-Trigger feuern in der
-- Reihenfolge ihrer Namen): Die gezogene Nummer geht danach durch die Prüfung gegen die
-- Buch-Barcodes wie jede andere.
--
-- Sperren: Der Trigger an leser hält die Zeile, bevor er den Advisory-Lock des Generators
-- nimmt. Umgekehrt (erst der Lock, dann eine Leserzeile) gehen der LUSD-Lauf, nur für
-- Schüler, an denen dieser Trigger nie zieht, und der aufgeschobene Trigger, nur für die
-- Zeile des eigenen Kontos. Warten müssten beide aufeinander nur, wenn zwei Vorgänge
-- dieselbe Person im selben Augenblick über zwei Türen schreiben; PostgreSQL bricht dann
-- einen mit 40P01 ab, laut und ohne Schreibwirkung.
--
-- Gemessen am Testserver am 24.09.2026: 9 aktive Konten, alle mit Nummer; kein Leser ohne
-- Nummer. Der Nachtrag unten trifft dort keine Zeile (lokal ebenso: 0 von 306); er steht
-- für eine Datenbank, auf der zwischen 136 und hier eine Nummer geleert wurde.

CREATE OR REPLACE FUNCTION aktives_konto_behaelt_ausweis()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.barcode_id IS NULL OR NEW.barcode_id IS NOT NULL
       OR NEW.deleted_at IS NOT NULL OR NEW.anonymized_at IS NOT NULL THEN
        RETURN NEW;
    END IF;
    IF EXISTS (SELECT 1 FROM benutzer WHERE leser_id = NEW.id AND aktiv) THEN
        NEW.barcode_id := ausweisnummer(ausweis_nummer_start());
    END IF;
    RETURN NEW;
END $$;

DROP TRIGGER IF EXISTS trg_leser_aktives_konto_behaelt_ausweis ON leser;
CREATE TRIGGER trg_leser_aktives_konto_behaelt_ausweis
BEFORE UPDATE OF barcode_id ON leser
FOR EACH ROW EXECUTE FUNCTION aktives_konto_behaelt_ausweis();

CREATE OR REPLACE FUNCTION aktives_konto_hat_ausweis()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NOT NEW.aktiv OR NEW.leser_id IS NULL THEN
        RETURN NULL;
    END IF;
    UPDATE leser
    SET barcode_id = ausweisnummer(ausweis_nummer_start()), aktualisiert_am = CURRENT_TIMESTAMP
    WHERE id = NEW.leser_id AND barcode_id IS NULL AND deleted_at IS NULL AND anonymized_at IS NULL;
    RETURN NULL;
END $$;

DROP TRIGGER IF EXISTS trg_aktives_konto_hat_ausweis ON benutzer;
CREATE CONSTRAINT TRIGGER trg_aktives_konto_hat_ausweis
AFTER INSERT OR UPDATE OF aktiv, leser_id ON benutzer
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION aktives_konto_hat_ausweis();

-- Nachtrag: jeder Leser mit aktivem Konto ohne Nummer, in der Reihenfolge seiner Anlage —
-- derselbe Weg wie der Nachtrag in 136. Ohne Treffer schreibt die Anweisung nichts; ein
-- zweiter Lauf findet keinen Leser mehr.
WITH start AS (
    SELECT ausweis_nummer_start() AS n
),
ohne AS (
    SELECT l.id, row_number() OVER (ORDER BY l.erstellt_am, l.id) - 1 AS k
    FROM leser l
    WHERE l.barcode_id IS NULL AND l.deleted_at IS NULL AND l.anonymized_at IS NULL
      AND EXISTS (SELECT 1 FROM benutzer b WHERE b.leser_id = l.id AND b.aktiv)
)
UPDATE leser l
SET barcode_id = ausweisnummer(start.n + ohne.k), aktualisiert_am = CURRENT_TIMESTAMP
FROM ohne, start
WHERE l.id = ohne.id;
