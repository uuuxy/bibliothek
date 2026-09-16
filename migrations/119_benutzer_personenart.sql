-- Migration 119: Personenart im Kollegium — Lehrkraft oder LiV (Lehrkraft im Vorbereitungsdienst).
--
-- Die Rolle (benutzer.rolle) sagt, was jemand in der Software darf: Admin, Mitarbeiter, Helfer,
-- Kollegium (nur „Mein Portal"). Wer jemand IST, stand nirgends; „Kollegium" stand für beides.
-- Eine Lehrkraft, die in der Bibliothek mitarbeitet, hat die Rolle Mitarbeiter und bleibt trotzdem
-- Lehrkraft (15.09.2026). Schüler brauchen das Feld nicht: Schüler ist, wer in der Tabelle
-- schueler steht (aus der LUSD).
--
-- Leer ist erlaubt: Helfer (Schülerhilfskräfte, Eltern), Mitarbeiter und Admin sind nicht zwingend
-- Lehrkräfte. Jedes vorhandene Kollegiumskonto wird Lehrkraft; eine LiV trägt die
-- Benutzerverwaltung von Hand ein. Sichtbar ist das Feld nur dort.

ALTER TABLE benutzer ADD COLUMN IF NOT EXISTS personenart VARCHAR(20);

ALTER TABLE benutzer DROP CONSTRAINT IF EXISTS chk_benutzer_personenart;
ALTER TABLE benutzer ADD CONSTRAINT chk_benutzer_personenart
    CHECK (personenart IN ('lehrkraft', 'liv'));

UPDATE benutzer SET personenart = 'lehrkraft'
 WHERE rolle = 'kollegium' AND personenart IS NULL;
