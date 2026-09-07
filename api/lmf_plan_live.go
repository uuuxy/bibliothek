package api

// lmf_plan_live.go — der LMF-Plan meldet seine Änderungen über die SSE-Leitung.
//
// Peter, 07.09.2026: „der lmf plan soll natürlich jederzeit sichtbar sein, auch
// Änderungen soll man live sehen — auch die Leute über Mein Portal."
//
// Bis hierher stimmte das nur im Moment des Öffnens: PortalLmfPlan lädt den Plan in
// `onMount` und danach nie wieder. Wer die Seite offen ließ — im Lehrerzimmer läuft sie
// nebenbei — sah eine verschobene Stunde erst nach F5. Dasselbe galt für den Planer
// selbst: Zwei Bibliothekskräfte an zwei PCs arbeiteten an derselben Reihenfolge, ohne
// voneinander zu wissen (Regel „geteilter Zustand immer zentral" — sie war eingehalten,
// nur erfuhr niemand davon).
//
// Das Signal trägt KEINE Daten, sondern nur den Anlass. Das ist hier nicht Sparsamkeit,
// sondern die Trennlinie: Auf der Leitung hängen auch Kollegiums-Sitzungen, und ein
// Entwurf geht sie nichts an (Migration 100). Jede Ansicht fragt nach dem Signal ihren
// EIGENEN Endpunkt neu — das Portal `GET /api/lmf-termine`, das nur veröffentlichte
// Pläne kennt, der Planer `GET /api/lmf-plan/{art}` hinter `edit_books`. Damit kann das
// Signal nichts ausleiten, was der Empfänger nicht ohnehin abrufen darf.

// meldeLmfPlanGeaendert sagt allen offenen Ansichten, dass sie den Plan neu holen
// sollen. Aufrufen NACH dem erfolgreichen Schreiben — ein Signal auf einen Schreibweg,
// der noch scheitern kann, schickte alle Ansichten auf den alten Stand zurück.
func (s *Server) meldeLmfPlanGeaendert() {
	if s.Broker != nil {
		s.Broker.Broadcast("lmf-plan", "{}")
	}
}
