-- PostgreSQL Schema for a School Library System (Bibliothek)
-- Designed for ~80,000 books and ~2,000 students.
-- Uses UUIDs, generated columns for full-text search, and explicit relationships.

-- Enable pgcrypto or uuid-ossp for UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- -------------------------------------------------------------
-- 1. ENUMS AND CUSTOM TYPES
-- -------------------------------------------------------------
CREATE TYPE benutzer_rolle AS ENUM ('admin', 'lehrer', 'mitarbeiter');

-- -------------------------------------------------------------
-- 2. REUSABLE TRIGGER FUNCTIONS
-- -------------------------------------------------------------

-- Automatically update the aktualisiert_am timestamp on row updates
CREATE OR REPLACE FUNCTION set_aktualisiert_am()
RETURNS TRIGGER AS $$
BEGIN
    NEW.aktualisiert_am = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------------------
-- 3. TABLES
-- -------------------------------------------------------------

-- Table: benutzer (System administrators, teachers, and library staff)
CREATE TABLE benutzer (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    barcode_id VARCHAR(100) UNIQUE,                   -- Barcode ID for fast login
    vorname VARCHAR(100) NOT NULL,
    nachname VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    passwort_hash VARCHAR(255) NOT NULL,
    rolle benutzer_rolle NOT NULL,
    aktiv BOOLEAN NOT NULL DEFAULT true,
    erstellt_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    aktualisiert_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Table: schueler (Students borrowing books)
CREATE TABLE schueler (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    barcode_id VARCHAR(100) UNIQUE NOT NULL,          -- Barcode ID on student ID card
    vorname VARCHAR(100) NOT NULL,
    nachname VARCHAR(100) NOT NULL,
    klasse VARCHAR(20) NOT NULL,                      -- e.g., '5a', '10b', 'Q2'
    abgaenger_jahr INTEGER NOT NULL,                  -- Graduation/leaving year (useful for batch archiving)
    ist_gesperrt BOOLEAN NOT NULL DEFAULT false,      -- Flag to suspend borrowing privileges
    lusd_id VARCHAR(64) UNIQUE,                       -- Integrated LUSD ID
    ist_abgaenger BOOLEAN NOT NULL DEFAULT false,     -- Integrated ist_abgaenger
    erstellt_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    aktualisiert_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Table: buecher_titel (Master book catalog metadata)
-- Under the strict rule: metadata is separated from physical copies
CREATE TABLE buecher_titel (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    titel VARCHAR(255) NOT NULL,
    untertitel VARCHAR(255),
    autor VARCHAR(255),
    isbn VARCHAR(20) UNIQUE,                          -- ISBN-10 or ISBN-13
    verlag VARCHAR(255),
    erscheinungsjahr INTEGER,
    beschreibung TEXT,
    meldebestand INTEGER NOT NULL DEFAULT 5,          -- Reorder threshold point
    cover_url VARCHAR(512),                           -- Integrated cover URL
    subject VARCHAR(100),                             -- Integrated from books table
    grade_level SMALLINT,                             -- Integrated from books table
    track VARCHAR(100),                               -- Integrated from books table
    stock INTEGER NOT NULL DEFAULT 0,                 -- Integrated from books table
    last_counted DATE,                                -- Integrated from books table
    sort_order SERIAL,                                -- Integrated from books table
    erstellt_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    aktualisiert_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Immutable generated column for German language full-text search indexing
    search_vector TSVECTOR GENERATED ALWAYS AS (
        to_tsvector('german', 
            coalesce(titel, '') || ' ' || 
            coalesce(untertitel, '') || ' ' || 
            coalesce(autor, '') || ' ' || 
            coalesce(verlag, '')
        )
    ) STORED
);

-- Table: buecher_exemplare (Physical items / book copies in circulation)
CREATE TABLE buecher_exemplare (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    titel_id UUID NOT NULL REFERENCES buecher_titel(id) ON DELETE CASCADE, -- Cascade delete copies if title is deleted
    barcode_id VARCHAR(100) UNIQUE NOT NULL,          -- Unique barcode sticker on the book copy
    zustand_notiz TEXT,                               -- Field for damage notes / physical condition remarks
    erworben_am DATE NOT NULL DEFAULT CURRENT_DATE,
    ist_ausleihbar BOOLEAN NOT NULL DEFAULT true,      -- Switch to block copies from being lent out
    inventur_geprueft_am TIMESTAMP WITH TIME ZONE,    -- Inventory scan check timestamp
    erstellt_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    aktualisiert_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Table: ausleihen (Tracking loans/transactions)
CREATE TABLE ausleihen (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exemplar_id UUID NOT NULL REFERENCES buecher_exemplare(id) ON DELETE RESTRICT,
    
    -- Polymorphic borrower association (loan to student OR user/staff)
    schueler_id UUID REFERENCES schueler(id) ON DELETE RESTRICT,
    ausleiher_benutzer_id UUID REFERENCES benutzer(id) ON DELETE SET NULL,
    
    ausgeliehen_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    rueckgabe_frist TIMESTAMP WITH TIME ZONE NOT NULL,
    rueckgabe_am TIMESTAMP WITH TIME ZONE,
    
    bearbeiter_id UUID REFERENCES benutzer(id) ON DELETE SET NULL,          -- Staff checking out the book (Nullable for GDPR anonymization)
    rueckgabe_bearbeiter_id UUID REFERENCES benutzer(id) ON DELETE SET NULL,          -- Staff checking in the book
    
    ist_fremdrueckgabe BOOLEAN NOT NULL DEFAULT false, -- True if returned by someone other than the borrower
    ist_handapparat BOOLEAN NOT NULL DEFAULT false,    -- True if borrowed by a teacher (handapparat)
    
    -- Constraint: Exactly one borrower must be associated with the loan, or both NULL when anonymized/deleted
    CONSTRAINT check_loan_borrower CHECK (
        (schueler_id IS NOT NULL AND ausleiher_benutzer_id IS NULL) OR
        (schueler_id IS NULL AND ausleiher_benutzer_id IS NOT NULL) OR
        (schueler_id IS NULL AND ausleiher_benutzer_id IS NULL)
    ),
    
    -- Constraint: Return timestamp cannot precede the loan timestamp
    CONSTRAINT check_return_date CHECK (
        rueckgabe_am IS NULL OR rueckgabe_am >= ausgeliehen_am
    )
);

-- Table: schadensfaelle (Incidents concerning damaged or lost books)
CREATE TABLE schadensfaelle (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exemplar_id UUID NOT NULL REFERENCES buecher_exemplare(id) ON DELETE RESTRICT,
    ausleihe_id UUID REFERENCES ausleihen(id) ON DELETE SET NULL, -- Optional link to corresponding checkout
    
    -- Target person responsible (either student OR user/staff)
    schueler_id UUID REFERENCES schueler(id) ON DELETE RESTRICT,
    benutzer_id UUID REFERENCES benutzer(id) ON DELETE SET NULL,
    
    beschreibung TEXT NOT NULL,
    betrag NUMERIC(10, 2) NOT NULL DEFAULT 0.00 CONSTRAINT check_positive_amount CHECK (betrag >= 0.00),
    ist_bezahlt BOOLEAN NOT NULL DEFAULT false,
    elternbrief_generiert BOOLEAN NOT NULL DEFAULT false,
    elternbrief_generiert_am TIMESTAMP WITH TIME ZONE,
    
    erstellt_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    aktualisiert_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraint: Exactly one responsible person must be associated, or both NULL when anonymized/deleted
    CONSTRAINT check_damage_responsible CHECK (
        (schueler_id IS NOT NULL AND benutzer_id IS NULL) OR
        (schueler_id IS NULL AND benutzer_id IS NOT NULL) OR
        (schueler_id IS NULL AND benutzer_id IS NULL)
    )
);

-- -------------------------------------------------------------
-- 4. INDEXES FOR RAPID SEARCH AND INTEGRITY
-- -------------------------------------------------------------

-- GIN index for full-text search across title, subtitle, author, and publisher
CREATE INDEX idx_buecher_titel_search ON buecher_titel USING GIN (search_vector);

-- Indexes for rapid login/barcode lookups (B-Tree)
CREATE INDEX idx_benutzer_barcode ON benutzer (barcode_id) WHERE barcode_id IS NOT NULL;
CREATE INDEX idx_schueler_barcode ON schueler (barcode_id);
CREATE INDEX idx_buecher_exemplare_barcode ON buecher_exemplare (barcode_id);

-- Foreign key indexes (speeds up JOINs and referential integrity checks)
CREATE INDEX idx_buecher_exemplare_titel ON buecher_exemplare (titel_id);
CREATE INDEX idx_ausleihen_exemplar ON ausleihen (exemplar_id);
CREATE INDEX idx_ausleihen_schueler ON ausleihen (schueler_id) WHERE schueler_id IS NOT NULL;
CREATE INDEX idx_ausleihen_benutzer ON ausleihen (ausleiher_benutzer_id) WHERE ausleiher_benutzer_id IS NOT NULL;
CREATE INDEX idx_schadensfaelle_exemplar ON schadensfaelle (exemplar_id);
CREATE INDEX idx_schadensfaelle_schueler ON schadensfaelle (schueler_id) WHERE schueler_id IS NOT NULL;
CREATE INDEX idx_schadensfaelle_benutzer ON schadensfaelle (benutzer_id) WHERE benutzer_id IS NOT NULL;

-- Query specific indexes
CREATE INDEX idx_ausleihen_aktive ON ausleihen (rueckgabe_am) WHERE rueckgabe_am IS NULL; -- Highly active lookup for current loans
CREATE INDEX idx_schadensfaelle_offene ON schadensfaelle (ist_bezahlt) WHERE ist_bezahlt = false; -- Fast extraction of unpaid fees
CREATE INDEX idx_ausleihen_rueckgabe_frist ON ausleihen (rueckgabe_frist);

-- -------------------------------------------------------------
-- 5. TRIGGERS FOR METADATA SYNCHRONIZATION
-- -------------------------------------------------------------

-- Trigger: Update benutzer timestamp
CREATE TRIGGER trg_benutzer_aktualisiert_am
BEFORE UPDATE ON benutzer
FOR EACH ROW EXECUTE FUNCTION set_aktualisiert_am();

-- Trigger: Update schueler timestamp
CREATE TRIGGER trg_schueler_aktualisiert_am
BEFORE UPDATE ON schueler
FOR EACH ROW EXECUTE FUNCTION set_aktualisiert_am();

-- Trigger: Update buecher_titel timestamp
CREATE TRIGGER trg_buecher_titel_aktualisiert_am
BEFORE UPDATE ON buecher_titel
FOR EACH ROW EXECUTE FUNCTION set_aktualisiert_am();

-- Trigger: Update buecher_exemplare timestamp
CREATE TRIGGER trg_buecher_exemplare_aktualisiert_am
BEFORE UPDATE ON buecher_exemplare
FOR EACH ROW EXECUTE FUNCTION set_aktualisiert_am();

-- Trigger: Update schadensfaelle timestamp
CREATE TRIGGER trg_schadensfaelle_aktualisiert_am
BEFORE UPDATE ON schadensfaelle
FOR EACH ROW EXECUTE FUNCTION set_aktualisiert_am();

-- Table: audit_log (Audit trail for immutable security logs)
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tabelle VARCHAR(50) NOT NULL,
    aktion VARCHAR(20) NOT NULL,
    datensatz_id UUID NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    bearbeiter_id UUID REFERENCES benutzer(id) ON DELETE SET NULL
);

-- Table: class_books (LMF class to book catalog metadata association)
CREATE TABLE class_books (
    class_name VARCHAR(50) NOT NULL,
    book_id UUID NOT NULL REFERENCES buecher_titel(id) ON DELETE CASCADE,
    PRIMARY KEY (class_name, book_id)
);

-- Table: subjects (Active subjects for inventory module)
CREATE TABLE subjects (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true
);

-- Seed default subjects
INSERT INTO subjects (name, is_active) VALUES
('Deutsch', true),
('Englisch', true),
('Mathematik', true),
('Physik', true),
('Chemie', true),
('Biologie', true),
('Geschichte', true),
('Geographie', true),
('Politik', true),
('Informatik', true),
('Kunst', true),
('Musik', true),
('Religion', true),
('Latein', true),
('Französisch', true),
('Naturwissenschaften', true)
ON CONFLICT (name) DO NOTHING;

-- -------------------------------------------------------------
-- 6. DEFAULT TEST SEED DATA
-- -------------------------------------------------------------

-- Default Admin User (Barcode: ADMIN-1)
INSERT INTO benutzer (id, barcode_id, vorname, nachname, email, passwort_hash, rolle, aktiv)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'ADMIN-1',
    'System',
    'Administrator',
    'admin@bibliothek.local',
    'dummy_passwort_hash',
    'admin',
    true
) ON CONFLICT (email) DO NOTHING;

-- Default Teacher User (Barcode: L-999)
INSERT INTO benutzer (id, barcode_id, vorname, nachname, email, passwort_hash, rolle, aktiv)
VALUES (
    '00000000-0000-0000-0000-000000000002',
    'L-999',
    'Maria',
    'Müller',
    'm.mueller@schule.de',
    'dummy_passwort_hash',
    'lehrer',
    true
) ON CONFLICT (email) DO NOTHING;

-- Default Student (Barcode: S-100)
INSERT INTO schueler (id, barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_gesperrt)
VALUES (
    '00000000-0000-0000-0000-000000000003',
    'S-100',
    'Max',
    'Mustermann',
    '10b',
    2028,
    false
) ON CONFLICT (barcode_id) DO NOTHING;

-- Default Book Title (ISBN: 9783127337001)
INSERT INTO buecher_titel (id, titel, untertitel, autor, isbn, verlag, erscheinungsjahr, meldebestand)
VALUES (
    '00000000-0000-0000-0000-000000000004',
    'LMF-Mathematik 10',
    'Gymnasium Bayern',
    'Dr. Arndt',
    '9783127337001',
    'Klett',
    2022,
    5
) ON CONFLICT (id) DO NOTHING;

-- Default Book Copy (Barcode: B-200)
INSERT INTO buecher_exemplare (id, barcode_id, titel_id, erworben_am, ist_ausleihbar)
VALUES (
    '00000000-0000-0000-0000-000000000005',
    'B-200',
    '00000000-0000-0000-0000-000000000004',
    CURRENT_DATE,
    true
) ON CONFLICT (barcode_id) DO NOTHING;
