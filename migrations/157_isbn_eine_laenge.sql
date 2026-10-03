-- =============================================================================
-- Migration 157: Die ISBN in einer Länge
-- =============================================================================
-- Seit Migration 133 hat eine ISBN eine Schreibweise: ohne Bindestriche und Leerzeichen,
-- Prüfzeichen groß. Die beiden Längen blieben getrennt. Unter der zehnstelligen und unter der
-- dreizehnstelligen Nummer desselben Buchs konnten zwei Titel stehen, mit zwei Beständen und
-- zwei Zeilen in der Nachbestellung. Maske, Bestellsuche und Bestelltür rechneten deshalb je
-- für sich in die andere Länge um und fragten nach; der Listenimport tat es nicht und legte
-- das Buch ein zweites Mal an.
--
-- Jetzt gehört die Länge zur Normalform (entschieden am 03.10.2026): Eine zehnstellige ISBN
-- mit richtigem Prüfzeichen wird zur dreizehnstelligen — 978, die neun Ziffern, die neu
-- berechnete Prüfziffer. Littera rechnet seit Version 4.6 genauso um. Eine zehnstellige
-- Nummer mit falschem Prüfzeichen bleibt, wie sie ist: Von ihr führte die Rechnung auf die
-- ISBN-13 eines anderen Buchs (am Testserver 3499500252). Alles Übrige gilt wie in 133.
--
-- Der Bestand wird nachgezogen wie in Migration 140: Eine Zeile, deren Normalform schon ein
-- anderer Titel trägt, bleibt stehen, und die Migration nennt die Zahl als NOTICE.
-- Zusammengelegt wird nichts. Gemessen am Testserver am 03.10.2026, lesend: 100 zehnstellige
-- Nummern unter 13.062 Titeln, keine mit richtigem Prüfzeichen — die Migration ändert dort
-- keine Zeile. Die Littera-Sicherung von 2010 und der Katalog-Export vom Juni 2026 tragen
-- ebenfalls keine gültige zehnstellige ISBN.
--
-- Idempotent: Beim zweiten Lauf weicht außer den stehengelassenen Zeilen keine mehr ab. Der
-- Trigger aus Migration 133 ruft die Funktion beim Schreiben und bleibt, wie er ist.
-- =============================================================================

CREATE OR REPLACE FUNCTION isbn_normalform(roh text)
RETURNS text LANGUAGE plpgsql IMMUTABLE STRICT AS $$
DECLARE
    ohne text := upper(regexp_replace(roh, '[- ]', '', 'g'));
    summe integer := 0;
BEGIN
    IF ohne = '' THEN
        RETURN NULL;
    END IF;
    IF ohne ~ '^[0-9]{13}$' THEN
        RETURN ohne;
    END IF;
    -- Was keine ISBN ist, bleibt, wie es geschrieben wurde: Es wird nicht still zu einer
    -- anderen Zeichenkette.
    IF ohne !~ '^[0-9]{9}[0-9X]$' THEN
        RETURN btrim(roh);
    END IF;
    -- Prüfzeichen der ISBN-10: Gewichte 10 bis 1, X zählt 10, die Summe teilt sich durch 11.
    FOR i IN 1..10 LOOP
        summe := summe + (11 - i)
            * CASE substr(ohne, i, 1) WHEN 'X' THEN 10 ELSE substr(ohne, i, 1)::integer END;
    END LOOP;
    IF summe % 11 <> 0 THEN
        RETURN ohne;
    END IF;
    -- Prüfziffer der ISBN-13: Gewichte 1 und 3 im Wechsel; 978 trägt 9 + 21 + 8 bei.
    summe := 38;
    FOR i IN 1..9 LOOP
        summe := summe + substr(ohne, i, 1)::integer * CASE WHEN i % 2 = 1 THEN 3 ELSE 1 END;
    END LOOP;
    RETURN '978' || left(ohne, 9) || ((10 - summe % 10) % 10)::text;
END $$;

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
        RAISE NOTICE 'Migration 157: % Titel behalten ihre ISBN, weil ein anderer Titel dieselbe trägt (Dublette, von Hand entscheiden)', rest;
    END IF;
END $$;
