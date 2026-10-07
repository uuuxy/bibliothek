-- =============================================================================
-- Migration 161: Gescheiterter Versand der Bestellmail (docs/OFFEN.md 5.5)
-- =============================================================================
-- Scheitert die Mail an den Lieferanten, ist die Bestellung trotzdem gespeichert. Bisher
-- stand der gescheiterte Versand nirgends an ihr: Die Meldung blieb fünf Sekunden auf dem
-- Bildschirm, danach sah die Bestellung aus wie jede andere.
--
-- mail_gescheitert_am trägt den Zeitpunkt des letzten gescheiterten Versuchs. NULL heißt
-- „kein gescheiterter Versand vermerkt": die Mail ging raus, oder die Bestellung stammt aus
-- der Zeit vor dieser Spalte. Bestehende Zeilen bleiben NULL, über sie ist nichts bekannt.
-- Ein gelungener erneuter Versand setzt die Spalte wieder auf NULL.

ALTER TABLE bestellungen_verlauf ADD COLUMN IF NOT EXISTS mail_gescheitert_am TIMESTAMPTZ;

COMMENT ON COLUMN bestellungen_verlauf.mail_gescheitert_am IS
    'Zeitpunkt des letzten gescheiterten Versands der Bestellmail. NULL = kein gescheiterter '
    'Versand vermerkt (Migration 161).';
