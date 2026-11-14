-- =============================================================================
-- Migration 172: Jedes Konto nennt, welcher Server sein Passwort prüft
-- =============================================================================
-- anmeldeverfahren: 'imap' prüft am Mailserver der Schule, 'ldap' am Verzeichnisdienst
-- (auth/ldap.go). Vorhandene Konten bleiben bei 'imap'.
--
-- ldap_kennung ist der Name, unter dem der Verzeichnisdienst das Konto findet (etwa
-- sAMAccountName oder userPrincipalName); LDAP_FILTER setzt ihn ein. Ein LDAP-Konto braucht ihn,
-- ein IMAP-Konto hat keinen. Wie die E-Mail ist er ohne Rücksicht auf Groß- und Kleinschreibung
-- eindeutig, denn die Anmeldung findet ein Konto auch über ihn.
--
-- Idempotent: Spalten, Bedingungen und Index entstehen nur, wenn es sie noch nicht gibt.
-- =============================================================================

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema() AND table_name = 'benutzer'
          AND column_name = 'anmeldeverfahren'
    ) THEN
        ALTER TABLE benutzer
            ADD COLUMN anmeldeverfahren TEXT NOT NULL DEFAULT 'imap',
            ADD COLUMN ldap_kennung VARCHAR(255),
            ADD CONSTRAINT benutzer_anmeldeverfahren_bekannt
                CHECK (anmeldeverfahren IN ('imap', 'ldap')),
            ADD CONSTRAINT benutzer_ldap_kennung_nur_bei_ldap
                CHECK ((anmeldeverfahren = 'ldap') = (ldap_kennung IS NOT NULL)),
            ADD CONSTRAINT benutzer_ldap_kennung_nicht_leer
                CHECK (ldap_kennung IS NULL OR btrim(ldap_kennung) <> '');
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS uniq_benutzer_ldap_kennung_lower
    ON benutzer (lower(ldap_kennung)) WHERE ldap_kennung IS NOT NULL;
