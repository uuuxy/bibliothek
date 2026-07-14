-- repair_titel_ortssuffix.sql
--
-- Einmalige Datenreparatur, NACH repair_titel_dubletten.sql ausführen.
--
-- Die aus dem Littera-PDF konvertierte Bestands-CSV (clean_import.csv) hat beim
-- Extrahieren häufig den VERLAGSORT an den Titel geklebt ("Fette Ferien Ravensburg"
-- statt "Fette Ferien"). Diese Titel konnten den sauberen Katalogisat-Titel (XML)
-- nie matchen — es existieren also zwei Zeilen: die CSV-Zeile trägt die Exemplare,
-- die XML-Zeile Signatur/ISBN.
--
-- Der Merge ist bewusst konservativ, alle vier Bedingungen müssen gelten:
--   1. Die CSV-Zeile hat weder Signatur noch ISBN, aber Exemplare.
--   2. Ihr letztes Wort ist ein BEKANNTER Verlagsort (aus der Littera-Zitierform
--      "Ort : Verlag" der eigenen Daten abgeleitet — keine geratene Ortsliste).
--   3. Der Titel ohne Suffix matcht (normalisiert) GENAU EINEN anderen Titel.
--   4. Das Ziel stammt aus dem Katalogisat (hat Signatur oder ISBN).
--
-- VORHER ein Backup ziehen: pg_dump -U postgres -d bibliothek | gzip > backup.sql.gz

BEGIN;

CREATE TEMP TABLE ortssuffix_merge ON COMMIT DROP AS
WITH orte AS (
    SELECT DISTINCT split_part(verlag, ' : ', 1) AS ort
    FROM buecher_titel
    WHERE verlag LIKE '% : %'
      AND split_part(verlag, ' : ', 1) ~ '^[A-ZÄÖÜ][a-zäöüß]+( am | an der | im )?[A-ZÄÖÜa-zäöüß.]*$'
), csv_only AS (
    SELECT t.id, t.titel
    FROM buecher_titel t
    WHERE (t.signatur IS NULL OR t.signatur = '') AND t.isbn IS NULL
      AND EXISTS (SELECT 1 FROM buecher_exemplare e WHERE e.titel_id = t.id)
), paar AS (
    SELECT c.id AS csv_id,
           btrim(regexp_replace(replace(regexp_replace(c.titel, ' [^ ]+$', ''), '"', ''), '\s+', ' ', 'g')) AS basis_key
    FROM csv_only c
    WHERE EXISTS (SELECT 1 FROM orte o WHERE o.ort = regexp_replace(c.titel, '^.* ', ''))
)
SELECT p.csv_id AS dup_id, x.id AS keeper_id
FROM paar p
JOIN buecher_titel x
  ON btrim(regexp_replace(replace(x.titel, '"', ''), '\s+', ' ', 'g')) = p.basis_key
 AND x.id <> p.csv_id
WHERE (x.signatur <> '' OR x.isbn IS NOT NULL)
  AND (SELECT count(*) FROM buecher_titel x2
       WHERE btrim(regexp_replace(replace(x2.titel, '"', ''), '\s+', ' ', 'g')) = p.basis_key
         AND x2.id <> p.csv_id) = 1;

-- Fachliche Werte der CSV-Zeile in den Keeper übernehmen (nur wo dort leer).
UPDATE buecher_titel k
SET autor            = COALESCE(NULLIF(k.autor, ''), d.autor),
    verlag           = COALESCE(NULLIF(k.verlag, ''), d.verlag),
    subject          = COALESCE(NULLIF(k.subject, ''), d.subject),
    erscheinungsjahr = COALESCE(k.erscheinungsjahr, d.erscheinungsjahr),
    aktualisiert_am  = CURRENT_TIMESTAMP
FROM (
    SELECT m.keeper_id,
           max(NULLIF(t.autor, ''))   AS autor,
           max(NULLIF(t.verlag, ''))  AS verlag,
           max(NULLIF(t.subject, '')) AS subject,
           max(t.erscheinungsjahr)    AS erscheinungsjahr
    FROM ortssuffix_merge m
    JOIN buecher_titel t ON t.id = m.dup_id
    GROUP BY m.keeper_id
) d
WHERE k.id = d.keeper_id;

-- Fremdschlüssel umhängen.
UPDATE buecher_exemplare e SET titel_id = m.keeper_id
FROM ortssuffix_merge m WHERE e.titel_id = m.dup_id;

UPDATE class_books c SET book_id = m.keeper_id
FROM ortssuffix_merge m WHERE c.book_id = m.dup_id;

UPDATE klassensatz_reservierungen k SET titel_id = m.keeper_id
FROM ortssuffix_merge m WHERE k.titel_id = m.dup_id;

UPDATE bestellungen_positionen b SET titel_id = m.keeper_id
FROM ortssuffix_merge m WHERE b.titel_id = m.dup_id;

-- Vormerkungen: UNIQUE (titel_id, schueler_id) — Kollisionen vorab entfernen.
DELETE FROM vormerkungen
WHERE id IN (
    SELECT id FROM (
        SELECT v.id,
               row_number() OVER (
                   PARTITION BY COALESCE(m.keeper_id, v.titel_id), v.schueler_id
                   ORDER BY v.id
               ) AS rn
        FROM vormerkungen v
        LEFT JOIN ortssuffix_merge m ON v.titel_id = m.dup_id
    ) nummeriert
    WHERE rn > 1
);

UPDATE vormerkungen v SET titel_id = m.keeper_id
FROM ortssuffix_merge m WHERE v.titel_id = m.dup_id;

-- CSV-Zeilen löschen und Ergebnis protokollieren.
DO $$
DECLARE
    gemergt integer;
BEGIN
    SELECT count(*) INTO gemergt FROM ortssuffix_merge;
    DELETE FROM buecher_titel WHERE id IN (SELECT dup_id FROM ortssuffix_merge);
    RAISE NOTICE 'Ortssuffix-Reparatur: % CSV-Titel in ihre Katalogisat-Titel gemergt.', gemergt;
END $$;

COMMIT;
