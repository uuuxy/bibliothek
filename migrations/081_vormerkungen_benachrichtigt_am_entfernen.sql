-- =============================================================================
-- Migration 081: tote Spalte vormerkungen.benachrichtigt_am entfernen
-- =============================================================================
-- Die Spalte stand im frühen schema.sql (Stand 05.06.2026) und wurde dort mit
-- db078c35 entfernt — aber ohne DROP-Migration. Datenbanken, die vor diesem Commit
-- aufgesetzt wurden, tragen sie seither als tote Spalte mit: Kein Go-Code liest
-- oder schreibt sie, keine Migration kennt sie. Frische Installationen haben sie
-- nicht — der Bestand war je nach Installationsdatum verschieden.
--
-- Gefunden vom Schema-Paritätstest (db/migrations_schema_paritaet_pg_test.go).
-- IF EXISTS: auf Datenbanken ohne die Spalte (alles ab db078c35) ein No-op.

ALTER TABLE vormerkungen DROP COLUMN IF EXISTS benachrichtigt_am;
