-- =============================================================================
-- Migration 076: Idempotenz-Schlüssel für Klassensatz-Reservierungen
-- =============================================================================
-- Betreiber-Entscheidung 19.08.2026 gegen Doppelklick-Duplikate: Der Client schickt
-- pro Absende-Vorgang einen stabilen Schlüssel. Eine zweite Anfrage mit DEMSELBEN
-- Schlüssel läuft am partiellen Unique-Index auf und wird serverseitig zum No-op
-- (die bereits angelegte Reservierung wird zurückgegeben). Eine BEWUSSTE zweite
-- Reservierung schickt einen neuen Schlüssel und ist weiterhin erlaubt.
--
-- Der Index ist partiell (WHERE ... IS NOT NULL): Alt-Zeilen ohne Schlüssel und
-- schlüssellose Requests kollidieren nie — die Idempotenz ist rein additiv.
-- =============================================================================

ALTER TABLE klassensatz_reservierungen
    ADD COLUMN IF NOT EXISTS idempotenz_schluessel UUID;

CREATE UNIQUE INDEX IF NOT EXISTS uniq_ksr_idempotenz
    ON klassensatz_reservierungen (idempotenz_schluessel)
    WHERE idempotenz_schluessel IS NOT NULL;
