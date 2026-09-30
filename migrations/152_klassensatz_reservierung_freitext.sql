-- =============================================================================
-- Migration 152: Die Klasse einer Klassensatz-Reservierung ist Freitext (docs/OFFEN.md 5.18)
-- =============================================================================
-- Festgelegt am 30.09.2026: Die Klasse an einer Klassensatz-Reservierung zeigt das Programm nur
-- an, es rechnet nicht damit — die Bibliothek sieht, für wen die Bücher sind. Seit Migration 079
-- trug die Reservierung trotzdem jede Eingabe als Klasse ins Vokabular ein: Ein Tippfehler im
-- Portal („07R44") legte eine Klasse an, die es an der Schule nicht gibt, und ein Kurs ging gar
-- nicht erst als Kurs durch. Jetzt bleibt der Text, wie die Lehrkraft ihn schreibt, wie schon bei
-- lehrer_anliegen.klasse (Migration 079: „bleibt Freitext, dort stehen auch Oberstufenkurse").
--
-- Das Kürzen und Trimmen übernimmt der Server (api/reservation.go), wie beim Anliegen.
--
-- Ohne Messung: Die Migration ändert keine Zeile. Vorhandene Reservierungen behalten ihren Text;
-- Klassen, die nur eine Reservierung angelegt hatte, bleiben im Vokabular stehen. Sichtbar sind
-- sie nirgends: Die Auswahllisten lesen die Klassen der Schüler (GET /api/klassen), nicht das
-- Vokabular. Umbenennen einer Klasse zieht den Text einer Reservierung nicht mehr mit.
-- =============================================================================

DROP TRIGGER IF EXISTS trg_ksr_klasse_vokabular ON klassensatz_reservierungen;

ALTER TABLE klassensatz_reservierungen DROP CONSTRAINT IF EXISTS fk_ksr_klasse_vokabular;
