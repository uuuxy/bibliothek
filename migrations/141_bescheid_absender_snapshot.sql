-- =============================================================================
-- Migration 141: der Bescheid hält auch die Angaben der Schule fest
-- =============================================================================
-- Der Nachdruck eines Bescheids soll derselbe Brief sein. Den Empfänger hält
-- empfaenger_snapshot seit Migration 110 fest; Schulanschrift, Geschäftszeichen,
-- Bearbeiter, Durchwahl, Zahlstelle, Bankverbindung, Aufsicht und Schulleitung las der
-- Nachdruck bis zum 23.09.2026 live aus den Einstellungen (docs/OFFEN.md 5.2). Nach einem
-- Wechsel der Schulleitung trug der Nachdruck eines alten Bescheids die neue Unterschrift,
-- nach einem Kontowechsel das neue Konto.
--
-- NULL heißt: entstanden vor dieser Migration — für solche Bescheide nimmt der Nachdruck
-- weiter die Einstellungen von heute. Keine Rückschreibung: Welche Angaben am Briefdatum
-- galten, steht nirgends.
--
-- Die Spalte trägt keine Schülerdaten; die Anonymisierung leert nur den Empfänger.
-- Idempotent: ADD COLUMN IF NOT EXISTS.

ALTER TABLE schadensersatz_bescheide ADD COLUMN IF NOT EXISTS absender_snapshot JSONB;
