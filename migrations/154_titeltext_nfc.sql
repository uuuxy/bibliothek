-- =============================================================================
-- Migration 154: Titeltexte in einer Unicode-Form (NFC) — an jeder Tür
-- =============================================================================
-- Die DNB liefert Umlaute und Akzente zerlegt: den Grundbuchstaben und das Zeichen dahinter
-- („u" + U+0308 statt „ü"; gemessen am 30.09.2026 an ihrer MARC21-Schnittstelle). Das Programm
-- speicherte das, wie es kam. Zerlegt und zusammengesetzt sehen gleich aus, sind für ILIKE, die
-- Volltextsuche und jeden Vergleich aber verschiedene Zeichenketten: Am Testserver fand der
-- Katalog (auch „Mein Portal") „Der Herr der Ringe - Anhänge und Register" mit „Anhänge" nicht,
-- weder über ILIKE noch über den Volltext — mit „Register" schon.
--
-- Wie die ISBN seit Migration 133 bringt jetzt die Datenbank selbst jeden geschriebenen
-- Titeltext in die zusammengesetzte Form (NFC). Die Titel entstehen an vielen Türen
-- (Buchformular, Bestellung per ISBN, Katalog-Import, Littera-Übernahme, Listen- und
-- CSV-Import); keine davon muss die Regel kennen. Die Signatur ist nicht dabei: Sie kommt aus
-- Littera und von Hand, nie aus der DNB, und bleibt Zeichen für Zeichen, wie sie am Buch steht.
--
-- Der Bestand wird in derselben Migration nachgezogen. Gemessen am Testserver am 30.09.2026:
-- 19 Titel mit zerlegten Zeichen (12 im Titel, 9 beim Autor, 2 beim Verlag; Untertitel und
-- Beschreibung keiner), lokal keiner. Keine der fünf Spalten ist eindeutig, das Nachziehen
-- kann also nicht an einem Index scheitern. Idempotent: Beim zweiten Lauf ist nichts zerlegt.
-- =============================================================================

CREATE OR REPLACE FUNCTION titel_text_in_nfc()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    NEW.titel := normalize(NEW.titel, NFC);
    NEW.untertitel := normalize(NEW.untertitel, NFC);
    NEW.autor := normalize(NEW.autor, NFC);
    NEW.verlag := normalize(NEW.verlag, NFC);
    NEW.beschreibung := normalize(NEW.beschreibung, NFC);
    RETURN NEW;
END $$;

DROP TRIGGER IF EXISTS trg_titel_text_nfc ON buecher_titel;
CREATE TRIGGER trg_titel_text_nfc
BEFORE INSERT OR UPDATE OF titel, untertitel, autor, verlag, beschreibung ON buecher_titel
FOR EACH ROW EXECUTE FUNCTION titel_text_in_nfc();

-- Der Bestand: Der Trigger läuft beim UPDATE mit und setzt dieselbe Form.
UPDATE buecher_titel
SET titel = normalize(titel, NFC),
    untertitel = normalize(untertitel, NFC),
    autor = normalize(autor, NFC),
    verlag = normalize(verlag, NFC),
    beschreibung = normalize(beschreibung, NFC)
WHERE NOT (coalesce(titel, '') IS NFC NORMALIZED)
   OR NOT (coalesce(untertitel, '') IS NFC NORMALIZED)
   OR NOT (coalesce(autor, '') IS NFC NORMALIZED)
   OR NOT (coalesce(verlag, '') IS NFC NORMALIZED)
   OR NOT (coalesce(beschreibung, '') IS NFC NORMALIZED);
