-- Migration 114: Titel mit Jahrgang 0–0 bekommen die Vorgabe 5–10.
--
-- Bis zum 10.09.2026 schrieben CreateBook und der Listenimport den Go-Nullwert
-- ausdrücklich in jahrgang_von/jahrgang_bis — der DEFAULT (5/10, Migration 008) griff
-- nie. Jeder über den Scanner-Dialog oder den Listenimport angelegte Titel stand deshalb
-- auf 0–0. Folgen: Im Mahnwesen-Modus „Jahrgang" (klasse_num > jahrgang_bis) erschien
-- jede offene Ausleihe eines solchen Titels für jeden Schüler mit Ziffernklasse, eine
-- Klassen-Inventur sah ihn nie, der Jahrgangsfilter des Portals blendete ihn aus
-- (Bestands-Durchgang, „DEFAULT-Umgehung durch Nullwert").
--
-- 0–0 ist keine fachliche Aussage (Jahrgänge laufen 5–13); die Reparatur setzt genau die
-- Vorgabe, die die Zwillinge (Littera, Bestands-CSV) schon immer gesetzt haben. Titel mit
-- einer echten Angabe bleiben unberührt.

UPDATE buecher_titel
   SET jahrgang_von = 5, jahrgang_bis = 10
 WHERE jahrgang_von = 0 AND jahrgang_bis = 0;
