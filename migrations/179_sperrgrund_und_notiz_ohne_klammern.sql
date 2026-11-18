-- =============================================================================
-- Migration 179: Sperrgründe der Abgänger und die Notiz bestellter Exemplare ohne Klammern
-- =============================================================================
-- M3 (Content design, Grammar and punctuation): Klammern stehen nur um ein Kürzel oder eine
-- Quelle (docs/STIL.md 11). Der Server schreibt die Sperrgründe der Abgänger
-- (repository.AbgaengerSperrgrundOffen und Geschwister) und die Notiz eines bestellten
-- Exemplars ohne Vorab-Barcode (internal/service/bestellung_anlegen.go) jetzt ohne Klammern;
-- gespeicherte Zeilen tragen noch die alte Fassung. Nachgezogen wird nur der genaue alte Text,
-- ein Grund von Hand bleibt stehen. Das Präfix „Automatisierte Abgänger-Sperre“, an dem SQL die
-- Automatik erkennt, bleibt gleich.
--
-- Gemessen am Testserver am 18.11.2026 (lesend): keine Zeile mit einem der drei Sperrgründe,
-- 3 Exemplare mit der Notiz „Bestellt (ohne Vorab-Barcode) - …“.
UPDATE schueler
   SET block_reason = CASE block_reason
       WHEN 'Automatisierte Abgänger-Sperre (offene Vorgänge)'
           THEN 'Automatisierte Abgänger-Sperre wegen offener Vorgänge'
       WHEN 'Automatisierte Abgänger-Sperre (Karenzzeit vor Anonymisierung)'
           THEN 'Automatisierte Abgänger-Sperre in der Karenzzeit vor der Anonymisierung'
       WHEN 'Automatisierte Abgänger-Sperre (Schuljahreswechsel)'
           THEN 'Automatisierte Abgänger-Sperre zum Schuljahreswechsel'
       END
 WHERE block_reason IN ('Automatisierte Abgänger-Sperre (offene Vorgänge)',
                        'Automatisierte Abgänger-Sperre (Karenzzeit vor Anonymisierung)',
                        'Automatisierte Abgänger-Sperre (Schuljahreswechsel)');

UPDATE buecher_exemplare
   SET zustand_notiz = 'Bestellt ohne Vorab-Barcode - ' || substr(zustand_notiz, length('Bestellt (ohne Vorab-Barcode) - ') + 1)
 WHERE zustand_notiz LIKE 'Bestellt (ohne Vorab-Barcode) - %';
