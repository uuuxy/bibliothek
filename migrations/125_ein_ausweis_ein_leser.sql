-- =============================================================================
-- Migration 125: Ein Ausweis, ein Leser, eine Ausleihe
-- =============================================================================
-- Bis heute konnte ein Mensch an ZWEI Orten stehen: als Leserzeile (`leser`) und
-- als Konto (`benutzer`). Beide trugen eine Ausweisnummer, beide konnten
-- ausleihen, und jede Ausleihe hatte deshalb zwei Ausleiher-Spalten, von denen
-- genau eine gefüllt sein durfte. Dieselbe Verdopplung noch einmal bei den
-- Nachbuch-Meldungen und bei den Schadensfällen.
--
-- Diese Migration legt beide Seiten zusammen: Jedes Konto bekommt seine
-- Leserzeile, der Ausweis zieht dorthin um, und die Ausleihe kennt nur noch
-- EINEN Ausleiher — den Leser. Das Konto behält, wofür es da ist: Anmeldung und
-- Rechte. Wer eine Buchung AUSGEFÜHRT hat (`bearbeiter_id`), bleibt davon
-- unberührt; das ist eine andere Frage als „wer hat das Buch".
--
-- Entschieden am 16.09.2026 (Peter): „eine Tabelle für alle".
--
-- Was dabei ERSATZLOS wegfällt, weil es die Zweiteilung nur verwaltet hat:
--
--   benutzer.personenart              — sagte, wer als Lehrkraft ausleihen darf.
--                                       Jetzt darf jeder aktive Leser ausleihen.
--   benutzer.barcode_id               — der Ausweis gehört zum Leser.
--   ausweis_eindeutig_ueber_personen  — hielt eine Nummer über ZWEI Tabellen
--                                       eindeutig (Migration 118). Es gibt nur
--                                       noch eine; uniq_schueler_barcode_active
--                                       tut es allein und ohne Trigger.
--
-- Zahlen vorher gemessen (lokal und auf dem Testserver, 16.09.2026):
-- 0 Ausleihen, 0 Nachbuch-Meldungen und 0 Schadensfälle hängen an einem Konto.
-- Die UPDATEs unten stehen trotzdem da: Sie sind die Regel, nicht die Statistik,
-- und eine weitere gewachsene Anlage darf nicht anders enden.
-- =============================================================================

-- ── 1. Der Kreuz-Wächter fällt ZUERST ────────────────────────────────────────
--
-- Er prüft bei jedem Schreiben in `leser`, ob die Nummer schon an einem Konto
-- hängt. Genau das ist im nächsten Schritt der Normalfall: Die Leserzeile des
-- Kontos trägt die Nummer, die das Konto in derselben Transaktion noch hat.
-- Bliebe er stehen, bräche die Migration an ihrer eigenen Umzugsbewegung ab.
DROP TRIGGER IF EXISTS trg_schueler_ausweis_eindeutig ON leser;
DROP TRIGGER IF EXISTS trg_benutzer_ausweis_eindeutig ON benutzer;
DROP FUNCTION IF EXISTS ausweis_eindeutig_ueber_personen();

-- ── 2. Jedes Konto bekommt seine Leserzeile ──────────────────────────────────
--
-- Zeilenweise und nicht als ein INSERT ... SELECT, weil jede neue Leser-ID sofort
-- an ihrem Konto vermerkt werden muss. Ein Sammel-INSERT gäbe die IDs zurück,
-- aber ohne verlässliche Zuordnung: Name und Ausweis taugen nicht als Schlüssel
-- (zwei Kollegen dürfen gleich heißen, und die meisten haben gar keine Nummer).
--
-- `art`: Wer eine Personenart trug, behält sie. Wer keine hatte — Admin,
-- Mitarbeiter, Helfer, Leitung — wird 'lehrkraft'. Das ist keine Aussage über
-- seine Rolle, sondern die Feststellung, dass er kein Schüler ist; die Art darf
-- die Leserdatei später jederzeit korrigieren. 'schueler' wäre falsch und würde
-- die Zeile dem LUSD-Abgleich ausliefern.
DO $$
DECLARE
    konto  record;
    neu_id uuid;
BEGIN
    FOR konto IN
        SELECT id, vorname, nachname, barcode_id, personenart
        FROM benutzer
        WHERE leser_id IS NULL
        ORDER BY nachname, vorname
    LOOP
        INSERT INTO leser (vorname, nachname, barcode_id, art)
        VALUES (konto.vorname, konto.nachname, konto.barcode_id,
                COALESCE(konto.personenart, 'lehrkraft'))
        RETURNING id INTO neu_id;

        UPDATE benutzer SET leser_id = neu_id WHERE id = konto.id;
    END LOOP;
END $$;

-- ── 3. Die Ausleihen ziehen auf den Leser um ─────────────────────────────────
--
-- Reihenfolge: erst umhängen, dann die Spalte entfernen. Andersherum wäre die
-- Zuordnung weg, bevor sie gelesen ist.
UPDATE ausleihen a
   SET schueler_id = b.leser_id
  FROM benutzer b
 WHERE a.ausleiher_benutzer_id = b.id
   AND b.leser_id IS NOT NULL;

