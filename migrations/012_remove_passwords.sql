-- Migration 012_remove_passwords.sql
-- Entfernt lokale Passwörter zugunsten von IMAP-Authentifizierung

ALTER TABLE benutzer DROP COLUMN IF EXISTS passwort_hash;
