-- Migration 110: Der Schadensersatz-Bescheid als eigener Datensatz — ein Brief, eine
-- Referenznummer, eine Frist.
--
-- Bis hierher kannte das System nur die FORDERUNG (schadensfaelle): Betrag, bezahlt,
-- storniert. Das Verfahren der Schule verlangt aber einen Brief mit Nummer und
-- Vierwochenfrist, der mehrere Bücher eines Schülers zusammenfasst, und nach Fristablauf
-- an die Schulaufsicht geht. Die drei bestehenden Briefe (Elternbrief, Rechnung,
-- Mahnbrief) sind keine solchen Bescheide: Sie verlangen Barzahlung „in der Bibliothek",
-- kennen keine Nummer und keine Frist.
--
-- Die POSITIONEN des Briefs sind die Forderungen selbst (schadensfaelle.bescheid_id) —
-- keine zweite Positionstabelle. Sperre, Löschblockade, Auskunft und Bezahlt/Storno
-- hängen an der Forderung und laufen unverändert weiter.
--
-- Zwei Fallgruppen, weil das Formular genau zwei Kästchen hat: „nicht ordnungsgemäß
-- zurückgegeben" und „so stark beschädigt zurückgegeben, dass eine Nutzung nicht mehr
-- möglich ist". Ein dritter Wert („Verlust") wäre ein Vokabular ohne Kästchen — Verlust
-- IST die erste Gruppe.
--
-- Die laufende Nummer kommt aus schadensersatz_nummern: EIN Generator je Topf und
-- Kassenjahr, UPDATE … RETURNING in derselben Transaktion wie der Bescheid. Eine Nummer
-- darf nie zweimal vergeben werden — sonst lässt sich eine Zahlung nicht zuordnen.

