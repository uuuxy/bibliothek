-- =============================================================================
-- Migration 181: Anschrift und Eltern-E-Mail nur beim Schüler
-- =============================================================================
-- Entscheidung vom 23.11.2026: Für das Kollegium (Lehrkraft, LiV, U-plus und die übrigen Arten
-- außer Schüler) führt das Programm keine Anschrift. Die Eltern-E-Mail gehört ebenso zum Schüler;
-- die Akte hatte sie dem Kollegium schon verschlossen.
--
-- Die Anweisung löscht, was bisher an Anschriften im Kollegium stand. Rückgängig machen lässt sie
-- sich nicht. Gemessen am 23.11.2026 am Testserver: Von 10 Lesern im Kollegium trägt einer eine
-- Anschrift und einer eine Eltern-E-Mail, keiner einen leeren String. Den Schulserver erreicht die
-- Entwicklung nicht; die Anweisung trifft dort jede Zeile im Kollegium mit einem dieser Felder. Über die Art wechselt kein Leser die Grenze zum Schüler (api.pruefeUndSetzeArt);
-- der Check kann deshalb an keiner Änderung der Art scheitern.
UPDATE leser
   SET strasse = NULL, hausnummer = NULL, plz = NULL, ort = NULL, eltern_email = NULL
 WHERE art <> 'schueler'
   AND (strasse IS NOT NULL OR hausnummer IS NOT NULL OR plz IS NOT NULL
        OR ort IS NOT NULL OR eltern_email IS NOT NULL);

ALTER TABLE leser DROP CONSTRAINT IF EXISTS chk_leser_anschrift_nur_schueler;
ALTER TABLE leser
    ADD CONSTRAINT chk_leser_anschrift_nur_schueler CHECK (
        art = 'schueler'
        OR (strasse IS NULL AND hausnummer IS NULL AND plz IS NULL
            AND ort IS NULL AND eltern_email IS NULL)
    );
