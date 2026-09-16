-- =============================================================================
-- Migration 123: Aus der Schülertabelle wird die Lesertabelle
-- =============================================================================
-- Entschieden am 16.09.2026: Alle Leser stehen an einem Ort, und
-- ausleihen darf jeder aktive Leser — nicht mehr eine Rolle. Bis heute lagen
-- Schüler und Kollegium in zwei Tabellen, eine Ausleihe zeigte auf genau eine
-- davon, und jede Frage über "Leser" musste zweimal gestellt werden.
--
-- Diese Migration ist ADDITIV: Sie legt die Form an und einen Schutz. Es
-- entsteht noch KEINE Leserzeile für ein Konto — das kommt mit der Stufe, die
-- auch die Ausleihe und die Theke umstellt. Ein Kollege in dieser Tabelle wäre
-- sonst sofort in jeder Schülerliste sichtbar, bevor irgendeine Liste weiß, dass
-- es zwei Arten gibt.
--
-- Die Spalten- und Tabellennamen bleiben zunächst ("schueler"); die Umbenennung
-- ist eine eigene, rein mechanische Stufe über 408 Fundstellen und gehört nicht
-- in denselben Commit wie eine Verhaltensänderung.
-- =============================================================================

-- 1. Die Art des Lesers. Vorgabe 'schueler', damit jede bestehende Zeile bleibt,
--    was sie ist. „LiV" statt „Referendar" ist das gewählte Wort (15.09.2026) und nur
--    eine Bezeichnung — entschieden wird daran nichts.
ALTER TABLE schueler
    ADD COLUMN IF NOT EXISTS art character varying(20) NOT NULL DEFAULT 'schueler';

ALTER TABLE schueler DROP CONSTRAINT IF EXISTS chk_leser_art;
ALTER TABLE schueler
    ADD CONSTRAINT chk_leser_art CHECK (art IN ('schueler', 'lehrkraft', 'liv'));

-- 2. Drei Spalten waren PFLICHT, weil jede Zeile ein Schüler war: Klasse,
--    Abgängerjahr und Ausweisnummer. Für einen Kollegen gilt das nicht — er hat
--    keine Klasse, kein Abgängerjahr, und einen Ausweis erst, wenn einer
--    gedruckt ist.
ALTER TABLE schueler ALTER COLUMN klasse DROP NOT NULL;
ALTER TABLE schueler ALTER COLUMN abgaenger_jahr DROP NOT NULL;
ALTER TABLE schueler ALTER COLUMN barcode_id DROP NOT NULL;

-- Die Pflicht wird nicht aufgegeben, sondern an die Art GEPAART. Ohne diese
-- Paarung wäre aus drei Pflichtfeldern für alle ein "darf leer sein" für alle
-- geworden — und ein Schüler ohne Klasse fällt in jeder Klassenliste und jeder
-- Mahnung lautlos hinten runter. Genau diese Bugklasse hat am 14.09.2026 die
-- LUSD-Klasse getroffen: Ein Gate prüfte den Wert, nicht die Paarung.
ALTER TABLE schueler DROP CONSTRAINT IF EXISTS chk_leser_schueler_pflichtfelder;
ALTER TABLE schueler
    ADD CONSTRAINT chk_leser_schueler_pflichtfelder CHECK (
        art <> 'schueler'
        OR (klasse IS NOT NULL AND abgaenger_jahr IS NOT NULL AND barcode_id IS NOT NULL)
    );

-- 3. Der Schutz, um den es hier wirklich geht.
--
-- Der LUSD-Abgleich lädt den Bestand mit "SELECT ... FROM schueler WHERE
-- deleted_at IS NULL" (repository/lusd_bestand.go) und behandelt JEDE Zeile, die
-- der Export nicht kennt, als Abgänger: ist_abgaenger, gesperrt, und nach der
-- Karenz Name, Adresse und Geburtsdatum anonymisiert. Ein Kollegium in derselben
-- Tabelle wäre damit beim ersten Import Freiwild — die Namen wären weg.
--
-- Die Abfrage wird im selben Schritt auf art='schueler' eingeschränkt. Aber ein
-- WHERE ist ein Versprechen, das der nächste Schreibweg nicht kennt: Es gibt in
-- diesem Projekt genug Fälle, in denen ein Schutz nur als Kommentar existierte
-- und die zweite Tür daran vorbeiführte. Deshalb steht er hier als CHECK: Nur
-- ein Schüler kann Abgänger sein, anonymisiert sein oder eine LUSD-ID tragen.
-- Greift der Abgleich doch einmal nach einem Kollegen, bricht die Transaktion
-- ab und der Import schlägt laut fehl — das ist das gewünschte Verhalten. Ein
-- fehlgeschlagener Import ist ein Ärgernis; anonymisierte Kollegennamen sind
-- ein Datenverlust, den niemand bemerkt, bis jemand den Namen sucht.
ALTER TABLE schueler DROP CONSTRAINT IF EXISTS chk_leser_nur_schueler_werden_abgaenger;
ALTER TABLE schueler
    ADD CONSTRAINT chk_leser_nur_schueler_werden_abgaenger CHECK (
        art = 'schueler'
        OR (ist_abgaenger = false AND anonymized_at IS NULL AND lusd_id IS NULL)
    );

-- 4. Die Verknüpfung Konto → Leserzeile.
--
-- Ein Konto ist die Anmeldung samt Rechten, eine Leserzeile ist der Mensch mit
-- seinen Büchern. Meist gehören sie zusammen, aber nicht immer: Ein Schüler hat
-- kein Konto, und ein Konto, mit dem nur gearbeitet wird, braucht keine
-- Leserzeile. ON DELETE SET NULL, weil das Löschen einer Leserzeile die
-- Anmeldung nicht mitnehmen darf.
ALTER TABLE benutzer
    ADD COLUMN IF NOT EXISTS leser_id uuid REFERENCES schueler(id) ON DELETE SET NULL;

-- Eine Leserzeile gehört höchstens einem Konto. Ohne diese Eindeutigkeit könnten
-- zwei Anmeldungen auf dieselbe Person zeigen, und an der Theke wäre nicht mehr
-- feststellbar, wessen Ausleihen man sieht.
CREATE UNIQUE INDEX IF NOT EXISTS uniq_benutzer_leser
    ON benutzer (leser_id) WHERE leser_id IS NOT NULL;

-- Nachschlagen in der Gegenrichtung (Leser → Konto) braucht die Theke bei jedem
-- Scan; der Index oben deckt das mit ab.

COMMENT ON COLUMN schueler.art IS
    'Art des Lesers: schueler | lehrkraft | liv. Entscheidet NICHT über Rechte — '
    'nur darüber, wer vom LUSD-Abgleich und vom Löschjob erfasst wird.';
COMMENT ON COLUMN benutzer.leser_id IS
    'Leserzeile dieses Kontos (NULL = dieses Konto leiht nichts aus).';
