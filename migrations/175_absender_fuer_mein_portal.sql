-- =============================================================================
-- Migration 175: Die Mails aus Mein Portal können unter eigenem Absender hinausgehen
-- =============================================================================
-- portal_absender_email ist der Absender der Mails an Lehrkräfte aus Mein Portal (Anliegen
-- erledigt, Klassensatz bereit). Leer heißt: Sie gehen wie alle anderen Mails über den
-- Postausgang darüber hinaus, so wie bisher.
--
-- portal_smtp_benutzer und portal_smtp_passwort_verschluesselt sind der eigene Zugang dieser
-- Adresse. Ein leerer Benutzer heißt, der Zugang des Postausgangs gilt auch für sie. Server und
-- Port teilen sich beide Absender. Das Passwort ist wie smtp_password_encrypted mit
-- APP_ENCRYPTION_KEY verschlüsselt (repository.VerschluesselteSpalten).
--
-- Idempotent: Die Spalten entstehen nur, wenn es sie noch nicht gibt.
-- =============================================================================

ALTER TABLE mail_settings_config
    ADD COLUMN IF NOT EXISTS portal_absender_email VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS portal_smtp_benutzer VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS portal_smtp_passwort_verschluesselt BYTEA;
