-- Manche Lieferanten (z. B. Naacher/Littera-Anbindung) schicken nach der Bestellung
-- einen eigenen Link: Dort wählt der Lieferant die Etikettengröße (klein/groß),
-- bestätigt die Bestellung und druckt/beklebt die Bücher selbst. Bibliosys erzeugt dabei
-- keine Etiketten — das bleibt vollständig beim Lieferanten. Diese Spalten sind reine
-- Merkfelder, damit der externe Bestätigungsschritt auch in unserem System sichtbar ist.
--
-- Opt-in pro Lieferant, exakt nach dem Muster von liefert_mit_barcode (Migration 056):
-- Vorgabe false, damit bestehende Lieferanten beim Update nicht plötzlich einen
-- Bestätigen-Schritt zeigen, den es für sie gar nicht gibt.
ALTER TABLE lieferanten
	ADD COLUMN IF NOT EXISTS bietet_bestellbestaetigung BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN lieferanten.bietet_bestellbestaetigung IS
	'Lieferant bietet nach der Bestellung eine eigene Etikettengrößen-Wahl + Bestätigung an (z. B. Naacher). Rein informativ — Bibliosys erzeugt hierfür keine Etiketten.';

-- bestaetigt_am ist NULL, solange niemand bestätigt hat — kein zusätzliches
-- Boolean-Feld nötig, das mit bestaetigt_am auseinanderlaufen könnte.
ALTER TABLE bestellungen_verlauf
	ADD COLUMN IF NOT EXISTS bestaetigt_am TIMESTAMPTZ,
	ADD COLUMN IF NOT EXISTS etiketten_groesse TEXT;

ALTER TABLE bestellungen_verlauf
	ADD CONSTRAINT bestellungen_verlauf_etiketten_groesse_check
	CHECK (etiketten_groesse IS NULL OR etiketten_groesse IN ('klein', 'gross'));

COMMENT ON COLUMN bestellungen_verlauf.bestaetigt_am IS
	'Zeitpunkt der externen Bestellbestätigung durch den Lieferanten (z. B. über den Naacher-Link). NULL = noch nicht bestätigt.';
COMMENT ON COLUMN bestellungen_verlauf.etiketten_groesse IS
	'Beim Bestätigen gewählte Etikettengröße (''klein''/''gross''), NULL solange unbestätigt.';