UPDATE nachbuch_meldungen m
   SET ausleiher_schueler_id = b.leser_id
  FROM benutzer b
 WHERE m.ausleiher_benutzer_id = b.id
   AND b.leser_id IS NOT NULL;

UPDATE nachbuch_meldungen m
   SET vorbesitzer_schueler_id = b.leser_id
  FROM benutzer b
 WHERE m.vorbesitzer_benutzer_id = b.id
   AND b.leser_id IS NOT NULL;

UPDATE schadensfaelle s
   SET schueler_id = b.leser_id
  FROM benutzer b
 WHERE s.benutzer_id = b.id
   AND b.leser_id IS NOT NULL;

-- ── 4. Die Zwillingsspalten fallen ───────────────────────────────────────────
--
-- Mit ihnen fallen die „entweder-oder"-Regeln: Wo es nur noch eine Spalte gibt,
-- ist nichts mehr auszuschließen. `schueler_id` bleibt NULLBAR — die
-- DSGVO-Anonymisierung löst die Ausleihe bewusst von der Person, die Buchung
-- selbst bleibt als Zahl im Bestand stehen.
ALTER TABLE ausleihen DROP CONSTRAINT IF EXISTS check_loan_borrower;
DROP INDEX IF EXISTS idx_ausleihen_benutzer;
ALTER TABLE ausleihen DROP COLUMN IF EXISTS ausleiher_benutzer_id;

ALTER TABLE nachbuch_meldungen DROP COLUMN IF EXISTS ausleiher_benutzer_id;
ALTER TABLE nachbuch_meldungen DROP COLUMN IF EXISTS vorbesitzer_benutzer_id;

ALTER TABLE schadensfaelle DROP CONSTRAINT IF EXISTS check_damage_responsible;
DROP INDEX IF EXISTS idx_schadensfaelle_benutzer;
ALTER TABLE schadensfaelle DROP COLUMN IF EXISTS benutzer_id;

-- ── 5. Das Konto gibt Ausweis und Personenart ab ─────────────────────────────
--
-- Mit der Personenart fällt auch ihr Wächter aus Migration 120: Er sorgte dafür,
-- dass ein Kollegiumskonto nie ohne Personenart entsteht. Die Frage, die er
-- bewachte, gibt es nicht mehr — jedes Konto hat jetzt eine Leserzeile, und die
-- trägt die Art.
DROP TRIGGER IF EXISTS trg_benutzer_kollegium_personenart ON benutzer;
DROP FUNCTION IF EXISTS kollegium_hat_personenart();

ALTER TABLE benutzer DROP CONSTRAINT IF EXISTS chk_benutzer_personenart;
ALTER TABLE benutzer DROP COLUMN IF EXISTS personenart;

DROP INDEX IF EXISTS idx_benutzer_barcode;
ALTER TABLE benutzer DROP CONSTRAINT IF EXISTS benutzer_barcode_id_key;
ALTER TABLE benutzer DROP COLUMN IF EXISTS barcode_id;

COMMENT ON COLUMN benutzer.leser_id IS
    'Die Leserzeile dieses Kontos: dort stehen Name, Ausweis und Ausleihen. '
    'Das Konto selbst ist nur noch Anmeldung und Rechte.';

-- ── 6. Ein Konto ohne Leserzeile darf nicht entstehen ────────────────────────
--
-- Konten entstehen an fünf Stellen: Benutzerverwaltung, Selbstanmeldung des
-- Kollegiums, Littera-Übernahme, Seed und Testaufbauten. Jede davon einzeln um
-- „und dann noch die Leserzeile" zu ergänzen heißt, dass die sechste Stelle es
-- vergisst — und ein Kollege, den die Theke nicht findet, ist genau der Fehler,
-- den dieser ganze Umbau abschafft.
--
-- Deshalb steht es an der EINEN Stelle, an der kein Schreibweg vorbeikommt.
-- Der Ausweis wird hier NICHT gesetzt: Den bekommt ein Leser, wenn einer
-- gedruckt oder eingetragen wird, und er gehört zur Leserzeile.
CREATE OR REPLACE FUNCTION konto_hat_leserzeile()
RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    neu_id uuid;
BEGIN
    IF NEW.leser_id IS NULL THEN
        INSERT INTO leser (vorname, nachname, art)
        VALUES (NEW.vorname, NEW.nachname, 'lehrkraft')
        RETURNING id INTO neu_id;
        NEW.leser_id := neu_id;
    END IF;
    RETURN NEW;
END $$;

DROP TRIGGER IF EXISTS trg_benutzer_hat_leserzeile ON benutzer;
CREATE TRIGGER trg_benutzer_hat_leserzeile
BEFORE INSERT ON benutzer
FOR EACH ROW EXECUTE FUNCTION konto_hat_leserzeile();
