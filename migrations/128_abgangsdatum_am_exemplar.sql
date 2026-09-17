-- Migration 128: Das Abgangsdatum am Exemplar — der fehlende Teil des Abgangsbuchs.
--
-- Protokoll des Medienzentrums vom 16.09.2026, Punkt 1: „Zugangs- und Abgangsbuch fehlen."
-- Der Zugang steht da (`erworben_am` trägt das echte Littera-Zugangsdatum), der Grund des
-- Abgangs auch (`aussonderung_grund`: VERLUST / AUSSORTIERT / BESTANDSKORREKTUR). Was
-- fehlt, ist der ZEITPUNKT: `ist_ausgesondert` ist ein Ja/Nein ohne Datum, und
-- `letzte_bewegung_am` wird von jeder späteren Bewegung überschrieben — ein Abgangsbuch,
-- das aus ihr rechnet, ändert rückwirkend seine Zeilen.
--
-- ── Warum ein TRIGGER und nicht sechs UPDATEs ────────────────────────────────────────
--
-- Ausgesondert wird an sechs Stellen im Code: Status-Editor (UpdateCopyStatus), Aussondern
-- (DecommissionCopy), Ausbuchen (audit_books.DeleteCopy), Schaden melden
-- (repository/schaden_melden.go) und zweimal die Bestandskorrektur der Inventur
-- (inventur/db_books_update.go, freie und verliehene Exemplare). Jede einzeln zu ändern
-- heißt, dass die siebte es vergisst — und ein Abgangsbuch mit Lücken ist schlimmer als
-- keins, weil niemand die fehlende Zeile sucht. Der Trigger hängt am Zustandswechsel
-- selbst; er gilt auch für Türen, die es noch nicht gibt.
--
-- ── KEINE Rückfüllung für den Altbestand ─────────────────────────────────────────────
--
-- Vorhandene ausgesonderte Exemplare bekommen NULL, nicht `erstellt_am`, nicht
-- `aktualisiert_am`, nicht CURRENT_DATE. Ein erfundenes Abgangsdatum wäre schlimmer als ein
-- leeres: Es sähe aus wie eine Tatsache, stünde in einem Bestandsnachweis und wäre falsch.
-- NULL heißt „Abgang vor dieser Migration, Zeitpunkt unbekannt" — das ist die Wahrheit über
-- diese Daten, und das Abgangsbuch schreibt sie genau so hin.
--
-- Aus demselben Grund stempelt der Trigger nur beim WECHSEL, nicht beim INSERT: Ein Import,
-- der eines Tages historischen Abgang einliest, trüge sonst das Datum seines Imports.

ALTER TABLE buecher_exemplare ADD COLUMN IF NOT EXISTS ausgesondert_am TIMESTAMP WITH TIME ZONE;

-- Zurückgeholt (repository/exemplar_rueckholen.go) heißt: Der Abgang war ein Irrtum, das
-- Exemplar steht wieder im Bestand. Dann muss das Datum weg — sonst führte das Abgangsbuch
-- Bücher, die im Regal stehen.
CREATE OR REPLACE FUNCTION stempel_ausgesondert_am()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.ist_ausgesondert THEN
        NEW.ausgesondert_am := CURRENT_TIMESTAMP;
    ELSE
        NEW.ausgesondert_am := NULL;
    END IF;
    RETURN NEW;
END $$;

-- BEFORE, weil der Trigger NEW noch ändern können muss. Die WHEN-Bedingung ist der Kern:
-- Nur der WECHSEL zählt. Ein zweites UPDATE auf ein bereits ausgesondertes Exemplar (etwa
-- eine Notiz) darf das Abgangsdatum nicht auf heute schieben.
DROP TRIGGER IF EXISTS trg_exemplar_abgangsdatum ON buecher_exemplare;
CREATE TRIGGER trg_exemplar_abgangsdatum
BEFORE UPDATE OF ist_ausgesondert ON buecher_exemplare
FOR EACH ROW
WHEN (NEW.ist_ausgesondert IS DISTINCT FROM OLD.ist_ausgesondert)
EXECUTE FUNCTION stempel_ausgesondert_am();

-- Das Abgangsbuch fragt „welche Abgänge im Zeitraum X?" — und zwar über den ganzen
-- Bestand, nicht je Titel. Der Teilindex bleibt klein: Er trägt nur die ausgesonderten
-- Zeilen, und genau die liest der Bericht.
CREATE INDEX IF NOT EXISTS idx_exemplare_ausgesondert_am
    ON buecher_exemplare (ausgesondert_am DESC)
    WHERE ist_ausgesondert = true;
