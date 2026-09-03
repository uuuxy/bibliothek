-- Räumt den Bodensatz weg, den die E2E-Suite in Bestand und Ausleihen hinterlässt.
--
-- WARUM NICHT IM GLOBALEN TEARDOWN (12.08.2026):
-- Der Teardown (frontend/e2e/global-teardown.js) räumt Lieferanten, Testbestellungen und
-- Klassensatz-Reservierungen ab — alles Tabellen, bei denen ein zu weites Muster wenig
-- anrichten kann. Bestand und Ausleihen sind eine andere Klasse:
--
--   ausleihen.exemplar_id      → RESTRICT   (blockiert das Löschen des Exemplars)
--   schadensfaelle.exemplar_id → RESTRICT
--
-- Ein Löschen der Titel scheitert also am Fremdschlüssel, solange Ausleihen daran hängen;
-- es bräuchte eine Kette über vier Tabellen. Eine automatisch nach jedem Lauf feuernde
-- Kette, die `ausleihen` anfasst, ist mir das nicht wert: Ein Fehler im Zuschnitt löscht
-- dort echte Ausleihhistorie. Deshalb liegt das hier als bewusst zu startendes Werkzeug —
-- dieselbe Entscheidung wie bei repair_*.sql und tabula_rasa.sql.
--
-- GEMESSEN (12.08.2026, Entwicklungsdatenbank): Ein vollständiger e2e-Lauf hinterlässt
-- rund 42 Titel, 75 Exemplare und 22 Ausleihen. Über Monate summiert sich das — dieselbe
-- Mechanik hatte schon 224 Testlieferanten und einen Reservierungszähler von 223 erzeugt.
--
-- ZUSCHNITT: ausschließlich Titel mit dem Präfix 'E2E ' — dasselbe Muster, das der
-- Teardown bereits für Reservierungen nutzt. Gegengeprüft: Es trifft null Titel aus dem
-- Littera-Altbestand (erweiterte_eigenschaften ? 'littera_id'). Schüler bleiben
-- UNANGETASTET: Ihre Testnamen sind uneinheitlich (Testschueler, Zweitleiher-…,
-- Klassenkind…, Zzunsichtbar…), ein Muster darauf wäre geraten.
--
--   Aufruf:  docker exec -i bibliothek-db-local psql -U postgres -d bibliothek \
--              -v ON_ERROR_STOP=1 < scripts/e2e_altlasten.sql
--
--   Vorher ein Backup. Das Skript löscht.

BEGIN;

-- Was steht an? (Vor dem Löschen, damit die Zahlen im Protokoll stehen.)
SELECT count(*) AS "E2E-Titel"      FROM buecher_titel WHERE titel LIKE 'E2E %';
SELECT count(*) AS "Exemplare daran" FROM buecher_exemplare e
  JOIN buecher_titel t ON t.id = e.titel_id WHERE t.titel LIKE 'E2E %';
SELECT count(*) AS "Ausleihen daran" FROM ausleihen a
  JOIN buecher_exemplare e ON e.id = a.exemplar_id
  JOIN buecher_titel t ON t.id = e.titel_id WHERE t.titel LIKE 'E2E %';

-- Reihenfolge folgt den RESTRICT-Fremdschlüsseln: erst was blockiert, dann das Blockierte.
DELETE FROM schadensfaelle s
 USING buecher_exemplare e, buecher_titel t
 WHERE s.exemplar_id = e.id AND e.titel_id = t.id AND t.titel LIKE 'E2E %';

DELETE FROM ausleihen a
 USING buecher_exemplare e, buecher_titel t
 WHERE a.exemplar_id = e.id AND e.titel_id = t.id AND t.titel LIKE 'E2E %';

-- Exemplare gehen per ON DELETE CASCADE mit dem Titel; ebenso Vormerkungen,
-- Klassensatz-Reservierungen, class_books und inventur_erfassungen.
DELETE FROM buecher_titel WHERE titel LIKE 'E2E %';

-- Ohne -v ausfuehren=ja bleibt es bei der Vorschau; die letzte Ausgabezeile sagt, was
-- passiert ist. Bis zum 03.09.2026 stand hier ein unbedingtes COMMIT und in SCRIPTS.md
-- die Empfehlung, es für den Probelauf von Hand gegen ROLLBACK zu tauschen — genau die
-- Handarbeit, die an repair_fach_kategorie.sql einen ganzen Abend gekostet hat: Der Lauf
-- druckt seine Zahlen, meldet keinen Fehler und ändert trotzdem nichts (oder alles).
\if :{?ausfuehren}
SELECT :'ausfuehren' = 'ja' AS schreiben \gset
\else
SELECT false AS schreiben \gset
\endif

\if :schreiben
COMMIT;
\else
\echo '>>> VORSCHAU — nichts gelöscht. Zum Ausführen: -v ausfuehren=ja'
ROLLBACK;
\endif
