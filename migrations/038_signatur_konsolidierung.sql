-- =============================================================================
-- Migration 038: Signatur-Konsolidierung — JSONB-Duplikat in die echte Spalte
-- =============================================================================
-- Es existierten ZWEI Signatur-Welten: Littera-Importe schrieben die Spalte
-- buecher_titel.signatur, das Buchformular pflegte parallel
-- erweiterte_eigenschaften->>'signatur' (JSONB). Die Signatur klebt physisch
-- auf dem Buchrücken — eine einzige Quelle der Wahrheit ist Pflicht.
--
-- Diese Migration übernimmt JSONB-Werte in die Spalte (nur wo die Spalte leer
-- ist — vorhandene Littera-Signaturen gewinnen) und entfernt den JSONB-Key,
-- damit keine Doppelpflege mehr entstehen kann. Das Formular schreibt ab
-- jetzt ausschließlich die Spalte.
-- =============================================================================

UPDATE buecher_titel
SET signatur = erweiterte_eigenschaften->>'signatur'
WHERE COALESCE(signatur, '') = ''
  AND COALESCE(erweiterte_eigenschaften->>'signatur', '') != '';

UPDATE buecher_titel
SET erweiterte_eigenschaften = erweiterte_eigenschaften - 'signatur'
WHERE erweiterte_eigenschaften ? 'signatur';
