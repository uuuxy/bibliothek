-- =============================================================================
-- Migration 165: Eine Reihenfolge für Texte, deutsch und mit Zahlen nach ihrer Größe
-- =============================================================================
-- Die Datenbank ordnet ohne Sprachregeln (datlocprovider = c), also nach Codepunkt:
-- „Mathe 10" vor „Mathe 5", „Zebra" vor „apfel", „Ägypten" hinter allem. Die Kollation
-- deutsch ordnet wie Intl.Collator('de', {numeric: true}) im Browser und wie
-- sortiereBuecherNachTitel in Go; alle drei prüft dieselbe Liste von Fällen
-- (frontend/src/lib/utils/titelReihenfolge.faelle.json).
--
-- Kein Index benutzt sie: Bringt ein neues Image eine andere ICU-Fassung, ändert sich nur die
-- Reihenfolge einer Ausgabe, kein gespeicherter Index. Postgres meldet die neue Fassung dann
-- als Warnung; ALTER COLLATION deutsch REFRESH VERSION stellt sie ab.
--
-- Idempotent: Beim zweiten Lauf gibt es die Kollation schon.
-- =============================================================================

CREATE COLLATION IF NOT EXISTS deutsch (provider = icu, locale = 'de-u-kn');
