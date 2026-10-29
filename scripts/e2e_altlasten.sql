-- Räumt den Bodensatz weg, den die E2E-Suite in Bestand und Ausleihen hinterlässt.
--
-- Warum nicht im globalen Teardown:
-- Der Teardown (frontend/e2e/global-teardown.js) räumt Lieferanten, Testbestellungen und
-- Klassensatz-Reservierungen ab — alles Tabellen, bei denen ein zu weites Muster wenig
-- anrichten kann. Bestand und Ausleihen sind eine andere Klasse:
--
--   ausleihen.exemplar_id      → RESTRICT   (blockiert das Löschen des Exemplars)
--   schadensfaelle.exemplar_id → RESTRICT
--
-- Ein Löschen der Titel scheitert also am Fremdschlüssel, solange Ausleihen daran hängen;
-- es bräuchte eine Kette über vier Tabellen. Eine automatisch nach jedem Lauf feuernde
-- Kette, die `ausleihen` anfasst, lohnt das nicht: Ein Fehler im Zuschnitt löscht dort
-- echte Ausleihhistorie. Deshalb liegt das hier als bewusst zu startendes Werkzeug —
-- dieselbe Entscheidung wie bei repair_*.sql und tabula_rasa.sql.
--
-- Ein vollständiger e2e-Lauf hinterlässt einige Dutzend Titel und rund hundert Exemplare,
-- und über Monate summiert sich das.
--
-- ZUSCHNITT: ausschließlich Titel, die mit 'E2E ' oder 'E2E-' beginnen; die Specs schreiben
-- beides („E2E Lehrerwunsch …", „E2E-Bestaetigung …"). Mit 'E2E ' allein blieben neun von
-- zehn Rest-Exemplaren liegen. Gegengeprüft: Es trifft null Titel aus dem
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
SELECT count(*) AS "E2E-Titel"      FROM buecher_titel WHERE titel ~ '^E2E[ -]';
SELECT count(*) AS "Exemplare daran" FROM buecher_exemplare e
  JOIN buecher_titel t ON t.id = e.titel_id WHERE t.titel ~ '^E2E[ -]';
SELECT count(*) AS "Ausleihen daran" FROM ausleihen a
  JOIN buecher_exemplare e ON e.id = a.exemplar_id
  JOIN buecher_titel t ON t.id = e.titel_id WHERE t.titel ~ '^E2E[ -]';

-- Reihenfolge folgt den RESTRICT-Fremdschlüsseln: erst was blockiert, dann das Blockierte.
DELETE FROM schadensfaelle s
 USING buecher_exemplare e, buecher_titel t
 WHERE s.exemplar_id = e.id AND e.titel_id = t.id AND t.titel ~ '^E2E[ -]';

DELETE FROM ausleihen a
 USING buecher_exemplare e, buecher_titel t
 WHERE a.exemplar_id = e.id AND e.titel_id = t.id AND t.titel ~ '^E2E[ -]';

-- Regel der Auflagen wie im Programm (repository.ladeWerkeDerTitel, raeumeWerkeAuf): unter
-- der Sperre der Auflagen die Werke der Titel merken, die gleich fallen.
DO $$ BEGIN PERFORM pg_advisory_xact_lock(7502026); END $$;
CREATE TEMP TABLE werke_der_geloeschten ON COMMIT DROP AS
    SELECT DISTINCT werk_id FROM buecher_titel WHERE titel ~ '^E2E[ -]' AND werk_id IS NOT NULL;

-- Exemplare gehen per ON DELETE CASCADE mit dem Titel; ebenso Vormerkungen,
-- Klassensatz-Reservierungen, class_books und inventur_erfassungen.
DELETE FROM buecher_titel WHERE titel ~ '^E2E[ -]';

-- Ein Werk mit weniger als zwei Titeln fällt; ein übriger Titel steht danach für sich.
UPDATE buecher_titel SET werk_id = NULL
 WHERE werk_id IN (SELECT werk_id FROM werke_der_geloeschten)
   AND werk_id IN (SELECT werk_id FROM buecher_titel WHERE werk_id IS NOT NULL
                   GROUP BY werk_id HAVING count(*) < 2);
DELETE FROM werke
 WHERE id IN (SELECT werk_id FROM werke_der_geloeschten)
   AND NOT EXISTS (SELECT 1 FROM buecher_titel t WHERE t.werk_id = werke.id);

-- Ohne -v ausfuehren=ja bleibt es bei der Vorschau; die letzte Ausgabezeile sagt, was
-- passiert ist. Kein Tausch von COMMIT gegen ROLLBACK von Hand: Ein solcher Lauf druckt
-- seine Zahlen, meldet keinen Fehler und ändert trotzdem nichts (oder alles).
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
