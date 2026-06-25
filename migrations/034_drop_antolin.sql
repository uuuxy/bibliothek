-- =============================================================================
-- Migration 034: Antolin-Spalten entfernen — Feature stillgelegt
-- =============================================================================
-- Die inoffizielle Antolin-JSON-Schnittstelle (www.antolin.de/all/jsonBuecher.do)
-- wurde mit dem Umzug zu Westermann abgeschaltet: Sie liefert kein JSON mehr,
-- sondern 301 → HTML-Startseite. Live-Abruf und Sync-Job liefen dauerhaft ins
-- Leere; der gesamte Anwendungs- und UI-Code wurde in Commit 0508855 entfernt.
--
-- Diese Spalten (in 015_antolin.sql angelegt, in 032 reconciled) sind seither
-- unreferenziert und durchgängig NULL — das Feature lieferte nie Daten. Sie
-- werden hier rückstandsfrei entfernt.
-- =============================================================================

ALTER TABLE buecher_titel DROP COLUMN IF EXISTS antolin_stufen;
ALTER TABLE buecher_titel DROP COLUMN IF EXISTS antolin_punkte;
ALTER TABLE buecher_titel DROP COLUMN IF EXISTS antolin_geprueft_am;
