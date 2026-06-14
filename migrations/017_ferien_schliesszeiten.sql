-- 017_ferien_schliesszeiten.sql
-- Tabelle für Schließzeiten / Ferien (Pausierung des Mahnwesens)

CREATE TABLE ferien_schliesszeiten (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bezeichnung VARCHAR(255) NOT NULL,
    start_datum DATE NOT NULL,
    end_datum DATE NOT NULL,
    erstellt_am TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
