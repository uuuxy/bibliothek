-- Migration 129: Das Zugangsdatum am Exemplar — der Tag, an dem das Buch wirklich da war.
--
-- Bis hierher las das Zugangsbuch `erworben_am`. Das ist der Tag, an dem die ZEILE entstand,
-- und im Bestellweg entsteht sie beim BESTELLEN: `api/order_service.go` legt die Exemplare
-- zusammen mit dem Bestellkopf an, `erworben_am` steht nicht in der Spaltenliste, also greift
-- der Vorgabewert CURRENT_DATE. Der Wareneingang (`BulkReceiveOrder`) setzt nur
-- `bestellstatus = NULL` und rührt das Datum nicht an. Zwei Folgen:
--
--   1. Ein im Dezember bestelltes, im Februar geliefertes Buch stand im Zugangsbuch im
--      DEZEMBER. Verlangt ist das Eingangsdatum der Lieferung.
--   2. Exemplare, die noch im Zulauf sind, standen bereits darin — Bücher, die niemand
--      ins Regal stellen kann, in einem Nachweis über den Bestand.
--
-- ── Warum eine eigene Spalte und nicht ein Filter in der Abfrage ─────────────────────
--
-- „Im Zulauf = kein Zugang" ließe sich auch als WHERE-Bedingung schreiben. Dann wanderte ein
-- im Dezember bestelltes Buch beim Wareneingang im Februar rückwirkend in das Halbjahr, das
-- jemand im Januar ausgedruckt und abgeheftet hat — dieselbe Klasse, wegen der das
-- Abgangsdatum in Migration 128 eine Spalte wurde. Ein Nachweis, dessen Zeilen sich
-- nachträglich ändern, ist keiner.
--
-- ── Warum ein TRIGGER ────────────────────────────────────────────────────────────────
--
-- Aus dem Zulauf heraus kommt ein Exemplar über mehrere Türen: den Wareneingang
-- (`BulkReceiveOrder`), das Freigeben im Status-Editor (`UpdateCopyStatus`) und das
-- Zurückholen (`repository/exemplar_rueckholen.go`). Der Trigger hängt am Zustand selbst,
-- nicht an den Schreibern: „steht im Bestand und hat noch kein Zugangsdatum" — das gilt auch
-- für Türen, die es noch nicht gibt. Gestempelt wird nur, solange die Spalte NULL ist; ein
-- zweites Update verschiebt nichts.
--
-- Ausgesondert wird NICHT gestempelt: Ein bestelltes Exemplar, das nie ankam und ausgebucht
-- wird, ist kein Zugang.
--
-- ── Die Rückfüllung ist keine Erfindung ──────────────────────────────────────────────
--
-- Alles, was nicht im Zulauf steht, bekommt `erworben_am` — genau die Zahl, die das
-- Zugangsbuch bisher gelesen hat. Ein Ausdruck von gestern bleibt damit derselbe. Exemplare
-- im Zulauf bekommen NULL: Sie waren nie ein Zugang, und ihr Datum entsteht beim Eintreffen.
--
-- Eine Unschärfe bleibt und soll benannt sein: Ein Exemplar, das vor dieser Migration
-- bestellt, nie geliefert und dann ausgebucht wurde, trägt jetzt sein Bestelldatum als
-- Zugang. Nachträglich unterscheiden lässt sich das nicht mehr — die Bestellung ist in beiden
-- Fällen dieselbe Zeile.

ALTER TABLE buecher_exemplare ADD COLUMN IF NOT EXISTS zugang_am DATE;

UPDATE buecher_exemplare
   SET zugang_am = erworben_am
 WHERE zugang_am IS NULL
   AND bestellstatus IS NULL;

-- Beim INSERT zählt das Datum, das die Zeile mitbringt: Die Littera-Übernahme trägt das echte
-- Zugangsdatum der Altanwendung ein, Handanlage und Listenimport den heutigen Tag. CURRENT_DATE
-- wäre hier falsch — es machte aus einem 2019 übernommenen Buch einen Zugang von heute.
-- Beim UPDATE ist es der Tag, an dem das Exemplar in den Bestand kommt.
CREATE OR REPLACE FUNCTION stempel_zugang_am()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        NEW.zugang_am := NEW.erworben_am;
    ELSE
        NEW.zugang_am := CURRENT_DATE;
    END IF;
    RETURN NEW;
END $$;

DROP TRIGGER IF EXISTS trg_exemplar_zugangsdatum_neu ON buecher_exemplare;
CREATE TRIGGER trg_exemplar_zugangsdatum_neu
BEFORE INSERT ON buecher_exemplare
FOR EACH ROW
WHEN (NEW.zugang_am IS NULL AND NEW.bestellstatus IS NULL)
EXECUTE FUNCTION stempel_zugang_am();

DROP TRIGGER IF EXISTS trg_exemplar_zugangsdatum ON buecher_exemplare;
CREATE TRIGGER trg_exemplar_zugangsdatum
BEFORE UPDATE ON buecher_exemplare
FOR EACH ROW
WHEN (NEW.zugang_am IS NULL AND NEW.bestellstatus IS NULL AND NEW.ist_ausgesondert = false)
EXECUTE FUNCTION stempel_zugang_am();

-- Das Zugangsbuch fragt „welche Zugänge im Zeitraum X?" über den ganzen Bestand. Der
-- Teilindex trägt nur Zeilen mit Datum — Exemplare im Zulauf stehen nicht darin.
CREATE INDEX IF NOT EXISTS idx_exemplare_zugang_am
    ON buecher_exemplare (zugang_am)
    WHERE zugang_am IS NOT NULL;
