-- =============================================================================
-- Migration 080: schueler.anonymized_at auf gewachsenen Datenbanken nachziehen
-- =============================================================================
-- Migration 022_dsgvo_anonymize_schueler.sql ist im goose-Format geschrieben —
-- unser Runner (db/migrations.go) kennt goose nicht und führt die DATEI ALS GANZES
-- aus: erst das ADD COLUMN aus dem Up-Teil, dann sofort das DROP COLUMN aus dem
-- Down-Teil. Netto-Effekt: nichts. Jede gewachsene Datenbank (auch Prod) hat die
-- Spalte deshalb NIE bekommen; schema.sql führt sie erst seit f71c3b57 (18.08.2026).
-- Ohne sie scheitert das UPDATE der DSGVO-Anonymisierung (jobs/cron_dsgvo.go) zur
-- Planzeit — still, nur als Logzeile.
--
-- Gefunden vom Schema-Paritätstest (db/migrations_schema_paritaet_pg_test.go), der
-- den gewachsenen Pfad (Baseline + Migrationen) gegen das heutige schema.sql diffe.
-- 022 bleibt unangetastet (historische Wahrheit); dieses IF NOT EXISTS ist auf
-- frischen Installationen (Spalte schon da) ein No-op.

ALTER TABLE schueler ADD COLUMN IF NOT EXISTS anonymized_at TIMESTAMP;
