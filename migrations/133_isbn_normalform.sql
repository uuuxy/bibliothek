-- Migration 133: Eine ISBN hat EINE Schreibweise in der Datenbank — an jeder Tür.
--
-- Der UNIQUE-Index auf buecher_titel.isbn fing nur die zeichengleiche Dublette:
-- „978-3-16-148410-0" und „9783161484100" waren zwei Titel, also zwei Bestände, zwei
-- Meldebestände und zwei Zeilen in der Nachbestellung (OFFEN.md 5.12, 4.18). Die Maske
-- prüfte seit dem 17.09.2026 beide Schreibweisen (inventur/dublettenkontrolle.go), die
-- Importe nicht — und jeder weitere Schreiber (Littera-Übernahme, Sammelimport,
-- Schnellanlage per ISBN, Skripte) müsste die Regel kennen. Fünf Go-Helfer bereinigten
-- die Nummer auf fünf Arten.
--
-- Jetzt bringt die Datenbank selbst jede geschriebene ISBN in die Normalform: ohne
-- Bindestriche und Leerzeichen, Prüfzeichen X groß — aber nur, wenn das Ergebnis eine
-- ISBN ist (10 oder 13 Zeichen aus Ziffern und X). Was keine ISBN ist, bleibt wie
-- geschrieben (Seed-Kennungen wie „ISBN-0000000001", Altwerte mit Beiwerk); leer wird
-- NULL, denn '' wäre eine zweite Art von „keine ISBN". Damit greift der vorhandene
-- UNIQUE-Index über alle Schreibweisen — ohne zweiten Index auf einem Ausdruck.
--
-- Bewusst KEIN Rückschreiben des Bestands: Zwei Altzeilen, die sich nur in der
-- Schreibweise unterscheiden, kollidierten dabei am Index, und ob es solche gibt, weiß
-- nur die Messung am Server (OFFEN.md 5.5). Bis dahin deckt die Dublettenkontrolle der
-- Maske die Zwischenzeit; das Rückschreiben ist eine eigene Migration nach der Zahl.

CREATE OR REPLACE FUNCTION isbn_normalform(roh text)
RETURNS text LANGUAGE sql IMMUTABLE AS $$
    -- Nur, was eine ISBN IST: Bindestriche und Leerzeichen weg, Prüfzeichen groß — und
    -- das Ergebnis muss 10 oder 13 Zeichen aus Ziffern und X haben. Alles andere bleibt,
    -- wie es geschrieben wurde (getrimmt): Ein Wert, der keine ISBN ist, wird nicht still
    -- zu einer anderen Zeichenkette oder zu NULL.
    SELECT CASE
        WHEN regexp_replace(roh, '[- ]', '', 'g') = '' THEN NULL
        WHEN upper(regexp_replace(roh, '[- ]', '', 'g')) ~ '^([0-9]{9}[0-9X]|[0-9]{13})$'
            THEN upper(regexp_replace(roh, '[- ]', '', 'g'))
        ELSE btrim(roh)
    END
$$;

CREATE OR REPLACE FUNCTION titel_isbn_in_normalform()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    NEW.isbn := isbn_normalform(NEW.isbn);
    RETURN NEW;
END $$;

DROP TRIGGER IF EXISTS trg_titel_isbn_normalform ON buecher_titel;
CREATE TRIGGER trg_titel_isbn_normalform
BEFORE INSERT OR UPDATE OF isbn ON buecher_titel
FOR EACH ROW EXECUTE FUNCTION titel_isbn_in_normalform();
