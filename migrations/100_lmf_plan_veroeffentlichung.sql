-- =============================================================================
-- Migration 100: LMF-Plan — Entwurf bis zur Veröffentlichung
-- =============================================================================
-- Peter 06.09.2026: „Stille Vorbereitung — der Plan nimmt die Schulleitung immer erst
-- ab." Bis hierher galt jeder gespeicherte Plan sofort: Das Portal zeigte ihn, das PDF
-- las ihn, und ein Rückgabe-Plan setzte beim Speichern die Fristen der Klassen.
-- Ungespeicherte Arbeit lebte nur im Browser-Tab — gegen die Regel „geteilter Zustand
-- immer zentral" (Multi-PC-Betrieb).
--
-- Jetzt ist ein Plan ohne veroeffentlicht_am ein ENTWURF: zentral gespeichert, auf jedem
-- PC gleich, aber für Portal, PDF des Kollegiums und Frist-Kopplung unsichtbar. Erst
-- „Veröffentlichen" stempelt ihn; danach gilt jede weitere Speicherung sofort (die
-- Korrektur-Mail von früher). Ein zweiter Entwurf neben einem veröffentlichten Plan gibt
-- es bewusst nicht — dann wüsste niemand, welcher gilt. Bestehende Pläne galten bereits:
-- Sie bekommen ihren Erstellzeitpunkt als Stempel, damit sich am Live-Betrieb nichts
-- ändert.
ALTER TABLE lmf_plaene
    ADD COLUMN IF NOT EXISTS veroeffentlicht_am TIMESTAMP WITH TIME ZONE;

COMMENT ON COLUMN lmf_plaene.veroeffentlicht_am IS
    'NULL = Entwurf (nur im Planer sichtbar, keine Fristen); gesetzt = gilt für Portal, PDF und Frist-Kopplung.';

UPDATE lmf_plaene SET veroeffentlicht_am = erstellt_am WHERE veroeffentlicht_am IS NULL;
