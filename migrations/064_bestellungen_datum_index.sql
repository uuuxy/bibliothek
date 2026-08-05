-- Die gedeckelte Bestellhistorie las trotzdem jede Bestellung.
--
-- Migration 063 kam mit einem LIMIT auf /api/bestellhistorie (2,45 MB → 0,10 MB). Der
-- Plan blieb aber `Seq Scan on bestellungen_verlauf (5.279 rows)` + `top-N heapsort`:
-- Postgres musste alle Zeilen anfassen, um die 200 neuesten zu finden. Das ist heute
-- 1,7 ms und wächst linear mit jedem Schuljahr — der Deckel begrenzte die Antwort, nicht
-- die Arbeit. Genau davor warnt das Projekt-Learning: LIMIT UND Index.
--
-- DESC passend zur Abfrage (ORDER BY bestelldatum DESC LIMIT n). Postgres kann einen
-- B-Tree zwar rückwärts lesen, aber die Richtung explizit zu nennen kostet nichts und
-- macht die Absicht lesbar.
--
-- Derselbe Index bedient den Bestellbericht: Der filtert
-- `bestelldatum >= $1 AND bestelldatum < $2` (api/bestellbericht_handler.go) und suchte
-- den Zeitraum bisher ebenfalls über einen vollen Durchlauf.
CREATE INDEX IF NOT EXISTS idx_bestellungen_verlauf_datum
	ON bestellungen_verlauf (bestelldatum DESC);

COMMENT ON INDEX idx_bestellungen_verlauf_datum IS
	'Neueste Bestellungen (Historie, LIMIT) und Zeitraumfilter (Bestellbericht).';

-- Der zweite, größere Teil der Bestellhistorie: „offene Etiketten je Titel".
--
-- Gemessen mit EXPLAIN ANALYZE auf echten Daten: Die Unterabfrage lief 237-mal (einmal je
-- geladener Bestellposition) und kostete 102 der 165 ms der gesamten Anfrage — mehr als
-- alles andere zusammen. Sie fand fast nie etwas (rows=0), musste dafür aber jedes Mal
-- ALLE Exemplare des Titels durchsehen; bei einem Klassensatz sind das schnell 200 Stück.
--
-- Der Teil-Index enthält nur die Exemplare, um die es geht: die ohne gedrucktes Etikett.
-- Das sind wenige, der Index bleibt klein, und die Suche endet sofort statt beim letzten
-- Exemplar eines Klassensatzes.
--
-- Die Bedingung ist wortgleich mit etikettenOffenBedingung (api/etiketten_offen.go) —
-- nur dann kann Postgres den Teil-Index überhaupt verwenden. Wer dort etwas ändert, muss
-- diesen Index mitziehen, sonst wird er still nutzlos.
--
-- Er bedient dieselbe Frage an drei Stellen: die Nachdruck-Liste, ihren Zähler und den
-- Verweis in der Bestellhistorie.
CREATE INDEX IF NOT EXISTS idx_buecher_exemplare_etikett_offen
	ON buecher_exemplare (titel_id)
	WHERE etikett_gedruckt = false AND ist_ausgesondert = false;

COMMENT ON INDEX idx_buecher_exemplare_etikett_offen IS
	'Exemplare ohne gedrucktes Etikett je Titel (Nachdruck-Liste, ihr Zähler, Bestellhistorie). Bedingung muss zu etikettenOffenBedingung passen.';
