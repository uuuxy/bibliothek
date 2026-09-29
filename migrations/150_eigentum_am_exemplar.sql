-- =============================================================================
-- Migration 150: Eigentum am Exemplar (docs/OFFEN.md 4.24, Stufe 1)
-- =============================================================================
-- Wem ein Exemplar gehört, bestimmte bisher allein die Regel repository.ExemplarTopfSQL: der
-- Topf der Bestellung, sonst ist_lernmittel am Titel. Für den Altbestand ist das eine
-- Faustregel. Littera führt das Eigentum je Exemplar (Exemplar.Eigentumsvermerk). In der
-- Medienliste vom 12.06.2026 steht „Land Hessen" an rund 11.200 Exemplaren ohne LMF-Signatur —
-- Lektüren, Ganzschriften, Nachschlagewerke und Zeitschriften, die nach dem Leitfaden des
-- Kultusministeriums aus LMF-Mitteln beschafft werden dürfen und dann im Eigentum des Landes
-- bleiben (Leitfaden „Lernmittelfreiheit in Hessen", Ziffern 2.1, 9.4.2 und 13).
--
-- Die Spalte hält das Eigentum, wo es ausdrücklich bekannt ist. NULL heißt „nicht ausdrücklich
-- gesetzt": Dann gilt weiter die Bestellung, sonst der Titel. Dasselbe Vokabular wie
-- bestellungen_verlauf.mittel (Migration 109).
--
-- Bestehende Zeilen bleiben NULL; an ihrem Eigentum ändert sich dadurch nichts.

ALTER TABLE buecher_exemplare ADD COLUMN IF NOT EXISTS eigentum TEXT;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_exemplar_eigentum') THEN
        ALTER TABLE buecher_exemplare ADD CONSTRAINT chk_exemplar_eigentum
            CHECK (eigentum IS NULL OR eigentum IN ('land', 'schultraeger'));
    END IF;
END $$;

COMMENT ON COLUMN buecher_exemplare.eigentum IS
    'Ausdrücklich gesetztes Eigentum (land, schultraeger), z. B. aus dem Littera-Vermerk. '
    'NULL = nicht gesetzt: Es gilt der Topf der Bestellung, sonst der Titel '
    '(repository.ExemplarTopfSQL, Migration 150).';
