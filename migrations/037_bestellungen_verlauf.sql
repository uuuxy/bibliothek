-- Bestellverlauf: ein Datensatz pro abgeschickter Bestellung
CREATE TABLE bestellungen_verlauf (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lieferant_id     UUID REFERENCES lieferanten(id) ON DELETE SET NULL,
    lieferant_name   TEXT NOT NULL,
    lieferant_email  TEXT NOT NULL,
    kundennummer     TEXT NOT NULL DEFAULT '',
    bestelldatum     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    gesamtbetrag     DECIMAL(10,2) NOT NULL DEFAULT 0.00,
    anzahl_exemplare INTEGER NOT NULL DEFAULT 0
);

-- Positionen: eine Zeile pro Titelposition in der Bestellung
CREATE TABLE bestellungen_positionen (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bestellung_id UUID NOT NULL REFERENCES bestellungen_verlauf(id) ON DELETE CASCADE,
    titel_id      UUID REFERENCES buecher_titel(id) ON DELETE SET NULL,
    titel_name    TEXT NOT NULL,
    isbn          TEXT NOT NULL DEFAULT '',
    menge         INTEGER NOT NULL,
    einzelpreis   DECIMAL(10,2) NOT NULL DEFAULT 0.00
);

CREATE INDEX idx_bestellpositionen_bestellung ON bestellungen_positionen(bestellung_id);
