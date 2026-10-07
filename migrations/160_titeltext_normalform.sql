-- =============================================================================
-- Migration 160: Titeltexte ohne Leerraum in Folge — an jeder Tür
-- =============================================================================
-- In der Littera-Sicherung von 2010 tragen 103 von 10.732 Titeln zwei bis vier Leerzeichen in
-- Folge („La  Peste"), dazu 73 Untertitel, 23 Verfassernamen und 2 Verlage. 11 Titel tragen
-- ein geschütztes Leerzeichen (U+00A0, „Mumienherz. Die Rückkehr des Seth / 1"). Gemessen am
-- 07.10.2026 an den Tabellen Titel, Personen und Verlag; Tabulator, Zeilenumbruch und andere
-- Leerzeichen kommen dort nicht vor. Das Programm speicherte den Text, wie er kam. Die
-- Titel-Verwaltung, die Lernmittel-Liste und die Liste der Etiketten vergleichen den Suchtext
-- mit dem Wortlaut (ILIKE): „La Peste" traf „La  Peste" nicht.
--
-- Die Datenbank bringt jetzt jeden geschriebenen Titeltext selbst in eine Form, wie seit
-- Migration 154 bei den zerlegten Umlauten: kein Leerraum am Rand, Leerraum in Folge wird ein
-- Leerzeichen, Umlaute und Akzente zusammengesetzt (NFC). Die Titel entstehen an vielen Türen
-- (Buchformular, Bestellung per ISBN, Katalog-Import, Littera-Übernahme, Listen- und
-- CSV-Import); keine davon muss die Regel kennen. Die Signatur ist weiter nicht dabei: Sie
-- bleibt Zeichen für Zeichen, wie sie am Buch steht.
--
-- Leerraum sind die 25 Zeichen mit der Unicode-Eigenschaft White_Space, hier einzeln genannt.
-- Die Klasse \s hängt an der Sprachumgebung des Servers: Gemessen am 07.10.2026 an PostgreSQL
-- 18.6 (musl) trifft sie das geschützte Leerzeichen nicht. Go zählt mit unicode.IsSpace
-- dieselben 25 Zeichen (repository.TiteltextNormalform).
--
-- Funktion und Trigger heißen nach dem, was sie tun, wie isbn_normalform und
-- trg_titel_isbn_normalform; die Namen aus Migration 154 entfallen.
--
-- Der Bestand wird in derselben Migration nachgezogen. Keine der vier Spalten ist eindeutig,
-- und ein Text wird dabei nie länger: Das Nachziehen kann an keinem Index und an keiner
-- Spaltenbreite scheitern. Keine Tür trennt in diesen Spalten mit Zeilenumbruch oder
-- Tabulator (mehrere Verfasser stehen mit „; " hintereinander): Es geht nur Leerraum verloren,
-- der nichts bedeutet. Gemessen an der lokalen Datenbank am 07.10.2026: Kein Titel ist
-- betroffen; am Testserver ist nicht gemessen. Idempotent: Beim zweiten Lauf steht jeder Text
-- schon in der Form.
-- =============================================================================

CREATE OR REPLACE FUNCTION titeltext_normalform(roh text)
RETURNS text LANGUAGE sql IMMUTABLE STRICT AS $$
    SELECT normalize(btrim(regexp_replace(roh,
        '[\u0009-\u000D\u0020\u0085\u00A0\u1680\u2000-\u200A\u2028\u2029\u202F\u205F\u3000]+',
        ' ', 'g')), NFC)
$$;

CREATE OR REPLACE FUNCTION titel_text_in_normalform()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    NEW.titel := titeltext_normalform(NEW.titel);
    NEW.untertitel := titeltext_normalform(NEW.untertitel);
    NEW.autor := titeltext_normalform(NEW.autor);
    NEW.verlag := titeltext_normalform(NEW.verlag);
    RETURN NEW;
END $$;

DROP TRIGGER IF EXISTS trg_titel_text_nfc ON buecher_titel;
DROP FUNCTION IF EXISTS titel_text_in_nfc();

DROP TRIGGER IF EXISTS trg_titel_text_normalform ON buecher_titel;
CREATE TRIGGER trg_titel_text_normalform
BEFORE INSERT OR UPDATE OF titel, untertitel, autor, verlag ON buecher_titel
FOR EACH ROW EXECUTE FUNCTION titel_text_in_normalform();

-- Der Bestand: Der Trigger läuft beim UPDATE mit und setzt dieselbe Form.
UPDATE buecher_titel
SET titel = titeltext_normalform(titel),
    untertitel = titeltext_normalform(untertitel),
    autor = titeltext_normalform(autor),
    verlag = titeltext_normalform(verlag)
WHERE titel IS DISTINCT FROM titeltext_normalform(titel)
   OR untertitel IS DISTINCT FROM titeltext_normalform(untertitel)
   OR autor IS DISTINCT FROM titeltext_normalform(autor)
   OR verlag IS DISTINCT FROM titeltext_normalform(verlag);
