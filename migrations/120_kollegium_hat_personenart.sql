-- Migration 120: Ein Kollegiumskonto hat immer eine Personenart — sie entscheidet jetzt, wer als
-- Lehrkraft ausleiht.
--
-- Seit dem 15.09.2026 erkennen Theke, Buch- und Geräteausleihe eine Lehrkraft an der Personenart
-- (Lehrkraft, LiV), nicht mehr an der Rolle kollegium: Eine Lehrkraft, die in der Bibliothek
-- mitarbeitet (Rolle Mitarbeiter), fand die Theke über ihren Ausweis nicht (Absprache: „an der
-- Personenart"). Damit darf ein Kollegiumskonto nicht ohne Personenart sein, sonst verlöre es die
-- Ausleihe als Lehrkraft. Bis hierher trugen nur die Anlage-Wege „lehrkraft" ein; ein Wechsel zur
-- Rolle kollegium oder „Keine Angabe" beim Ändern ließen sie leer (Rasterdurchgang 15.09.2026).
--
-- Der Trigger setzt „lehrkraft", sobald ein Kollegiumskonto ohne Personenart geschrieben wird —
-- über jeden Weg, auch Testdaten und künftige Schreiber. Eine LiV bleibt LiV; andere Rollen
-- dürfen leer bleiben.

UPDATE benutzer SET personenart = 'lehrkraft'
 WHERE rolle = 'kollegium' AND personenart IS NULL;

CREATE OR REPLACE FUNCTION kollegium_hat_personenart()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.rolle = 'kollegium' AND NEW.personenart IS NULL THEN
        NEW.personenart := 'lehrkraft';
    END IF;
    RETURN NEW;
END $$;

DROP TRIGGER IF EXISTS trg_benutzer_kollegium_personenart ON benutzer;
CREATE TRIGGER trg_benutzer_kollegium_personenart
BEFORE INSERT OR UPDATE OF rolle, personenart ON benutzer
FOR EACH ROW EXECUTE FUNCTION kollegium_hat_personenart();
