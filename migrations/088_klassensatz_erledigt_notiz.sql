-- =============================================================================
-- Migration 088: Notiz der Bibliothek beim Abschließen einer Klassensatz-Reservierung
-- =============================================================================
-- Anlass (27.08.2026, Peter): Reservieren und Anliegen-Melden lösen beide eine Mail
-- aus — aber nur beim Anliegen konnte die Theke beim Abhaken etwas dazuschreiben
-- (075: erledigt_notiz). Die Bereit-Mail des Klassensatzes war ein fester Text
-- „liegt bereit" und ließ sich weder ergänzen („24 von 30, Rest bei der 8a") noch
-- korrigieren (Vorgang geschlossen, Satz kommt nicht). Die Asymmetrie war
-- historisch, nicht fachlich: beide Wege bekommen jetzt dasselbe Muster.
--
-- NOT NULL DEFAULT '' wie bei lehrer_anliegen — kein drittes Nullable-Feld, das ein
-- Scan in einen Go-String umkippen lässt (NULL-Scan-Bugklasse).
-- =============================================================================

ALTER TABLE klassensatz_reservierungen
    ADD COLUMN IF NOT EXISTS erledigt_notiz TEXT NOT NULL DEFAULT '';
