-- Migration: Create schueler_fotos table for encrypted photo storage

CREATE TABLE schueler_fotos (
    schueler_id UUID PRIMARY KEY REFERENCES schueler(id) ON DELETE CASCADE,
    foto_encrypted BYTEA NOT NULL,
    aktualisiert_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Trigger for auto-updating aktualisiert_am
CREATE TRIGGER trg_schueler_fotos_aktualisiert_am
BEFORE UPDATE ON schueler_fotos
FOR EACH ROW EXECUTE FUNCTION set_aktualisiert_am();
