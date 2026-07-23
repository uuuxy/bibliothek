-- scripts/seed_demo.sql
-- Realistischer DEMO-Datensatz für Pilot/Schulung:
--   ~2000 Schüler über echte Klassenstruktur, ein Teil Abgänger,
--   ausleihbare Exemplare mit Barcodes, aktive + überfällige Ausleihen (Mahnwesen).
--
-- SICHERHEIT:
--   * ALLE Daten tragen ein DEMO-Präfix (DEMO-S-, DEMO-B-, "DEMO-Titel ") und sind
--     über den CLEANUP-Block unten rückstandsfrei entfernbar.
--   * Eltern-Mails sind @example.invalid (RFC 6761) — ein Mahnlauf kann NIEMANDEN
--     erreichen, selbst wenn er versehentlich ausgelöst wird.
--   * Läuft in EINER Transaktion mit ON_ERROR_STOP: bei jedem Fehler kompletter Rollback.
--
-- Aufruf lokal:  docker exec -i bibliothek-db-local psql -U postgres -d bibliothek -v ON_ERROR_STOP=1 < scripts/seed_demo.sql
-- Cleanup:       nur den DELETE-Block (Abschnitt 1) ausführen.

BEGIN;

-- 1) Idempotenz / Cleanup: Reste eines früheren DEMO-Laufs entfernen.
DELETE FROM ausleihen WHERE exemplar_id IN (SELECT id FROM buecher_exemplare WHERE barcode_id LIKE 'DEMO-B-%');
DELETE FROM buecher_exemplare WHERE barcode_id LIKE 'DEMO-B-%';
DELETE FROM buecher_titel   WHERE titel LIKE 'DEMO-Titel %';
DELETE FROM schueler        WHERE barcode_id LIKE 'DEMO-S-%';

-- 2) ~2000 Schüler, über realistische Klassen (5a–Q4) verteilt.
--    ~8 % Abgänger (ist_abgaenger), ~2 % gesperrt (mit Pflicht-Grund).
INSERT INTO schueler (barcode_id, vorname, nachname, klasse, geburtsdatum,
                      abgaenger_jahr, ist_abgaenger, eltern_email, ist_gesperrt, block_reason, ort, plz)
SELECT
    'DEMO-S-' || i,
    -- Echtes 2D-Namensraster: Vorname = (i-1) mod |vn|, Nachname = (i-1) div |vn| mod |nn|.
    -- Bei |vn|=|nn|=50 (2500 Kombis > 2000) bekommt JEDER Schüler einen EINDEUTIGEN vollen
    -- Namen. Die frühere Rechnung (kleiner Pool + /7) koppelte Vor-/Nachname → Dutzende
    -- identische "Ben Bauer"/"Felix Demir".
    p.vn[1 + ((i - 1) % array_length(p.vn, 1))],
    p.nn[1 + (((i - 1) / array_length(p.vn, 1)) % array_length(p.nn, 1))],
    p.kl[1 + (i % array_length(p.kl, 1))],
    DATE '2007-01-01' + ((i * 37) % 2600),
    CASE WHEN i % 12 = 0 THEN 2024 + (i % 3) ELSE 2028 + (i % 5) END,
    (i % 12 = 0),
    'demo' || i || '@example.invalid',
    (i % 50 = 0),
    CASE WHEN i % 50 = 0 THEN 'Demo-Sperre (Testdaten)' ELSE NULL END,
    'Musterstadt',
    '12345'
