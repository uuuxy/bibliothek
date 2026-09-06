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

-- 2) ~2000 Schüler, über die Klassen DIESER Schule verteilt (Peter, 06.09.2026: „es gibt
--    kein 10a" — Anzeigeform des Vokabulars, Migration 087): Förderstufe 5F/6F und
--    Gymnasialzweig 5G/6G, ab 7 die Zweige H (bis 9), R (bis 10) und G (bis 10), dann die
--    Oberstufe als Tutorien ET/12T/13T — so, wie sie im LMF-Plan der Schule stehen.
--    ~8 % Abgänger (NUR aus Abschlussklassen, s. u.), ~2 % gesperrt (mit Pflicht-Grund).
INSERT INTO schueler (barcode_id, vorname, nachname, klasse, geburtsdatum,
                      abgaenger_jahr, ist_abgaenger, eltern_email, ist_gesperrt, block_reason, ort, plz)
SELECT
    'DEMO-S-' || s.i,
    s.vorname,
    s.nachname,
    s.klasse,
    DATE '2007-01-01' + ((s.i * 37) % 2600),
    -- Abgänger aufs AKTUELLE Jahr: der GDPR-Job löscht Abgänger mit abgaenger_jahr <
    -- aktuellem Jahr (jobs/cron.go). CURRENT_DATE hält sie in der Karenzzeit und macht
    -- den Seed selbstwartend. Aktive Schüler: künftiges Abgangsjahr.
    CASE WHEN s.ist_abgaenger THEN EXTRACT(YEAR FROM CURRENT_DATE)::int ELSE 2028 + (s.i % 5) END,
    s.ist_abgaenger,
    'demo' || s.i || '@example.invalid',
    (s.i % 50 = 0),
    CASE WHEN s.i % 50 = 0 THEN 'Demo-Sperre (Testdaten)' ELSE NULL END,
    'Musterstadt',
    '12345'
FROM (
    SELECT
        i,
        -- Echtes 2D-Namensraster: Vorname = (i-1) mod |vn|, Nachname = (i-1) div |vn| mod |nn|.
        -- Bei |vn|=|nn|=50 (2500 Kombis > 2000) bekommt JEDER Schüler einen EINDEUTIGEN
        -- vollen Namen.
        p.vn[1 + ((i - 1) % array_length(p.vn, 1))] AS vorname,
        p.nn[1 + (((i - 1) / array_length(p.vn, 1)) % array_length(p.nn, 1))] AS nachname,
        p.kl[1 + (i % array_length(p.kl, 1))] AS klasse,
        -- Abgänger NUR aus Abschlussklassen (dieselbe Regel wie repository.AbschlussklasseSQL:
        -- H ab 9, R ab 10, Jahrgang 13) — ein „abgehender Fünftklässler" wäre fachlich
        -- Unsinn. i % 7 (teilerfremd zu 73 = |Klassen|) streut die Abgänger gleichmäßig
        -- INNERHALB dieser Klassen, statt ganze Jahrgänge komplett zu treffen.
        (p.kl[1 + (i % array_length(p.kl, 1))] IN ('09H1','09H2','10R1','10R2','10R3','13T1','13T2','13T3')
            AND i % 7 < 2) AS ist_abgaenger
    FROM generate_series(1, 2000) AS i
    CROSS JOIN (SELECT
    ARRAY['Lukas','Leon','Finn','Noah','Elias','Paul','Ben','Jonas','Luca','Felix','Maximilian','Jakob','David','Tim','Moritz','Julian','Niklas','Simon','Fabian','Tom','Emma','Mia','Hannah','Emilia','Sofia','Lina','Marie','Lena','Sophie','Charlotte','Clara','Johanna','Laura','Anna','Leonie','Amelie','Nele','Ida','Frieda','Greta','Yusuf','Ali','Mert','Emir','Can','Aylin','Elif','Zeynep','Mohammed','Duc'] AS vn,
    ARRAY['Müller','Schmidt','Schneider','Fischer','Weber','Meyer','Wagner','Becker','Schulz','Hoffmann','Koch','Bauer','Richter','Klein','Wolf','Schröder','Neumann','Schwarz','Zimmermann','Braun','Krüger','Hofmann','Hartmann','Lange','Schmitt','Werner','Krause','Meier','Lehmann','Schmitz','Yılmaz','Kaya','Demir','Çelik','Şahin','Yıldız','Nguyen','Popović','Novak','Kowalski','Weiß','Jung','Hahn','Vogel','Friedrich','Keller','Günther','Frank','Berger','Winkler'] AS nn,
    ARRAY['05F1','05F2','05F3','05F4','05G1','05G2','05G3','05G4','05G5','05G6',
          '06F1','06F2','06F3','06F4','06G1','06G2','06G3','06G4','06G5','06G6',
          '07H1','07H2','07R1','07R2','07R3','07G1','07G2','07G3','07G4','07G5','07G6',
          '08H1','08H2','08H3','08H4','08R1','08R2','08R3','08G1','08G2','08G3','08G4','08G5',
          '09H1','09H2','09R1','09R2','09R3','09G1','09G2','09G3','09G4','09G5',
          '10R1','10R2','10R3','10G1','10G2','10G3','10G4','10G5','10G6',
          'ET1','ET2','ET3','12T1','12T2','12T3','12T4','12T5','13T1','13T2','13T3'] AS kl
    ) p
) s;

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

-- 5) Ein Teil der Abgänger hat noch offene Bücher. NUR DIESE erscheinen in der
--    Abgänger-Ansicht (sie listet gezielt Abgänger mit nicht zurückgegebenen Medien —
--    JOIN ausleihen ... rueckgabe_am IS NULL). Ohne offene Ausleihe gäbe es dort nichts.
--    ~60 % überfällig. Nebeneffekt: diese Abgänger sind vor der GDPR-Löschung geschützt
--    (Löschung verlangt "keine offenen Ausleihen"). Exemplare 1551–1670 (vom Block oben
--    ungenutzt), damit kein Exemplar doppelt verliehen wird.
WITH ab AS (
    SELECT id, row_number() OVER (ORDER BY barcode_id) AS rn
    FROM schueler WHERE barcode_id LIKE 'DEMO-S-%' AND ist_abgaenger = true
),
e AS (
    SELECT id, row_number() OVER (ORDER BY barcode_id) AS rn
    FROM buecher_exemplare WHERE barcode_id LIKE 'DEMO-B-%'
)
INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist,
                      rueckgabe_am, mahnstufe, letztes_mahndatum)
SELECT
    e.id, ab.id,
    now() - ((30 + ab.rn % 60) * INTERVAL '1 day'),
    CASE WHEN ab.rn % 5 < 3 THEN now() - ((1 + ab.rn % 40) * INTERVAL '1 day')
         ELSE now() + ((5 + ab.rn % 20) * INTERVAL '1 day') END,
    NULL,
    CASE WHEN ab.rn % 5 < 3 AND ab.rn % 2 = 0 THEN 1 ELSE 0 END,
    CASE WHEN ab.rn % 5 < 3 AND ab.rn % 2 = 0 THEN now() - (7 * INTERVAL '1 day') ELSE NULL END
FROM ab JOIN e ON e.rn = 1550 + ab.rn
WHERE ab.rn <= 120;

COMMIT;
