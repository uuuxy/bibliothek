-- =============================================================================
-- Migration 148: Auflagen eines Schulbuchs zusammenfassen (docs/OFFEN.md 4.18)
-- =============================================================================
-- Eine neue Auflage ist ein eigener Titel mit eigener ISBN — richtig so, sie hat andere
-- Seitenzahlen, und jedes Exemplar bleibt an seiner Auflage. Gezählt wird aber am Buch:
-- Die Nachbestell-Liste soll ein Buch in zwei Auflagen einmal nennen, mit dem Bestand
-- beider.
--
-- Die Form (entschieden am 17.09.2026): Ein Werk ist die Kennung, an der die Titel hängen,
-- die dasselbe Buch sind. werk_id ist NULLBAR, gelesen wird über COALESCE(werk_id, id):
-- Ein Titel ohne Werk ist sein eigenes. Nachzutragen ist deshalb nichts, und jeder
-- Lesepfad, der nicht nach Auflagen fragt, bleibt, wie er ist.
--
-- Ohne Spalte für einen Namen (entschieden am 25.09.2026): Jede Ansicht zeigt die neueste
-- Auflage mit ihrem Titel. Ein Name, den keine Ansicht liest und niemand pflegt, liefe
-- auseinander wie meldebestand.
--
-- Geschrieben wird werk_id nur in repository/auflagen.go — nicht beim Speichern der
-- Titelmaske (UpdateBook). Die Regeln stehen dort an einer Stelle: nur Lernmittel, zwei
-- Gruppen werden eine, ein Werk mit weniger als zwei Titeln fällt.
--
-- ON DELETE SET NULL: Ein Werk löscht nur repository/auflagen.go, und erst, nachdem es den
-- letzten Titel daran gelöst hat. Der SET NULL fängt einen Titel auf, der dabei übersehen
-- würde — er steht danach allein, wie vor dem Zusammenfassen.

CREATE TABLE IF NOT EXISTS werke (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    angelegt_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE werke IS
    'Ein Buch über seinen Auflagen (Migration 148): Titel mit derselben werk_id sind '
    'Auflagen desselben Buchs. Geschrieben nur über repository/auflagen.go.';

ALTER TABLE buecher_titel
    ADD COLUMN IF NOT EXISTS werk_id UUID
        CONSTRAINT fk_titel_werk REFERENCES werke (id) ON DELETE SET NULL;

-- Trägt „alle Auflagen dieses Buchs" und die Gruppierung der Nachbestell-Liste. Teilindex:
-- Die meisten Titel gehören zu keinem Werk.
CREATE INDEX IF NOT EXISTS idx_buecher_titel_werk_id
    ON buecher_titel (werk_id) WHERE werk_id IS NOT NULL;
