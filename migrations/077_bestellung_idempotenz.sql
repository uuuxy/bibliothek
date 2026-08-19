-- =============================================================================
-- Migration 077: Idempotenz-Schlüssel für Bestellungen
-- =============================================================================
-- Betreiber-Entscheidung 19.08.2026 (Server-Idempotenz), Fortsetzung von Migration
-- 076 (Klassensatz). Die Bestell-Anlage hatte keine Doppelklick-Sperre — zweimal
-- klicken erzeugte zwei Bestellungen, zwei Lieferanten-Mails und doppelte Platzhalter-
-- Exemplare. Der Client schickt jetzt pro Absende-Vorgang einen Schlüssel; eine zweite
-- Anfrage mit DEMSELBEN Schlüssel läuft am partiellen Unique-Index auf, die bereits
-- angelegte Bestellung wird zurückgegeben und KEINE zweite Mail verschickt.
--
-- Partiell (WHERE ... IS NOT NULL): Alt-Bestellungen ohne Schlüssel und schlüssellose
-- Requests kollidieren nie — rein additiv.
-- =============================================================================

ALTER TABLE bestellungen_verlauf
    ADD COLUMN IF NOT EXISTS idempotenz_schluessel UUID;

CREATE UNIQUE INDEX IF NOT EXISTS uniq_bestellung_idempotenz
    ON bestellungen_verlauf (idempotenz_schluessel)
    WHERE idempotenz_schluessel IS NOT NULL;
