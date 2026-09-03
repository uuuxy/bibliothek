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
--   1b. Schulbücher (ist_lernmittel) bekommen Fach und Jahrgang aus Signatur bzw.
--      Standorttext („Deu 12", „Buch Bio 7/Nat 106 …") — dieselben Kürzel wie pkg/lmf.
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

-- Schritt 1b: Schulbücher bekommen Fach und Jahrgang aus dem, was Littera hinterließ.
-- Die Signatur („Deu 12", „Ges7", „Re1213", „Bio 7/Nat 106") und der Standorttext
-- („Buch Bio 7/Nat 106 Exemplare …") tragen beides; die Kürzel sind dieselben wie in
-- pkg/lmf (fachKuerzel). Nur Lernmittel, nur wo das Fach noch kein kanonisches ist bzw.
-- der Jahrgang noch auf der Vorgabe 5–10 steht.
CREATE TEMP TABLE kuerzel (k, fach) ON COMMIT DROP AS VALUES
    ('ma','Mathematik'), ('m','Mathematik'), ('deu','Deutsch'), ('d','Deutsch'),
    ('eng','Englisch'), ('e','Englisch'), ('fra','Französisch'), ('f','Französisch'),
    ('lat','Latein'), ('l','Latein'), ('spa','Spanisch'), ('ges','Geschichte'), ('g','Geschichte'),
    ('powi','Politik und Wirtschaft'), ('powie','Politik und Wirtschaft'),
    ('erd','Erdkunde'), ('erdat','Erdkunde'), ('ek','Erdkunde'), ('bio','Biologie'),
    ('che','Chemie'), ('ch','Chemie'), ('phy','Physik'), ('ph','Physik'), ('mus','Musik'),
    ('ku','Kunst'), ('kun','Kunst'), ('rel','Religion'), ('re','Religion'), ('eth','Ethik'),
    ('phil','Philosophie'), ('inf','Informatik'), ('info','Informatik'),
    ('spo','Sport'), ('sposi','Sport'), ('sposii','Sport'), ('arb','Arbeitslehre'),
    ('dsp','Darstellendes Spiel'), ('nawi','Naturwissenschaften');

CREATE TEMP TABLE ableitung ON COMMIT DROP AS
WITH quelle AS (
    SELECT id, subject, jahrgang_von, jahrgang_bis,
           regexp_replace(
               COALESCE(NULLIF(btrim(signatur), ''), regexp_replace(subject, '^Buch ', '')),
               '^LMF[ -]*', '', 'i') AS s
      FROM buecher_titel
     WHERE ist_lernmittel
), zerlegt AS (
    SELECT id, subject, jahrgang_von, jahrgang_bis,
           lower(substring(s from '^([A-Za-zÄÖÜäöüß]+)')) AS kz,
           -- Ziffern direkt hinter dem Kürzel, vor dem „/": „7", „12", „1213" (12–13)
           substring(regexp_replace(s, '^[A-Za-zÄÖÜäöüß]+ ?', '') from '^([0-9]{1,4})') AS ziffern
      FROM quelle
)
SELECT z.id, z.subject AS alt, k.fach,
       CASE WHEN length(ziffern) = 4 THEN left(ziffern, 2)::int
            WHEN ziffern IS NOT NULL THEN ziffern::int END AS von,
       CASE WHEN length(ziffern) = 4 THEN right(ziffern, 2)::int
            WHEN ziffern IS NOT NULL THEN ziffern::int END AS bis,
       z.jahrgang_von AS alt_von, z.jahrgang_bis AS alt_bis
  FROM zerlegt z
  LEFT JOIN kuerzel k ON k.k = z.kz;

SELECT 'Schulbücher gesamt' AS was, count(*)::text FROM buecher_titel WHERE ist_lernmittel
UNION ALL
SELECT 'davon Fach aus Signatur/Standort ableitbar', count(*)::text FROM ableitung WHERE fach IS NOT NULL
UNION ALL
SELECT 'davon Jahrgang ableitbar (5–13)', count(*)::text FROM ableitung WHERE von BETWEEN 5 AND 13 AND bis BETWEEN 5 AND 13
UNION ALL
SELECT 'Fach-Verteilung danach', string_agg(fach || ' ' || n, ', ' ORDER BY n DESC)
  FROM (SELECT fach, count(*) n FROM ableitung WHERE fach IS NOT NULL GROUP BY fach) x;

UPDATE buecher_titel t
   SET subject = (SELECT s.bezeichnung FROM systematik_kategorien s WHERE lower(s.bezeichnung) = lower(a.fach)),
       aktualisiert_am = CURRENT_TIMESTAMP
  FROM ableitung a
 WHERE t.id = a.id AND a.fach IS NOT NULL
   AND (t.subject IS NULL OR t.subject NOT IN (SELECT bezeichnung FROM kanon));

UPDATE buecher_titel t
   SET jahrgang_von = a.von, jahrgang_bis = a.bis,
       grade_level = CASE WHEN a.von = a.bis THEN a.von ELSE grade_level END,
       aktualisiert_am = CURRENT_TIMESTAMP
  FROM ableitung a
 WHERE t.id = a.id AND a.von BETWEEN 5 AND 13 AND a.bis BETWEEN 5 AND 13 AND a.von <= a.bis
   AND (t.jahrgang_von, t.jahrgang_bis) IN ((5, 10), (0, 0));

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
