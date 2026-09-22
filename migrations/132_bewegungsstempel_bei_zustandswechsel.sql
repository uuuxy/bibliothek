-- Migration 132: Jeder Zustandswechsel eines Exemplars stempelt die Bewegung — als Regel,
-- nicht als Auswahl.
--
-- Migration 116 hat letzte_bewegung_am eingeführt; der Wächter des Nachbuchens verlässt
-- sich darauf: Ein Offline-Scan, der älter ist als die letzte Bewegung, wird gemeldet statt
-- gebucht. Gesetzt wurde der Stempel von den Schreibern — und die Auswahl stand nirgends:
-- Aussondern, Bestandskorrektur, Schadensmeldung und Soft-Delete stempelten, „Verloren"
-- und Reaktivieren im Status-Editor, der Inventur-Abschluss und repository/damage.go nicht
-- (OFFEN.md 5.15, Durchgang vom 15.09.2026). Der Test dazu prüfte eine feste Liste; ein
-- neuer Schreiber ohne Stempel bliebe grün.
--
-- Jetzt stempelt die Datenbank, sobald ist_ausgesondert oder ist_ausleihbar kippt und der
-- Schreiber den Stempel nicht selbst gesetzt hat. Einen gesetzten Stempel lässt sie stehen:
-- Das Rückholen beim Nachbuchen stempelt mit der SCAN-Zeit (rückdatiert), und jeder
-- spätere Offline-Scan desselben Bandes gälte sonst als veraltet. Ausleihe und Rückgabe
-- ändern keinen dieser beiden Zustände; ihr Stempel bleibt Sache der Schreiber
-- (sqlStempelVor mit Scan-Zeit).
--
-- GREATEST: Der Stempel läuft nie rückwärts, auch nicht aus einer Transaktion, die vor der
-- letzten Bewegung begonnen hat (CURRENT_TIMESTAMP ist ihr Beginn). GREATEST übergeht NULL —
-- ein Exemplar ohne Stempel bekommt den Zeitpunkt.

CREATE OR REPLACE FUNCTION stempel_bewegung_bei_zustandswechsel()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF (NEW.ist_ausgesondert IS DISTINCT FROM OLD.ist_ausgesondert
        OR NEW.ist_ausleihbar IS DISTINCT FROM OLD.ist_ausleihbar)
       AND NEW.letzte_bewegung_am IS NOT DISTINCT FROM OLD.letzte_bewegung_am THEN
        NEW.letzte_bewegung_am := GREATEST(OLD.letzte_bewegung_am, CURRENT_TIMESTAMP);
    END IF;
    RETURN NEW;
END $$;

DROP TRIGGER IF EXISTS trg_exemplar_bewegung_bei_zustandswechsel ON buecher_exemplare;
CREATE TRIGGER trg_exemplar_bewegung_bei_zustandswechsel
BEFORE UPDATE OF ist_ausgesondert, ist_ausleihbar ON buecher_exemplare
FOR EACH ROW EXECUTE FUNCTION stempel_bewegung_bei_zustandswechsel();
