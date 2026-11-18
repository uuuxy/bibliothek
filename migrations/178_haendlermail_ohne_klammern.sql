-- =============================================================================
-- Migration 178: Die Händler-Mail ohne Klammern und Gedankenstrich
-- =============================================================================
-- M3 (Content design, Grammar and punctuation): Klammern stehen nur um ein Kürzel oder eine
-- Quelle, und „Em dashes are best avoided in UX writing“ (docs/STIL.md 11). Wie in Migration 109
-- wird nur die unveränderte Seed-Fassung nachgezogen; eine vom Sekretariat umformulierte Vorlage
-- bleibt, wie sie ist. Die neue Fassung ist dieselbe wie die Vorgabe im Code
-- (service.bestellMailVorgabeBetreff und bestellMailVorgabeText).
--
-- Gemessen am Testserver am 18.11.2026 (lesend, über md5 von Betreff und Text): Es gibt eine Vorlage
-- BESTELLUNG_HAENDLER, und sie trägt Betreff und Text der Saat unverändert; die Migration zieht
-- genau sie nach. Eine umformulierte Vorlage trifft die Bedingung nicht und bleibt stehen.
UPDATE mail_vorlagen
   SET betreff   = 'Buchbestellung {{.Mittel}} vom {{.Datum}}, Kundennummer {{.Kundennummer}}',
       text_body = 'Sehr geehrte Damen und Herren,

anbei erhalten Sie unsere Buchbestellung vom {{.Datum}} zur Kundennummer {{.Kundennummer}} sowie den zugehörigen Barcode-Bogen zur Vorab-Beklebung der Exemplare.

Diese Bestellung: {{.Mittel}}. Den Vermerk finden Sie auch im Anschreiben; bitte führen Sie ihn auf der Rechnung.

Bestellte Titel: {{.AnzahlTitel}}
Gesamtanzahl Exemplare: {{.AnzahlExemplare}}

Mit freundlichen Grüßen
Schulbibliothek'
 WHERE typ = 'BESTELLUNG_HAENDLER'
   AND betreff = 'Buchbestellung {{.Mittel}} - {{.Datum}} (Kundennummer {{.Kundennummer}})'
   AND text_body = 'Sehr geehrte Damen und Herren,

anbei erhalten Sie unsere Buchbestellung vom {{.Datum}} (Kundennummer: {{.Kundennummer}}) sowie den zugehörigen Barcode-Bogen zur Vorab-Beklebung der Exemplare.

Diese Bestellung: {{.Mittel}} — den Vermerk finden Sie auch im Anschreiben; bitte führen Sie ihn auf der Rechnung.

Bestellte Titel: {{.AnzahlTitel}}
Gesamtanzahl Exemplare: {{.AnzahlExemplare}}

Mit freundlichen Grüßen,
Schulbibliothek';
