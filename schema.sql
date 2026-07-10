-- PostgreSQL Schema for a School Library System (Bibliothek)
-- Designed for ~80,000 books and ~2,000 students.
-- Uses UUIDs, generated columns for full-text search, and explicit relationships.

-- Enable pgcrypto or uuid-ossp for UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

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
-- 3. TABLES, INDEXES, AND TRIGGERS
-- -------------------------------------------------------------

-- Table: system_einstellungen (Configurable key-value system settings)
CREATE TABLE system_einstellungen (
    schluessel VARCHAR(100) PRIMARY KEY,
    wert TEXT,
    aktualisiert_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Table: ferien_schliesszeiten (Pausierung des Mahnwesens)
CREATE TABLE ferien_schliesszeiten (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bezeichnung VARCHAR(255) NOT NULL,
    start_datum DATE NOT NULL,
    end_datum DATE NOT NULL,
    erstellt_am TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Seed default system settings
INSERT INTO system_einstellungen (schluessel, wert) VALUES
    ('ferien_leseclub_aktiv', 'false'),
    ('ferien_leseclub_zieldatum', NULL),
    ('lmf_stichtag', '07-31'),
    ('max_ausleihen_schueler', '5'),
    ('frist_buch_tage', '21'),
    ('frist_medien_tage', '7'),
    ('max_overdue_days', '14'),
    ('max_overdue_items', '1'),
    ('schule_name', ''),
    ('schule_strasse', ''),
    ('schule_plz', ''),
    ('schule_ort', '')
ON CONFLICT (schluessel) DO NOTHING;


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


-- Table: benutzer (System administrators, teachers, and library staff)
CREATE TABLE benutzer (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    barcode_id VARCHAR(100) UNIQUE,                   -- Barcode ID for fast login
    vorname VARCHAR(100) NOT NULL,
    nachname VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    -- Passwörter wurden mit Migration 012 entfernt (Barcode-basierte Anmeldung).
    rolle benutzer_rolle NOT NULL,
    aktiv BOOLEAN NOT NULL DEFAULT true,
    erstellt_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    aktualisiert_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_benutzer_barcode ON benutzer (barcode_id) WHERE barcode_id IS NOT NULL;

CREATE TRIGGER trg_benutzer_aktualisiert_am
BEFORE UPDATE ON benutzer
FOR EACH ROW EXECUTE FUNCTION set_aktualisiert_am();


-- Table: benutzer_rollen (Mapping users to their roles)
CREATE TABLE benutzer_rollen (
    benutzer_id UUID PRIMARY KEY REFERENCES benutzer(id) ON DELETE CASCADE,
    rolle VARCHAR(50) NOT NULL CHECK (rolle IN ('ADMIN', 'MITARBEITER', 'LEHRER', 'HELFER'))
);


-- Table: schueler (Students borrowing books)
-- DSGVO Art. 5 Abs. 1 lit. c – Datensparsamkeit:
-- Erlaubte Felder (ausschließlich aus LUSD-Import): vorname, nachname, klasse,
-- geburtsdatum, lusd_id. Adress- und Kontaktdaten dürfen NICHT gespeichert werden.
CREATE TABLE schueler (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    barcode_id VARCHAR(100) UNIQUE NOT NULL,          -- Barcode ID on student ID card
    vorname VARCHAR(100) NOT NULL,
    nachname VARCHAR(100) NOT NULL,
    klasse VARCHAR(20) NOT NULL,                      -- e.g., '5a', '10b', 'Q2'
    geburtsdatum DATE DEFAULT NULL,                   -- LUSD-Feld; NULL für Altdatensätze
    abgaenger_jahr INTEGER NOT NULL,                  -- Graduation/leaving year (useful for batch archiving)
    ist_gesperrt BOOLEAN NOT NULL DEFAULT false,      -- Flag to suspend borrowing privileges
    lusd_id VARCHAR(64),                              -- Integrated LUSD ID (Eindeutigkeit: partieller Index uniq_schueler_lusd_id_active, nur aktive Zeilen)
    ist_abgaenger BOOLEAN NOT NULL DEFAULT false,     -- Integrated ist_abgaenger
    strasse VARCHAR(255),
    hausnummer VARCHAR(50),
    plz VARCHAR(20),
    ort VARCHAR(255),
    eltern_email VARCHAR(255),
    erstellt_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    aktualisiert_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_manually_blocked BOOLEAN DEFAULT false,
    block_reason TEXT,
    deleted_at TIMESTAMP WITH TIME ZONE DEFAULT NULL
);

CREATE INDEX idx_schueler_barcode ON schueler (barcode_id);
CREATE INDEX idx_schueler_vorname_trgm ON schueler USING gin (vorname gin_trgm_ops);
CREATE INDEX idx_schueler_nachname_trgm ON schueler USING gin (nachname gin_trgm_ops);
CREATE UNIQUE INDEX unique_schueler_name_gebdatum ON schueler (vorname, nachname, coalesce(geburtsdatum, '1900-01-01'::DATE));
-- lusd_id ist nur unter AKTIVEN Schülern eindeutig; eine soft-gelöschte lusd_id
-- darf bei Wiederanmeldung neu vergeben werden (siehe Migration 035).
CREATE UNIQUE INDEX uniq_schueler_lusd_id_active ON schueler (lusd_id) WHERE deleted_at IS NULL AND lusd_id IS NOT NULL;

CREATE TRIGGER trg_schueler_aktualisiert_am
BEFORE UPDATE ON schueler
FOR EACH ROW EXECUTE FUNCTION set_aktualisiert_am();


-- Table: schueler_fotos (Encrypted student photos)
CREATE TABLE schueler_fotos (
    schueler_id UUID PRIMARY KEY REFERENCES schueler(id) ON DELETE CASCADE,
    foto_encrypted BYTEA NOT NULL,
    aktualisiert_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER trg_schueler_fotos_aktualisiert_am
BEFORE UPDATE ON schueler_fotos
FOR EACH ROW EXECUTE FUNCTION set_aktualisiert_am();


-- Table: klassen_lehrer_mapping (Class → class teacher e-mail for automated reminders)
CREATE TABLE klassen_lehrer_mapping (
    klasse       VARCHAR(50)  PRIMARY KEY,
    lehrer_email VARCHAR(255) NOT NULL,
    erstellt_am  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);


-- Table: systematik_kategorien
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

-- Table: lesergruppen
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

-- Table: mail_vorlagen
CREATE TABLE mail_vorlagen (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    typ VARCHAR(100) UNIQUE NOT NULL,
    betreff VARCHAR(255) NOT NULL,
    text_body TEXT NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

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

-- Table: revoked_tokens
CREATE TABLE revoked_tokens (
    token_signature VARCHAR(255) PRIMARY KEY,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE INDEX idx_revoked_tokens_expires_at ON revoked_tokens(expires_at);


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
    cover_status VARCHAR(50) DEFAULT 'PENDING',       -- Added for async cover fetching
    signatur VARCHAR(255),                            -- Signature (e.g. from MAB 700)
    subject VARCHAR(100),                             -- Integrated from books table
    grade_level SMALLINT,                             -- Integrated from books table
    track VARCHAR(100),                               -- Integrated from books table
    stock INTEGER NOT NULL DEFAULT 0,                 -- Integrated from books table
    last_counted DATE,                                -- Integrated from books table
    sort_order SERIAL,                                -- Integrated from books table
    medientyp VARCHAR(100) NOT NULL DEFAULT 'Buch',   -- Media type (Book, CD, DVD, etc.)
    erweiterte_eigenschaften JSONB NOT NULL DEFAULT '{}', -- Flexible key-value metadata (e.g. shelf location, notes)
    ziel_jahrgang INTEGER NOT NULL DEFAULT 0,          -- Target grade level for loan duration calculation (0 = 1 year default)
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

CREATE INDEX idx_buecher_titel_search ON buecher_titel USING GIN (search_vector);
CREATE INDEX idx_buecher_titel_trgm ON buecher_titel USING gin (titel gin_trgm_ops);
CREATE INDEX idx_buecher_autor_trgm ON buecher_titel USING gin (autor gin_trgm_ops);
CREATE INDEX idx_buecher_isbn_trgm ON buecher_titel USING gin (isbn gin_trgm_ops);

CREATE TRIGGER trg_buecher_titel_aktualisiert_am
BEFORE UPDATE ON buecher_titel
FOR EACH ROW EXECUTE FUNCTION set_aktualisiert_am();


-- Table: buecher_exemplare (Physical items / book copies in circulation)
CREATE TABLE buecher_exemplare (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    titel_id UUID NOT NULL REFERENCES buecher_titel(id) ON DELETE CASCADE, -- Cascade delete copies if title is deleted
    barcode_id VARCHAR(100) UNIQUE NOT NULL,          -- Unique barcode sticker on the book copy
    zustand_notiz TEXT,                               -- Field for damage notes / physical condition remarks
    erworben_am DATE NOT NULL DEFAULT CURRENT_DATE,
    ist_ausleihbar BOOLEAN NOT NULL DEFAULT true,      -- Switch to block copies from being lent out
    inventur_geprueft_am TIMESTAMP WITH TIME ZONE,    -- Inventory scan check timestamp
    inventur_status VARCHAR(20) DEFAULT NULL,         -- 'ausstehend' or 'erfasst' during inventory
    ist_ausgesondert BOOLEAN NOT NULL DEFAULT false,   -- Decommissioned copies: hidden from catalog/kiosk/inventory, kept for statistics
    etikett_gedruckt BOOLEAN NOT NULL DEFAULT false,   -- True if barcode label has been printed
    erweiterte_eigenschaften JSONB NOT NULL DEFAULT '{}', -- Flexible key-value metadata (e.g. shelf position, condition details)
    einkaufspreis DECIMAL(10,2) NOT NULL DEFAULT 0.00,
    erstellt_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    aktualisiert_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_buecher_exemplare_barcode ON buecher_exemplare (barcode_id);
CREATE INDEX idx_buecher_exemplare_titel ON buecher_exemplare (titel_id);

CREATE TRIGGER trg_buecher_exemplare_aktualisiert_am
BEFORE UPDATE ON buecher_exemplare
FOR EACH ROW EXECUTE FUNCTION set_aktualisiert_am();


-- Table: class_books (LMF class to book catalog metadata association)
CREATE TABLE class_books (
    class_name VARCHAR(50) NOT NULL,
    book_id UUID NOT NULL REFERENCES buecher_titel(id) ON DELETE CASCADE,
    PRIMARY KEY (class_name, book_id)
);


-- Table: geraete (Hardware devices like iPads, Laptops)
CREATE TABLE geraete (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    modellname VARCHAR(255) NOT NULL,
    seriennummer VARCHAR(255) UNIQUE,
    barcode_id VARCHAR(100) UNIQUE NOT NULL,
    zubehoer TEXT NOT NULL DEFAULT '', -- Comma separated checklist items
    ist_ausleihbar BOOLEAN NOT NULL DEFAULT true,
    ist_ausgesondert BOOLEAN NOT NULL DEFAULT false,
    zustand_notiz TEXT,
    erstellt_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    aktualisiert_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);


-- Table: ausleihen (Tracking loans/transactions)
CREATE TABLE ausleihen (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exemplar_id UUID REFERENCES buecher_exemplare(id) ON DELETE RESTRICT,
    geraet_id UUID REFERENCES geraete(id) ON DELETE RESTRICT,
    
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
    
    -- Constraint: Exactly one item must be borrowed (book or device)
    CONSTRAINT check_loan_item CHECK (
        (exemplar_id IS NOT NULL AND geraet_id IS NULL) OR
        (exemplar_id IS NULL AND geraet_id IS NOT NULL)
    ),
    
    -- Constraint: Return timestamp cannot precede the loan timestamp
    CONSTRAINT check_return_date CHECK (
        rueckgabe_am IS NULL OR rueckgabe_am >= ausgeliehen_am
    )
);

CREATE INDEX idx_ausleihen_exemplar ON ausleihen (exemplar_id);
CREATE INDEX idx_ausleihen_schueler ON ausleihen (schueler_id) WHERE schueler_id IS NOT NULL;
CREATE INDEX idx_ausleihen_benutzer ON ausleihen (ausleiher_benutzer_id) WHERE ausleiher_benutzer_id IS NOT NULL;
CREATE INDEX idx_ausleihen_aktive ON ausleihen (rueckgabe_am) WHERE rueckgabe_am IS NULL; -- Highly active lookup for current loans
-- Datenintegrität: höchstens EINE aktive Ausleihe je Exemplar bzw. Gerät (siehe Migration 033).
CREATE UNIQUE INDEX IF NOT EXISTS uniq_ausleihen_aktiv_exemplar ON ausleihen (exemplar_id) WHERE rueckgabe_am IS NULL AND exemplar_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uniq_ausleihen_aktiv_geraet ON ausleihen (geraet_id) WHERE rueckgabe_am IS NULL AND geraet_id IS NOT NULL;
CREATE INDEX idx_ausleihen_rueckgabe_frist ON ausleihen (rueckgabe_frist);


-- Table: schadensfaelle (Incidents concerning damaged or lost books)
CREATE TABLE schadensfaelle (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exemplar_id UUID REFERENCES buecher_exemplare(id) ON DELETE RESTRICT,
    geraet_id UUID REFERENCES geraete(id) ON DELETE RESTRICT,
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
    storniert_am TIMESTAMP WITH TIME ZONE DEFAULT NULL,
    storniert_von UUID REFERENCES benutzer(id) ON DELETE SET NULL DEFAULT NULL,
    stornierungsgrund TEXT DEFAULT NULL,
    
    -- Constraint: Exactly one responsible person must be associated, or both NULL when anonymized/deleted
    CONSTRAINT check_damage_responsible CHECK (
        (schueler_id IS NOT NULL AND benutzer_id IS NULL) OR
        (schueler_id IS NULL AND benutzer_id IS NOT NULL) OR
        (schueler_id IS NULL AND benutzer_id IS NULL)
    ),
    
    -- Constraint: Exactly one item must be associated
    CONSTRAINT check_damage_item CHECK (
        (exemplar_id IS NOT NULL AND geraet_id IS NULL) OR
        (exemplar_id IS NULL AND geraet_id IS NOT NULL)
    )
);

CREATE INDEX idx_schadensfaelle_exemplar ON schadensfaelle (exemplar_id);
CREATE INDEX idx_schadensfaelle_schueler ON schadensfaelle (schueler_id) WHERE schueler_id IS NOT NULL;
CREATE INDEX idx_schadensfaelle_benutzer ON schadensfaelle (benutzer_id) WHERE benutzer_id IS NOT NULL;
CREATE INDEX idx_schadensfaelle_offene ON schadensfaelle (ist_bezahlt) WHERE ist_bezahlt = false; -- Fast extraction of unpaid fees

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
    bearbeiter_id UUID REFERENCES benutzer(id) ON DELETE SET NULL,
    details JSONB DEFAULT NULL,
    akteur VARCHAR(10) NOT NULL DEFAULT 'USER' CHECK (akteur IN ('USER', 'SYSTEM')),
    kontext TEXT DEFAULT NULL
);

-- Table: audit_logs (Admin-spezifische Eingriffe)
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id UUID REFERENCES benutzer(id) ON DELETE SET NULL,
    aktion VARCHAR(255) NOT NULL,
    details JSONB NOT NULL DEFAULT '{}',
    ip_adresse VARCHAR(45),
    zeitstempel TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);


-- Table: lieferanten (Book suppliers)
CREATE TABLE lieferanten (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    kundennummer VARCHAR(100) NOT NULL,
    erstellt_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);


-- Table: bestellungen_verlauf (Order history — one record per submitted order)
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

-- Table: bestellungen_positionen (Line items per order)
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


-- Table: vormerkungen (Individual book reservations / waitlist)
CREATE TABLE vormerkungen (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    titel_id    UUID NOT NULL REFERENCES buecher_titel(id) ON DELETE CASCADE,
    schueler_id UUID REFERENCES schueler(id) ON DELETE CASCADE,
    notiz       TEXT,
    erstellt_am TIMESTAMPTZ NOT NULL DEFAULT now(),
    status      VARCHAR(50) DEFAULT 'wartend' NOT NULL,
    bereitgestellt_exemplar_id UUID REFERENCES buecher_exemplare(id) ON DELETE SET NULL,
    
    UNIQUE(titel_id, schueler_id)
);

CREATE INDEX idx_vormerkungen_titel_id ON vormerkungen(titel_id);


-- Table: klassensatz_reservierungen (Teacher-submitted class-set reservation requests)
CREATE TABLE klassensatz_reservierungen (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    titel_id         UUID NOT NULL REFERENCES buecher_titel(id) ON DELETE CASCADE,
    klasse           VARCHAR(50) NOT NULL,
    anzahl           INTEGER NOT NULL DEFAULT 1,
    notiz            TEXT,
    angefordert_von  UUID REFERENCES benutzer(id) ON DELETE SET NULL,
    erledigt         BOOLEAN NOT NULL DEFAULT false,
    erstellt_am      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- -------------------------------------------------------------
-- 4. MARK MIGRATIONS AS APPLIED
-- -------------------------------------------------------------
CREATE TABLE IF NOT EXISTS schema_migrations (
    version     VARCHAR(255) PRIMARY KEY,
    applied_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- WICHTIG: Diese Liste MUSS exakt allen Dateien in migrations/*.sql entsprechen.
-- schema.sql baut bei einer Neuinstallation bereits das vollständige, aktuelle Schema auf.
-- Damit der Migrations-Runner (db/migrations.go) beim Start NICHT versucht, dieselben
-- Migrationen erneut anzuwenden (→ "column already exists"-Fehler → os.Exit(1)), werden
-- hier ALLE Migrationsversionen als bereits angewendet markiert. Fehlt hier eine Datei,
-- läuft sie beim Start gegen das fertige Schema und bricht den Serverstart ab.
INSERT INTO schema_migrations (version) VALUES
('001_initial_baseline.sql'),
('002_dsgvo_audit_hardening.sql'),
('003_audit_log_append_only.sql'),
('003_dsgvo_lusd_datensparsamkeit.sql'),
('004_aussonderung.sql'),
('005_add_etikett_gedruckt.sql'),
('006_create_geraete.sql'),
('007_audit_log_missing_columns.sql'),
('008_jahrgaenge.sql'),
('008_schueler_fotos_bytea.sql'),
('009_erweiterte_verwaltung.sql'),
('010_jwt_blacklist.sql'),
('011_performance_indexes.sql'),
('012_remove_passwords.sql'),
('013_view_buecher_bestand.sql'),
('014_barcode_sequence.sql'),
('015_antolin.sql'),
('016_fix_ghost_vormerkungen.sql'),
('017_ferien_schliesszeiten.sql'),
('018_vormerkungen_status.sql'),
('019_performance_indexe.sql'),
('020_audit_logs_admin.sql'),
('021_signatures_mdm.sql'),
('021_soft_delete_schueler.sql'),
('022_dsgvo_anonymize_schueler.sql'),
('022_vormerkungen_ablauf.sql'),
('023_seed_mail_vorlagen.sql'),
('024_inventur_status.sql'),
('025_hybrid_blocks.sql'),
('026_cover_status.sql'),
('027_mail_settings.sql'),
('028_idempotency_keys.sql'),
('029_nutzungsdauer_jahre.sql'),
('030_ziel_jahrgang.sql'),
('031_inventur_geprueft_am.sql'),
('032_reconcile_titel_columns.sql'),
('033_unique_active_loan.sql'),
('034_drop_antolin.sql'),
('035_lusd_id_partial_unique.sql'),
('036_schule_einstellungen.sql'),
('037_bestellungen_verlauf.sql')
ON CONFLICT DO NOTHING;

-- -------------------------------------------------------------
-- 5. PERFORMANCE INDEXES
-- -------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_ausleihen_geraet ON ausleihen (geraet_id) WHERE geraet_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_schadensfaelle_ausleihe ON schadensfaelle (ausleihe_id) WHERE ausleihe_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_vormerkungen_schueler ON vormerkungen(schueler_id) WHERE schueler_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_klassensatz_titel ON klassensatz_reservierungen(titel_id);
CREATE INDEX IF NOT EXISTS idx_class_books_klasse ON class_books(class_name);
CREATE INDEX IF NOT EXISTS idx_schueler_klasse ON schueler(klasse);
CREATE INDEX IF NOT EXISTS idx_vormerkungen_status ON vormerkungen(status);

-- -------------------------------------------------------------
-- 6. VIEWS
-- -------------------------------------------------------------
CREATE OR REPLACE VIEW view_buecher_bestand AS
SELECT 
    bt.id AS titel_id,
    bt.titel,
    COUNT(be.id) FILTER (WHERE be.ist_ausgesondert = false) AS gesamtbestand,
    COUNT(be.id) FILTER (WHERE a.id IS NULL AND be.ist_ausgesondert = false) AS verfuegbar
FROM buecher_titel bt
LEFT JOIN buecher_exemplare be ON bt.id = be.titel_id
LEFT JOIN ausleihen a ON be.id = a.exemplar_id AND a.rueckgabe_am IS NULL
GROUP BY bt.id, bt.titel;

CREATE SEQUENCE IF NOT EXISTS barcode_seq START 10000;
SELECT setval('barcode_seq', (
    SELECT COALESCE(MAX(CAST(SUBSTRING(barcode_id FROM '^B-([0-9]+)$') AS INTEGER)), 10000)
    FROM buecher_exemplare
    WHERE barcode_id ~ '^B-[0-9]+$'
));

CREATE TABLE IF NOT EXISTS idempotency_keys (
    idempotency_key UUID PRIMARY KEY,
    response_data JSONB NOT NULL,
    status_code INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_idempotency_keys_created_at ON idempotency_keys(created_at);
