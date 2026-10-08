-- =============================================================================
-- Migration 163: Die „Klasse" am Titel entfällt (docs/OFFEN.md 5.5, Stufe 3)
-- =============================================================================
-- buecher_titel trug zwei Angaben zum Jahrgang: grade_level („Klasse", eine Zahl) und die Spanne
-- jahrgang_von bis jahrgang_bis. Mit der Klasse rechnete kein Ablauf: Inventur nach Klasse,
-- der Filter „Jahrgang" im Portal und die Schulbuchliste lesen die Spanne. Kein Bildschirm
-- zeigt die Klasse mehr, und kein Weg schreibt sie noch.
--
-- Die Werte verfallen (entschieden am 08.10.2026), in die Spanne wird nichts übernommen: Was
-- die Zahl meinte, ist nicht belegt, und mehrjährige Bände trugen ein einziges Jahr.
--
-- Gemessen am Testserver am 08.10.2026 (Stand 009f5189, Migrationen bis 161, lesend): 13.062
-- Titel, 155 mit einer Klasse von 1 bis 13 (22 davon die 5, die Vorgabe der früheren Maske),
-- 12.907 ohne. 150 der 155 tragen die Vorgabe-Spanne 5 bis 10 und sind nach Migration 162 ohne
-- Jahrgang; fünf tragen eine eigene Spanne, die Klasse liegt bei allen darin. Kein Wert
-- außerhalb von 0 bis 13.
--
-- Nicht umkehrbar: Die Werte stehen danach nur noch in der Sicherung, die update.sh vor dem
-- Einspielen anlegt.
--
-- Idempotent: Beim zweiten Lauf fehlen Spalte und Bedingung schon.
-- =============================================================================

DO $$
DECLARE
    mit_klasse BIGINT;
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema = current_schema() AND table_name = 'buecher_titel'
                 AND column_name = 'grade_level') THEN
        EXECUTE 'SELECT count(*) FROM buecher_titel WHERE grade_level BETWEEN 1 AND 13' INTO mit_klasse;
        RAISE NOTICE 'Migration 163: % Titel tragen eine Klasse; die Angabe verfällt mit der Spalte.', mit_klasse;
    END IF;
END $$;

ALTER TABLE buecher_titel DROP CONSTRAINT IF EXISTS chk_grade_level_bereich;
ALTER TABLE buecher_titel DROP COLUMN IF EXISTS grade_level;
