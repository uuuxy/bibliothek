-- Migration 104: Der Nummernkreis der automatisch erzeugten Exemplare bekommt eine
-- Heimat.
--
-- `sys_barcode_seq` vergibt die Nummern der Exemplare, die das System selbst anlegt
-- („SYS-…", Bestandskorrektur in der Buchmaske und Sammelimport). Sie stand bis heute
-- (07.09.2026) in KEINER Migration und in keiner Zeile von schema.sql: Der Go-Code führte
-- vor jedem Insert ein `CREATE SEQUENCE IF NOT EXISTS` aus — mitten im Schreibpfad,
-- innerhalb der Transaktion, die gerade das Buch anlegt.
--
-- Am echten Postgres gemessen: Solange die Sequenz noch nicht existiert, hält dieses DDL
-- eine Sperre bis zum Commit. Eine zweite Transaktion, die dasselbe versucht, wartet die
-- volle Laufzeit der ersten ab (3,06 s bei 4 s Vorlauf); existiert die Sequenz, kostet es
-- nichts (0,11 s). Der Preis fällt also genau EINMAL an — beim ersten Buch mit Bestand
-- nach einer frischen Installation, wo ihn niemand erwartet und das Frontend nach 10 s
-- abbricht (frontend/src/lib/apiFetch.js).
--
-- Warum keine Ratsche das gefunden hat: Die Schema-Parität (db/migrations_schema_-
-- paritaet_pg_test.go) vergleicht Sequenzen ausdrücklich mit, hält dabei aber den
-- gewachsenen gegen den frischen Weg. Ein Objekt, das erst der laufende Anwendungscode
-- erzeugt, fehlt in BEIDEN — und ist damit für jeden Vergleich der beiden unsichtbar.
-- Das neue Gate prüft deshalb die Quelle: inventur/kein_ddl_im_schreibpfad_test.go.
--
-- Diese Migration ändert an bestehenden Installationen NICHTS: Wo die Sequenz durch den
-- alten Pfad schon entstanden ist, lässt IF NOT EXISTS sie samt Stand unangetastet. Der
-- setval-Nachzug hebt sie nur an, falls vergebene SYS-Nummern über ihrem Stand liegen —
-- dieselbe Vorsichtsmaßnahme wie in Migration 068 für barcode_seq, damit keine Nummer
-- ein zweites Mal vergeben wird.
--
-- ACHTUNG, kein Rückbau dieser Zeilen: „SYS-…" ist ein ZWEITER Nummernkreis neben
-- barcode_seq („B-…", die Quelle der von Hand vergebenen Exemplarnummern). Dass es zwei
-- gibt, ist hier nur festgeschrieben, nicht entschieden — die Präfixe halten sie
-- auseinander, eine Zusammenlegung wäre eine eigene Entscheidung.

CREATE SEQUENCE IF NOT EXISTS sys_barcode_seq START 100000;

SELECT setval('sys_barcode_seq', GREATEST(
    (SELECT last_value FROM sys_barcode_seq),
    (SELECT COALESCE(MAX(CAST(SUBSTRING(barcode_id FROM '^SYS-([0-9]{1,15})$') AS BIGINT)), 100000)
     FROM buecher_exemplare
     WHERE barcode_id ~ '^SYS-[0-9]{1,15}$')
));
