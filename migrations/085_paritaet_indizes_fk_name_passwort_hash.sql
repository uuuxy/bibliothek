-- =============================================================================
-- Migration 085: Parität für Installationen aus schema.sql 14.06.–21.08.2026
-- =============================================================================
-- Die Paritäts-Ratsche (db/migrations_schema_paritaet_pg_test.go) spielt nur die älteste
-- eingefrorene Baseline (05.06.) durch. Eine Installation, die aus einem schema.sql
-- ZWISCHEN dem 14.06. (011 geseedet) bzw. 21.06. (021 geseedet) und dem 21.08. entstand,
-- hat vier Indexe nie bekommen — sie standen nur in den Migrationen 011/021, die dort
-- als „angewendet" galten, und kamen in schema.sql erst mit 8409dc1e zurück; 082 heilte
-- bewusst nur die Gegenrichtung. Außerdem trägt eine Baseline vom 21.06. die tote Spalte
-- benutzer.passwort_hash (012 geseedet, erst 24.06. aus schema.sql entfernt), und der
-- Fremdschlüssel aus 063 heißt gewachsen anders als frisch (Prüfung 22.08.2026).
--
-- Alles idempotent: IF NOT EXISTS / IF EXISTS, auf frischen Datenbanken No-ops.
-- =============================================================================

CREATE INDEX IF NOT EXISTS idx_ausleihen_ausgeliehen_am ON ausleihen(ausgeliehen_am);
CREATE INDEX IF NOT EXISTS idx_ausleihen_rueckgabe_am ON ausleihen(rueckgabe_am);
CREATE INDEX IF NOT EXISTS idx_buecher_titel_erstellt_am ON buecher_titel(erstellt_am);
CREATE INDEX IF NOT EXISTS idx_schueler_deleted_at ON schueler(deleted_at) WHERE deleted_at IS NULL;

-- Tote Spalte aus der Zeit vor der IMAP-Anmeldung (Migration 012): Es wird nirgends
-- ein Passwort gespeichert (docs/FACHKONZEPT.md 12.1).
ALTER TABLE benutzer DROP COLUMN IF EXISTS passwort_hash;

-- Fremdschlüssel-Name angleichen: 063 legte ihn ohne Namen an (Postgres:
-- buecher_exemplare_bestellung_id_fkey), schema.sql nennt ihn
-- buecher_exemplare_bestellung_fkey. Der Paritätstest vergleicht seit 22.08. auch
-- Constraint-NAMEN — wie sein Kommentar es immer behauptet hat.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'buecher_exemplare_bestellung_id_fkey')
       AND NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'buecher_exemplare_bestellung_fkey') THEN
        ALTER TABLE buecher_exemplare RENAME CONSTRAINT buecher_exemplare_bestellung_id_fkey TO buecher_exemplare_bestellung_fkey;
    END IF;
END $$;
