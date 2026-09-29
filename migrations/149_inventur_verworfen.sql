-- =============================================================================
-- Migration 149: Eine verworfene Inventur ist kein Abschluss (docs/OFFEN.md 5.32)
-- =============================================================================
-- „Verwerfen" (AbortInventurSession) beendet eine Inventur ohne Verlustbuchung. Es schrieb
-- dieselben Spalten wie der Abschluss — abgeschlossen_am und verloren_gemeldet = 0 —, und die
-- Liste „Frühere Inventuren" las die Zeile als Abschluss ohne Fehlbestand: „vollständig".
-- Wer die Liste las, hielt den Bereich für geprüft.
--
-- abgeschlossen_am bleibt das Ende JEDER Inventur. Die eine offene Inventur je Bereich
-- (idx_inv_session_offen_*), der Scan und der Abschluss fragen nach abgeschlossen_am IS NULL,
-- und ein verworfener Bereich muss für einen Neustart frei werden. Die neue Spalte sagt nur,
-- wie die Inventur endete.
--
-- Bestehende Zeilen bleiben false: Das Verwerfen schrieb nichts anderes als ein Abschluss
-- ohne Verlust und keinen Protokolleintrag. Welche Zeile verworfen wurde, ist nachträglich
-- nicht zu erkennen.

ALTER TABLE inventur_sessions ADD COLUMN IF NOT EXISTS verworfen BOOLEAN NOT NULL DEFAULT false;

-- Eine laufende Inventur ist nie verworfen. Einziger Schreiber ist AbortInventurSession, das
-- beide Spalten in einer Anweisung setzt.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_inv_session_verworfen_beendet') THEN
        ALTER TABLE inventur_sessions ADD CONSTRAINT chk_inv_session_verworfen_beendet
            CHECK (NOT verworfen OR abgeschlossen_am IS NOT NULL);
    END IF;
END $$;

COMMENT ON COLUMN inventur_sessions.verworfen IS
    'true = mit „Verwerfen" beendet, ohne Verlustbuchung; abgeschlossen_am ist dann der '
    'Zeitpunkt des Verwerfens (Migration 149).';
