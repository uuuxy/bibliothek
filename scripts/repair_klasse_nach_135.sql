-- repair_klasse_nach_135.sql
--
-- Einmalige Reparatur für den Testserver. Migration 135 (ddfc1900) lief dort am 22.09.2026
-- um 17:40 Uhr, bevor sie mit 7981e347 zurückgenommen wurde. Sie hat 129 Titeln die Klasse
-- als einjährige Spanne gesetzt und die Spalte grade_level gelöscht. Der Code nach der
-- Rücknahme liest die Spalte wieder; ohne sie brechen Titelliste, Buchakte, Anlegen,
-- Ändern und die Importe mit „column grade_level does not exist" ab.
--
-- Die alten Werte kommen aus der Vorab-Sicherung vordeploy_20260922_153818 (17:38 Uhr,
-- vor 135). Das Skript
--   1. legt grade_level und chk_grade_level_bereich wieder an,
--   2. setzt die Spanne zurück, wo 135 sie gesetzt hat: in der Sicherung Klasse 6 bis 13
--      bei der Spanne 5 bis 10, heute genau „Klasse bis Klasse",
--   3. schreibt jede Klasse der Sicherung zurück,
--   4. streicht 135 aus schema_migrations.
-- Titel, die nach 17:38 Uhr angelegt wurden, stehen nicht in der Sicherung und bleiben ohne
-- Klasse, so wie sie angelegt wurden.
--
-- ABLAUF auf dem Server, im Ordner /root/bibliothek, Zeile für Zeile:
--
--   Vorbereitung — Sicherung entschlüsseln, in die Hilfsdatenbank bibliothek_vor135
--   einspielen und Klassen und Spannen als CSV ablegen. Die Sicherung geht über die
--   Standardeingabe in den Backend-Container wie in scripts/backup_krypto.sh
--   (pruefe_enc_rundweg): Mit docker cp läge sie dort nur für root lesbar, das Werkzeug
--   läuft als appuser. Der entschlüsselte Dump berührt die Platte nicht.
--     docker exec bibliothek-db createdb -U postgres bibliothek_vor135
--     docker exec -i bibliothek-backend sh -c 'tmp=$(mktemp) && cat > "$tmp" && ./restore-backup "$tmp"; rc=$?; rm -f "$tmp"; exit $rc' < backups/vordeploy_20260922_153818.sql.gz.enc | docker exec -i bibliothek-db psql -U postgres -d bibliothek_vor135 -q -o /dev/null
--     docker exec bibliothek-db psql -U postgres -d bibliothek_vor135 -c "\copy (SELECT id, grade_level, jahrgang_von, jahrgang_bis FROM buecher_titel) TO '/tmp/vor135_titel.csv' CSV"
--
--   1. Vorschau (ändert NICHTS, endet mit ROLLBACK):
--     docker exec -i bibliothek-db psql -U postgres -d bibliothek -v ON_ERROR_STOP=1 < scripts/repair_klasse_nach_135.sql
--
--   2. Ausführen (endet mit COMMIT) — dieselbe Zeile plus -v ausfuehren=ja:
--     docker exec -i bibliothek-db psql -U postgres -d bibliothek -v ON_ERROR_STOP=1 -v ausfuehren=ja < scripts/repair_klasse_nach_135.sql
--
--   Danach: Backend neu starten, Hilfsdatenbank und CSV entfernen:
--     docker restart bibliothek-backend
--     docker exec bibliothek-db dropdb -U postgres bibliothek_vor135
--     docker exec bibliothek-db rm -f /tmp/vor135_titel.csv
--
-- Die letzte Zeile der Ausgabe sagt, was passiert ist: „ROLLBACK" = Vorschau,
-- „COMMIT" = geschrieben.

BEGIN;

CREATE TEMP TABLE vor135 (
    id           uuid PRIMARY KEY,
    grade_level  smallint,
    jahrgang_von integer,
    jahrgang_bis integer
) ON COMMIT DROP;

\copy vor135 FROM '/tmp/vor135_titel.csv' CSV

DO $$
BEGIN
    IF (SELECT count(*) FROM vor135) = 0 THEN
        RAISE EXCEPTION 'Die CSV aus der Sicherung ist leer — Vorbereitung prüfen, nichts geändert';
    END IF;
END $$;

SELECT count(*) AS titel_in_sicherung,
       count(*) FILTER (WHERE grade_level IS NOT NULL AND grade_level <> 0) AS mit_klasse
FROM vor135;

ALTER TABLE buecher_titel ADD COLUMN IF NOT EXISTS grade_level SMALLINT;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_grade_level_bereich') THEN
        ALTER TABLE buecher_titel ADD CONSTRAINT chk_grade_level_bereich
            CHECK (grade_level IS NULL OR grade_level BETWEEN 0 AND 13);
    END IF;
END $$;

WITH zurueck AS (
    UPDATE buecher_titel t
       SET jahrgang_von = v.jahrgang_von, jahrgang_bis = v.jahrgang_bis
      FROM vor135 v
     WHERE t.id = v.id
       AND v.grade_level BETWEEN 6 AND 13
       AND v.jahrgang_von = 5 AND v.jahrgang_bis = 10
       AND t.jahrgang_von = v.grade_level AND t.jahrgang_bis = v.grade_level
    RETURNING 1
)
SELECT count(*) AS spanne_zurueckgesetzt FROM zurueck;

WITH klasse AS (
    UPDATE buecher_titel t
       SET grade_level = v.grade_level
      FROM vor135 v
     WHERE t.id = v.id AND v.grade_level IS NOT NULL
    RETURNING t.grade_level
)
SELECT count(*) AS klasse_zurueckgeschrieben,
       count(*) FILTER (WHERE grade_level <> 0) AS davon_nicht_null
FROM klasse;

WITH gestrichen AS (
    DELETE FROM schema_migrations WHERE version = '135_klasse_in_spanne.sql' RETURNING 1
)
SELECT count(*) AS migration_135_gestrichen FROM gestrichen;

-- Gegenprobe: Titel, deren Klasse oder Spanne jetzt von der Sicherung abweicht.
SELECT count(*) AS abweichend_von_sicherung
FROM buecher_titel t JOIN vor135 v ON v.id = t.id
WHERE t.grade_level IS DISTINCT FROM v.grade_level
   OR t.jahrgang_von IS DISTINCT FROM v.jahrgang_von
   OR t.jahrgang_bis IS DISTINCT FROM v.jahrgang_bis;

-- Ohne -v ausfuehren=ja bleibt es bei der Vorschau. Geprüft wird der WERT, nicht nur ob
-- die Variable gesetzt ist — sonst schriebe auch `-v ausfuehren=nein`.
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
