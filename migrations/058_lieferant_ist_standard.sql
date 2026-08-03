-- Ein Lieferant kann als Vorauswahl für neue Bestellungen gesetzt werden.
--
-- Bisher gewann der alphabetisch erste: loadSuppliers nahm suppliers[0], und die Liste
-- kommt mit ORDER BY name ASC. Wer immer beim selben Händler bestellt, musste ihn also
-- bei jeder Bestellung neu auswählen — und einmal vergessen heisst, die Bestellung geht
-- an den falschen Händler raus.
ALTER TABLE lieferanten
	ADD COLUMN IF NOT EXISTS ist_standard BOOLEAN NOT NULL DEFAULT false;

-- HÖCHSTENS EINER. Der Teil-Index erlaubt nur eine einzige Zeile mit true; false-Zeilen
-- sind nicht betroffen und dürfen beliebig oft vorkommen.
--
-- Warum die Datenbank und nicht nur der Handler: Zwei Standardlieferanten wären ein
-- stiller Fehler — die Oberfläche zeigte einen davon, welchen, entschiede die
-- Sortierung. Ein solcher Zustand darf gar nicht erst entstehen können, auch nicht
-- durch zwei gleichzeitige Speichervorgänge an zwei Arbeitsplätzen.
CREATE UNIQUE INDEX IF NOT EXISTS idx_lieferanten_ein_standard
	ON lieferanten (ist_standard)
	WHERE ist_standard;

COMMENT ON COLUMN lieferanten.ist_standard IS
	'Vorauswahl im Bestellformular. Höchstens ein Lieferant darf true tragen (idx_lieferanten_ein_standard).';
