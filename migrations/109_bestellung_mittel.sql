-- Migration 109: Eine Bestellung kennt ihren Topf — Lernmittelfreiheit (Land) oder
-- Schülerbücherei (Schulträger).
--
-- Bis hierher trug bestellungen_verlauf Lieferant, Kundennummer, Betrag, Exemplare und
-- Bestätigung — aber nicht, aus welchen Mitteln bestellt wurde. Das Anschreiben behauptete
-- für JEDE Bestellung „für unsere Schulbibliothek", auch für einen Lernmittel-Klassensatz.
-- Der Händler gewährt auf Lernmittel-Sammelbestellungen einen anderen Nachlass als auf
-- Bibliotheksbestände, und die Rechnungen der beiden Töpfe werden getrennt geführt.
-- Deshalb steht der Topf ab jetzt auf der Bestellung selbst — eine Bestellung = ein Topf,
-- ein gemischter Warenkorb wird beim Auslösen in zwei Bestellungen geteilt (bestätigt von
-- der EDV-Servicestelle für Schulbibliotheken, 10.09.2026; docs/mittel_konzept.md, Teil B).
--
-- Vokabular: 'land' (Lernmittelfreiheit, Eigentum des Landes) und 'schultraeger'
-- (Schülerbücherei, Mittel des Schulträgers). Neue Bestellungen MÜSSEN einen Topf tragen
-- (Go-Guard in api/order_service.go, 400 an der Tür); die Spalte bleibt nullbar für
-- Alt-Bestellungen, denen sich der Topf nicht eindeutig zuordnen lässt.
--
-- Backfill: Alt-Bestellungen bekommen den Topf NUR, wenn er aus den Positionen eindeutig
-- ist — alle Titel Lernmittel → 'land', kein Titel Lernmittel → 'schultraeger'. Gemischte
-- Bestellungen und solche mit inzwischen gelöschten Titeln (titel_id ist ON DELETE SET
-- NULL) bleiben NULL und erscheinen in Berichten als „ohne Zuordnung" — nie geraten.
--
-- lieferanten.kundennummer_schultraeger: Händler führen für Lernmittel und Bibliothek oft
-- getrennte Kundenkonten (anderer Nachlass, andere Rechnungsstelle). Leer = dieselbe
-- Nummer wie kundennummer.
--
-- IF NOT EXISTS mit Absicht: api/bestellung_mittel_backfill_pg_test.go führt diese Datei
-- gegen eine Datenbank aus, die die Spalten aus schema.sql bereits hat, und prüft nur den
-- Backfill. Die Parität beider Wege prüft db/migrations_schema_paritaet_pg_test.go.

ALTER TABLE bestellungen_verlauf
    ADD COLUMN IF NOT EXISTS mittel TEXT
        CONSTRAINT bestellungen_verlauf_mittel_check
        CHECK (mittel IS NULL OR mittel IN ('land', 'schultraeger'));

ALTER TABLE lieferanten
    ADD COLUMN IF NOT EXISTS kundennummer_schultraeger VARCHAR(100) NOT NULL DEFAULT '';

WITH lage AS (
    SELECT p.bestellung_id,
           bool_and(t.id IS NOT NULL)   AS alle_titel_bekannt,
           bool_and(t.ist_lernmittel)   AS alle_lernmittel,
           bool_or(t.ist_lernmittel)    AS irgendein_lernmittel
    FROM bestellungen_positionen p
    LEFT JOIN buecher_titel t ON t.id = p.titel_id
    GROUP BY p.bestellung_id
)
UPDATE bestellungen_verlauf b
   SET mittel = CASE
                    WHEN l.alle_lernmittel          THEN 'land'
                    WHEN NOT l.irgendein_lernmittel THEN 'schultraeger'
                END
  FROM lage l
 WHERE l.bestellung_id = b.id
   AND b.mittel IS NULL
   AND l.alle_titel_bekannt;

-- Die Seed-Vorlage der Händler-Mail (Migration 052) nennt den Topf jetzt im Betreff und
-- im Text ({{.Mittel}}). Nur die UNVERÄNDERTE Seed-Fassung wird nachgezogen — eine vom
-- Sekretariat umformulierte Vorlage bleibt, wie sie ist; fehlt ihr der Platzhalter, hängt
-- der Versand den Vermerk automatisch als eigenen Absatz an (api/bestellmail_text.go).
UPDATE mail_vorlagen
   SET betreff   = 'Buchbestellung {{.Mittel}} - {{.Datum}} (Kundennummer {{.Kundennummer}})',
       text_body = 'Sehr geehrte Damen und Herren,

anbei erhalten Sie unsere Buchbestellung vom {{.Datum}} (Kundennummer: {{.Kundennummer}}) sowie den zugehörigen Barcode-Bogen zur Vorab-Beklebung der Exemplare.

Diese Bestellung: {{.Mittel}} — den Vermerk finden Sie auch im Anschreiben; bitte führen Sie ihn auf der Rechnung.

Bestellte Titel: {{.AnzahlTitel}}
Gesamtanzahl Exemplare: {{.AnzahlExemplare}}

Mit freundlichen Grüßen,
Schulbibliothek'
 WHERE typ = 'BESTELLUNG_HAENDLER'
   AND betreff = 'Buchbestellung Schulbibliothek - {{.Datum}} (Kundennummer {{.Kundennummer}})'
   AND text_body = 'Sehr geehrte Damen und Herren,

anbei erhalten Sie unsere Buchbestellung vom {{.Datum}} (Kundennummer: {{.Kundennummer}}) sowie den zugehörigen Barcode-Bogen zur Vorab-Beklebung der Exemplare.

Bestellte Titel: {{.AnzahlTitel}}
Gesamtanzahl Exemplare: {{.AnzahlExemplare}}

Mit freundlichen Grüßen,
Schulbibliothek';
