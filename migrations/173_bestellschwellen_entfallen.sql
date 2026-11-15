-- =============================================================================
-- Migration 173: Die Spalte meldebestand und die Einstellung bestellbedarf_schwelle entfallen
-- =============================================================================
-- Beide waren Schwellen des Bestellbedarfs. Er rechnet nach der Zahl der Kinder, die ein
-- Schulbuch im nächsten Schuljahr brauchen (repository/bestellbedarf.go); seitdem liest und
-- schreibt keine Stelle die Spalte buecher_titel.meldebestand oder die Einstellung
-- bestellbedarf_schwelle. Mit der Spalte fällt ihre Bedingung chk_meldebestand_nonneg.
--
-- Gemessen am Testserver am 15.11.2026 (lesend): Alle 13.060 Titel tragen meldebestand = 5, die
-- Vorgabe der Spalte; gepflegt hat sie niemand.
--
-- Nicht umkehrbar: Die Werte stehen danach nur noch in der Sicherung, die update.sh vor dem
-- Einspielen anlegt.
--
-- Idempotent: Beim zweiten Lauf fehlen Spalte und Einstellung schon.
-- =============================================================================

ALTER TABLE buecher_titel DROP COLUMN IF EXISTS meldebestand;

DELETE FROM system_einstellungen WHERE schluessel = 'bestellbedarf_schwelle';
