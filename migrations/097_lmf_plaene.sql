-- =============================================================================
-- Migration 097: LMF-Plan als Reihenfolge — Rahmen je Art und Schuljahr
-- =============================================================================
-- Peter 05.09.2026 abends, nach dem ersten Blick auf die Seite aus Migration 096: Der
-- Plan der Schule ist keine Liste einzeln angelegter Zeilen, sondern eine REIHENFOLGE
-- von Klassen, die auf Schultage × Stunden gegossen wird — Abschlussklassen zuerst,
-- dann jeder Schultag Stunde 1–6, eine Klasse je Stunde, die Reihenfolge läuft über
-- die Tage weiter; manche teilen sich eine Stunde („10R1/10R2"), am Ende Zeilen ohne
-- Klasse („Nachzügler", „Aufräumen"). Wochentag und Datum von Hand waren im Excel
-- zweimal falsch.
--
-- Deshalb trägt ein Plan (je Art und Schuljahr) den RAHMEN — erster Tag, Startstunde am
-- ersten Tag, Stunden je Tag — und seine Zeilen in lmf_termine bekommen eine Position.
-- Datum und Stunde jeder Zeile rechnet der Server aus Rahmen und Position und schreibt
-- sie weiterhin in lmf_termine: Portal, PDF und die Frist-Kopplung lesen sie wie bisher.
-- Klassen, die der Plan bewusst auslässt (die Oberstufe organisiert sich an dieser
-- Schule selbst), stehen in lmf_plan_ausgelassen — sie werden nicht als „ohne Termin"
-- angemahnt, und der Plan des nächsten Jahres übernimmt die Auslassung.
CREATE TABLE lmf_plaene (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    art              TEXT NOT NULL CONSTRAINT chk_lmf_plaene_art CHECK (art IN ('rueckgabe', 'ausgabe')),
    schuljahr_beginn DATE NOT NULL,   -- 1. August des Schuljahres, in dem der erste Tag liegt
    erster_tag       DATE NOT NULL,
    startstunde      SMALLINT NOT NULL DEFAULT 1 CONSTRAINT chk_lmf_plaene_startstunde CHECK (startstunde BETWEEN 1 AND 12),
    stunden_je_tag   SMALLINT NOT NULL DEFAULT 6 CONSTRAINT chk_lmf_plaene_stunden CHECK (stunden_je_tag BETWEEN 1 AND 12),
    erstellt_am      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    aktualisiert_am  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uniq_lmf_plaene_art_schuljahr UNIQUE (art, schuljahr_beginn)
);

CREATE TRIGGER trg_lmf_plaene_aktualisiert_am
BEFORE UPDATE ON lmf_plaene
FOR EACH ROW EXECUTE FUNCTION set_aktualisiert_am();

ALTER TABLE lmf_termine
    ADD COLUMN plan_id  UUID REFERENCES lmf_plaene(id) ON DELETE CASCADE,
    ADD COLUMN position INTEGER;

CREATE INDEX idx_lmf_termine_plan ON lmf_termine (plan_id, position);

CREATE TABLE lmf_plan_ausgelassen (
    plan_id UUID NOT NULL REFERENCES lmf_plaene(id) ON DELETE CASCADE,
    klasse  VARCHAR(50) NOT NULL,
    PRIMARY KEY (plan_id, klasse),
    CONSTRAINT fk_lmf_plan_ausgelassen_vokabular
        FOREIGN KEY (klasse) REFERENCES klassen (name)
        ON UPDATE CASCADE ON DELETE RESTRICT
);

CREATE TRIGGER trg_lmf_plan_ausgelassen_vokabular
BEFORE INSERT OR UPDATE OF klasse ON lmf_plan_ausgelassen
FOR EACH ROW EXECUTE FUNCTION klasse_kanonisieren();
