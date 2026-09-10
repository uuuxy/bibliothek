-- Migration 111: bestellstatus gilt nur im Zulauf — als Regel der Datenbank, nicht als
-- Verabredung zwischen fünf Schreibpfaden.
--
-- bestellstatus (Migration 071) heißt „bestellt, noch nicht da". Gesetzt wird er beim
-- Bestellen, geräumt beim Wareneingang. Jeder ANDERE Ausgang aus dem Zulauf musste ihn
-- ebenfalls räumen — der Status-Editor der Buchakte, das Aussondern, das Ausbuchen und die
-- Bestandskorrektur taten es nicht (Bestands-Durchgang 10.09.2026). Ein so freigegebenes
-- Exemplar war ausleihbar, aber OPAC, Inventur, Lernmittel-Übersicht und Katalog/Monitor
-- (alle `bestellstatus IS NULL`) zählten es nie; ein ausgesondertes stand für immer im
-- Wareneingang.
--
-- Die Schreibpfade räumen seit demselben Tag. Diese Bedingung sorgt dafür, dass ein
-- künftiger sechster Ausgang, der es vergisst, LAUT scheitert statt still zu zählen.
--
-- Vorher die Altlast: Zeilen, die schon im Widerspruch stehen, sind genau die Opfer des
-- Fehlers — ausleihbar oder ausgesondert, aber noch „im Zulauf". Ihr Status wird geräumt;
-- danach zählen sie, wie sie es von Anfang an hätten sollen.

UPDATE buecher_exemplare
   SET bestellstatus = NULL
 WHERE bestellstatus IS NOT NULL
   AND (ist_ausleihbar OR ist_ausgesondert);

DO $$
BEGIN
	IF NOT EXISTS (
		SELECT 1 FROM pg_constraint WHERE conname = 'chk_exemplar_bestellstatus_nur_im_zulauf'
	) THEN
		ALTER TABLE buecher_exemplare
			ADD CONSTRAINT chk_exemplar_bestellstatus_nur_im_zulauf
			CHECK (bestellstatus IS NULL OR (ist_ausleihbar = false AND ist_ausgesondert = false));
	END IF;
END $$;
