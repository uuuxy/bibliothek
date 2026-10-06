-- =============================================================================
-- Migration 158: Standort am Exemplar (docs/OFFEN.md 5.53, Schritt 1)
-- =============================================================================
-- Wo ein Buch steht, wenn es nicht an seinem Platz nach der Signatur steht, führt Littera je
-- Exemplar (Exemplar.Sonderstandort: „Videoschrank", „Lehrerschrank"). Zusätzlich hat die
-- Bibliothek es an Titeln vermerkt („Bibliothek Klassensatz Regal 11", „Schulseelsorge").
-- Von einem Schulbuch stehen 30 Exemplare im Lernmittelbestand und zwei in der Bücherei: Der
-- Standort gehört deshalb zum Exemplar, nicht zum Titel.
--
-- NULL heißt „kein besonderer Standort": Das Exemplar steht dort, wo die Signatur es
-- hinstellt. Ein gesetzter Wert ist nicht leer und höchstens 255 Zeichen lang, so breit wie
-- Litteras Feld.
--
-- Bestehende Zeilen bleiben NULL. Der Standort am Titel (erweiterte_eigenschaften.standort)
-- bleibt von dieser Migration unberührt.

ALTER TABLE buecher_exemplare ADD COLUMN IF NOT EXISTS standort TEXT;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_exemplar_standort') THEN
        ALTER TABLE buecher_exemplare ADD CONSTRAINT chk_exemplar_standort
            CHECK (standort IS NULL OR (btrim(standort) <> '' AND char_length(standort) <= 255));
    END IF;
END $$;

COMMENT ON COLUMN buecher_exemplare.standort IS
    'Wo das Exemplar steht, wenn nicht an seinem Platz nach der Signatur (Freitext, z. B. '
    '„Bibliothek Klassensatz Regal 11"). NULL = kein besonderer Standort (Migration 158).';
