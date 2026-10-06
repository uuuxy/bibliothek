-- =============================================================================
-- Migration 159: Der Standort am Titel entfällt (docs/OFFEN.md 5.53, Schritt 3)
-- =============================================================================
-- Seit Migration 158 steht der Standort am Exemplar. Was die Titelmaske als „Standort / Regal"
-- am Titel geführt hat (erweiterte_eigenschaften.standort), geht an die Exemplare des Titels,
-- die keinen eigenen Standort tragen. Danach fällt der Schlüssel am Titel weg; die Maske führt
-- das Feld nicht mehr.
--
-- Ein Titel ohne Exemplar hat kein Buch, das irgendwo steht: Sein Wert entfällt mit dem
-- Schlüssel. Ein Wert über 255 Zeichen wird auf die Länge der Spalte gekürzt.
--
-- Gemessen am Testserver am 06.10.2026: Bei keinem der 13.062 Titel ist der Standort
-- ausgefüllt. Gemessen an der lokalen Datenbank am 06.10.2026: Kein Titel trägt den Schlüssel.

UPDATE buecher_exemplare e
SET standort = btrim(left(btrim(t.erweiterte_eigenschaften->>'standort'), 255))
FROM buecher_titel t
WHERE t.id = e.titel_id
  AND e.standort IS NULL
  AND btrim(coalesce(t.erweiterte_eigenschaften->>'standort', '')) <> '';

UPDATE buecher_titel
SET erweiterte_eigenschaften = erweiterte_eigenschaften - 'standort'
WHERE erweiterte_eigenschaften ? 'standort';
