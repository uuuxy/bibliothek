-- =============================================================================
-- Migration 047: Sperrgrund-Pflicht (block_reason) für gesperrte Schüler
-- =============================================================================
-- Ein gesperrter Schüler (ist_gesperrt = true) ohne block_reason ist ein toter Zustand
-- ("Zombie-Sperre"): Im Profil sieht das Personal nur das rote Flag und muss die
-- Ausleih-Historie durchwühlen, um den Grund zu erraten (Vandalismus? Gebühren?
-- Abgänger-Automatik?). Die DB erzwingt jetzt: gesperrt ⇒ Grund nicht leer.
--
-- Alle automatischen Sperr-Pfade setzen ihren Grund selbst mit:
--   * Abgänger-Sync (api/lusd_apply.go: sperreAbgaenger / anonymisiereAbgaenger),
--   * Schuljahreswechsel (api/student_promotion.go),
--   * Soft-Delete (repository/audit_users.go: DeleteStudent).
-- Die MANUELLE Sperre läuft über is_manually_blocked und ist NICHT betroffen.
--
-- Wie 039/040: erst Verletzer bereinigen, dann CHECK setzen (idempotent per DO-Guard),
-- damit die Migration auf Bestandsdaten nie fehlschlägt.
-- =============================================================================

-- 1. Bestehende gesperrte Schüler ohne Grund rückwirkend kennzeichnen.
UPDATE schueler
SET block_reason = 'Sperre (Grund nicht erfasst — Altbestand vor Migration 047)'
WHERE ist_gesperrt = true
  AND btrim(coalesce(block_reason, '')) = '';

-- 2. Constraint ergänzen (idempotent).
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_schueler_block_reason') THEN
        ALTER TABLE schueler ADD CONSTRAINT chk_schueler_block_reason
            CHECK (ist_gesperrt = false OR btrim(coalesce(block_reason, '')) <> '');
    END IF;
END $$;
