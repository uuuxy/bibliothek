-- =============================================================================
-- Migration 041: Wertemenge für buecher_titel.cover_status
-- =============================================================================
-- Das Vokabular ist geschlossen und wird ausschließlich vom CoverService gesetzt:
--   PENDING   – noch nie versucht (Default)
--   FOUND     – Cover gefunden/gesetzt
--   FAILED    – Abruf fehlgeschlagen (wird erneut versucht)
--   NOT_FOUND – zur ISBN existiert kein Cover (bewusst KEIN Retry, siehe
--               internal/service/cover_service.go: Kandidaten-Query)
--
-- Ein Tippfehler in einem dieser Werte würde stillschweigend die Retry-Auswahl
-- verändern (Titel bekämen nie ein Cover oder würden endlos abgefragt) — genau
-- deshalb gehört die Menge in die DB.
--
-- Wie 039/040: erst Verletzer bereinigen, dann CHECK (idempotent per DO-Guard).
-- =============================================================================

-- Unbekannte Werte auf PENDING zurücksetzen: unbekannt = "nochmal versuchen".
UPDATE buecher_titel
SET cover_status = 'PENDING'
WHERE cover_status IS NOT NULL
  AND cover_status NOT IN ('PENDING', 'FOUND', 'FAILED', 'NOT_FOUND');

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_cover_status') THEN
        ALTER TABLE buecher_titel ADD CONSTRAINT chk_cover_status
            CHECK (cover_status IS NULL OR cover_status IN ('PENDING', 'FOUND', 'FAILED', 'NOT_FOUND'));
    END IF;
END $$;
