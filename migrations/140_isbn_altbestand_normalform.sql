-- =============================================================================
-- Migration 140: ISBNs aus der Zeit vor Migration 133 in die Normalform
-- =============================================================================
-- Seit Migration 133 bringt ein Trigger jede GESCHRIEBENE ISBN in eine Schreibweise
-- (isbn_normalform: ohne Bindestriche und Leerzeichen, X groß, leer wird NULL). Zeilen, die
-- vorher entstanden, blieben, wie sie waren. Gemessen am Testserver am 23.09.2026: 10.061
-- Titel mit ISBN, genau einer weicht ab („978-3-12-622042-2"), keine zwei werden nach dem
-- Normalisieren gleich.
--
-- Warum überhaupt: Die UNIQUE-Regel auf isbn vergleicht die gespeicherte Schreibweise. Neben
-- „978-3-12-622042-2" ließe sie „9783126220422" als zweiten Titel zu; die Dublettenkontrolle
-- der Oberfläche vergleicht die Normalform, die Datenbank nicht.
--
-- Eine Zeile, deren Normalform auch ein ANDERER Titel ergibt, bleibt stehen: Zwei Titel mit
-- derselben ISBN sind eine Dublette, und ob man sie zusammenlegt, entscheidet ein Mensch
-- (docs/OFFEN.md 4.18: „Zusammengelegt wird nichts"). Die Migration scheitert daran nicht,
-- sie nennt die Zahl als NOTICE.
--
-- Idempotent: Beim zweiten Lauf gibt es nichts mehr abzuweichen. Der Trigger auf
-- UPDATE OF isbn läuft mit und schreibt dieselbe Normalform.

UPDATE buecher_titel t
SET isbn = isbn_normalform(t.isbn)
WHERE t.isbn IS DISTINCT FROM isbn_normalform(t.isbn)
  AND NOT EXISTS (
      SELECT 1 FROM buecher_titel a
      WHERE a.id <> t.id
        AND isbn_normalform(a.isbn) = isbn_normalform(t.isbn)
  );

DO $$
DECLARE
    rest integer;
BEGIN
    SELECT count(*) INTO rest FROM buecher_titel WHERE isbn IS DISTINCT FROM isbn_normalform(isbn);
    IF rest > 0 THEN
        RAISE NOTICE 'Migration 140: % Titel behalten ihre Schreibweise, weil ein anderer Titel dieselbe ISBN trägt (Dublette, von Hand entscheiden)', rest;
    END IF;
END $$;
