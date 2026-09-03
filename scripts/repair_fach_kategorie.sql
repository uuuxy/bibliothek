-- repair_fach_kategorie.sql
--
-- Einmalige Datenreparatur für Bestände, die VOR dem 03.09.2026 über den CSV-/Excel-
-- Bestandsimport geladen wurden.
--
-- Die Kategorie-Spalte der aus dem Littera-PDF gewonnenen Bestands-CSV enthält keine
-- Fächer, sondern Standorttexte („Buch Pg/Kaf 078829 1. Aufl.", „Buch Deu 6/Cha 126
-- Exemplare 1. Auflage"). Der Import nahm die Spalte ungeprüft als Fach-Rückfall: Jeder
-- dieser Texte wurde eine eigene Zeile in systematik_kategorien und stand als „Fach" am
-- Titel — im Portal-Reiter Schulbücher eine Kachel je Titel, in der Fach-Auswahl der
-- Buchmaske 1.677 Einträge. Seit 03.09.2026 wird die Kategorie nur noch übernommen, wenn
-- sie als Ganzes ein Fach benennt (lmf.FachExakt); dieses Skript räumt den Altbestand ab.
--
-- Regeln (bewusst konservativ):
--   1. Fach am Titel bleibt nur, wenn es ein kanonisches Fach ist (Liste unten) oder eine
--      bekannte Schreibvariante („Mathe", „Geographie", „Politik"), die auf das kanonische
--      Fach gezogen wird.
--   2. Alles andere am Titel wird zu NULL („ohne Fach"). Der Text geht nicht verloren, er
--      steht weiterhin in der Systematik-Zeile, bis Schritt 3 sie löscht.
--   3. Systematik-Zeilen werden nur gelöscht, wenn nach Schritt 2 KEIN Titel mehr daran
--      hängt UND sie kein kanonisches Fach sind. Handgepflegte Fächer ohne Titel sind
--      davon ebenfalls betroffen — vorher die Liste prüfen (Ausgabe „wird gelöscht").
--
-- Ablauf: erst mit ROLLBACK am Ende laufen lassen und die Ausgaben lesen, dann COMMIT.
-- VORHER ein Backup ziehen: pg_dump -U postgres -d bibliothek | gzip > backup.sql.gz

BEGIN;

CREATE TEMP TABLE kanon (bezeichnung) ON COMMIT DROP AS VALUES
    ('Mathematik'), ('Deutsch'), ('Englisch'), ('Französisch'), ('Latein'), ('Spanisch'),
    ('Geschichte'), ('Politik und Wirtschaft'), ('Erdkunde'), ('Biologie'), ('Chemie'),
    ('Physik'), ('Musik'), ('Kunst'), ('Religion'), ('Ethik'), ('Philosophie'),
    ('Informatik'), ('Sport'), ('Arbeitslehre'), ('Darstellendes Spiel'), ('Naturwissenschaften');

-- Schreibvarianten → kanonisch (dieselbe Zuordnung wie pkg/lmf schlagwortFach).
CREATE TEMP TABLE variante (alt, neu) ON COMMIT DROP AS VALUES
    ('mathe', 'Mathematik'), ('geographie', 'Erdkunde'), ('geografie', 'Erdkunde'),
    ('politik', 'Politik und Wirtschaft'), ('sozialkunde', 'Politik und Wirtschaft');

-- Vorschau: was passiert.
SELECT 'Titel mit Fach gesamt' AS was, count(*) FROM buecher_titel WHERE subject IS NOT NULL AND subject <> ''
UNION ALL
SELECT 'davon bleibt (kanonisch)', count(*) FROM buecher_titel WHERE subject IN (SELECT bezeichnung FROM kanon)
UNION ALL
SELECT 'davon wird umbenannt (Variante)', count(*) FROM buecher_titel WHERE lower(subject) IN (SELECT alt FROM variante)
UNION ALL
SELECT 'davon wird "ohne Fach"', count(*) FROM buecher_titel
 WHERE subject IS NOT NULL AND subject <> ''
   AND subject NOT IN (SELECT bezeichnung FROM kanon) AND lower(subject) NOT IN (SELECT alt FROM variante);

-- Schritt 1: kanonische Zeilen sicherstellen (Kürzel wie repository.registriereFach: ohne Leerzeichen).
INSERT INTO systematik_kategorien (kuerzel, bezeichnung)
SELECT replace(bezeichnung, ' ', ''), bezeichnung FROM kanon
ON CONFLICT (lower(bezeichnung)) DO NOTHING;

-- Schritt 2a: Varianten auf das kanonische Fach ziehen.
UPDATE buecher_titel t
   SET subject = (SELECT s.bezeichnung FROM systematik_kategorien s
                   WHERE lower(s.bezeichnung) = lower(v.neu)),
       aktualisiert_am = CURRENT_TIMESTAMP
  FROM variante v
 WHERE lower(t.subject) = v.alt;

-- Schritt 2b: alles andere wird „ohne Fach".
UPDATE buecher_titel
   SET subject = NULL, aktualisiert_am = CURRENT_TIMESTAMP
 WHERE subject IS NOT NULL AND subject <> ''
   AND subject NOT IN (SELECT bezeichnung FROM kanon);

-- Schritt 3: verwaiste Nicht-Fächer aus der Systematik — vorher zeigen, dann löschen.
SELECT 'wird gelöscht' AS was, s.bezeichnung
  FROM systematik_kategorien s
 WHERE s.bezeichnung NOT IN (SELECT bezeichnung FROM kanon)
   AND NOT EXISTS (SELECT 1 FROM buecher_titel t WHERE t.subject = s.bezeichnung)
 ORDER BY s.bezeichnung;

DELETE FROM systematik_kategorien s
 WHERE s.bezeichnung NOT IN (SELECT bezeichnung FROM kanon)
   AND NOT EXISTS (SELECT 1 FROM buecher_titel t WHERE t.subject = s.bezeichnung);

-- Ergebnis.
SELECT 'Fächer danach' AS was, string_agg(subject || ' ' || n, ', ' ORDER BY n DESC)
  FROM (SELECT subject, count(*) n FROM buecher_titel WHERE subject IS NOT NULL GROUP BY subject) x
UNION ALL
SELECT 'Systematik-Zeilen danach', count(*)::text FROM systematik_kategorien;

-- Erst lesen, dann entscheiden:
ROLLBACK;
-- COMMIT;
