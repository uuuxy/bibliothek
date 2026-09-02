-- =============================================================================
-- Migration 093: Lernmittel wird ein Feld, kein Textpräfix mehr
-- =============================================================================
-- Betreiber-Entscheidung 02.09.2026: Das Kennzeichen „LMF" (Lernmittelfreiheit,
-- also ein Schulbuch, das die Schule dem Schüler fürs Schuljahr leiht) war eine
-- Konvention über zwei Freitextfelder — Titel oder Signatur begannen mit „LMF".
-- Sie stammte aus Littera, wo sie den Schulbüchern eine Sehhilfe in Listen gab.
-- Bei uns hing daran aber Verhalten: Schuljahresfrist statt 21 Tage, keine
-- Anrechnung aufs Ausleihlimit, unsichtbar im öffentlichen Katalog, 730 statt
-- 90 Tage Löschfrist, Bestellbedarf-Vorauswahl, Massenverlängerung — und die
-- DSGVO-Trennung in zwei Verarbeitungstätigkeiten (docs/SECURITY.md).
--
-- Vier Stellen lasen die Konvention mit drei verschiedenen Mustern (Go, SQL,
-- Svelte), zwei Schreibwege setzten sie an verschiedene Orte, und zweimal lief es
-- in Produktion falsch (85bee7b9: handangelegte Schulbücher mit falscher Frist im
-- öffentlichen Katalog; 6d79b4b1: ein führendes Leerzeichen reichte dafür).
--
-- Ab jetzt: EINE Spalte. Alle Regeln lesen ist_lernmittel. Titel und Signatur
-- sind wieder freier Text. Die Schulbücher tragen ohnehin kein Rückenetikett —
-- ihre Littera-Signatur „LMF Bio 7" war nie eine Regaladresse, nur eine
-- Einordnung; der Katalogisat-Import löst sie ab jetzt in Fach und Jahrgang auf.
-- =============================================================================

-- 1) Die Spalte.
ALTER TABLE buecher_titel
    ADD COLUMN IF NOT EXISTS ist_lernmittel BOOLEAN NOT NULL DEFAULT false;

-- 2) Backfill aus der bisherigen Konvention — exakt das Muster, das pkg/lmf bis
--    zu dieser Migration in Go und SQL las (btrim wie strings.TrimSpace, Trenner
--    Leerzeichen oder Bindestrich nach dem Kürzel). pkg/lmf.HatKennung liest es
--    weiter, damit der Import Litteras Kennzeichen erkennt; der Test dort hält das
--    Muster gegen diese Datei.
UPDATE buecher_titel
SET ist_lernmittel = true
WHERE LOWER(btrim(titel, E' \t\n\r')) ~ '^lmf[ -]'
   OR LOWER(btrim(COALESCE(signatur, ''), E' \t\n\r')) ~ '^lmf[ -]';

-- 3) Das Titel-Präfix „LMF-" war NIE Littera, sondern unsere eigene Import-
--    Konvention vom 11.07.2026 (742a82bc): Der Katalogisat-Import schnitt „LMF" aus
--    der Signatur und stellte es dem Titel voran, weil damals nur der Titel gelesen
--    wurde. Zurückdrehen, damit der Titel auf Cover-Kacheln, im Portal und beim
--    nächsten Import (Titel-Matching!) wieder der Buchtitel ist. Ein Titel, der
--    danach leer wäre, bleibt stehen.
UPDATE buecher_titel
SET titel = btrim(regexp_replace(titel, '^\s*lmf\s*-\s*|^\s*lmf\s+', '', 'i'))
WHERE ist_lernmittel
  AND titel ~* '^\s*lmf[ -]'
  AND btrim(regexp_replace(titel, '^\s*lmf\s*-\s*|^\s*lmf\s+', '', 'i')) <> '';

-- 4) Teilindex: Die Lernmittel sind die Minderheit (474 von 14.858 Titeln im
--    Katalogisat), aber die Massenverlängerung, der Bestellbedarf und die
--    Statistik fragen genau nach ihnen.
CREATE INDEX IF NOT EXISTS idx_titel_lernmittel
    ON buecher_titel (ist_lernmittel) WHERE ist_lernmittel;
