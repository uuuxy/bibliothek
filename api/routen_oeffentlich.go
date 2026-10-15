package api

import (
	"net/http"
)

func (s *Server) registriereOeffentlicheRouten(mux *http.ServeMux) {
	// ── PUBLIC ENDPOINTS ──
	mux.HandleFunc("GET /api/public/opac/suche", s.portal().OPACSuche())
	mux.HandleFunc("GET /api/monitor/slides", s.portal().MonitorFolien())

	// Bestätigungs-Link an den Lieferanten (Migration 063). Kein Login: Der Token aus dem
	// Link ist der Ausweis, und er öffnet ausschließlich diese eine Bestellung. Das
	// Bestätigen ist bewusst die einzige schreibende Route ohne Anmeldung im System —
	// Zuschnitt und Schadensobergrenze stehen in api/bestellung/bestellbestaetigung_oeffentlich.go.
	mux.HandleFunc("GET /api/public/bestellung/{token}", s.bestellung().OeffentlicheBestellung())
	mux.HandleFunc("GET /api/public/bestellung/{token}/etiketten/{groesse}", s.bestellung().OeffentlicheEtiketten())
	mux.HandleFunc("POST /api/public/bestellung/{token}/bestaetigen", s.bestellung().OeffentlichBestaetigen())
}
