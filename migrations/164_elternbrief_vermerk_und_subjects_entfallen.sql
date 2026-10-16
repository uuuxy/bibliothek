-- =============================================================================
-- Migration 164: Der Elternbrief-Vermerk und die Tabelle subjects entfallen (docs/OFFEN.md 5.67)
-- =============================================================================
-- schadensfaelle.elternbrief_generiert und elternbrief_generiert_am hielten fest, ob der
-- Elternbrief zu einem Schadensfall gedruckt war. Den Brief gibt es nicht mehr; seitdem liest
-- und schreibt keine Stelle die beiden Spalten.
--
-- subjects legte schema.sql mit 16 Fächern an. Kein Programmteil und keine Sicht liest oder
-- schreibt die Tabelle; die Fächer des Programms stehen in systematik_kategorien.
--
-- Gemessen am Testserver am 16.10.2026 (Stand 50f9951f, lesend): subjects hält die 16 Fächer
-- aus schema.sql; von 3 Schadensfällen trägt einer den Vermerk samt Zeitpunkt.
--
-- Nicht umkehrbar: Die Werte stehen danach nur noch in der Sicherung, die update.sh vor dem
-- Einspielen anlegt.
--
-- Idempotent: Beim zweiten Lauf fehlen Spalten und Tabelle schon.
-- =============================================================================

DO $$
DECLARE
    mit_vermerk BIGINT;
    faecher BIGINT;
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema = current_schema() AND table_name = 'schadensfaelle'
                 AND column_name = 'elternbrief_generiert') THEN
        EXECUTE 'SELECT count(*) FROM schadensfaelle WHERE elternbrief_generiert' INTO mit_vermerk;
        RAISE NOTICE 'Migration 164: % Schadensfälle tragen den Elternbrief-Vermerk; er verfällt mit der Spalte.', mit_vermerk;
    END IF;
    IF to_regclass('subjects') IS NOT NULL THEN
        EXECUTE 'SELECT count(*) FROM subjects' INTO faecher;
        RAISE NOTICE 'Migration 164: subjects hält % Fächer; sie verfallen mit der Tabelle.', faecher;
    END IF;
END $$;

ALTER TABLE schadensfaelle
    DROP COLUMN IF EXISTS elternbrief_generiert,
    DROP COLUMN IF EXISTS elternbrief_generiert_am;

DROP TABLE IF EXISTS subjects;
