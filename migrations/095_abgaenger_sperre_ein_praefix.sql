-- =============================================================================
-- Migration 095: Abgänger-Sperre — ein Präfix für alle drei Türen
-- =============================================================================
-- Anlass (Rasterdurchgang 02.09.2026): Die Versetzung (POST /api/students/promote)
-- schrieb als Sperrgrund „Automatische Abgänger-Sperre (Schuljahreswechsel)"; der
-- Rückkehrer-Pfad des LUSD-Imports und das Zusammenführen erkennen die Automatik aber
-- am Präfix „Automatisierte Abgänger-Sperre". Ein Versetzungs-Abgänger, der per LUSD
-- zurückkam, wurde aktiv, blieb aber gesperrt (Ghost-Block). Der Code schreibt seit
-- dieser Migration nur noch das eine Präfix (repository.AbgaengerSperrPraefix); hier
-- werden die Bestandszeilen nachgezogen. Idempotent.
UPDATE schueler
SET block_reason = 'Automatisierte Abgänger-Sperre (Schuljahreswechsel)'
WHERE block_reason = 'Automatische Abgänger-Sperre (Schuljahreswechsel)';
