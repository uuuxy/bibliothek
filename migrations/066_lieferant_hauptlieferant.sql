-- Drei Schalter für EINE Tatsache werden zu einem.
--
-- Eine Schule bestellt in der Regel bei einem Händler (hier: Naacher). Der bekommt statt
-- der reinen Bestellmail einen Link, wählt darüber große oder kleine Etiketten, beklebt
-- die Bücher selbst und bestätigt damit die Bestellung. Alle anderen Händler bekommen
-- einfach nur die Bestellmail.
--
-- Im Datenmodell standen dafür bisher DREI unabhängige Boolean-Spalten:
--
--   ist_standard               → Vorauswahl im Bestellformular
--   bietet_bestellbestaetigung → bekommt den Bestelllink
--   liefert_mit_barcode        → seine Exemplare gelten sofort als etikettiert und
--                                stehen nicht auf der Nachdruck-Liste
--
-- Sie beschreiben denselben Händler, waren aber einzeln zu setzen — und eine Kombination
-- davon war eine stille Falle: Bestelllink OHNE liefert_mit_barcode heißt, der Händler
-- beklebt die Bücher, die Bibliothek druckt die Etiketten aber trotzdem noch einmal. Kein
-- Fehler, keine Meldung, nur doppelte Arbeit am Regal. Genau dieser Zustand war beim
-- Betreiber eingestellt, als es auffiel.
--
-- Der Preis dieser Zusammenlegung, bewusst in Kauf genommen: Vorauswahl und Bestelllink
-- lassen sich nicht mehr auf zwei verschiedene Händler verteilen.

ALTER TABLE lieferanten
	ADD COLUMN IF NOT EXISTS ist_hauptlieferant BOOLEAN NOT NULL DEFAULT false;

-- Übernahme in der Reihenfolge der Aussagekraft: Wer den Bestelllink trug, war der
-- Hauptlieferant — diese Spalte durfte seit Migration 065 ohnehin nur einer tragen.
-- Gab es keinen, gilt der Standardlieferant (auch davon höchstens einer, Migration 058).
-- Gab es beides nicht, bleibt es bei false; dann ist noch nichts eingerichtet.
--
-- Im DO-Block, weil dieselbe Migration die gelesenen Spalten weiter unten LÖSCHT: Ein
-- nacktes UPDATE bräche beim zweiten Lauf mit „column does not exist" ab. Alle anderen
-- Anweisungen hier sind über IF (NOT) EXISTS wiederholbar, diese eine wäre es nicht — und
-- eine Migration, die nur beim ersten Mal durchläuft, ist eine Falle für jeden, der einmal
-- von Hand nachhilft (genau das ist beim Einbau passiert).
DO $$
BEGIN
	IF EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_name = 'lieferanten' AND column_name = 'bietet_bestellbestaetigung'
	) THEN
		UPDATE lieferanten
		SET ist_hauptlieferant = true
		WHERE id = (
			SELECT id FROM lieferanten
			WHERE bietet_bestellbestaetigung OR ist_standard
			ORDER BY bietet_bestellbestaetigung DESC, ist_standard DESC, erstellt_am DESC, id DESC
			LIMIT 1
		);
	END IF;
END $$;

-- Höchstens EINER, aus demselben Grund wie bei den Vorgängerspalten: Zwei Hauptlieferanten
-- wären ein stiller Fehler — beide bekämen den Link zur selben Bestellung, wer zuerst
-- bestätigt, gewinnt. Der Teil-Index lässt nur eine Zeile mit true zu.
CREATE UNIQUE INDEX IF NOT EXISTS idx_lieferanten_ein_hauptlieferant
	ON lieferanten (ist_hauptlieferant)
	WHERE ist_hauptlieferant;

COMMENT ON COLUMN lieferanten.ist_hauptlieferant IS
	'Der eine Händler, über den die Schule bestellt: vorausgewählt im Bestellformular, bekommt den Bestelllink (Etikettengröße + Bestätigung) und beklebt die Bücher selbst, weshalb seine Exemplare nicht auf der Nachdruck-Liste stehen. Höchstens einer (idx_lieferanten_ein_hauptlieferant).';

DROP INDEX IF EXISTS idx_lieferanten_ein_standard;
DROP INDEX IF EXISTS idx_lieferanten_ein_bestelllink;

ALTER TABLE lieferanten
	DROP COLUMN IF EXISTS ist_standard,
	DROP COLUMN IF EXISTS liefert_mit_barcode,
	DROP COLUMN IF EXISTS bietet_bestellbestaetigung;
