-- entferne_demo_daten.sql
--
-- Entfernt den Datensatz aus scripts/seed_demo.sql (Schüler DEMO-S-*, Exemplare DEMO-B-*,
-- Titel „DEMO-Titel …") von einem System, das seitdem BENUTZT wurde.
--
-- Warum nicht der Block am Anfang von seed_demo.sql: Der löscht nur Ausleihen auf
-- Demo-Exemplaren, dann Exemplare, Titel und Schüler. Auf einem benutzten System hängt
-- mehr daran — Schäden (RESTRICT, der DELETE bricht ab), Ausleihen echter Exemplare an
-- Demo-Schüler (RESTRICT), Bescheide und Inventur-Verluste (SET NULL, bleiben als Reste
-- ohne Bezug stehen), Protokollzeilen. Dieses Skript räumt all das in einer Transaktion
-- und zeigt VORHER jede Stelle, an der Demo-Daten mit echten verflochten sind.
--
-- Die Demo-Exemplare hängen an ECHTEN Titeln (seed_demo.sql nimmt vorhandene Titel);
-- gelöscht werden nur die Exemplare. „DEMO-Titel" entstehen nur in einer leeren DB und
-- werden nur gelöscht, wenn kein echtes Exemplar daran hängt.
--
-- ABLAUF — zwei Aufrufe, die Datei bleibt unverändert:
--
--   1. Vorschau (ändert NICHTS, endet mit ROLLBACK):
--        docker exec -i bibliothek-db sh -c 'psql -U $POSTGRES_USER -d $POSTGRES_DB -v ON_ERROR_STOP=1' < scripts/entferne_demo_daten.sql
--
--   2. Ausführen (endet mit COMMIT) — dieselbe Zeile plus -v ausfuehren=ja:
--        docker exec -i bibliothek-db sh -c 'psql -U $POSTGRES_USER -d $POSTGRES_DB -v ON_ERROR_STOP=1 -v ausfuehren=ja' < scripts/entferne_demo_daten.sql
--
-- Die letzte Zeile der Ausgabe sagt, was passiert ist: „ROLLBACK" = Vorschau,
-- „COMMIT" = geschrieben.
--
-- VORHER ein Backup ziehen (siehe docs/DEPLOYMENT.md); der einzige Rückweg ist das Backup.

\set QUIET on
\pset footer off
BEGIN;

CREATE TEMP TABLE demo_s ON COMMIT DROP AS
    SELECT id FROM schueler WHERE barcode_id LIKE 'DEMO-S-%';
CREATE TEMP TABLE demo_b ON COMMIT DROP AS
    SELECT id FROM buecher_exemplare WHERE barcode_id LIKE 'DEMO-B-%';
CREATE TEMP TABLE demo_t ON COMMIT DROP AS
    SELECT t.id FROM buecher_titel t
     WHERE t.titel LIKE 'DEMO-Titel %'
       AND NOT EXISTS (SELECT 1 FROM buecher_exemplare e
                        WHERE e.titel_id = t.id AND e.id NOT IN (SELECT id FROM demo_b));
CREATE TEMP TABLE demo_ausleihen ON COMMIT DROP AS
    SELECT id FROM ausleihen
     WHERE schueler_id IN (SELECT id FROM demo_s) OR exemplar_id IN (SELECT id FROM demo_b);

\echo
\echo '=== Was entfernt wird ==='
SELECT 'Demo-Schüler' AS was, count(*) AS anzahl FROM demo_s
UNION ALL SELECT 'Demo-Exemplare', count(*) FROM demo_b
UNION ALL SELECT 'Demo-Titel ohne echtes Exemplar', count(*) FROM demo_t
UNION ALL SELECT 'Demo-Titel, die BLEIBEN (echtes Exemplar hängt dran)', count(*) FROM buecher_titel
           WHERE titel LIKE 'DEMO-Titel %' AND id NOT IN (SELECT id FROM demo_t)
UNION ALL SELECT 'Ausleihen', count(*) FROM demo_ausleihen
UNION ALL SELECT 'davon offen', count(*) FROM ausleihen WHERE id IN (SELECT id FROM demo_ausleihen) AND rueckgabe_am IS NULL
UNION ALL SELECT 'Schadensfälle', count(*) FROM schadensfaelle
           WHERE schueler_id IN (SELECT id FROM demo_s) OR exemplar_id IN (SELECT id FROM demo_b)
UNION ALL SELECT 'Bescheide an Demo-Schüler', count(*) FROM schadensersatz_bescheide WHERE schueler_id IN (SELECT id FROM demo_s)
UNION ALL SELECT 'Inventur-Verluste auf Demo-Exemplaren', count(*) FROM inventur_verluste WHERE exemplar_id IN (SELECT id FROM demo_b)
UNION ALL SELECT 'Vormerkungen von Demo-Schülern', count(*) FROM vormerkungen WHERE schueler_id IN (SELECT id FROM demo_s)
UNION ALL SELECT 'Protokollzeilen audit_log', count(*) FROM audit_log
           WHERE (tabelle = 'schueler' AND datensatz_id IN (SELECT id FROM demo_s))
              OR (tabelle = 'ausleihen' AND details->>'schueler_id' IN (SELECT id::text FROM demo_s))
UNION ALL SELECT 'Protokollzeilen audit_logs', count(*) FROM audit_logs
           WHERE details->>'schueler_id' IN (SELECT id::text FROM demo_s);

\echo
\echo '=== Verflechtung mit ECHTEN Daten — vor dem Ausführen lesen ==='
\echo 'Leere Tabellen sind gut. Jede Zeile hier betrifft einen echten Schüler oder ein echtes Exemplar.'
\echo
\echo '1) Echte Exemplare, offen an Demo-Schüler verliehen — sind danach wieder frei:'
SELECT e.barcode_id AS exemplar, t.titel, s.barcode_id AS demo_schueler, a.ausgeliehen_am::date
  FROM ausleihen a
  JOIN buecher_exemplare e ON e.id = a.exemplar_id
  JOIN buecher_titel t ON t.id = e.titel_id
  JOIN schueler s ON s.id = a.schueler_id
 WHERE a.schueler_id IN (SELECT id FROM demo_s) AND a.exemplar_id NOT IN (SELECT id FROM demo_b)
   AND a.rueckgabe_am IS NULL
 ORDER BY e.barcode_id LIMIT 50;

\echo '2) Echte Schüler mit Ausleihen auf Demo-Exemplaren — diese Einträge verschwinden aus ihrer Ausleih-Historie:'
SELECT s.barcode_id AS schueler, s.klasse, e.barcode_id AS demo_exemplar, a.ausgeliehen_am::date,
       CASE WHEN a.rueckgabe_am IS NULL THEN 'OFFEN' ELSE 'zurück' END AS stand
  FROM ausleihen a
  JOIN buecher_exemplare e ON e.id = a.exemplar_id
  JOIN schueler s ON s.id = a.schueler_id
 WHERE a.exemplar_id IN (SELECT id FROM demo_b) AND a.schueler_id NOT IN (SELECT id FROM demo_s)
 ORDER BY s.barcode_id LIMIT 50;

\echo '3) Schäden echter Schüler auf Demo-Exemplaren — werden mit gelöscht:'
SELECT s.barcode_id AS schueler, e.barcode_id AS demo_exemplar, sf.betrag, sf.ist_bezahlt, sf.bescheid_id IS NOT NULL AS auf_bescheid
  FROM schadensfaelle sf
  JOIN buecher_exemplare e ON e.id = sf.exemplar_id
  LEFT JOIN schueler s ON s.id = sf.schueler_id
 WHERE sf.exemplar_id IN (SELECT id FROM demo_b)
   AND (sf.schueler_id IS NULL OR sf.schueler_id NOT IN (SELECT id FROM demo_s))
 ORDER BY 1 LIMIT 50;

\echo '4) Echte Exemplare im Abholfach eines Demo-Schülers — die Vormerkung fällt, der Nächste auf der Warteliste rückt NICHT von selbst nach:'
SELECT e.barcode_id AS exemplar, t.titel, s.barcode_id AS demo_schueler
  FROM vormerkungen v
  JOIN buecher_exemplare e ON e.id = v.bereitgestellt_exemplar_id
  JOIN buecher_titel t ON t.id = v.titel_id
  JOIN schueler s ON s.id = v.schueler_id
 WHERE v.schueler_id IN (SELECT id FROM demo_s) AND v.status = 'abholbereit'
   AND v.bereitgestellt_exemplar_id NOT IN (SELECT id FROM demo_b)
 ORDER BY e.barcode_id LIMIT 50;

\echo '5) Vormerkungen echter Schüler, bereitgestellt auf einem Demo-Exemplar — gehen zurück auf „wartend":'
SELECT s.barcode_id AS schueler, t.titel, e.barcode_id AS demo_exemplar
  FROM vormerkungen v
  JOIN buecher_exemplare e ON e.id = v.bereitgestellt_exemplar_id
  JOIN buecher_titel t ON t.id = v.titel_id
  JOIN schueler s ON s.id = v.schueler_id
 WHERE v.bereitgestellt_exemplar_id IN (SELECT id FROM demo_b)
   AND v.schueler_id NOT IN (SELECT id FROM demo_s)
 ORDER BY s.barcode_id LIMIT 50;

-- SPERRE (17.09.2026): Ein Schadensfall eines ECHTEN Schülers auf einem Demo-Exemplar
-- wird NICHT mitgelöscht — das Skript bricht ab.
--
-- Punkt 3 der Vorschau zeigt diese Zeilen seit jeher an; gelöscht wurden sie trotzdem,
-- auch bezahlte und solche auf einem Bescheid. Das ist keine Aufräumarbeit mehr: Eine
-- bezahlte Forderung ist ein Kassenbeleg, eine auf einem Bescheid ist Teil eines
-- Verwaltungsakts, und eine offene ist Geld, das jemand schuldet. Auf dem Server war die
-- Vorschau am 13.09.2026 leer (0 Zeilen) — der Lauf blieb also folgenlos; beim nächsten
-- Mal muss es keiner sein.
--
-- Wer solche Zeilen hat, entscheidet sie einzeln (stornieren, umbuchen) und lässt das
-- Skript danach erneut laufen. Die Meldung nennt die Zahl, die Liste steht darüber.
DO $$
DECLARE betroffen int;
BEGIN
    SELECT count(*) INTO betroffen
      FROM schadensfaelle sf
     WHERE sf.exemplar_id IN (SELECT id FROM demo_b)
       AND (sf.schueler_id IS NULL OR sf.schueler_id NOT IN (SELECT id FROM demo_s));
    IF betroffen > 0 THEN
        RAISE EXCEPTION 'Abbruch: % Schadensfall/-fälle echter Leser hängen an Demo-Exemplaren (Liste in Punkt 3 der Vorschau). Diese Fälle erst einzeln entscheiden — das Skript würde Kassenbelege und Forderungen mitlöschen.', betroffen;
    END IF;
END $$;

-- Löschen, von den Blättern zur Wurzel. Reihenfolge wegen RESTRICT: Schäden und Ausleihen
-- vor Exemplaren und Schülern.
DELETE FROM schadensfaelle
 WHERE schueler_id IN (SELECT id FROM demo_s) OR exemplar_id IN (SELECT id FROM demo_b);
-- Bescheide an Demo-Schüler. Die Nummer wird dabei nicht wieder frei: Sie kommt aus
-- schadensersatz_nummern, nicht aus dieser Tabelle.
DELETE FROM schadensersatz_bescheide WHERE schueler_id IN (SELECT id FROM demo_s);
DELETE FROM ausleihen WHERE id IN (SELECT id FROM demo_ausleihen);
-- SET NULL ließe Verlustzeilen ohne Exemplar stehen, die der Fehlbestandsbericht zeigt.
DELETE FROM inventur_verluste WHERE exemplar_id IN (SELECT id FROM demo_b);
DELETE FROM audit_log
 WHERE (tabelle = 'schueler' AND datensatz_id IN (SELECT id FROM demo_s))
    OR (tabelle = 'ausleihen' AND details->>'schueler_id' IN (SELECT id::text FROM demo_s));
DELETE FROM audit_logs WHERE details->>'schueler_id' IN (SELECT id::text FROM demo_s);
-- Exemplare: der Trigger trg_exemplar_geloescht_abholfach setzt Vormerkungen, die auf
-- ein Demo-Exemplar zeigen, auf „wartend"; inventur_erfassungen fallen per CASCADE.
DELETE FROM buecher_exemplare WHERE id IN (SELECT id FROM demo_b);
-- Schüler: Fotos und Vormerkungen fallen per CASCADE.
DELETE FROM schueler WHERE id IN (SELECT id FROM demo_s);
DELETE FROM buecher_titel WHERE id IN (SELECT id FROM demo_t);

\echo
\echo '=== Danach (muss überall 0 sein, außer bei bleibenden Demo-Titeln oben) ==='
SELECT 'Demo-Schüler' AS was, count(*) AS anzahl FROM schueler WHERE barcode_id LIKE 'DEMO-S-%'
UNION ALL SELECT 'Demo-Exemplare', count(*) FROM buecher_exemplare WHERE barcode_id LIKE 'DEMO-B-%'
UNION ALL SELECT 'Demo-Titel ohne echtes Exemplar', count(*) FROM buecher_titel t
           WHERE t.titel LIKE 'DEMO-Titel %' AND NOT EXISTS (SELECT 1 FROM buecher_exemplare e WHERE e.titel_id = t.id)
UNION ALL SELECT 'Inventur-Verluste ohne Exemplar', count(*) FROM inventur_verluste WHERE exemplar_id IS NULL AND barcode_id LIKE 'DEMO-B-%'
UNION ALL SELECT 'Abholbereit ohne Exemplar', count(*) FROM vormerkungen WHERE status = 'abholbereit' AND bereitgestellt_exemplar_id IS NULL;

-- Ohne -v ausfuehren=ja bleibt es bei der Vorschau. Geprüft wird der WERT, nicht nur ob
-- die Variable gesetzt ist (Muster aus repair_fach_kategorie.sql).
\if :{?ausfuehren}
SELECT :'ausfuehren' = 'ja' AS schreiben \gset
\else
SELECT false AS schreiben \gset
\endif

\if :schreiben
\echo 'COMMIT'
COMMIT;
\else
\echo '>>> VORSCHAU — nichts geschrieben. Zum Ausführen: -v ausfuehren=ja'
\echo 'ROLLBACK'
ROLLBACK;
\endif
