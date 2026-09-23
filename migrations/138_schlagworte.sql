-- Migration 138: Schlagworte am Titel — frei eintragbar, mit Vorschlägen aus dem Bestand
--
-- Entschieden am 22.09.2026 (docs/OFFEN.md 4.20: Themen werden gebaut), am 23.09.2026
-- geändert: statt eines geschlossenen Vokabulars von zwölf bis zwanzig Wörtern die freie
-- Eintragung, wie Littera sie hat („Schlagworte (Wertehilfe)": mehrere je Titel,
-- Vorschläge beim Tippen). Zusammengehalten wird die Liste durch eine Pflegeseite mit
-- Umbenennen, Zusammenführen und Verweisen — das ist Stufe 2, die ihre eigene Migration
-- mitbringt, weil dort die Verweise entstehen.
--
-- Eine Tabelle der Wörter und eine Verbindung zum Titel, keine Textspalte am Titel. Der
-- Grund ist die Pflege: Wer „Fantasie" in „Fantasy" umbenennt, ändert hier EINE Zeile,
-- und alle Titel folgen. In einer Spalte am Titel müsste dieselbe Korrektur jede Zeile
-- umschreiben, die das Wort trägt — und jede übersehene Zeile wäre eine vierte
-- Schreibweise. Dasselbe gilt für das Zählen: „wie viele Titel tragen das Wort" ist hier
-- ein JOIN über einen Index.
--
-- Die Identität eines Wortes ist seine Kleinschreibung (eindeutiger Index auf
-- lower(wort)). Wer „fantasy" tippt, bekommt das vorhandene „Fantasy" — die zuerst
-- angelegte Schreibweise gewinnt, wie bei den Fächern (repository.StelleFaecherSicher).
-- Die Form (getrimmt, nicht leer, höchstens 80 Zeichen) prüft die Datenbank selbst; das
-- Zusammenziehen innerer Leerzeichen macht der Schreibpfad (repository.NormalisiereSchlagworte).

CREATE TABLE IF NOT EXISTS schlagworte (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wort TEXT NOT NULL,
    angelegt_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_schlagwort_form
        CHECK (wort = btrim(wort) AND wort <> '' AND char_length(wort) <= 80)
);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_schlagworte_wort ON schlagworte (lower(wort));

COMMENT ON TABLE schlagworte IS
    'Schlagworte des Bestands (Migration 138), frei eintragbar wie in Littera. Ein Wort ist '
    'case-insensitiv eindeutig; die zuerst angelegte Schreibweise gewinnt. Geschrieben nur '
    'über repository.SetzeSchlagworte.';

-- Ein Titel trägt beliebig viele Wörter, ein Wort beliebig viele Titel. Wird ein Titel
-- gelöscht, fallen seine Verbindungen mit; wird ein Wort gelöscht, verschwindet es aus
-- allen Titeln (Littera: „Einträge können gelöscht werden (auch aus allen Medien)").
CREATE TABLE IF NOT EXISTS titel_schlagworte (
    titel_id UUID NOT NULL REFERENCES buecher_titel(id) ON DELETE CASCADE,
    schlagwort_id UUID NOT NULL REFERENCES schlagworte(id) ON DELETE CASCADE,
    PRIMARY KEY (titel_id, schlagwort_id)
);
-- Der Primärschlüssel beginnt mit titel_id und trägt „die Wörter eines Titels". Für die
-- Gegenrichtung — „wie viele Titel tragen das Wort", später „alle Titel zum Wort" in
-- Katalog und Portal — braucht es den zweiten Index.
CREATE INDEX IF NOT EXISTS idx_titel_schlagworte_schlagwort ON titel_schlagworte (schlagwort_id);

COMMENT ON TABLE titel_schlagworte IS
    'Welcher Titel welches Schlagwort trägt (Migration 138). Geschrieben nur über '
    'repository.SetzeSchlagworte, das die Menge eines Titels als Ganzes ersetzt.';
