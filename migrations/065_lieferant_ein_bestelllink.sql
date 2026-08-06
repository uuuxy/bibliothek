-- Den Bestelllink bekommt HÖCHSTENS EINER.
--
-- Über diesen Link wählt der Händler die Etikettengröße (klein/groß) und bestätigt die
-- Bestellung; die Bestätigung landet automatisch in bestellungen_verlauf. In der Praxis
-- ist das genau ein Händler — bisher war das aber nur eine Gewohnheit: Die Spalte war ein
-- freies Boolean, und in der lokalen Stack-Datenbank trugen es 25 Lieferanten
-- gleichzeitig. Ein zweiter Händler mit Bestelllink ist kein sichtbarer Fehler, sondern
-- ein stiller: Beide bekämen den Link zur selben Bestellung, und wer zuerst bestätigt,
-- entscheidet — der andere läuft in ein 409.
--
-- Dieselbe Regel und dieselbe Begründung wie bei ist_standard (Migration 058), deshalb
-- auch dieselbe Bauform: ein Teil-Index. Die Datenbank und nicht nur der Handler, weil
-- zwei Arbeitsplätze gleichzeitig speichern können.

-- Vor dem Index aufräumen: Bestehende Mehrfach-Setzungen würden das CREATE INDEX sonst
-- abbrechen lassen, und zwar erst beim Kunden. Es überlebt der zuletzt angelegte
-- Lieferant — willkürlich, aber deterministisch, und die Wahl ist in der Oberfläche mit
-- einem Klick zu korrigieren. id als zweites Kriterium, damit gleiche Zeitstempel nicht
-- doch wieder zwei Gewinner zulassen.
UPDATE lieferanten
SET bietet_bestellbestaetigung = false
WHERE bietet_bestellbestaetigung
  AND id <> (
	SELECT id FROM lieferanten
	WHERE bietet_bestellbestaetigung
	ORDER BY erstellt_am DESC, id DESC
	LIMIT 1
  );

CREATE UNIQUE INDEX IF NOT EXISTS idx_lieferanten_ein_bestelllink
	ON lieferanten (bietet_bestellbestaetigung)
	WHERE bietet_bestellbestaetigung;

COMMENT ON COLUMN lieferanten.bietet_bestellbestaetigung IS
	'Dieser Lieferant bekommt den Bestelllink: Er wählt darüber die Etikettengröße und bestätigt die Bestellung selbst. Höchstens einer darf true tragen (idx_lieferanten_ein_bestelllink).';
