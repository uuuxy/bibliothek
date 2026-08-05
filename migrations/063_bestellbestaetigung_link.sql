-- Der Bestätigungs-Link gehört UNS, nicht dem Lieferanten.
--
-- Migration 062 hat den Ablauf falsch herum modelliert: Sie ging davon aus, der Link
-- komme vom Lieferanten und die Bibliothek trage das Ergebnis von Hand nach. Richtig
-- ist das Gegenteil — Bibliosys erzeugt beim Absenden der Bestellung einen Link und
-- schickt ihn dem Lieferanten (z. B. Naacher) mit der Bestellmail. Dort wählt und
-- druckt der Lieferant die Etiketten und bestätigt die Bestellung; erst dadurch wird
-- der Vorgang in unserer Historie automatisch sichtbar. Der manuelle Nachtrag bleibt
-- als Rückfallebene bestehen (etwa wenn jemand telefonisch bestätigt).

-- NUR DER HASH. Ein Datenbank-Auszug oder ein Backup in falschen Händen enthält damit
-- keine funktionierenden Links: Aus dem SHA-256 lässt sich der Token nicht zurückrechnen.
-- Der Klartext-Token existiert genau einmal — in der Mail an den Lieferanten.
ALTER TABLE bestellungen_verlauf
	ADD COLUMN IF NOT EXISTS bestaetigungs_token_hash TEXT,
	ADD COLUMN IF NOT EXISTS token_gueltig_bis TIMESTAMPTZ,
	ADD COLUMN IF NOT EXISTS bestaetigt_durch TEXT;

-- Teil-Index statt UNIQUE-Spalte: Bestellungen ohne Link (alle Lieferanten ohne den
-- externen Schritt) tragen NULL, und NULL darf beliebig oft vorkommen. Der Index ist
-- zugleich der Zugriffspfad des öffentlichen Endpunkts, der nur den Hash kennt.
CREATE UNIQUE INDEX IF NOT EXISTS idx_bestellungen_token_hash
	ON bestellungen_verlauf (bestaetigungs_token_hash)
	WHERE bestaetigungs_token_hash IS NOT NULL;

-- Wer bestätigt hat, ist keine Nebensache: „der Lieferant hat selbst bestätigt" ist die
-- Aussage, wegen der es den Link gibt. Ein manueller Nachtrag aus der Bibliothek trägt
-- dieselbe Statuszeile, meint aber etwas anderes — ohne diese Spalte wären beide in der
-- Historie nicht auseinanderzuhalten.
ALTER TABLE bestellungen_verlauf
	ADD CONSTRAINT bestellungen_verlauf_bestaetigt_durch_check
	CHECK (bestaetigt_durch IS NULL OR bestaetigt_durch IN ('lieferant', 'bibliothek'));

COMMENT ON COLUMN bestellungen_verlauf.bestaetigungs_token_hash IS
	'SHA-256 des Bestätigungs-Tokens aus dem Link an den Lieferanten. NULL = kein Link vergeben. Klartext steht nur in der Bestellmail.';
COMMENT ON COLUMN bestellungen_verlauf.token_gueltig_bis IS
	'Ablauf des Links. Danach ist er tot — alte Mails in fremden Postfächern öffnen keine Seite mehr.';
COMMENT ON COLUMN bestellungen_verlauf.bestaetigt_durch IS
	'''lieferant'' = über den Link bestätigt, ''bibliothek'' = manuell nachgetragen. NULL solange unbestätigt.';

-- Welches Exemplar aus welcher Lieferung stammt, stand bisher nirgends (siehe Kommentar
-- zu EtikettenOffen in api/bestellhistorie_handler.go). Für den Link ist genau das nötig:
-- Die Seite muss GENAU die Etiketten dieser einen Bestellung drucken können — nicht alle
-- Exemplare des Titels, von denen die meisten längst im Regal stehen.
ALTER TABLE buecher_exemplare
	ADD COLUMN IF NOT EXISTS bestellung_id UUID REFERENCES bestellungen_verlauf(id) ON DELETE SET NULL;

-- Teil-Index: Nur die Exemplare aus dem Bestellwesen tragen den Verweis. Der Altbestand
-- (Import, Handanlage) bleibt NULL und würde den Index sonst nur aufblähen.
CREATE INDEX IF NOT EXISTS idx_buecher_exemplare_bestellung
	ON buecher_exemplare (bestellung_id)
	WHERE bestellung_id IS NOT NULL;

-- Der Barcodebogen der Bestellmail enthält NICHT alle Exemplare: Positionen ohne
-- Vorab-Barcode sind bewusst nicht darauf. Die Etikettenseite hinter dem Link muss
-- exakt dieselbe Auswahl drucken wie der Anhang — sonst bekleben zwei Wege dasselbe
-- Buch unterschiedlich. Ohne diese Spalte stünde die Auswahl nirgends.
--
-- Vorgabe false für Altbestellungen: Deren Etiketten sind längst gedruckt, und ein
-- nachträglich behaupteter Vorab-Barcode wäre schlechter als gar keine Angabe.
ALTER TABLE bestellungen_positionen
	ADD COLUMN IF NOT EXISTS mit_vorab_barcode BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN bestellungen_positionen.mit_vorab_barcode IS
	'Position stand auf dem Barcodebogen der Bestellmail. Steuert, welche Etiketten die öffentliche Bestätigungsseite druckt.';

COMMENT ON COLUMN buecher_exemplare.bestellung_id IS
	'Bestellung, aus der dieses Exemplar entstanden ist (ON DELETE SET NULL — das Exemplar überlebt die gelöschte Bestellung). NULL bei Altbestand und Handanlage.';
