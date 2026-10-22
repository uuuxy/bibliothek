-- =============================================================================
-- Migration 166: Ein altes LMF-Etikett scannt weiter
-- =============================================================================
-- Manche Lernmittel tragen ein Etikett mit einer LMF-Nummer (LMF-, ISBN-13, laufende Nummer,
-- etwa LMF-9783060158423-001). Steht die Nummer hier, finden Theke, Nachbuchen, Medienkatalog,
-- Inventur und Theken-Auskunft das Exemplar auch über dieses Etikett, wie über barcode_id
-- (repository.FindeExemplar und die Abfragen daneben). So kann ein solches Exemplar eine kurze
-- Nummer tragen, ohne dass sein altes Etikett aufhört zu scannen.
--
-- NULL heißt „kein altes LMF-Etikett". Ein gesetzter Wert beginnt mit LMF- und gehört genau
-- einem Exemplar.
--
-- Idempotent: Spalte, Regel und Eindeutigkeit entstehen nur, wenn es sie noch nicht gibt.
-- =============================================================================

ALTER TABLE buecher_exemplare ADD COLUMN IF NOT EXISTS lmf_barcode TEXT;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_exemplar_lmf_barcode') THEN
        ALTER TABLE buecher_exemplare ADD CONSTRAINT chk_exemplar_lmf_barcode
            CHECK (lmf_barcode IS NULL OR lmf_barcode LIKE 'LMF-%');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'buecher_exemplare_lmf_barcode_key') THEN
        ALTER TABLE buecher_exemplare ADD CONSTRAINT buecher_exemplare_lmf_barcode_key
            UNIQUE (lmf_barcode);
    END IF;
END $$;

COMMENT ON COLUMN buecher_exemplare.lmf_barcode IS
    'LMF-Nummer eines alten Etiketts, über die das Exemplar weiter gefunden wird. NULL = kein '
    'altes LMF-Etikett (Migration 166).';