CREATE TABLE IF NOT EXISTS schadensersatz_bescheide (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- SET NULL und nullbar, wie bei der Ausleihhistorie: Der Bescheid überlebt die
    -- DSGVO-Löschung als BELEG OHNE PERSON — die Referenznummer und der Betrag bleiben
    -- (über die Nummer werden Zahlungen zugeordnet, Rechnungsunterlagen liegen Jahre),
    -- der Klarname im Snapshot wird bei der Tilgung geleert (DSGVO-Paar).
    --
    -- Bewusst NICHT RESTRICT: Das hätte die berechtigte Löschung blockiert, solange
    -- irgendein alter Bescheid existiert. Die Löschsperre bei OFFENEN Vorgängen liegt
    -- da, wo sie hingehört — an der unbezahlten Forderung
    -- (repository/audit_users.go, blockiereBeiOffenenVorgaengen).
    schueler_id         UUID REFERENCES schueler(id) ON DELETE SET NULL,
    -- Aus welchem Topf: dasselbe Vokabular wie bestellungen_verlauf.mittel (Migration 109).
    mittel              TEXT NOT NULL
        CONSTRAINT chk_bescheid_mittel CHECK (mittel IN ('land', 'schultraeger')),
    kassenjahr          INTEGER NOT NULL,
    laufende_nr         INTEGER NOT NULL
        CONSTRAINT chk_bescheid_laufende_nr CHECK (laufende_nr >= 1),
    -- Die fertige Nummer, wie sie im Brief steht (vier Blöcke, durch Leerzeichen getrennt).
    -- UNIQUE ist die eigentliche Zusicherung: nie zweimal.
    referenznummer      TEXT NOT NULL UNIQUE,
    brief_datum         DATE NOT NULL DEFAULT CURRENT_DATE,
    frist_bis           DATE NOT NULL,
    gesamtbetrag        NUMERIC(10,2) NOT NULL DEFAULT 0.00
        CONSTRAINT chk_bescheid_betrag CHECK (gesamtbetrag >= 0.00),
    -- Anrede, Name und Anschrift zum Briefdatum. Der Nachdruck muss dasselbe Blatt
    -- ergeben wie das Original („Kopie verbleibt in der Schule"), auch wenn die Familie
    -- inzwischen umgezogen ist.
    empfaenger_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    status              TEXT NOT NULL DEFAULT 'offen'
        CONSTRAINT chk_bescheid_status CHECK (status IN ('offen', 'uebergeben', 'erledigt')),
    uebergeben_am       TIMESTAMPTZ,
    erledigt_am         TIMESTAMPTZ,
    -- Rückgabe NACH der Übergabe: Die Schulaufsicht ist unverzüglich zu informieren.
    -- Ein Merker, den die Oberfläche zeigt — kein Automatismus.
    rueckgabe_nach_uebergabe BOOLEAN NOT NULL DEFAULT false,
    erstellt_von        UUID REFERENCES benutzer(id) ON DELETE SET NULL,
    erstellt_am         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    letzter_druck_am    TIMESTAMPTZ,
    CONSTRAINT uniq_bescheid_nummer UNIQUE (mittel, kassenjahr, laufende_nr)
);

-- Die Arbeitsliste fragt „welche Frist ist abgelaufen?" — offene Bescheide nach Frist.
CREATE INDEX IF NOT EXISTS idx_bescheide_offen_frist
    ON schadensersatz_bescheide (frist_bis)
    WHERE status = 'offen';

-- Die Akte fragt „welche Bescheide hat dieser Schüler?".
CREATE INDEX IF NOT EXISTS idx_bescheide_schueler
    ON schadensersatz_bescheide (schueler_id, brief_datum DESC);

-- Der EINE Nummerngenerator. Eine Zeile je Topf und Kassenjahr; gezogen wird mit
-- UPDATE … RETURNING, das die Zeile für die Dauer der Transaktion sperrt. Kein MAX+1
-- über die Bescheide — zwei gleichzeitige Briefe bekämen dieselbe Nummer, und genau
-- das darf nie passieren (Registereintrag „Zwei Generatoren, ein Nummernkreis").
CREATE TABLE IF NOT EXISTS schadensersatz_nummern (
    mittel     TEXT NOT NULL
        CONSTRAINT chk_nummern_mittel CHECK (mittel IN ('land', 'schultraeger')),
    kassenjahr INTEGER NOT NULL,
    letzte_nr  INTEGER NOT NULL DEFAULT 0
        CONSTRAINT chk_nummern_letzte_nr CHECK (letzte_nr >= 0),
    PRIMARY KEY (mittel, kassenjahr)
);

-- Die Fallgruppe der Forderung. Vorgabe 'beschaedigt', weil das der bisherige Weg ist
-- (ReportDamage bei der Rückgabe).
ALTER TABLE schadensfaelle
    ADD COLUMN IF NOT EXISTS art TEXT NOT NULL DEFAULT 'beschaedigt'
        CONSTRAINT chk_schaden_art CHECK (art IN ('nicht_zurueckgegeben', 'beschaedigt'));

-- Die Zugehörigkeit zu einem Brief. SET NULL: Ein gelöschter Bescheid (den es im
-- Betrieb nicht gibt) darf die Forderung nicht mitnehmen — sie trägt die Sperre.
ALTER TABLE schadensfaelle
    ADD COLUMN IF NOT EXISTS bescheid_id UUID REFERENCES schadensersatz_bescheide(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_schadensfaelle_bescheid
    ON schadensfaelle (bescheid_id)
    WHERE bescheid_id IS NOT NULL;

-- Backfill der Fallgruppe: Wo das Exemplar als VERLUST ausgesondert ist, war es „nicht
-- zurückgegeben"; alles andere bleibt bei 'beschaedigt'. Nicht geraten — nur dieser eine
-- eindeutige Fall wird umgesetzt.
UPDATE schadensfaelle s
   SET art = 'nicht_zurueckgegeben'
  FROM buecher_exemplare e
 WHERE e.id = s.exemplar_id
   AND e.ist_ausgesondert
   AND e.aussonderung_grund = 'VERLUST';
