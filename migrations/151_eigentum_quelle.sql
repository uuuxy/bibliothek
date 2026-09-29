-- =============================================================================
-- Migration 151: Woher das Eigentum am Exemplar kommt (docs/OFFEN.md 4.24, Stufe 3)
-- =============================================================================
-- Die Exemplarkarte nennt das Eigentum mit seiner Herkunft: „laut Littera" oder „von Hand
-- gesetzt" (sonst „aus der Bestellung" oder „Vorgabe"). Das Eigentum selbst steht seit
-- Migration 150 in buecher_exemplare.eigentum; woher es kam, sagt diese Spalte.
--
-- Die zweite Bedingung koppelt beide Spalten: Wer eigentum schreibt, nennt die Quelle. Ein
-- künftiger dritter Schreiber läuft sonst nicht still als „laut Littera" durch, sondern gegen
-- die Datenbank.
--
-- Bis hierher schrieb allein die Littera-Übernahme eigentum (internal/littera, Migration 150);
-- vorhandene Werte bekommen deshalb die Quelle littera.
--
-- Ohne Messung: Die Spalte eigentum entstand am 29.09.2026 mit Migration 150, und ihr einziger
-- Schreiber, die Littera-Übernahme, läuft erst beim Umstieg (docs/OFFEN.md 7.2); die
-- Generalprobe läuft in einer eigenen Wegwerf-Datenbank. Das UPDATE trifft deshalb 0 Zeilen und
-- steht für den Fall, dass zwischen 150 und 151 doch eine Übernahme lief. Nachprüfbar mit
-- SELECT count(*) FROM buecher_exemplare WHERE eigentum IS NOT NULL.

ALTER TABLE buecher_exemplare ADD COLUMN IF NOT EXISTS eigentum_quelle TEXT;

UPDATE buecher_exemplare SET eigentum_quelle = 'littera'
 WHERE eigentum IS NOT NULL AND eigentum_quelle IS NULL;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_exemplar_eigentum_quelle') THEN
        ALTER TABLE buecher_exemplare ADD CONSTRAINT chk_exemplar_eigentum_quelle
            CHECK (eigentum_quelle IS NULL OR eigentum_quelle IN ('littera', 'hand'));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_exemplar_eigentum_mit_quelle') THEN
        ALTER TABLE buecher_exemplare ADD CONSTRAINT chk_exemplar_eigentum_mit_quelle
            CHECK ((eigentum IS NULL) = (eigentum_quelle IS NULL));
    END IF;
END $$;

COMMENT ON COLUMN buecher_exemplare.eigentum_quelle IS
    'Woher eigentum kommt: littera (Übernahme) oder hand (Buchakte, PUT /api/exemplare/eigentum). '
    'NULL genau dann, wenn eigentum NULL ist (Migration 151).';
