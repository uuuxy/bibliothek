-- =============================================================================
-- Migration 185: Gelöschte Exemplare behalten Einkaufspreis und Art des Titels
-- =============================================================================
-- Die Statistik zählt Verluste wie das Abgangsbuch, auch endgültig gelöschte
-- (repository.LadeBestandKennzahlen). Löschte das Team verlorene Bücher nach der Inventur oder
-- einen ganzen Titel, sanken „Verluste & Schäden“, die Quote und der Wiederbeschaffungswert
-- rückwirkend. Für den Wiederbeschaffungswert braucht die Zeile den Einkaufspreis, für den Filter
-- Lernmittel/Freihand die Art des Titels (buecher_titel.ist_lernmittel); beides schreibt
-- repository.loescheExemplareMitBuchzeile.
--
-- Zeilen von vor dieser Migration tragen beides nicht (NULL): Ihr Preis fehlt im
-- Wiederbeschaffungswert, und sie zählen nur ohne Filter. Ohne Messung: Die Anweisungen ändern
-- keine Zeile. Idempotent: IF NOT EXISTS.
-- =============================================================================

ALTER TABLE geloeschte_exemplare ADD COLUMN IF NOT EXISTS einkaufspreis DECIMAL(10,2);
ALTER TABLE geloeschte_exemplare ADD COLUMN IF NOT EXISTS ist_lernmittel BOOLEAN;
