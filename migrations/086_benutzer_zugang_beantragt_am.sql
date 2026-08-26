-- =============================================================================
-- Migration 086: benutzer.zugang_beantragt_am — „diese Zeile hat die Selbstanmeldung
-- angelegt, und sie wartet auf die Freischaltung"
-- =============================================================================
-- Anlass (26.08.2026): Die Selbstanmeldung (auth/selbstanmeldung.go) legt ein
-- INAKTIVES Konto an. In der Benutzerverwaltung war das von einem bewusst
-- deaktivierten Konto nicht zu unterscheiden — beides aktiv = false, beides ein grauer
-- Punkt „Inaktiv" irgendwo zwischen 160 Zeilen. Niemand schaltet frei, was niemand
-- sieht; die Lehrkraft wartet, die Bibliothek weiß von nichts.
--
-- Das ist der Spezialwert-Fall aus dem Raster (Frage 2): aktiv = false trug zwei
-- Bedeutungen. Die Spalte gibt der zweiten einen eigenen Ort. NULL = kein offener
-- Antrag. Gesetzt = wartet seit diesem Zeitpunkt; die Freischaltung (aktiv = true über
-- die Benutzerverwaltung) setzt sie zurück, damit ein SPÄTERES Deaktivieren nicht
-- wieder wie ein Antrag aussieht.
--
-- Backfill: Bestehende inaktive Kollegiums-Konten ohne Barcode und ohne Bearbeitung
-- seit der Anlage sind mit hoher Wahrscheinlichkeit liegengebliebene Anträge — sie
-- bekommen erstellt_am als Antragszeit, damit sie nach dem Update sichtbar werden.
-- Ein Fehlgriff kostet nichts: Die Bibliothek sieht Name und Adresse und entscheidet.
-- Idempotent: ADD COLUMN IF NOT EXISTS, Backfill nur WHERE … IS NULL.
-- =============================================================================

ALTER TABLE benutzer ADD COLUMN IF NOT EXISTS zugang_beantragt_am TIMESTAMP WITH TIME ZONE;

UPDATE benutzer
   SET zugang_beantragt_am = erstellt_am
 WHERE zugang_beantragt_am IS NULL
   AND aktiv = false
   AND rolle = 'kollegium'
   AND barcode_id IS NULL
   AND aktualisiert_am = erstellt_am;
