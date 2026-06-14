-- 016_fix_ghost_vormerkungen.sql
-- Behebt den blinden Fleck: Ghost-Vormerkungen von gelöschten Schülern blockieren Bücher.

-- 1. Lösche alle bestehenden Ghost-Vormerkungen (wo der Schüler durch DSGVO-Löschung bereits NULL ist)
DELETE FROM vormerkungen WHERE schueler_id IS NULL;

-- 2. Ändere den Foreign Key Constraint von ON DELETE SET NULL auf ON DELETE CASCADE
ALTER TABLE vormerkungen DROP CONSTRAINT IF EXISTS vormerkungen_schueler_id_fkey;

ALTER TABLE vormerkungen ADD CONSTRAINT vormerkungen_schueler_id_fkey
FOREIGN KEY (schueler_id) REFERENCES schueler(id) ON DELETE CASCADE;
