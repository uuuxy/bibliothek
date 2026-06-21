-- Migration: 021_soft_delete_schueler.sql
-- Description: Fügt die Spalte deleted_at zur schueler-Tabelle hinzu, um Soft-Deletes zu ermöglichen.

ALTER TABLE schueler ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE DEFAULT NULL;
CREATE INDEX IF NOT EXISTS idx_schueler_deleted_at ON schueler(deleted_at) WHERE deleted_at IS NULL;
