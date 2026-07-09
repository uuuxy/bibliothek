-- =============================================================================
-- Signatur-Harmonisierungs-Report (nach Littera-Import / Migration 038)
-- Aufruf: docker exec -i bibliothek-db-local psql -U postgres -d bibliothek < scripts/signatur_report.sql
-- (oder direkt in psql/DBeaver gegen die Zieldatenbank)
-- =============================================================================

-- 0) Überblick: Wie steht es um die Signaturen insgesamt?
SELECT
    COUNT(*)                                                          AS titel_gesamt,
    COUNT(*) FILTER (WHERE COALESCE(signatur, '') = '')               AS ohne_signatur,
    COUNT(*) FILTER (WHERE length(trim(signatur)) = 1)                AS verdaechtig_kurz,
    COUNT(*) FILTER (WHERE COALESCE(erweiterte_eigenschaften->>'signatur', '') != '')
                                                                      AS noch_im_jsonb_altbestand,
    ROUND(COUNT(*) FILTER (WHERE COALESCE(signatur, '') = '') * 100.0
          / GREATEST(COUNT(*), 1), 1)                                 AS quote_ohne_signatur_pct
FROM buecher_titel;

-- 1) Anomalien: Titel ohne Signatur bzw. mit verdächtig kurzer Signatur
--    (leer, NULL oder < 2 Zeichen) — Kandidaten für Nachpflege.
SELECT
    t.id,
    t.titel,
    COALESCE(t.autor, '—')                       AS autor,
    COALESCE(t.isbn, '—')                        AS isbn,
    COALESCE(NULLIF(trim(t.signatur), ''), '∅')  AS signatur,
    COUNT(e.id)                                  AS exemplare
FROM buecher_titel t
LEFT JOIN buecher_exemplare e ON e.titel_id = t.id AND e.ist_ausgesondert = false
WHERE COALESCE(trim(t.signatur), '') = ''
   OR length(trim(t.signatur)) < 2
GROUP BY t.id, t.titel, t.autor, t.isbn, t.signatur
ORDER BY exemplare DESC, t.titel
LIMIT 100;

-- 2) Bonus: Systematik-Übersicht — welche Kürzel (erstes „Wort" bzw. die
--    ersten 3 Zeichen) dominieren den Bestand? Deckt Tippfehler-Cluster auf
--    („Bio" vs. „BIO" vs. „Bio5").
SELECT
    COALESCE(NULLIF(split_part(trim(signatur), ' ', 1), ''),
             left(trim(signatur), 3))            AS systematik_kuerzel,
    COUNT(*)                                     AS titel,
    MIN(trim(signatur))                          AS beispiel_min,
    MAX(trim(signatur))                          AS beispiel_max
FROM buecher_titel
WHERE COALESCE(trim(signatur), '') != ''
GROUP BY 1
ORDER BY titel DESC
LIMIT 40;
