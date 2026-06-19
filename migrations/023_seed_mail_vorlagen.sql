-- Migration 023: Seed Mail-Vorlagen

INSERT INTO mail_vorlagen (typ, betreff, text_body) 
VALUES 
(
    'MAHNUNG_ELTERN', 
    'Erinnerung: Bitte Bücher in die Schulbibliothek zurückbringen',
    'Liebe Eltern von {{.Vorname}} {{.Nachname}},

wir möchten Sie höflich daran erinnern, dass die Leihfrist für folgende Medien abgelaufen ist:

{{.BuchListe}}

Bitte sorgen Sie dafür, dass Ihr Kind die Medien zeitnah in der Bibliothek abgibt.
Ursprüngliche Frist: {{.Frist}}

Vielen Dank für Ihre Mithilfe!
Ihre Schulbibliothek'
)
ON CONFLICT (typ) DO NOTHING;

INSERT INTO mail_vorlagen (typ, betreff, text_body)
VALUES
(
    'BESTELLUNG_EINGETROFFEN',
    'Dein vorgemerktes Buch ist abholbereit!',
    'Hallo {{.Vorname}} {{.Nachname}},

dein vorgemerktes Buch ist in der Schulbibliothek für dich eingetroffen:

{{.BuchListe}}

Bitte hole das Buch bis zum {{.Frist}} ab, da es ansonsten an den nächsten Schüler auf der Warteliste weitergegeben wird.

Viele Grüße,
Deine Schulbibliothek'
)
ON CONFLICT (typ) DO NOTHING;
