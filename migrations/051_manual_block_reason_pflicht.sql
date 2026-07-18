-- =============================================================================
-- Migration 051: Auch manuelle Sperren verlangen einen Grund
-- =============================================================================
-- chk_schueler_block_reason (Migration 047) erzwang einen block_reason nur bei
-- ist_gesperrt = true. Die zweite Sperrquelle is_manually_blocked war ausgenommen —
-- eine manuelle Sperre ohne Text erzeugte damit genau die "Zombie-Sperre", die der
-- Constraint eigentlich verhindern sollte (Personal sieht nur das Flag, keinen Grund).
-- Der dedizierte Lock-Endpoint setzte keinen Grund.
--
-- Fix: Der Constraint deckt jetzt BEIDE Sperrquellen ab. Vorher bestehende
-- grundlose manuelle Sperren werden mit einem Platzhalter aufgefüllt.
-- =============================================================================

-- 1. Backfill: bestehende manuelle Sperren ohne Grund auffüllen (sonst schlägt der
--    strengere Constraint beim Anlegen fehl).
UPDATE schueler
SET block_reason = 'Manuell gesperrt (Grund nachträglich zu erfassen)'
WHERE COALESCE(is_manually_blocked, false) = true
  AND btrim(coalesce(block_reason, '')) = '';

-- 2. Constraint erweitern: jede Sperre (System ODER manuell) verlangt einen Grund.
ALTER TABLE schueler DROP CONSTRAINT IF EXISTS chk_schueler_block_reason;
ALTER TABLE schueler ADD CONSTRAINT chk_schueler_block_reason
    CHECK (
        (ist_gesperrt = false AND COALESCE(is_manually_blocked, false) = false)
        OR btrim(coalesce(block_reason, '')) <> ''
    );
