package api

import (
	"bibliothek/repository"
	"net/http"
)

func (s *Server) registerBookRoutes(mux *http.ServeMux, bookRepo repository.BookRepository, auditRepo repository.AuditRepository) {
	// ── BUECHER (Titles & Copies) ──
	mux.Handle("DELETE /api/buecher/titel/{id}", s.RequirePermission("delete_books")(s.DeleteTitleHandler(auditRepo)))

	// Exemplare (Copies)
	mux.Handle("GET /api/buecher/titel/{id}/exemplare", s.RequirePermission("view_books")(s.GetTitleCopiesHandler()))
	mux.Handle("GET /api/buecher/titel/{id}/ausleiher", s.RequirePermission("view_books")(s.GetTitleBorrowersHandler()))
	mux.Handle("GET /api/buecher/titel/{id}/historie", s.RequirePermission("view_books")(s.GetTitleHistoryHandler()))
	mux.Handle("GET /api/buecher/titel/{id}/etiketten", s.RequirePermission("view_books")(s.LabelsHandler()))

	mux.Handle("DELETE /api/buecher/exemplare/{id}", s.RequirePermission("delete_books")(s.DeleteCopyHandler(auditRepo)))

	// Update specific copy fields
	mux.Handle("POST /api/buecher/exemplare/{id}/schadensnotiz", s.RequirePermission("edit_books")(s.UpdateDamageNoteHandler(bookRepo)))
	mux.Handle("PUT /api/buecher/exemplare/{id}/barcode", s.RequirePermission("edit_books")(s.UpdateCopyBarcodeHandler(bookRepo)))
	mux.Handle("PUT /api/buecher/exemplare/{id}/status", s.RequirePermission("edit_books")(s.UpdateCopyStatusHandler(bookRepo)))
	mux.Handle("POST /api/buecher/exemplare/{id}/defekt", s.RequirePermission("edit_books")(s.MarkCopyDefektHandler()))
	mux.Handle("POST /api/buecher/exemplare/{id}/aussondern", s.RequirePermission("edit_books")(s.AussondernCopyHandler(bookRepo)))

	// ── AUSLEIHEN (Loans) ──
	mux.Handle("DELETE /api/ausleihen/{id}/rueckgabe", s.RequirePermission("view_students")(s.UndoReturnHandler()))
	mux.Handle("POST /api/ausleihen/{ausleihe_id}/verlaengern", s.RequirePermission("edit_books")(s.ExtendLoanHandler()))
	mux.Handle("POST /api/ausleihen/global-extend-lmf", s.RequirePermission("edit_books")(s.GlobalExtendLMFHandler()))

	// Live ISBN Lookup
	mux.Handle("POST /api/buecher/aus-isbn", s.RequirePermission("create_orders")(s.ISBNZuTitelHandler()))

	// Vormerkungen
	mux.Handle("GET /api/vormerkungen", s.RequirePermission("view_books")(s.ListVormerkungHandler()))
	mux.Handle("POST /api/vormerkungen", s.RequirePermission("view_books")(s.CreateVormerkungHandler()))
	mux.Handle("DELETE /api/vormerkungen/{id}", s.RequirePermission("view_books")(s.DeleteVormerkungHandler()))

	// Klassensatz Reservierungen
	mux.Handle("POST /api/reservierungen/klassensatz", s.RequirePermission("view_students")(s.CreateKlassensatzReservierungHandler()))
	mux.Handle("GET /api/reservierungen/klassensatz", s.RequirePermission("view_orders")(s.GetKlassensatzReservierungenHandler()))
	mux.Handle("GET /api/reservierungen/klassensatz/anzahl", s.RequirePermission("view_orders")(s.GetKlassensatzReservierungenAnzahlHandler()))
	mux.Handle("PUT /api/reservierungen/klassensatz/{id}/erledigen", s.RequirePermission("create_orders")(s.ErledigeKlassensatzReservierungHandler()))
}
