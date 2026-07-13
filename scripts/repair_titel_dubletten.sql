-- repair_titel_dubletten.sql
--
-- Einmalige Datenreparatur: führt Titel-Dubletten in buecher_titel zusammen,
-- die die Import-Pfade vor den Fixes vom 13.07.2026 angelegt haben:
--   1. Der Katalogisat-Import (XML) matchte Datensätze MIT ISBN nur über die
--      ISBN — die per Bestands-CSV angelegten Titel (ohne ISBN) wurden nie
--      gefunden und jedes Mal neu angelegt.
--   2. Die aus dem Littera-PDF konvertierte Bestands-CSV enthält Titel mit
--      Streu-Anführungszeichen an den Rändern (`"Elemente Chemie 1`), die den
--      Katalogisat-Titel nie matchen konnten.
--
-- Vorgehen (eine Transaktion, idempotent):
--   1. Streu-Anführungszeichen an den Titel-Rändern entfernen.
--   2. Je Titel-String einen Keeper wählen (meiste Exemplare > ISBN vorhanden >
--      Signatur vorhanden > ältester Datensatz) und alle Fremdschlüssel der
--      Dubletten auf ihn umhängen.
--   3. Fachliche Werte (ISBN, Signatur, Autor, …) aus den Dubletten in den
--      Keeper übernehmen, sofern dort leer.
--   4. Dubletten löschen.
--
-- VORHER ein Backup ziehen: pg_dump -U postgres -d bibliothek | gzip > backup.sql.gz

BEGIN;

-- 1. Streu-Anführungszeichen (PDF-Konvertierungs-Artefakte) an den Rändern
--    entfernen. Anführungszeichen IM Titel bleiben erhalten — identisch zur
--    Import-Bereinigung (bereinigeImportTitel).
UPDATE buecher_titel
SET titel = btrim(btrim(btrim(titel), '"')), aktualisiert_am = CURRENT_TIMESTAMP
WHERE titel <> btrim(btrim(btrim(titel), '"'));

-- 2. Keeper je Titel-Gruppe bestimmen. Gruppiert wird über den NORMALISIERTEN
--    Titel (Anführungszeichen raus, Whitespace kollabiert) — dieselbe Logik wie
--    repository.NormalisiereTitelKey in den Import-Pfaden, damit z. B.
--    `"Kein Bock auf Lernen?"` und `Kein Bock auf Lernen?` zusammenfinden.
CREATE TEMP TABLE titel_merge ON COMMIT DROP AS
SELECT id AS dup_id, keeper_id
FROM (
    SELECT id,
           first_value(id) OVER (
               PARTITION BY btrim(regexp_replace(replace(titel, '"', ''), '\s+', ' ', 'g'))
               ORDER BY
                   (SELECT count(*) FROM buecher_exemplare e WHERE e.titel_id = t.id) DESC,
                   (isbn IS NOT NULL) DESC,
                   (signatur IS NOT NULL AND signatur <> '') DESC,
                   erstellt_am ASC,
                   id ASC
           ) AS keeper_id
    FROM buecher_titel t
) kandidaten
WHERE id <> keeper_id;

-- 3. Fachliche Werte der Dubletten in den Keeper übernehmen (nur wo leer).
--    Die Werte werden VOR dem ISBN-Freigeben in eine Temp-Tabelle gesichert:
--    die UNIQUE-Prüfung auf isbn greift sofort beim UPDATE, die Dublette wird
--    aber erst in Schritt 5 gelöscht — ihre ISBN muss daher zuerst genullt werden.
CREATE TEMP TABLE dup_werte ON COMMIT DROP AS
SELECT m.keeper_id,
       max(t.isbn)                       AS isbn,
       max(NULLIF(t.signatur, ''))       AS signatur,
       max(NULLIF(t.autor, ''))          AS autor,
       max(NULLIF(t.verlag, ''))         AS verlag,
       max(NULLIF(t.untertitel, ''))     AS untertitel,
       max(NULLIF(t.beschreibung, ''))   AS beschreibung,
       max(NULLIF(t.subject, ''))        AS subject,
       max(NULLIF(t.cover_url, ''))      AS cover_url,
       max(t.erscheinungsjahr)           AS erscheinungsjahr
FROM titel_merge m
JOIN buecher_titel t ON t.id = m.dup_id
GROUP BY m.keeper_id;

UPDATE buecher_titel t SET isbn = NULL
FROM titel_merge m
WHERE t.id = m.dup_id AND t.isbn IS NOT NULL;

UPDATE buecher_titel k
SET isbn             = COALESCE(k.isbn, d.isbn),
    signatur         = COALESCE(NULLIF(k.signatur, ''), d.signatur),
    autor            = COALESCE(NULLIF(k.autor, ''), d.autor),
    verlag           = COALESCE(NULLIF(k.verlag, ''), d.verlag),
    untertitel       = COALESCE(NULLIF(k.untertitel, ''), d.untertitel),
    beschreibung     = COALESCE(NULLIF(k.beschreibung, ''), d.beschreibung),
    subject          = COALESCE(NULLIF(k.subject, ''), d.subject),
    cover_url        = COALESCE(NULLIF(k.cover_url, ''), d.cover_url),
    erscheinungsjahr = COALESCE(k.erscheinungsjahr, d.erscheinungsjahr),
    aktualisiert_am  = CURRENT_TIMESTAMP
FROM dup_werte d
WHERE k.id = d.keeper_id;

-- 4. Fremdschlüssel der Dubletten auf den Keeper umhängen.
UPDATE buecher_exemplare e SET titel_id = m.keeper_id
FROM titel_merge m WHERE e.titel_id = m.dup_id;

UPDATE class_books c SET book_id = m.keeper_id
FROM titel_merge m WHERE c.book_id = m.dup_id;

UPDATE klassensatz_reservierungen k SET titel_id = m.keeper_id
FROM titel_merge m WHERE k.titel_id = m.dup_id;

UPDATE bestellungen_positionen b SET titel_id = m.keeper_id
FROM titel_merge m WHERE b.titel_id = m.dup_id;

-- Vormerkungen: UNIQUE (titel_id, schueler_id) — vor dem Umhängen alle
-- Vormerkungen entfernen, die nach dem Merge doppelt wären (älteste bleibt).
DELETE FROM vormerkungen
WHERE id IN (
    SELECT id
    FROM (
        SELECT v.id,
               row_number() OVER (
                   PARTITION BY COALESCE(m.keeper_id, v.titel_id), v.schueler_id
                   ORDER BY v.id
               ) AS rn
        FROM vormerkungen v
        LEFT JOIN titel_merge m ON v.titel_id = m.dup_id
    ) nummeriert
    WHERE rn > 1
);

UPDATE vormerkungen v SET titel_id = m.keeper_id
FROM titel_merge m WHERE v.titel_id = m.dup_id;

-- 5. Dubletten löschen.
DELETE FROM buecher_titel WHERE id IN (SELECT dup_id FROM titel_merge);

-- Ergebnis protokollieren.
DO $$
DECLARE
    verbleibend integer;
BEGIN
    SELECT count(*) INTO verbleibend FROM buecher_titel;
    RAISE NOTICE 'Reparatur fertig: % Titel verbleiben.', verbleibend;
END $$;

COMMIT;
