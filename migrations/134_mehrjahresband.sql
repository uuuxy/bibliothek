-- Migration 134: Ein Mehrjahresband ist ein Schalter am Werk, keine zweite Jahreszahl.
--
-- Antwort der Schule vom 22.09.2026 auf Protokoll 5 der Sichtung vom 16.09.2026: „Die
-- Einstellung ‚Mehrjahres-Band' wird am Titel/Werk hinterlegt." Am selben Tag bekam die
-- Spalte ziel_jahrgang (Migration 030, bis dahin ohne Schreiber) dafür eine Tür — eine
-- zweite Jahreszahl neben jahrgang_bis, die dasselbe sagt: bis zu welchem Jahrgang das
-- Buch gebraucht wird. Zwei Felder für eine Zahl laufen auseinander, und der
-- Littera-Import schreibt nur die Spanne. Deshalb: Die Zahl kommt aus jahrgang_bis, der
-- Schalter sagt, ob das Buch über die Spanne beim Kind bleibt oder wie jedes Schulbuch am
-- Rückgabetermin der Klasse zurückkommt (docs/OFFEN.md 9.6). Gemessen am Testserver am
-- 22.09.2026: kein Titel trug einen ziel_jahrgang — die Spalte fällt ohne Datenverlust.
--
-- Der Schalter gilt nur an einem Lernmittel (bei einem Buch der Schülerbücherei liest ihn
-- keine Regel) und nur mit einer Spanne über mehr als einen Jahrgang: Ein „Mehrjahresband
-- 7 bis 7" hätte keine Wirkung, die Maske sagte aber „bleibt beim Kind". Dieselbe Regel
-- prüfen die Türen der Titel-Verwaltung (inventur/mehrjahresband.go); die Datenbank hält
-- sie für jeden anderen Schreiber.

ALTER TABLE buecher_titel ADD COLUMN IF NOT EXISTS mehrjahresband BOOLEAN NOT NULL DEFAULT false;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_mehrjahresband_spanne') THEN
        ALTER TABLE buecher_titel ADD CONSTRAINT chk_mehrjahresband_spanne
            CHECK (NOT mehrjahresband
                   OR (ist_lernmittel AND coalesce(jahrgang_bis, 0) > coalesce(jahrgang_von, 0)));
    END IF;
END $$;

ALTER TABLE buecher_titel DROP COLUMN IF EXISTS ziel_jahrgang;
