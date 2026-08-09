-- Rückstand von barcode_seq gegenüber dem tatsächlichen Bestand aufholen.
--
-- Bis hierher gab es ZWEI Vergabestellen für Exemplarnummern: das Bestellwesen zog aus
-- barcode_seq, der Knopf "Interne ID generieren" rechnete MAX(Nummer)+1 aus der
-- Tabelle. Beide lieferten dieselbe nächste Nummer, aber nur die Sequenz zählte weiter.
-- Jede von Hand vergebene Nummer ließ die Sequenz also zurückfallen — und die nächste
-- Bestellung mit Vorab-Barcodes zog genau diese Nummer erneut. Das endete in einer
-- UNIQUE-Verletzung, die wegen des CopyFrom in einer Transaktion nicht die Position,
-- sondern die ganze Bestellung kostete.
--
-- Ab jetzt ist barcode_seq die einzige Quelle (repository/barcode_vergabe.go). Diese
-- Migration schließt den bereits entstandenen Rückstand, damit die erste Bestellung
-- nach dem Update nicht in die alte Falle läuft.
--
-- GREATEST statt eines direkten setval: Die Sequenz darf nur steigen, nie sinken.
-- Die Ziffernlänge ist begrenzt, damit eine von Hand eingetippte Fantasienummer den
-- Cast auf bigint nicht sprengt. Wiederholbar — ein zweiter Lauf ändert nichts.
SELECT setval('barcode_seq', GREATEST(
    (SELECT last_value FROM barcode_seq),
    (SELECT coalesce(max(substr(barcode_id, 3)::bigint), 0)
       FROM buecher_exemplare
      WHERE barcode_id ~ '^B-[0-9]{1,15}$')));
