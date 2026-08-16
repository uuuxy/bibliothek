-- Einmalige Prod-Reparatur (16.08.2026): Importierter Bestand gilt als etikettiert.
--
-- Befund: Das Druck-Center-Badge zählt Exemplare mit etikett_gedruckt = false und
-- stand auf Prod dauerhaft bei 999+ — gemessen waren es 30.658 importierte
-- Littera-Bücher (die ihre Etiketten physisch tragen) und nur 16 echte B-Nummern
-- aus dem Bestellwesen. Die Import-Pfade setzen das Flag seit diesem Datum selbst
-- (internal/service/import_dynamic.go, internal/littera/schreiber_bestand.go);
-- dieses Skript zieht den bereits importierten Bestand nach.
--
-- Abgrenzung über den Barcode: App-eigene Neuzugänge tragen B-Nummern ("B-…"),
-- nur sie durchlaufen den echten Etikettendruck. Alles andere ist Import.
--
-- Ausführen (auf dem Server, nach Backup):
--   docker exec -i bibliothek-db psql -U postgres -d bibliothek < scripts/repair_altbestand_etiketten.sql

BEGIN;

UPDATE buecher_exemplare
SET etikett_gedruckt = true
WHERE etikett_gedruckt = false
  AND barcode_id NOT LIKE 'B-%';

-- Erwartung laut Messung vom 16.08.: ~30.658 Zeilen. Danach zählt das Badge nur
-- noch echte offene Etiketten (gemessen: 16).
SELECT count(*) AS verbleibend_offen FROM buecher_exemplare
WHERE etikett_gedruckt = false AND ist_ausgesondert = false;

COMMIT;
