-- =============================================================================
-- Migration 039: Non-Negativitäts-/Positivitäts-Constraints für Zählwerte & Geld
-- =============================================================================
-- Zählwerte (Bestand, Mengen) und Geldbeträge können fachlich nie negativ sein.
-- Bisher erlaubte die DB solche Werte (z. B. der von CodeQL gefundene int32-Stock-
-- Overflow konnte negative Bestände erzeugen). Diese Migration schiebt die
-- Invariante von der App-Ebene (umgehbar) in die DB (unumgehbar).
--
-- Vorgehen: erst vorhandene VERLETZER auf einen sinnvollen Boden clampen, damit das
-- ADD CONSTRAINT auf Bestandsdaten nicht fehlschlägt (Serverstart würde sonst
-- abbrechen). Danach die CHECKs ergänzen. Der DO-Guard macht die Migration
-- idempotent (schadet nicht, falls ein Constraint schon existiert).
-- =============================================================================

-- 1. Korrupte Bestandswerte auf den fachlichen Boden clampen.
UPDATE buecher_titel              SET stock = 0            WHERE stock < 0;
UPDATE buecher_titel              SET meldebestand = 0     WHERE meldebestand < 0;
UPDATE buecher_exemplare          SET einkaufspreis = 0    WHERE einkaufspreis < 0;
UPDATE bestellungen_positionen    SET menge = 1            WHERE menge < 1;
UPDATE bestellungen_positionen    SET einzelpreis = 0      WHERE einzelpreis < 0;
UPDATE bestellungen_verlauf       SET gesamtbetrag = 0     WHERE gesamtbetrag < 0;
UPDATE bestellungen_verlauf       SET anzahl_exemplare = 0 WHERE anzahl_exemplare < 0;
UPDATE klassensatz_reservierungen SET anzahl = 1           WHERE anzahl < 1;

-- 2. Constraints ergänzen (idempotent).
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_stock_nonneg') THEN
        ALTER TABLE buecher_titel ADD CONSTRAINT chk_stock_nonneg CHECK (stock >= 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_meldebestand_nonneg') THEN
        ALTER TABLE buecher_titel ADD CONSTRAINT chk_meldebestand_nonneg CHECK (meldebestand >= 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_einkaufspreis_nonneg') THEN
        ALTER TABLE buecher_exemplare ADD CONSTRAINT chk_einkaufspreis_nonneg CHECK (einkaufspreis >= 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_pos_menge_positiv') THEN
        ALTER TABLE bestellungen_positionen ADD CONSTRAINT chk_pos_menge_positiv CHECK (menge >= 1);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_pos_einzelpreis_nonneg') THEN
        ALTER TABLE bestellungen_positionen ADD CONSTRAINT chk_pos_einzelpreis_nonneg CHECK (einzelpreis >= 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_verlauf_gesamtbetrag_nonneg') THEN
        ALTER TABLE bestellungen_verlauf ADD CONSTRAINT chk_verlauf_gesamtbetrag_nonneg CHECK (gesamtbetrag >= 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_verlauf_anzahl_nonneg') THEN
        ALTER TABLE bestellungen_verlauf ADD CONSTRAINT chk_verlauf_anzahl_nonneg CHECK (anzahl_exemplare >= 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_ksr_anzahl_positiv') THEN
        ALTER TABLE klassensatz_reservierungen ADD CONSTRAINT chk_ksr_anzahl_positiv CHECK (anzahl >= 1);
    END IF;
END $$;
