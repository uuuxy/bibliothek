package lernmittel

// Der LMF-Plan meldet seine Änderungen über die SSE-Leitung: Eine offene Ansicht, das Portal
// im Lehrerzimmer oder der Planer am zweiten Arbeitsplatz, holt den Plan dann neu, statt bis
// zum Neuladen den alten Stand zu zeigen.
//
// Das Signal trägt keine Daten, nur den Anlass. Auf der Leitung hängen auch Sitzungen des
// Kollegiums, und ein Entwurf geht sie nichts an. Jede Ansicht fragt nach dem Signal ihren
// eigenen Endpunkt neu: das Portal GET /api/lmf-termine, das nur veröffentlichte Pläne
// kennt, der Planer GET /api/lmf-plan/{art} hinter edit_books. So leitet das Signal nichts
// aus, was der Empfänger nicht ohnehin abrufen darf.

// meldeLMFPlanGeaendert sagt allen offenen Ansichten, dass sie den Plan neu holen
// sollen. Aufrufen nach dem erfolgreichen Schreiben: ein Signal auf einen Schreibweg,
// der noch scheitern kann, schickte alle Ansichten auf den alten Stand zurück.
func (t *Tueren) meldeLMFPlanGeaendert() {
	if t.Broker != nil {
		t.Broker.Broadcast("lmf-plan", "{}")
	}
}