FROM generate_series(1, 2000) AS i
CROSS JOIN (SELECT
    ARRAY['Lukas','Leon','Finn','Noah','Elias','Paul','Ben','Jonas','Luca','Felix','Maximilian','Jakob','David','Tim','Moritz','Julian','Niklas','Simon','Fabian','Tom','Emma','Mia','Hannah','Emilia','Sofia','Lina','Marie','Lena','Sophie','Charlotte','Clara','Johanna','Laura','Anna','Leonie','Amelie','Nele','Ida','Frieda','Greta','Yusuf','Ali','Mert','Emir','Can','Aylin','Elif','Zeynep','Mohammed','Duc'] AS vn,
    ARRAY['Müller','Schmidt','Schneider','Fischer','Weber','Meyer','Wagner','Becker','Schulz','Hoffmann','Koch','Bauer','Richter','Klein','Wolf','Schröder','Neumann','Schwarz','Zimmermann','Braun','Krüger','Hofmann','Hartmann','Lange','Schmitt','Werner','Krause','Meier','Lehmann','Schmitz','Yılmaz','Kaya','Demir','Çelik','Şahin','Yıldız','Nguyen','Popović','Novak','Kowalski','Weiß','Jung','Hahn','Vogel','Friedrich','Keller','Günther','Frank','Berger','Winkler'] AS nn,
    ARRAY['5a','5b','5c','5d','6a','6b','6c','6d','7a','7b','7c','7d','8a','8b','8c','8d','9a','9b','9c','9d','10a','10b','10c','10d','E1','E2','Q1','Q2','Q3','Q4'] AS kl
) p;

-- 3) 2500 ausleihbare Exemplare mit gedruckten Barcodes.
--    Wenn echte (importierte) Titel existieren, hängen wir die DEMO-Exemplare an eine
--    Stichprobe DAVON — dann zeigen Ausleihe & Mahnwesen echte Buchnamen. Nur wenn gar
--    keine Titel vorhanden sind (z. B. leere lokale DB), werden DEMO-Titel angelegt.
--    Cleanup entfernt nur DEMO-Exemplare/-Titel, echte Titel bleiben unangetastet.
DO $$
BEGIN
    IF (SELECT count(*) FROM buecher_titel WHERE titel NOT LIKE 'DEMO-Titel %') = 0 THEN
        INSERT INTO buecher_titel (titel, autor, isbn)
        SELECT 'DEMO-Titel ' || i, 'Autor ' || (1 + i % 60), '978' || lpad(i::text, 10, '0')
        FROM generate_series(1, 200) AS i;
    END IF;
END $$;

WITH n AS (SELECT count(*)::int AS c FROM buecher_titel),
titel AS (SELECT id, row_number() OVER (ORDER BY random()) AS rn FROM buecher_titel)
INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, etikett_gedruckt)
SELECT t.id, 'DEMO-B-' || g, true, true
FROM generate_series(1, 2500) AS g
CROSS JOIN n
JOIN titel t ON t.rn = 1 + (g % n.c);

-- 4) 1550 Ausleihen: 350 überfällig (Frist in der Vergangenheit, nicht zurückgegeben →
--    speist das Mahnwesen; ein Drittel bereits einmal gemahnt) + 1200 aktiv (Frist in
--    der Zukunft). Jede Ausleihe nutzt ein eigenes Exemplar und einen eigenen
--    (nicht-Abgänger) Schüler.
WITH s AS (
    SELECT id, row_number() OVER (ORDER BY barcode_id) AS rn
    FROM schueler WHERE barcode_id LIKE 'DEMO-S-%' AND ist_abgaenger = false
),
e AS (
    SELECT id, row_number() OVER (ORDER BY barcode_id) AS rn
    FROM buecher_exemplare WHERE barcode_id LIKE 'DEMO-B-%'
)
INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist,
                      rueckgabe_am, mahnstufe, letztes_mahndatum)
SELECT
    e.id, s.id,
    CASE WHEN e.rn <= 350 THEN now() - ((21 + e.rn % 80) * INTERVAL '1 day')
         ELSE now() - ((e.rn % 25) * INTERVAL '1 day') END,
    CASE WHEN e.rn <= 350 THEN now() - ((1 + e.rn % 60) * INTERVAL '1 day')
         ELSE now() + ((7 + e.rn % 21) * INTERVAL '1 day') END,
    NULL,
    CASE WHEN e.rn <= 350 AND e.rn % 3 = 0 THEN 1 ELSE 0 END,
    CASE WHEN e.rn <= 350 AND e.rn % 3 = 0 THEN now() - (7 * INTERVAL '1 day') ELSE NULL END
FROM e JOIN s ON s.rn = e.rn
WHERE e.rn <= 1550;

COMMIT;
