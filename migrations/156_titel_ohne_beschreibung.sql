-- =============================================================================
-- Migration 156: Titel tragen keine Beschreibung mehr
-- =============================================================================
-- Das Feld „Beschreibung / Klappentext" am Titel entfällt (entschieden am 03.10.2026). Gezeigt
-- hat es nur die Maske „Buch bearbeiten"; der Katalog für Leser gab es nie aus. Ein
-- Klappentext gehört dem Verlag: Die DNB reicht ihn nur als Verweis weiter und nennt für die
-- Nachnutzung das Verzeichnis Lieferbarer Bücher. Gefüllt war die Spalte nicht. Gemessen am
-- 03.10.2026, lesend: am Testserver bei 0 von 13.062 Titeln, lokal bei 0 von 10.416.
--
-- Mit der Spalte ändern sich die drei Dinge, die sie nennen: der Volltext-Suchvektor
-- (Migration 050) umfasst danach Titel, Untertitel, Autor, Verlag und ISBN; Funktion und
-- Trigger für die Zeichenform der Titeltexte (Migration 154) führen vier statt fünf Spalten.
-- Ein Text, der in der Spalte steht, ist mit ihr gelöscht.
--
-- Idempotent: Beim zweiten Lauf fehlt die Spalte schon; Suchvektor und Index entstehen neu.
-- =============================================================================

CREATE OR REPLACE FUNCTION titel_text_in_nfc()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    NEW.titel := normalize(NEW.titel, NFC);
    NEW.untertitel := normalize(NEW.untertitel, NFC);
    NEW.autor := normalize(NEW.autor, NFC);
    NEW.verlag := normalize(NEW.verlag, NFC);
    RETURN NEW;
END $$;

DROP TRIGGER IF EXISTS trg_titel_text_nfc ON buecher_titel;
CREATE TRIGGER trg_titel_text_nfc
BEFORE INSERT OR UPDATE OF titel, untertitel, autor, verlag ON buecher_titel
FOR EACH ROW EXECUTE FUNCTION titel_text_in_nfc();

-- Der Ausdruck einer GENERATED-Spalte lässt sich nicht ändern: verwerfen und neu anlegen.
-- Der GIN-Index fällt mit der Spalte.
ALTER TABLE buecher_titel DROP COLUMN IF EXISTS search_vector;
ALTER TABLE buecher_titel DROP COLUMN IF EXISTS beschreibung;

ALTER TABLE buecher_titel ADD COLUMN search_vector TSVECTOR GENERATED ALWAYS AS (
    to_tsvector('german',
        coalesce(titel, '') || ' ' ||
        coalesce(untertitel, '') || ' ' ||
        coalesce(autor, '') || ' ' ||
        coalesce(verlag, '') || ' ' ||
        coalesce(isbn, '')
    )
) STORED;

CREATE INDEX idx_buecher_titel_search ON buecher_titel USING GIN (search_vector);
