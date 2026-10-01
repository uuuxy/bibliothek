-- =============================================================================
-- Migration 155: Sitzungen — die Sperre nach Inaktivität gilt am Server
-- =============================================================================
-- Eine Zeile je Anmeldung. An ihr hängt die Sperre nach Inaktivität: Sie gilt am Server und
-- überdauert die Erneuerung des Tokens, das ihre Kennung trägt.
--
-- passwort_pruefwert schließt die Sperre auf, wenn der Mailserver der Schule nicht erreichbar
-- ist — sonst sperrte sein Ausfall die Theke zu. Der Wert ist Argon2id über das Passwort mit
-- einem Schlüssel, der nicht in der Datenbank steht (auth/pruefwert.go). Abmelden löscht die
-- Zeile, abgelaufene räumt der Server ab.
--
-- Ohne Messung: Die Tabelle ist neu und leer. Anmeldungen, die beim Einspielen laufen, haben
-- keine Zeile; für sie bleibt die Sperre bis zu ihrem Ablauf ein Sichtschutz.
-- =============================================================================

CREATE TABLE IF NOT EXISTS sitzungen (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    benutzer_id UUID NOT NULL REFERENCES benutzer(id) ON DELETE CASCADE,
    passwort_pruefwert TEXT NOT NULL,
    gesperrt_seit TIMESTAMP WITH TIME ZONE,
    laeuft_ab TIMESTAMP WITH TIME ZONE NOT NULL,
    erstellt_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sitzungen_laeuft_ab ON sitzungen(laeuft_ab);
CREATE INDEX IF NOT EXISTS idx_sitzungen_benutzer ON sitzungen(benutzer_id);
