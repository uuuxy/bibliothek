-- =============================================================================
-- Migration 082: Index- und Unique-Parität zwischen gewachsenen und frischen DBs
-- =============================================================================
-- Der Schema-Paritätstest (db/migrations_schema_paritaet_pg_test.go) fand zwei
-- Sorten Drift zwischen dem gewachsenen Pfad (Baseline + Migrationen, wie Prod)
-- und dem frischen Pfad (heutiges schema.sql):
--
-- 1. Fünf Indexe stehen NUR in schema.sql — sie kamen dort ohne Migration dazu.
--    Gewachsene Datenbanken haben sie nie bekommen.
-- 2. UNIQUE (titel_id, schueler_id) auf vormerkungen steht NUR im CREATE TABLE
--    von schema.sql. Auf gewachsenen Datenbanken kann derselbe Schüler denselben
--    Titel deshalb MEHRFACH vormerken — auf frischen ist genau das verboten.
--
-- (Die Gegenrichtung — vier Indexe aus 011/021, die schema.sql verloren hatte —
-- heilt der schema.sql-Nachzug im selben Commit, nicht diese Migration.)
--
-- Alles idempotent: auf frischen Datenbanken ist jede Anweisung ein No-op.

CREATE INDEX IF NOT EXISTS idx_ausleihen_geraet ON ausleihen (geraet_id) WHERE geraet_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_class_books_klasse ON class_books (class_name);
CREATE INDEX IF NOT EXISTS idx_klassensatz_titel ON klassensatz_reservierungen (titel_id);
CREATE INDEX IF NOT EXISTS idx_schadensfaelle_ausleihe ON schadensfaelle (ausleihe_id) WHERE ausleihe_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_vormerkungen_schueler ON vormerkungen (schueler_id) WHERE schueler_id IS NOT NULL;

-- Vorhandene Doppel-Vormerkungen zusammenführen, bevor der Unique greift:
-- Es bleibt die ÄLTESTE (der Platz in der Warteschlange), Nachzügler fliegen.
-- Anonyme Vormerkungen (schueler_id IS NULL) sind nicht betroffen — NULLs gelten
-- im Unique ohnehin als verschieden.
DELETE FROM vormerkungen a
USING vormerkungen b
WHERE a.titel_id = b.titel_id
  AND a.schueler_id = b.schueler_id
  AND a.schueler_id IS NOT NULL
  AND (a.erstellt_am, a.id) > (b.erstellt_am, b.id);

-- Constraint-Name exakt wie ihn das CREATE TABLE in schema.sql erzeugt.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'vormerkungen_titel_id_schueler_id_key'
          AND conrelid = 'vormerkungen'::regclass
    ) THEN
        ALTER TABLE vormerkungen
            ADD CONSTRAINT vormerkungen_titel_id_schueler_id_key UNIQUE (titel_id, schueler_id);
    END IF;
END $$;
