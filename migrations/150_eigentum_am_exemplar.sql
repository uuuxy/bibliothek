-- =============================================================================
-- Migration 150: Eigentum am Exemplar (docs/OFFEN.md 4.24, Stufe 1)
-- =============================================================================
-- Wem ein Exemplar gehört, bestimmte bisher allein die Regel repository.ExemplarTopfSQL: der
-- Topf der Bestellung, sonst ist_lernmittel am Titel. Für den Altbestand ist das eine
-- Faustregel. Littera führt das Eigentum je Exemplar (Exemplar.Eigentumsvermerk). In der
-- Medienliste vom 12.06.2026 steht „Land Hessen" an 11.160 Exemplaren ohne LMF-Signatur, davon
-- 3.682 aus Titeln mit 20 und mehr Stück (Klassensätze) und 528 Zeitschriften. Nach dem Leitfaden
-- „Lernmittelfreiheit in Hessen" des Kultusministeriums dürfen aus LMF-Mitteln auch Lektüren und
-- Ganzschriften gekauft werden (Ziffer 2.1), mit einer Vereinbarung mit dem Schulträger bis zu
-- 5 % auch Lehrmittel (13); was aus Landesmitteln beschafft ist, wird als Eigentum des Landes
-- gekennzeichnet (11.1, 11.4).
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
