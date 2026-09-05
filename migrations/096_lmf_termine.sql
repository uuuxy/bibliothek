-- =============================================================================
-- Migration 096: LMF-Plan — Rückgabe- und Ausgabetermine je Klasse
-- =============================================================================
-- Betreiber-Entscheidung 05.09.2026 (Register, Entscheidung 3): Die Schule führte den
-- Plan als Excel-Tabelle (Wochentag, Datum, Stunde, Klasse(n), Besonderheiten) und
-- mailte ihn dem Kollegium; Korrekturen kamen als Folge-Mail. Vor den Sommerferien
-- „Bücherrückgabe" für alle Klassen, Abschlussklassen zuerst — wer keine Abschluss-
-- klasse ist, bekommt am selben Termin die neuen Bücher; nach den Ferien
-- „Bücherausgabe" für die neu gebildeten Klassen (5er, 7er) plus Nachzügler;
-- dazwischen Zeilen „Bücher setzen" ohne Klasse. Klassenzahl schwankt je Jahr.
--
-- Zwei Tabellen: ein Termin (Datum, Stunde, Art, Vermerk) und seine 0..n Klassen aus
-- dem Vokabular (Migration 079) — „6F1/6F2" in einer Stunde ist erlaubt, „Bücher
-- setzen" ohne Klasse auch. Der Rückgabe-Termin einer Klasse wird die Frist ihrer
-- Lernmittel (Entscheidung 3a); Ausgabe-Zeilen setzen keine Frist.
CREATE TABLE lmf_termine (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    datum           DATE NOT NULL,
    stunde          SMALLINT NOT NULL CONSTRAINT chk_lmf_termine_stunde CHECK (stunde BETWEEN 1 AND 12),
    art             TEXT NOT NULL CONSTRAINT chk_lmf_termine_art CHECK (art IN ('rueckgabe', 'ausgabe')),
    vermerk         TEXT NOT NULL DEFAULT '',
    erstellt_am     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    aktualisiert_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER trg_lmf_termine_aktualisiert_am
BEFORE UPDATE ON lmf_termine
FOR EACH ROW EXECUTE FUNCTION set_aktualisiert_am();

CREATE INDEX idx_lmf_termine_datum ON lmf_termine (datum, stunde);

CREATE TABLE lmf_termin_klassen (
    termin_id UUID NOT NULL REFERENCES lmf_termine(id) ON DELETE CASCADE,
    klasse    VARCHAR(50) NOT NULL,
    PRIMARY KEY (termin_id, klasse),
    CONSTRAINT fk_lmf_termin_klassen_vokabular
        FOREIGN KEY (klasse) REFERENCES klassen (name)
        ON UPDATE CASCADE ON DELETE RESTRICT
);

CREATE TRIGGER trg_lmf_termin_klassen_vokabular
BEFORE INSERT OR UPDATE OF klasse ON lmf_termin_klassen
FOR EACH ROW EXECUTE FUNCTION klasse_kanonisieren();

CREATE INDEX idx_lmf_termin_klassen_klasse ON lmf_termin_klassen (klasse);
