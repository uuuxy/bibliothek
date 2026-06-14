CREATE OR REPLACE VIEW view_buecher_bestand AS
SELECT 
    bt.id AS titel_id,
    bt.titel,
    COUNT(be.id) FILTER (WHERE be.ist_ausgesondert = false) AS gesamtbestand,
    COUNT(be.id) FILTER (WHERE a.id IS NULL AND be.ist_ausgesondert = false) AS verfuegbar
FROM buecher_titel bt
LEFT JOIN buecher_exemplare be ON bt.id = be.titel_id
LEFT JOIN ausleihen a ON be.id = a.exemplar_id AND a.rueckgabe_am IS NULL
GROUP BY bt.id, bt.titel;
