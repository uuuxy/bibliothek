-- =============================================================================
-- Migration 186: Die Titelsuche liest einen vorbereiteten Suchtext je Titel
-- =============================================================================
-- Die Titelsuche des Programms (repository.sqlTitelTrifft) verglich je Titel und Suchwort vier
-- Felder über suchnorm, also über unaccent mit seinem Wörterbuch. Bei 13.674 Titeln kostete das
-- jede Suche rund 40 ms, auch wenn nichts traf (gemessen am lokalen Stack), an der Theke bei
-- jedem Tippen. suchtext trägt die vier Felder schon normalisiert, getrennt durch einen
-- Zeilenumbruch; ein Suchwort enthält keinen und trifft deshalb nie über die Grenze zweier
-- Felder. Die Regel bleibt dieselbe.
--
-- Ohne Messung: Die Spalte wird aus vorhandenen Feldern berechnet und ändert keinen Wert. Das
-- Anlegen schreibt die Tabelle einmal neu (lokal 0,3 s bei 13.674 Titeln).
-- =============================================================================

ALTER TABLE buecher_titel ADD COLUMN IF NOT EXISTS suchtext TEXT GENERATED ALWAYS AS (
    suchnorm(coalesce(titel, '')) || E'\n' ||
    suchnorm(coalesce(autor, '')) || E'\n' ||
    suchnorm(coalesce(subject, '')) || E'\n' ||
    suchnorm(coalesce(signatur, ''))
) STORED;
