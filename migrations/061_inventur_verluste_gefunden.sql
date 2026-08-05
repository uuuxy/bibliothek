-- Handlungsspielraum für den Fehlbestandsbericht: bisher war er reine Anzeige, die
-- „Gefunden"-Spalte ein Kästchen ohne Funktion (☐, kein <input>). Peter (05.08.2026):
-- „was mache ich [dann]? ich kann nicht weiter damit machen!"
--
-- gefunden_am hält fest, dass ein als Verlust gebuchtes Exemplar beim Regal-Absuchen
-- doch wieder aufgetaucht ist — gesetzt vom neuen "Gefunden"-Knopf, der zugleich das
-- Exemplar selbst wieder in Umlauf bringt (ist_ausgesondert = false). NULL heisst: noch
-- offen, entweder nie gefunden oder das Exemplar wurde inzwischen endgültig gelöscht.
ALTER TABLE inventur_verluste
	ADD COLUMN IF NOT EXISTS gefunden_am TIMESTAMP WITH TIME ZONE;

COMMENT ON COLUMN inventur_verluste.gefunden_am IS
	'Zeitpunkt, zu dem ein gebuchter Verlust beim Nachsuchen doch gefunden wurde. NULL = weiterhin offen.';
