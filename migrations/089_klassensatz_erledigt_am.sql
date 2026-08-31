-- =============================================================================
-- Migration 089: Zeitpunkt des Abschlusses einer Klassensatz-Reservierung
-- =============================================================================
-- Anlass (31.08.2026, Raster-Durchgang über e4d0b889): Das Anliegen speichert
-- erledigt_am, die Reservierung nur erledigt=true/false — „erledigt am …" ließ sich
-- nirgends anzeigen, und die Bibliotheks-Notiz (Migration 088) hatte außerhalb der
-- Bereit-Mail keinen dauerhaften Rückweg (Portal und Theke lesen sie erst seit
-- diesem Stand).
--
-- Nullable, KEIN Backfill: NULL heißt „vor dieser Migration abgeschlossen" — der
-- echte Zeitpunkt ist nicht rekonstruierbar, und ein erfundener wäre schlechter
-- als ein ehrliches Leerfeld. Lesende Pfade scannen deshalb in *time.Time
-- (NULL-Scan-Bugklasse).
-- =============================================================================

ALTER TABLE klassensatz_reservierungen
    ADD COLUMN IF NOT EXISTS erledigt_am TIMESTAMP WITH TIME ZONE;
