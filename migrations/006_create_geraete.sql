CREATE TABLE geraete (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    modellname VARCHAR(255) NOT NULL,
    seriennummer VARCHAR(255) UNIQUE,
    barcode_id VARCHAR(100) UNIQUE NOT NULL,
    zubehoer TEXT NOT NULL DEFAULT '',
    ist_ausleihbar BOOLEAN NOT NULL DEFAULT true,
    ist_ausgesondert BOOLEAN NOT NULL DEFAULT false,
    zustand_notiz TEXT,
    erstellt_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    aktualisiert_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER trg_geraete_aktualisiert_am
BEFORE UPDATE ON geraete
FOR EACH ROW EXECUTE FUNCTION set_aktualisiert_am();

-- Modify ausleihen
ALTER TABLE ausleihen ADD COLUMN IF NOT EXISTS geraet_id UUID REFERENCES geraete(id) ON DELETE RESTRICT;
ALTER TABLE ausleihen ALTER COLUMN exemplar_id DROP NOT NULL;

ALTER TABLE ausleihen ADD CONSTRAINT check_loan_item CHECK (
    (exemplar_id IS NOT NULL AND geraet_id IS NULL) OR
    (exemplar_id IS NULL AND geraet_id IS NOT NULL)
);

-- Modify schadensfaelle
ALTER TABLE schadensfaelle ADD COLUMN IF NOT EXISTS geraet_id UUID REFERENCES geraete(id) ON DELETE RESTRICT;
ALTER TABLE schadensfaelle ALTER COLUMN exemplar_id DROP NOT NULL;

ALTER TABLE schadensfaelle ADD CONSTRAINT check_damage_item CHECK (
    (exemplar_id IS NOT NULL AND geraet_id IS NULL) OR
    (exemplar_id IS NULL AND geraet_id IS NOT NULL)
);
