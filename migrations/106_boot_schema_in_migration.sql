-- Migration 106: Was der Boot bisher per DDL anlegte, steht jetzt im Schema.
--
-- db/seed.go führte bei JEDEM Start Schema-Anweisungen aus: CREATE EXTENSION pg_trgm,
-- fünf CREATE INDEX (GIN-Trigramm für die Suche), CREATE TABLE role_permissions,
-- CREATE TABLE lieferanten und ein ALTER TABLE, das eine ENUM-Spalte auf VARCHAR hob.
-- Vier davon waren längst tot — Extension, Indexe und lieferanten stehen in jeder
-- Baseline seit e5740b95 (05.06.2026). Eine war lebendig und unsichtbar: role_permissions
-- stand in KEINER Migration und in keiner Zeile von schema.sql. Eine frische Anlage
-- bekam die Rechte-Tabelle ausschließlich dadurch, dass der Go-Prozess sie beim ersten
-- Start erzeugte.
--
-- Warum keine Ratsche das sah: Die Schema-Parität (db/migrations_schema_paritaet_pg_test.go)
-- vergleicht den gewachsenen mit dem frischen Weg — und in BEIDEN läuft derselbe Boot.
-- Was der Boot auf beiden Seiten anlegt, ist für den Vergleich unsichtbar; der Test
-- musste role_permissions sogar von Hand vorab anlegen, weil Migration 055 sie
-- voraussetzt. Dieselbe Blindheit wie bei sys_barcode_seq (Migrationen 104/105): Ein
-- Objekt, das erst der laufende Code erzeugt, fehlt in jedem verglichenen Schema.
-- Seit 07.09.2026 prüft inventur/kein_ddl_im_schreibpfad_test.go deshalb die Quelle —
-- und seit dieser Migration auch db/.
--
-- Alles hier ist idempotent (IF NOT EXISTS, DO-Block mit Prüfung): Auf jeder heutigen
-- Anlage ist es ein No-op, weil der Boot es längst erledigt hat. Der Wert liegt nicht in
-- der Wirkung auf Prod, sondern darin, dass eine Neuinstallation und der Replay der
-- Paritäts-Ratsche das Schema jetzt ohne den Go-Prozess vollständig herstellen.
--
-- Die Vorab-Anlage im Paritäts-Test bleibt trotzdem nötig — 055 läuft VOR 106, und eine
-- Migration kann nicht rückwirkend vor eine ältere treten. Der Test friert dafür das
-- historische Boot-SQL ein, das seed.go nicht mehr enthält.

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS idx_buecher_titel_trgm ON buecher_titel USING gin (titel gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_buecher_autor_trgm ON buecher_titel USING gin (autor gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_buecher_isbn_trgm ON buecher_titel USING gin (isbn gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_schueler_vorname_trgm ON schueler USING gin (vorname gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_schueler_nachname_trgm ON schueler USING gin (nachname gin_trgm_ops);

-- Rechte je Rolle (GROSS-Vokabular; die Middleware verbindet es per UPPER() mit
-- benutzer.rolle, siehe api/permission_middleware.go). Inhalt kommt weiter aus dem Seed
-- (db.RechteVorgabe, ON CONFLICT DO NOTHING) — nur die Struktur zieht hierher um.
CREATE TABLE IF NOT EXISTS role_permissions (
    role VARCHAR(50) NOT NULL,
    permission VARCHAR(100) NOT NULL,
    allowed BOOLEAN NOT NULL DEFAULT false,
    PRIMARY KEY (role, permission)
);

-- Altlast aus der Zeit, als role ein ENUM war: Der Boot hob die Spalte beim Start auf
-- VARCHAR(50). Auf keiner heutigen Anlage mehr wirksam; hier festgehalten, damit der
-- Boot es nicht mehr tun muss.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'role_permissions'
          AND column_name = 'role' AND data_type = 'USER-DEFINED'
    ) THEN
        ALTER TABLE role_permissions ALTER COLUMN role TYPE VARCHAR(50);
    END IF;
END $$;
