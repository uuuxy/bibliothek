-- =============================================================================
-- Migration 168: Eine LMF-Nummer ist nie die Nummer eines Exemplars (OFFEN 5.94, Stufe 3)
-- =============================================================================
-- Seit Migration 167 trägt jedes Exemplar eine kurze Nummer; die LMF-Nummer eines alten
-- Etiketts steht in lmf_barcode. Die Regel hält das für jeden Schreiber fest: Eine LMF-Nummer
-- (LMF-, ISBN-13, laufende Nummer) kommt so auf kein neues Etikett, wo ihr Strichcode Striche von
-- 0,16 mm hätte, und ein Scan trifft höchstens ein Exemplar, über barcode_id oder lmf_barcode.
-- Der Barcode-Editor nennt die Ablehnung (repository.IstLMFNummerAlsBarcode).
--
-- Bekam ein Exemplar zwischen Migration 167 und dieser noch eine LMF-Nummer, etwa im
-- Barcode-Editor, stellt der erste Teil es wie in Migration 167 um; sonst ändert er nichts.
-- Ohne Messung: Migration 167 hat den gemessenen Bestand umgestellt, hier bleibt höchstens, was
-- seitdem von Hand dazukam.
--
-- Idempotent: Die Regel entsteht nur, wenn es sie noch nicht gibt.
-- =============================================================================

SELECT setval('barcode_seq', GREATEST(
    (SELECT last_value FROM barcode_seq),
    (SELECT coalesce(max(substr(barcode_id, 3)::bigint), 0)
       FROM buecher_exemplare
      WHERE barcode_id ~ '^B-[0-9]{1,15}$')));

DO $$
DECLARE
    exemplar RECORD;
    nummer bigint;
BEGIN
    FOR exemplar IN
        SELECT id, barcode_id FROM buecher_exemplare
        WHERE barcode_id ILIKE 'LMF-%'
        ORDER BY barcode_id
    LOOP
        nummer := nextval('barcode_seq');
        UPDATE buecher_exemplare
        SET lmf_barcode = 'LMF-' || substr(exemplar.barcode_id, 5),
            barcode_id = 'B-' || CASE WHEN length(nummer::text) < 5 THEN lpad(nummer::text, 5, '0') ELSE nummer::text END
        WHERE id = exemplar.id;
    END LOOP;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_exemplar_barcode_keine_lmf_nummer') THEN
        ALTER TABLE buecher_exemplare ADD CONSTRAINT chk_exemplar_barcode_keine_lmf_nummer
            CHECK (barcode_id NOT ILIKE 'LMF-%');
    END IF;
END $$;
