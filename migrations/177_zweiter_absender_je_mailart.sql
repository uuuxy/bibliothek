-- =============================================================================
-- Migration 177: Welche Mails über den zweiten Absender gehen
-- =============================================================================
-- Die Spalten portal_* (Migration 175) sind der zweite Absender neben dem Postausgang.
-- portal_absender_fuer nennt die Arten von Mails, die über ihn hinausgehen:
--   portal      Mails aus Mein Portal (Anliegen erledigt, Klassensatz bereit)
--   mahnliste   Mahnlisten an die Klassenleitungen
--   abgaenger   Kontoauszüge der Abgänger an die Klassenleitungen
--   bestellung  Bestellmail an den Händler
-- Alle übrigen Mails, und alle, solange der zweite Absender keine Adresse hat, gehen über den
-- Postausgang. Ab Werk steht nur portal darin, wie bisher (mailservice.Mailart).
--
-- Idempotent: Spalte und Bedingung entstehen nur, wenn es sie noch nicht gibt.
-- =============================================================================

ALTER TABLE mail_settings_config
    ADD COLUMN IF NOT EXISTS portal_absender_fuer TEXT[] NOT NULL DEFAULT '{portal}';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'mail_settings_config_absender_fuer_chk'
    ) THEN
        ALTER TABLE mail_settings_config ADD CONSTRAINT mail_settings_config_absender_fuer_chk
            CHECK (portal_absender_fuer <@ ARRAY['portal', 'mahnliste', 'abgaenger', 'bestellung']::TEXT[]);
    END IF;
END $$;
