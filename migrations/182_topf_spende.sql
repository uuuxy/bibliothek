-- =============================================================================
-- Migration 182: Der Topf Spende
-- =============================================================================
-- Entscheidung vom 23.11.2026: Bücher lassen sich ohne Topf bestellen, etwa mit Geld des
-- Fördervereins. Der dritte Wert heißt 'spende'; beim Händler, auf dem Etikett und beim Ersatz
-- gilt er wie die Schülerbücherei (mitteltopf.WieSchuelerbuecherei), im Zugangs- und im
-- Abgangsbuch steht er in einer eigenen Liste.
--
-- Erweitert werden die Checks, die denselben Wert tragen wie bestellungen_verlauf.mittel:
-- die Bestellung, das Eigentum am Exemplar und die beiden Topf-Spalten gelöschter Exemplare.
-- Der Bescheid bleibt bei Land und Schulträger (chk_bescheid_mittel, chk_nummern_mittel): Er
-- entsteht nur für Lernmittel des Landes.
--
-- Ohne Messung: Die Anweisungen ändern keine Zeile, sie lassen nur einen Wert mehr zu.
ALTER TABLE bestellungen_verlauf DROP CONSTRAINT IF EXISTS bestellungen_verlauf_mittel_check;
ALTER TABLE bestellungen_verlauf
    ADD CONSTRAINT bestellungen_verlauf_mittel_check
    CHECK (mittel IS NULL OR mittel IN ('land', 'schultraeger', 'spende'));

ALTER TABLE buecher_exemplare DROP CONSTRAINT IF EXISTS chk_exemplar_eigentum;
ALTER TABLE buecher_exemplare
    ADD CONSTRAINT chk_exemplar_eigentum
    CHECK (eigentum IS NULL OR eigentum IN ('land', 'schultraeger', 'spende'));

ALTER TABLE geloeschte_exemplare DROP CONSTRAINT IF EXISTS chk_geloeschte_exemplare_topf_belegt;
ALTER TABLE geloeschte_exemplare
    ADD CONSTRAINT chk_geloeschte_exemplare_topf_belegt
    CHECK (topf_belegt IN ('land', 'schultraeger', 'spende'));

ALTER TABLE geloeschte_exemplare DROP CONSTRAINT IF EXISTS chk_geloeschte_exemplare_topf;
ALTER TABLE geloeschte_exemplare
    ADD CONSTRAINT chk_geloeschte_exemplare_topf
    CHECK (topf IN ('land', 'schultraeger', 'spende'));
