package api

import (
	"net/http"

	"bibliothek/internal/service"
	"bibliothek/repository"
)

func (s *Server) registerOrderRoutes(mux *http.ServeMux, orderSvc *service.OrderService) {
	// Bestellungen & Lieferanten
	// Anzeige-Regeln des Bestellwesens (z. B. ob mit Preisen gearbeitet wird).
	// view_orders statt manage_users: Wer bestellen darf, muss dafuer keine
	// Benutzerverwaltungsrechte bekommen (siehe api/bestellung/bestellkonfiguration.go).
	mux.Handle("GET /api/bestellungen/konfiguration", s.RequirePermission("view_orders")(s.bestellung().BestellKonfigurationHandler(
		repository.NewSystemSettingsRepository(s.DB.Pool), repository.NewSupplierRepository(s.DB.Pool))))
	mux.Handle("GET /api/bestellungen", s.RequirePermission("view_orders")(s.bestellung().GetReordersHandler()))
	mux.Handle("GET /api/bestellungen/pdf", s.RequirePermission("view_orders")(s.bestellung().ExportReordersPDFHandler()))
	mux.Handle("GET /api/bestellhistorie", s.RequirePermission("view_orders")(s.bestellung().GetBestellhistorieHandler()))
	mux.Handle("GET /api/bestellhistorie/uebersicht", s.RequirePermission("view_orders")(s.bestellung().GetBestellhistorieUebersichtHandler()))
	mux.Handle("GET /api/bestellhistorie/bericht", s.RequirePermission("view_orders")(s.bestellung().GetBestellBerichtPDFHandler()))
	// Nach den beiden festen Pfaden eingetragen, gilt aber unabhängig von der
	// Reihenfolge: Der ServeMux von Go 1.22 bevorzugt das genauere Muster, /uebersicht
	// und /bericht gewinnen also gegen {id}.
	mux.Handle("GET /api/bestellhistorie/{id}", s.RequirePermission("view_orders")(
		s.bestellung().GetBestelldetailHandler(repository.NewBestelldetailRepository(s.DB.Pool))))
	mux.Handle("GET /api/lieferanten", s.RequirePermission("view_orders")(s.bestellung().ListSuppliersHandler()))
	mux.Handle("POST /api/lieferanten", s.RequirePermission("create_orders")(s.bestellung().CreateSupplierHandler()))
	mux.Handle("PUT /api/lieferanten/{id}", s.RequirePermission("create_orders")(s.bestellung().UpdateSupplierHandler()))
	mux.Handle("DELETE /api/lieferanten/{id}", s.RequirePermission("create_orders")(s.bestellung().DeleteSupplierHandler()))
	mux.Handle("POST /api/bestellungen", s.RequirePermission("create_orders")(s.bestellung().SubmitOrderHandler(orderSvc)))
	mux.Handle("GET /api/bestellungen/zulauf", s.RequirePermission("view_orders")(s.bestellung().GetIncomingShipmentsHandler()))
	mux.Handle("POST /api/bestellungen/suche", s.RequirePermission("view_orders")(s.bestellung().SearchOrdersHandler()))
	mux.Handle("POST /api/bestellungen/bulk-receive", s.RequirePermission("create_orders")(s.bestellung().BulkReceiveOrderHandler()))
	mux.Handle("PUT /api/bestellungen/{id}/bestaetigen", s.RequirePermission("create_orders")(s.bestellung().BestaetigenBestellungHandler()))
	mux.Handle("PUT /api/bestellungen/{id}/bestaetigungs-link", s.RequirePermission("create_orders")(s.bestellung().NeuerBestaetigungsLinkHandler()))
	// Erneuter Versand der Bestellmail nach einem gescheiterten Versuch (Migration 161).
	mux.Handle("POST /api/bestellungen/{id}/mail", s.RequirePermission("create_orders")(s.bestellung().SendeBestellmailErneutHandler()))
	// Derselbe Vermerk ohne Versand: Der Händler hat die Bestellung auf anderem Weg erhalten.
	mux.Handle("DELETE /api/bestellungen/{id}/mail", s.RequirePermission("create_orders")(s.bestellung().NimmBestellmailVermerkHandler()))
	// Rückweg für den Topf (Migration 109): mit Pflicht-Grund, im Admin-Audit-Log.
	mux.Handle("PUT /api/bestellungen/{id}/mittel", s.RequirePermission("create_orders")(s.bestellung().UpdateBestellungMittelHandler()))
}
