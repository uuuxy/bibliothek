-- 073: buecher_titel.stock entfernt (Befund F5 der unabhängigen Prüfung).
--
-- Die Spalte war eine zweite, ungepflegte Antwort auf "wie viele Exemplare?":
-- Das Hauptprogramm zählt live aus buecher_exemplare (OPAC, Bestellsuche,
-- Verfügbarkeit), aber Inventur-Modul und Littera-Migration schrieben parallel
-- eine Zahl hierher, die niemand las — und der alte Excel-Import legte sogar
-- Titel an, deren Stückzahl NUR hier stand und nirgends mehr auftauchte.
-- Seit diesem Stand übersetzen alle Importe ihre Stückzahlen in echte
-- Exemplar-Zeilen; die eine Wahrheit ist die Live-Zählung.
ALTER TABLE buecher_titel DROP COLUMN IF EXISTS stock;
