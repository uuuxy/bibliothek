-- Migration 103: Die Zahl „erfasst" eines abgeschlossenen Inventur-Durchgangs wird
-- festgeschrieben.
--
-- Frage 12 „Gegenrichtung Schema" (06.09.2026): `inventur_erfassungen.exemplar_id` steht
-- auf ON DELETE CASCADE. Die Zahl der erfassten Exemplare wurde bisher bei JEDEM Blick
-- live gezählt — ein später gelöschtes Exemplar (Verlust endgültig, Titel gelöscht,
-- ausgesondert) senkte damit rückwirkend das Ergebnis eines längst abgeschlossenen
-- Durchgangs. In derselben Zeile stand `verloren_gemeldet` fest: „312 erfasst, 4
-- verloren" wurde über die Monate zu „298 erfasst, 4 verloren". Zwei Zahlen desselben
-- Berichts, von denen nur eine altert.
--
-- Ein Inventur-Durchgang ist die Abschrift einer körperlichen Zählung. Was einmal gezählt
-- wurde, ändert sich nicht mehr, weil ein Buch später aus dem Bestand fällt.
--
-- Für BESTEHENDE abgeschlossene Zeilen wird der heutige (bereits abgedriftete) Stand
-- eingefroren — mehr ist nicht rekonstruierbar. Offene Sessions bleiben NULL und zählen
-- weiter live; sie werden beim Abschluss gestempelt.

ALTER TABLE inventur_sessions ADD COLUMN IF NOT EXISTS erfasst_gemeldet INT;

UPDATE inventur_sessions s
SET erfasst_gemeldet = (SELECT count(*) FROM inventur_erfassungen e WHERE e.session_id = s.id)
WHERE s.abgeschlossen_am IS NOT NULL AND s.erfasst_gemeldet IS NULL;

COMMENT ON COLUMN inventur_sessions.erfasst_gemeldet IS
  'Beim Abschluss festgeschriebene Zahl der Erfassungen. NULL = noch offen (dann live zählen). Migration 103.';
