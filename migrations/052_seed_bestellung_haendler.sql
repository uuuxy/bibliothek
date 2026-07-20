-- Migration 052: Seed Mail-Vorlage für die Buchbestellung an den Händler/Lieferanten.
-- Bis hierher war Betreff/Text der Bestellmail hartkodiert in api/pdf_service.go und
-- daher nicht über den Vorlagen-Editor pflegbar. Diese Vorlage macht ihn editierbar;
-- der Versand (DispatchOrderEmail) fällt bei fehlender Vorlage weiter auf denselben
-- Standardtext zurück, bleibt also immer versandfähig.
-- Platzhalter: {{.Datum}}, {{.Kundennummer}}, {{.AnzahlTitel}}, {{.AnzahlExemplare}}.

INSERT INTO mail_vorlagen (typ, betreff, text_body)
VALUES
(
    'BESTELLUNG_HAENDLER',
    'Buchbestellung Schulbibliothek - {{.Datum}} (Kundennummer {{.Kundennummer}})',
    'Sehr geehrte Damen und Herren,

anbei erhalten Sie unsere Buchbestellung vom {{.Datum}} (Kundennummer: {{.Kundennummer}}) sowie den zugehörigen Barcode-Bogen zur Vorab-Beklebung der Exemplare.

Bestellte Titel: {{.AnzahlTitel}}
Gesamtanzahl Exemplare: {{.AnzahlExemplare}}

Mit freundlichen Grüßen,
Schulbibliothek'
)
ON CONFLICT (typ) DO NOTHING;
