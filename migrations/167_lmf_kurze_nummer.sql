-- =============================================================================
-- Migration 167: Lernmittel mit LMF-Nummer bekommen eine kurze Nummer (OFFEN 5.94, Stufe 2)
-- =============================================================================
-- Jedes Exemplar trägt eine kurze Nummer wie in Littera, auch ein Lernmittel, das mit seiner
-- LMF-Nummer (LMF-, ISBN-13, laufende Nummer, etwa LMF-9783060158423-001) übernommen wurde. Die
-- LMF-Nummer wandert nach lmf_barcode (Migration 166): Über sie finden Theke, Katalog, Inventur
-- und Theken-Auskunft das Exemplar weiter, ein neues Etikett trägt die kurze Nummer.
--
-- Die kurzen Nummern kommen aus barcode_seq wie jede Exemplarnummer, in derselben Form: „B-" und
-- mindestens fünf Ziffern (repository/barcode_vergabe.go, exemplarNummer; lpad allein schnitte
-- eine sechsstellige Zahl ab). Vorher steigt die Sequenz über die höchste vergebene B-Nummer
-- wie in Migration 068, damit keine eine von Hand eingetragene trifft. Vergeben wird in der
-- Reihenfolge der LMF-Nummern: Die Exemplare einer ISBN bekommen aufeinanderfolgende Nummern.
-- Die Vorsilbe steht in lmf_barcode groß, so wie der Scan sie liefert (normalisiereScan).
--
-- etikett_gedruckt bleibt, wie er ist: Das alte Etikett klebt und scannt. Wiederholbar: Ein
-- zweiter Lauf findet in barcode_id keine LMF-Nummer mehr und ändert nichts.
--
-- Gemessen am Testserver am 22.10.2026 (nur lesend): 4.065 Exemplare tragen eine LMF-Nummer als
-- eigene Nummer, an 130 Titeln, alle aus einem Lauf des Bestands-Imports und als gedruckt
-- vermerkt. Der echte Betrieb beginnt mit einer leeren Datenbank; dort legt der Bestands-Import
-- solche Exemplare gleich mit kurzer Nummer an.
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
END $$;
