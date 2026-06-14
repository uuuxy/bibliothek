-- Migration 009: Erweiterte Verwaltung (Adressdaten, Systematik, Mail-Vorlagen)

-- 1. SCHÜLER / LESER ERWEITERN
ALTER TABLE schueler
ADD COLUMN strasse VARCHAR(255),
ADD COLUMN hausnummer VARCHAR(50),
ADD COLUMN plz VARCHAR(20),
ADD COLUMN ort VARCHAR(255),
ADD COLUMN eltern_email VARCHAR(255);

-- 2. SYSTEMATIK & LESERGRUPPEN
CREATE TABLE systematik_kategorien (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kuerzel VARCHAR(50) UNIQUE NOT NULL,
    bezeichnung VARCHAR(255) NOT NULL,
    erstellt_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    aktualisiert_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER trg_systematik_kategorien_aktualisiert_am
BEFORE UPDATE ON systematik_kategorien
FOR EACH ROW EXECUTE FUNCTION set_aktualisiert_am();

CREATE TABLE lesergruppen (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kuerzel VARCHAR(50) UNIQUE NOT NULL,
    bezeichnung VARCHAR(255) NOT NULL,
    erstellt_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    aktualisiert_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER trg_lesergruppen_aktualisiert_am
BEFORE UPDATE ON lesergruppen
FOR EACH ROW EXECUTE FUNCTION set_aktualisiert_am();

-- 3. MAIL-VORLAGEN
CREATE TABLE mail_vorlagen (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    typ VARCHAR(100) UNIQUE NOT NULL,
    betreff VARCHAR(255) NOT NULL,
    text_body TEXT NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Trigger für updated_at
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_mail_vorlagen_updated_at
BEFORE UPDATE ON mail_vorlagen
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
