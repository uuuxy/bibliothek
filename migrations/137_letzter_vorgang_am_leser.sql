-- Migration 137: Die Karenz-Uhr bekommt eine eigene Spalte — leser.letzter_vorgang_am
--
-- Entschieden am 16.09.2026, freigegeben am 22.09.2026 (docs/OFFEN.md 4.12).
--
-- Der Fall. Die Uhr vor der Anonymisierung (repository.KarenzUhr) rechnete den letzten
-- Vorgang über zwei Unterabfragen aus: die späteste Rückgabe in `ausleihen` und den
-- spätesten Abschluss in `schadensfaelle`, beide über `schueler_id`. Genau diese Spalte
-- leert der Lesehistorie-Lauf (jobs/cron_dsgvo_lesehistorie.go, `schueler_id = NULL`).
-- Ist die Karenz länger eingestellt als die Lesehistorie-Frist — Karenz 90 gegen
-- Schülerbücherei 90 ist die Vorgabe, aber beide sind einstellbar —, verschwindet die
-- Rückgabe aus der Rechnung, die Uhr fällt auf den Abgang zurück und die Zeile wird
-- FRÜHER anonymisiert als eingestellt. Ein Lauf, der Daten sparen soll, verkürzt dabei
-- still die Frist eines anderen.
--
-- Nicht die beiden Fristen aneinander binden. Die Kopplung liefe in die falsche
-- Richtung: Um eine längere Karenz zu bekommen, müsste die Schule die Lesehistorie
-- verlängern — also mehr Personendaten länger aufbewahren. Die beiden Fristen
-- beantworten verschiedene Fragen und dürfen sich nicht gegenseitig binden.
--
-- Stattdessen speichert die Uhr ihren Zeitpunkt selbst. Die Spalte hängt am LESER und
-- überlebt das Trennen der Ausleihe; sie trägt keinen Hinweis darauf, WAS gelesen wurde,
-- nur WANN zuletzt ein Vorgang abgeschlossen war. Das ist genau die Angabe, die die
-- Karenz braucht, und die sparsamste Form davon.
--
-- Gepflegt von Triggern, nicht von Schreibpfaden. Eine Rückgabe entsteht an mehreren
-- Stellen (Theke, Nachbuchen eines Offline-Scans, Sammelrückgabe, Reparaturen); jede
-- einzeln um „und dann noch den Stempel" zu ergänzen heißt, dass die nächste es vergisst.
-- Dasselbe Muster wie konto_hat_leserzeile (Migration 125): an der einen Stelle, an der
-- kein Schreibweg vorbeikommt.
--
-- Die Uhr geht nie zurück. Wird eine Rückgabe storniert, bleibt der Stempel stehen,
-- während die alte Unterabfrage ihn verloren hätte. Praktisch ändert das nichts: Eine
-- stornierte Rückgabe heißt, das Buch ist wieder offen, und eine offene Ausleihe schützt
-- die Zeile ohnehin vor der Anonymisierung (PredikatAnonymisierung). Die Richtung ist
-- die unschädliche — es wird später anonymisiert, nicht früher.

-- ── 1. Die Spalte ────────────────────────────────────────────────────────────
ALTER TABLE leser ADD COLUMN IF NOT EXISTS letzter_vorgang_am TIMESTAMPTZ;

COMMENT ON COLUMN leser.letzter_vorgang_am IS
    'Zeitpunkt des letzten ABGESCHLOSSENEN Vorgangs dieses Lesers: letzte Rückgabe einer '
    'Ausleihe oder letzter Abschluss eines Schadensfalls (bezahlt oder storniert). Die Uhr '
    'der Karenzzeit vor der Anonymisierung (repository.KarenzUhr). Von Triggern gepflegt, '
    'nie von Hand gesetzt; überlebt das Trennen der Ausleihe durch den Lesehistorie-Lauf.';

-- ── 2. Die Sicht muss die Spalte mitbekommen ─────────────────────────────────
--
-- `CREATE VIEW schueler AS SELECT * FROM leser` (Migration 124) friert die Spaltenliste
-- beim Anlegen ein. Eine neue Spalte in leser erscheint dort NICHT von selbst — und
-- PredikatAnonymisierung liest `schueler`, nicht `leser`. Ohne diese Zeile schlüge die
-- Löschuhr mit „column schueler.letzter_vorgang_am does not exist" fehl, und zwar erst
-- nachts im Cron. CREATE OR REPLACE genügt, weil ADD COLUMN hinten anhängt und eine
-- ersetzte Sicht Spalten nur ANHÄNGEN darf.
CREATE OR REPLACE VIEW schueler AS
    SELECT * FROM leser WHERE art = 'schueler'
    WITH CHECK OPTION;

