-- =============================================================================
-- Migration 171: Die Bestandsbücher behalten die Zeile eines gelöschten Exemplars
-- =============================================================================
-- Zugangs- und Abgangsbuch lasen nur buecher_exemplare. Wer einen Titel löschte oder einen
-- Verlust nach der Inventur endgültig löschte, nahm die Exemplare aus beiden Büchern, auch aus
-- Halbjahren, deren Ausdruck schon abgeheftet war; unter dem Abgangsbuch stand nur ihre Zahl.
-- Das Verfahren der Schule verlangt den Nachweis der Aussonderungen (docs/mittel_konzept.md 1.1:
-- „nie löschen, immer aussondern").
--
-- Jede Zeile entsteht in derselben Anweisung, die das Exemplar löscht
-- (repository.loescheExemplareMitBuchzeile), und hält fest, was beide Bücher zeigen. Kein
-- Fremdschlüssel: Das Exemplar gibt es danach nicht mehr. Keine Personendaten.
--
-- Ohne Messung: Die Tabelle ist neu und leer; vorhandene Zeilen ändert die Migration nicht.
-- Idempotent: IF NOT EXISTS.
-- =============================================================================

CREATE TABLE IF NOT EXISTS geloeschte_exemplare (
    -- Die Kennung des gelöschten Exemplars. Über sie erkennt das Abgangsbuch, welche Spur im
    -- Protokoll schon eine Zeile hat.
    exemplar_id UUID PRIMARY KEY,
    barcode     TEXT NOT NULL,
    titel       TEXT NOT NULL,
    signatur    TEXT NOT NULL DEFAULT '',
    -- Nur Exemplare, die im Bestand waren: Ein bestelltes, nie eingetroffenes steht in keinem
    -- der beiden Bücher (repository.sqlWarImBestand).
    zugang_am   DATE NOT NULL,
    lieferant   TEXT NOT NULL DEFAULT '',
    -- Die beiden Topf-Regeln der Bücher: belegt (Zugangsbuch, exemplarTopfBelegtSQL) und mit
    -- der Vorgabe des Titels (Abgangsbuch, ExemplarTopfSQL).
    topf_belegt TEXT
        CONSTRAINT chk_geloeschte_exemplare_topf_belegt CHECK (topf_belegt IN ('land', 'schultraeger')),
    topf        TEXT NOT NULL
        CONSTRAINT chk_geloeschte_exemplare_topf CHECK (topf IN ('land', 'schultraeger')),
    -- War das Exemplar schon ausgesondert, Tag und Grund von damals; sonst der Zeitpunkt des
    -- Löschens und der Grund, den der Mensch gewählt hat. NULL nur bei Altbestand, der ohne
    -- Datum ausgesondert ist (Abgangsbuch: „ohne Zeitpunkt").
    abgang_am   TIMESTAMP WITH TIME ZONE,
    grund       VARCHAR(20) NOT NULL
        CONSTRAINT chk_geloeschte_exemplare_grund
        CHECK (grund IN ('VERLUST', 'BESCHAEDIGUNG', 'AUSSORTIERT', 'BESTANDSKORREKTUR')),
    geloescht_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_geloeschte_exemplare_abgang_am ON geloeschte_exemplare (abgang_am);
CREATE INDEX IF NOT EXISTS idx_geloeschte_exemplare_zugang_am ON geloeschte_exemplare (zugang_am);
