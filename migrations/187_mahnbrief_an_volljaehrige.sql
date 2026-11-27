-- =============================================================================
-- Migration 187: Vorlage für den Mahnbrief an volljährige Schüler
-- =============================================================================
-- Der Mahnbrief ging immer an „Eltern von …“. Einen volljährigen Schüler schreibt er jetzt selbst
-- an, wie Bescheid und Ersatzforderung (repository.sqlVolljaehrig); dafür diese zweite Vorlage,
-- die die Schule unter Einstellungen bearbeitet wie die an die Eltern (MAHNUNG_ELTERN).
-- Idempotent: ON CONFLICT.
-- =============================================================================

INSERT INTO mail_vorlagen (typ, betreff, text_body)
VALUES (
    'MAHNUNG_VOLLJAEHRIG',
    'Erinnerung: Bitte Bücher in die Schulbibliothek zurückbringen',
    'Guten Tag {{.Vorname}} {{.Nachname}},

wir möchten Sie höflich daran erinnern, dass die Leihfrist für folgende Medien abgelaufen ist:

{{.BuchListe}}

Bitte geben Sie die Medien zeitnah in der Bibliothek ab.
Ursprüngliche Frist: {{.Frist}}

Vielen Dank!
Ihre Schulbibliothek'
)
ON CONFLICT (typ) DO NOTHING;