-- ── 3. Ein Vorgang ist keine Änderung am Leser ───────────────────────────────
--
-- `trg_schueler_aktualisiert_am` stempelt bei jedem UPDATE auf leser die Spalte
-- aktualisiert_am. Die Trigger unten schreiben letzter_vorgang_am — jede Rückgabe an der
-- Theke wäre damit eine „Änderung" am Schülerdatensatz. Das ist nicht nur kosmetisch
-- falsch: aktualisiert_am ist der Rückfall von repository.AbgangSeit für Altzeilen ohne
-- Abgangsstempel. Eine Rückgabe würde also für genau diese Zeilen die Uhr vorschieben —
-- an derselben Stelle, die diese Migration in Ordnung bringt.
--
-- Deshalb bekommt leser eine eigene Stempel-Funktion: Ändert sich NUR letzter_vorgang_am
-- und sonst kein Feld, bleibt aktualisiert_am stehen. Der Vergleich läuft über den ganzen
-- Datensatz (to_jsonb), nicht über eine Aufzählung von Spalten — eine Aufzählung veraltet
-- still, sobald jemand eine Spalte ergänzt. Der teure Zweig wird nur betreten, wenn sich
-- letzter_vorgang_am überhaupt geändert hat; ein LUSD-Massenlauf baut nie ein jsonb.
--
-- Der Triggername bleibt `trg_schueler_aktualisiert_am` (er steht in den Ratschen); nur
-- die Funktion dahinter wechselt. set_aktualisiert_am() bleibt unverändert und gilt
-- weiter für alle anderen Tabellen.
CREATE OR REPLACE FUNCTION leser_aktualisiert_am()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.letzter_vorgang_am IS DISTINCT FROM OLD.letzter_vorgang_am
       AND to_jsonb(NEW) - 'letzter_vorgang_am' - 'aktualisiert_am'
         = to_jsonb(OLD) - 'letzter_vorgang_am' - 'aktualisiert_am' THEN
        NEW.aktualisiert_am := OLD.aktualisiert_am;
        RETURN NEW;
    END IF;
    NEW.aktualisiert_am := CURRENT_TIMESTAMP;
    RETURN NEW;
END $$;

DROP TRIGGER IF EXISTS trg_schueler_aktualisiert_am ON leser;
CREATE TRIGGER trg_schueler_aktualisiert_am
BEFORE UPDATE ON leser
FOR EACH ROW EXECUTE FUNCTION leser_aktualisiert_am();

-- ── 4. Die Rückgabe stempelt ─────────────────────────────────────────────────
--
-- Gestempelt wird nur nach VORNE: Das WHERE lässt das UPDATE aus, wenn der Stempel schon
-- später steht. Damit läuft der Trigger bei den allermeisten Rückgaben (Stempel = NULL
-- oder älter) genau einmal und sonst gar nicht — und eine nachgebuchte alte Rückgabe
-- kann die Uhr nicht zurückdrehen.
CREATE OR REPLACE FUNCTION leser_stempel_rueckgabe()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.schueler_id IS NULL OR NEW.rueckgabe_am IS NULL THEN
        RETURN NULL;
    END IF;
    UPDATE leser SET letzter_vorgang_am = NEW.rueckgabe_am
     WHERE id = NEW.schueler_id
       AND (letzter_vorgang_am IS NULL OR letzter_vorgang_am < NEW.rueckgabe_am);
    RETURN NULL;
END $$;

DROP TRIGGER IF EXISTS trg_leser_stempel_rueckgabe ON ausleihen;
CREATE TRIGGER trg_leser_stempel_rueckgabe
AFTER INSERT OR UPDATE OF rueckgabe_am, schueler_id ON ausleihen
FOR EACH ROW EXECUTE FUNCTION leser_stempel_rueckgabe();

-- ── 5. Der abgeschlossene Schadensfall stempelt ──────────────────────────────
--
-- Abgeschlossen heißt ist_bezahlt — das setzt sowohl die Bezahlung als auch das Storno
-- (repository/audit_system.go). Der Zeitpunkt ist der spätere von aktualisiert_am und
-- storniert_am; GREATEST übergeht dabei ein NULL.
CREATE OR REPLACE FUNCTION leser_stempel_schaden()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE
    zeitpunkt TIMESTAMPTZ;
BEGIN
    IF NEW.schueler_id IS NULL OR NOT NEW.ist_bezahlt THEN
        RETURN NULL;
    END IF;
    zeitpunkt := GREATEST(NEW.aktualisiert_am, NEW.storniert_am);
    UPDATE leser SET letzter_vorgang_am = zeitpunkt
     WHERE id = NEW.schueler_id
       AND (letzter_vorgang_am IS NULL OR letzter_vorgang_am < zeitpunkt);
    RETURN NULL;
END $$;

DROP TRIGGER IF EXISTS trg_leser_stempel_schaden ON schadensfaelle;
CREATE TRIGGER trg_leser_stempel_schaden
AFTER INSERT OR UPDATE OF ist_bezahlt, storniert_am, aktualisiert_am, schueler_id ON schadensfaelle
FOR EACH ROW EXECUTE FUNCTION leser_stempel_schaden();

-- ── 6. Rückfüllung über den ganzen Bestand ───────────────────────────────────
--
-- Genau die Rechnung, die KarenzUhr bis heute zur Laufzeit angestellt hat — einmal
-- ausgeführt und festgehalten. Was der Lesehistorie-Lauf VOR dieser Migration schon
-- getrennt hat, ist nicht mehr auffindbar; diese Zeilen behalten NULL und rechnen wie
-- bisher allein ab dem Abgang. Das ist kein Rückschritt gegenüber heute, nur keine
-- rückwirkende Heilung. Die Sperre gegen aktualisiert_am aus Abschnitt 3 gilt hier schon:
-- Der Massenlauf berührt nur letzter_vorgang_am und lässt jeden Änderungsstempel stehen.
UPDATE leser l
   SET letzter_vorgang_am = q.zeitpunkt
  FROM (
    SELECT leser.id,
           GREATEST(
               (SELECT max(a.rueckgabe_am) FROM ausleihen a WHERE a.schueler_id = leser.id),
               (SELECT max(GREATEST(sf.aktualisiert_am, sf.storniert_am)) FROM schadensfaelle sf
                 WHERE sf.schueler_id = leser.id AND sf.ist_bezahlt)
           ) AS zeitpunkt
      FROM leser
  ) q
 WHERE l.id = q.id
   AND q.zeitpunkt IS NOT NULL
   AND l.letzter_vorgang_am IS DISTINCT FROM q.zeitpunkt;
