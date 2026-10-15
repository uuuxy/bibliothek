package api

import (
	"net/http"

	"bibliothek/internal/service"
	"bibliothek/repository"
)

func (s *Server) registriereBestellRouten(mux *http.ServeMux, bestellDienst *service.BestellDienst) {
	// Bestellungen & Lieferanten
	// Anzeige-Regeln des Bestellwesens (z. B. ob mit Preisen gearbeitet wird).
	// view_orders statt manage_users: Wer bestellen darf, muss dafuer keine
	// Benutzerverwaltungsrechte bekommen (siehe api/bestellung/bestellkonfiguration.go).
	mux.Handle("GET /api/bestellungen/konfiguration", s.verlangeRecht("view_orders")(s.bestellung().BestellKonfiguration(
		repository.NewEinstellungsAbfragen(s.DB.Pool), repository.NewLieferantenAbfragen(s.DB.Pool))))
	mux.Handle("GET /api/bestellungen", s.verlangeRecht("view_orders")(s.bestellung().Bestellbedarf()))
	mux.Handle("GET /api/bestellungen/pdf", s.verlangeRecht("view_orders")(s.bestellung().BestellbedarfPDF()))
	mux.Handle("GET /api/bestellhistorie", s.verlangeRecht("view_orders")(s.bestellung().Bestellhistorie()))
	mux.Handle("GET /api/bestellhistorie/uebersicht", s.verlangeRecht("view_orders")(s.bestellung().BestellhistorieUebersicht()))
	mux.Handle("GET /api/bestellhistorie/bericht", s.verlangeRecht("view_orders")(s.bestellung().BestellberichtPDF()))
	// Nach den beiden festen Pfaden eingetragen, gilt aber unabhängig von der
	// Reihenfolge: Der ServeMux von Go 1.22 bevorzugt das genauere Muster, /uebersicht
	// und /bericht gewinnen also gegen {id}.
	mux.Handle("GET /api/bestellhistorie/{id}", s.verlangeRecht("view_orders")(
		s.bestellung().Bestelldetail(repository.NewBestelldetailAbfragen(s.DB.Pool))))
	mux.Handle("GET /api/lieferanten", s.verlangeRecht("view_orders")(s.bestellung().Lieferanten()))
	mux.Handle("POST /api/lieferanten", s.verlangeRecht("create_orders")(s.bestellung().LieferantAnlegen()))
	mux.Handle("PUT /api/lieferanten/{id}", s.verlangeRecht("create_orders")(s.bestellung().LieferantAendern()))
	mux.Handle("DELETE /api/lieferanten/{id}", s.verlangeRecht("create_orders")(s.bestellung().LieferantLoeschen()))
	mux.Handle("POST /api/bestellungen", s.verlangeRecht("create_orders")(s.bestellung().BestellungAnlegen(bestellDienst)))
	mux.Handle("GET /api/bestellungen/zulauf", s.verlangeRecht("view_orders")(s.bestellung().Zulauf()))
	mux.Handle("POST /api/bestellungen/suche", s.verlangeRecht("view_orders")(s.bestellung().BestellSuche()))
	mux.Handle("POST /api/bestellungen/bulk-receive", s.verlangeRecht("create_orders")(s.bestellung().WareneingangBuchen()))
	mux.Handle("PUT /api/bestellungen/{id}/bestaetigen", s.verlangeRecht("create_orders")(s.bestellung().BestellungBestaetigen()))
	mux.Handle("PUT /api/bestellungen/{id}/bestaetigungs-link", s.verlangeRecht("create_orders")(s.bestellung().BestaetigungsLinkErneuern()))
	// Erneuter Versand der Bestellmail nach einem gescheiterten Versuch (Migration 161).
	mux.Handle("POST /api/bestellungen/{id}/mail", s.verlangeRecht("create_orders")(s.bestellung().BestellmailErneutSenden()))
	// Derselbe Vermerk ohne Versand: Der Händler hat die Bestellung auf anderem Weg erhalten.
	mux.Handle("DELETE /api/bestellungen/{id}/mail", s.verlangeRecht("create_orders")(s.bestellung().BestellmailVermerkNehmen()))
	// Rückweg für den Topf (Migration 109): mit Pflicht-Grund, im Admin-Audit-Log.
	mux.Handle("PUT /api/bestellungen/{id}/mittel", s.verlangeRecht("create_orders")(s.bestellung().BestellungMittelKorrigieren()))
}
