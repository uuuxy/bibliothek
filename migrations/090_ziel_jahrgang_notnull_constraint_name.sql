-- Migration 030 hat die Spalte nutzungsdauer_jahre in ziel_jahrgang umbenannt —
-- der NOT-NULL-Constraint behielt dabei seinen alten, aus dem Spaltennamen
-- abgeleiteten Namen. Unsichtbar blieb das drei Monate lang, weil Postgres bis 17
-- NOT NULL gar nicht als benannten Constraint katalogisiert; seit dem Upgrade auf 18
-- (31.08.2026) steht der Name in pg_constraint, und die Schema-Paritäts-Ratsche
-- (db/migrations_schema_paritaet_pg_test.go) meldet zu Recht: gewachsen heißt er
-- buecher_titel_nutzungsdauer_jahre_not_null, frisch (schema.sql) und nach jedem
-- Dump/Restore buecher_titel_ziel_jahrgang_not_null.
--
-- Idempotent über alle drei Welten: existiert der alte Name (gewachsener Pfad),
-- wird umbenannt; existiert er nicht (frische DB, restaurierte DB, Postgres < 18),
-- passiert nichts.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'buecher_titel_nutzungsdauer_jahre_not_null'
          AND conrelid = 'buecher_titel'::regclass
    ) THEN
        ALTER TABLE buecher_titel
            RENAME CONSTRAINT buecher_titel_nutzungsdauer_jahre_not_null
            TO buecher_titel_ziel_jahrgang_not_null;
    END IF;
END $$;
