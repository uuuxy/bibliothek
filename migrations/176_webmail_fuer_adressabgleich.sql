-- =============================================================================
-- Migration 176: Zugang zum Webmail für den Abgleich der Adressen
-- =============================================================================
-- webmail_adresse, webmail_benutzer und webmail_passwort_verschluesselt sind der Zugang zum
-- Webmail der Schule (Roundcube). Mit ihm holt der Adressabgleich das globale Adressbuch und
-- trägt Konten mit Platzhalter-Adresse ihre Adresse der Schule ein (internal/adressabgleich).
-- Eine leere Adresse heißt: kein Abgleich. Das Passwort ist wie smtp_password_encrypted mit
-- APP_ENCRYPTION_KEY verschlüsselt (repository.VerschluesselteSpalten).
--
-- Idempotent: Die Spalten entstehen nur, wenn es sie noch nicht gibt.
-- =============================================================================

ALTER TABLE mail_settings_config
    ADD COLUMN IF NOT EXISTS webmail_adresse VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS webmail_benutzer VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS webmail_passwort_verschluesselt BYTEA;
