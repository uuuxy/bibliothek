-- =============================================================================
-- Migration 169: Eine Anmeldung gilt höchstens zehn Stunden nach der Passworteingabe (OFFEN 5.96)
-- =============================================================================
-- Bisher hielt jedes offene Fenster seine Anmeldung ohne Grenze offen: Die Erneuerung schob
-- laeuft_ab immer weiter hinaus. Wer ein Token in die Hand bekam, blieb damit angemeldet, ohne das
-- Passwort zu kennen.
--
-- passwort_geprueft_am trägt den Zeitpunkt, zu dem das Passwort zuletzt geprüft wurde: bei der
-- Anmeldung und beim Aufschließen der Sperre. Liegt er zehn Stunden zurück, ist die Anmeldung
-- gesperrt, bis das Passwort wieder eingegeben ist (auth/sitzungen.go). Die Erneuerung des Tokens
-- ändert ihn nicht.
--
-- Laufende Anmeldungen bekommen den Zeitpunkt ihrer Anmeldung: Später geprüft wurde ihr Passwort
-- nur beim Aufschließen, und das ist nicht vermerkt. Wer beim Einspielen länger als zehn Stunden
-- angemeldet ist, sieht den Sperrbildschirm und schließt mit dem Passwort auf.
-- Ohne Messung: Die Tabelle hält nur laufende Anmeldungen, eine Zeile je Anmeldung.
--
-- Idempotent: Die Spalte entsteht und wird gefüllt nur, wenn es sie noch nicht gibt.
-- =============================================================================

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema() AND table_name = 'sitzungen'
          AND column_name = 'passwort_geprueft_am'
    ) THEN
        ALTER TABLE sitzungen
            ADD COLUMN passwort_geprueft_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP;
        UPDATE sitzungen SET passwort_geprueft_am = erstellt_am;
    END IF;
END $$;
