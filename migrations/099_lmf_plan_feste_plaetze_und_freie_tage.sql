-- =============================================================================
-- Migration 099: LMF-Plan — feste Plätze und freie Tage
-- =============================================================================
-- Peter 05.09.2026, nach dem ersten Plan aus Migration 097: „manchmal gibt es ja auch
-- noch gesetzliche Feiertage, oder ich muss eine Klasse komplett auf einen anderen Tag
-- schieben, weil sie einen Ausflug haben."
--
-- Zwei Dinge, die die Reihenfolge allein nicht ausdrücken kann:
--
-- 1. Eine Zeile mit FESTEM Platz: Die Klasse mit dem Ausflug bekommt Datum und Stunde
--    von Hand; die übrigen Zeilen fließen weiter über die Schultage und lassen den
--    belegten Platz aus. lmf_termine.fest sagt, dass Datum und Stunde dieser Zeile
--    vorgegeben und nicht gerechnet sind — beim Bearbeiten des Plans bleibt die Vorgabe
--    erhalten, im Vorschlag fürs nächste Jahr fällt sie weg (der Ausflug war dieses Jahr).
--
-- 2. FREIE TAGE des Plans: bewegliche Ferientage, pädagogische Tage, der Brückentag nach
--    Fronleichnam. Gesetzliche Feiertage (Hessen) rechnet der Server selbst
--    (pkg/lmfplan/feiertage.go); alles, was nur diese Schule weiß, steht hier am Plan.
--    Die Tabelle ferien_schliesszeiten (Migration 017) hat bis heute keinen Schreiber in
--    der Oberfläche — der Plan übersprang deshalb in der Praxis nur Wochenenden.
ALTER TABLE lmf_termine
    ADD COLUMN fest BOOLEAN NOT NULL DEFAULT false;

CREATE TABLE lmf_plan_freie_tage (
    plan_id UUID NOT NULL REFERENCES lmf_plaene(id) ON DELETE CASCADE,
    datum   DATE NOT NULL,
    grund   TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (plan_id, datum)
);
