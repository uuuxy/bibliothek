package api

import (
	"bibliothek/repository"
	"net/http"
)

func (s *Server) registerBookRoutes(mux *http.ServeMux, bookRepo repository.BookRepository, auditRepo repository.AuditRepository) {
	// ── BUECHER (Titles & Copies) ──
	mux.Handle("DELETE /api/buecher/titel/{id}", s.RequirePermission("delete_books")(s.DeleteTitleHandler(auditRepo)))
	// Bestellkorb-Korrektur der Signatur (create_orders, nicht edit_books — dieselbe
	// Berechtigung wie das Anlegen des Titels über /aus-isbn, siehe api/isbn_handler.go).
	mux.Handle("PUT /api/buecher/titel/{id}/signatur", s.RequirePermission("create_orders")(s.UpdateTitelSignaturHandler()))

	// Exemplare (Copies)
	mux.Handle("GET /api/buecher/titel/{id}/exemplare", s.RequirePermission("view_books")(s.GetTitleCopiesHandler()))
	mux.Handle("GET /api/buecher/titel/{id}/ausleiher", s.RequirePermission("view_books")(s.GetTitleBorrowersHandler()))
	mux.Handle("GET /api/buecher/titel/{id}/historie", s.RequirePermission("view_books")(s.GetTitleHistoryHandler()))
	mux.Handle("GET /api/buecher/titel/{id}/etiketten", s.RequirePermission("view_books")(s.LabelsHandler()))
	mux.Handle("POST /api/print/labels", s.RequirePermission("view_books")(s.PrintLabelsHandler()))

	// Nachdruck: Welche Exemplare haben noch kein Etikett, und Gegenbuchung nach dem Druck.
	// edit_books statt view_books — die Gegenbuchung ändert Bestandsdaten.
	mux.Handle("GET /api/exemplare/etiketten-offen", s.RequirePermission("edit_books")(s.EtikettenOffenHandler()))
	mux.Handle("GET /api/exemplare/etiketten-offen/anzahl", s.RequirePermission("edit_books")(s.EtikettenOffenAnzahlHandler()))
	mux.Handle("POST /api/exemplare/etiketten-gedruckt", s.RequirePermission("edit_books")(s.EtikettenGedrucktHandler()))
	// Der Weg zurück nach Papierstau oder zu weitem Altbestands-Stichtag.
	mux.Handle("POST /api/exemplare/etiketten-zuruecksetzen", s.RequirePermission("edit_books")(s.EtikettenZuruecksetzenHandler()))
	mux.Handle("POST /api/exemplare/etiketten-altbestand", s.RequirePermission("edit_books")(s.EtikettenAltbestandHandler()))

	mux.Handle("DELETE /api/buecher/exemplare/{id}", s.RequirePermission("delete_books")(s.DeleteCopyHandler(auditRepo)))

	// Update specific copy fields
	damageRepo := repository.NewDamageRepository(s.DB.Pool)
	mux.Handle("POST /api/buecher/exemplare/{id}/schadensnotiz", s.RequirePermission("edit_books")(s.UpdateDamageNoteHandler(bookRepo)))
	mux.Handle("PUT /api/buecher/exemplare/{id}/barcode", s.RequirePermission("edit_books")(s.UpdateCopyBarcodeHandler(bookRepo)))
	mux.Handle("PUT /api/buecher/exemplare/{id}/status", s.RequirePermission("edit_books")(s.UpdateCopyStatusHandler(bookRepo)))
	mux.Handle("POST /api/buecher/exemplare/{id}/defekt", s.RequirePermission("edit_books")(s.MarkCopyDefektHandler(damageRepo)))
	mux.Handle("POST /api/buecher/exemplare/{id}/aussondern", s.RequirePermission("edit_books")(s.AussondernCopyHandler(bookRepo)))

	// ── AUSLEIHEN (Loans) ──
	settingsRepo := repository.NewSystemSettingsRepository(s.DB.Pool)
	mux.Handle("POST /api/ausleihen/{ausleihe_id}/verlaengern", s.RequirePermission("edit_books")(s.ExtendLoanHandler(settingsRepo)))
	mux.Handle("POST /api/ausleihen/global-extend-lmf", s.RequirePermission("edit_books")(s.GlobalExtendLMFHandler()))
	mux.Handle("PATCH /api/admin/ausleihen/{id}/faelligkeit", s.RequirePermission("edit_books")(s.OverrideDueDateHandler()))

	// Live ISBN Lookup
	mux.Handle("POST /api/buecher/aus-isbn", s.RequirePermission("create_orders")(s.ISBNZuTitelHandler()))

	// Vormerkungen
	vormerkungRepo := repository.NewVormerkungRepository(s.DB.Pool)
	mux.Handle("GET /api/vormerkungen", s.RequirePermission("view_books")(s.ListVormerkungHandler(vormerkungRepo)))
	mux.Handle("POST /api/vormerkungen", s.RequirePermission("view_books")(s.CreateVormerkungHandler(vormerkungRepo)))
	mux.Handle("DELETE /api/vormerkungen/{id}", s.RequirePermission("view_books")(s.DeleteVormerkungHandler(vormerkungRepo)))

	// Klassensatz Reservierungen
	//
	// create_reservations statt view_students (10.08.2026): Der Handler fasst keine
	// Schülerdaten an — er nimmt Titel-ID, Klasse als Freitext und Anzahl entgegen. Mit
	// view_students war das trotzdem gekoppelt: Wer im Kollegiums-Portal reservieren
	// dürfen sollte, brauchte dasselbe Recht, an dem Schülerdatei und Druck-Center
	// hängen — die einzige Portal-Funktion zog also die komplette Schülerdatei auf.
	mux.Handle("POST /api/reservierungen/klassensatz", s.RequirePermission("create_reservations")(s.CreateKlassensatzReservierungHandler()))
	mux.Handle("GET /api/reservierungen/klassensatz", s.RequirePermission("view_orders")(s.GetKlassensatzReservierungenHandler()))
	mux.Handle("GET /api/reservierungen/klassensatz/anzahl", s.RequirePermission("view_orders")(s.GetKlassensatzReservierungenAnzahlHandler()))
	// Warteschlangen-Sicht fürs Kollegium: dasselbe Recht wie das Anlegen — wer
	// reservieren darf, darf sehen, wer vor ihm dran ist.
	mux.Handle("GET /api/reservierungen/klassensatz/offen", s.RequirePermission("create_reservations")(s.OffeneKlassensatzReservierungenHandler()))
	mux.Handle("PUT /api/reservierungen/klassensatz/{id}/erledigen", s.RequirePermission("create_orders")(s.ErledigeKlassensatzReservierungHandler()))
}
