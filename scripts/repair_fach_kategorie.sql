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
--   2. Alles andere am Titel wird zu NULL („ohne Fach"). ACHTUNG: Der Standorttext ist
--      danach WEG — Schritt 3 löscht die Systematik-Zeile elf Zeilen später in derselben
--      Transaktion, und die Zuordnung Titel↔Text steht nirgends sonst (kein Trigger, kein
--      audit_log-Eintrag). Der einzige Rückweg ist das Backup.
--   3. Systematik-Zeilen werden gelöscht, wenn nach Schritt 2 kein Titel mehr daran hängt
--      und sie kein kanonisches Fach sind. Weil Schritt 2b JEDES nicht-kanonische Fach vom
--      Titel löst, trifft das auch handgepflegte Fächer MIT Büchern — die Liste „wird
--      gelöscht" führt deshalb die Titelzahl mit. Vorher lesen.
--
-- ABLAUF — zwei Aufrufe, die Datei bleibt unverändert:
--
--   1. Vorschau (ändert NICHTS, endet mit ROLLBACK):
--        docker exec -i bibliothek-db sh -c 'psql -U $POSTGRES_USER -d $POSTGRES_DB -v ON_ERROR_STOP=1' \
--          < scripts/repair_fach_kategorie.sql
--
--   2. Ausführen (endet mit COMMIT) — dieselbe Zeile plus -v ausfuehren=ja:
--        docker exec -i bibliothek-db sh -c 'psql -U $POSTGRES_USER -d $POSTGRES_DB -v ON_ERROR_STOP=1 -v ausfuehren=ja' \
--          < scripts/repair_fach_kategorie.sql
--
-- Die letzte Zeile der Ausgabe sagt, was passiert ist: „ROLLBACK" = Vorschau,
-- „COMMIT" = geschrieben. Bis zum 03.09.2026 stand hier stattdessen die Bitte, ROLLBACK
-- von Hand gegen COMMIT zu tauschen — der Lauf sah dann aus wie ein Erfolg, änderte aber
-- nichts, und die Oberfläche zeigte danach unverändert die alten Daten.
--
-- VORHER ein Backup ziehen: pg_dump -U postgres -d bibliothek | gzip > backup.sql.gz
-- (Ein solches Backup lässt sich NICHT einfach über die laufende Datenbank zurückspielen:
-- pg_dump ohne --clean legt keine Tabellen neu an, und jedes COPY bricht am
-- Primärschlüssel ab. Zum echten Zurücksetzen die Datenbank erst leeren.)

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

-- Vorschau: was passiert. Die Verlustzahl steht bewusst NICHT hier, sondern nach Schritt
-- 1b — an dieser Stelle wäre sie zu hoch, weil die Rettung aus Signatur/Standorttext noch
-- nicht gelaufen ist und der Ausführende an genau dieser Zahl entscheidet.
SELECT 'Titel mit Fach gesamt' AS was, count(*) FROM buecher_titel WHERE subject IS NOT NULL AND subject <> ''
UNION ALL
SELECT 'davon bleibt (kanonisch)', count(*) FROM buecher_titel WHERE subject IN (SELECT bezeichnung FROM kanon)
UNION ALL
SELECT 'davon wird umbenannt (Variante)', count(*) FROM buecher_titel WHERE lower(subject) IN (SELECT alt FROM variante);

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

-- Ehrliche Verlustzahl: NACH der Rettung und der Varianten-Umbenennung.
SELECT 'wird jetzt noch "ohne Fach"' AS was, count(*) FROM buecher_titel
 WHERE subject IS NOT NULL AND subject <> '' AND subject NOT IN (SELECT bezeichnung FROM kanon);

-- Offene Inventuren, die auf ein gleich gelöschtes Fach zeigen: Sie zählten danach null
-- Exemplare, und jeder Scan liefe auf „außer Scope". Migration 078 und PatchSystematik
-- ziehen scope_subject bei Umbenennungen mit; dieses Skript kann es nicht, weil das Fach
-- ersatzlos verschwindet — also wird es genannt, damit niemand blind weiterzählt.
SELECT 'ACHTUNG offene Inventur auf gelöschtem Fach' AS was, s.id, s.scope_subject
  FROM inventur_sessions s
 WHERE s.abgeschlossen_am IS NULL
   AND s.scope_subject IS NOT NULL
   AND s.scope_subject NOT IN (SELECT bezeichnung FROM kanon);

-- Schritt 2b: alles andere wird „ohne Fach".
UPDATE buecher_titel
   SET subject = NULL, aktualisiert_am = CURRENT_TIMESTAMP
 WHERE subject IS NOT NULL AND subject <> ''
   AND subject NOT IN (SELECT bezeichnung FROM kanon);

-- Schritt 3: verwaiste Nicht-Fächer aus der Systematik — vorher zeigen, dann löschen.
SELECT 'wird gelöscht' AS was, s.bezeichnung,
       (SELECT count(*) FROM buecher_titel t WHERE t.subject = s.bezeichnung) AS titel_vorher
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

-- Ohne -v ausfuehren=ja bleibt es bei der Vorschau. Geprüft wird der WERT, nicht nur ob
-- die Variable gesetzt ist — sonst schriebe auch `-v ausfuehren=nein`, und der Schalter
-- hielte weniger, als der Kopf verspricht. psql kann Zeichenketten nicht selbst
-- vergleichen; der Vergleich läuft deshalb über die Datenbank in eine \gset-Variable.
\if :{?ausfuehren}
SELECT :'ausfuehren' = 'ja' AS schreiben \gset
\else
SELECT false AS schreiben \gset
\endif

\if :schreiben
COMMIT;
\else
\echo '>>> VORSCHAU — nichts geschrieben. Zum Ausführen: -v ausfuehren=ja'
ROLLBACK;
\endif
