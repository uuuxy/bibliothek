package api

import (
	"net/http"
)

func (s *Server) registriereOeffentlicheRouten(mux *http.ServeMux) {
	// ── Ohne Anmeldung ──
	mux.HandleFunc("GET /api/public/opac/suche", s.portal().OPACSuche())
	mux.HandleFunc("GET /api/monitor/slides", s.portal().MonitorFolien())

	// Der Bestätigungs-Link an den Lieferanten. Keine Anmeldung: Der Token aus dem Link ist der
	// Ausweis und öffnet nur diese eine Bestellung. Das Bestätigen ist die einzige schreibende
	// Route ohne Anmeldung; Zuschnitt und Obergrenze des Schadens stehen in
	// api/bestellung/bestellbestaetigung_oeffentlich.go.
	mux.HandleFunc("GET /api/public/bestellung/{token}", s.bestellung().OeffentlicheBestellung())
	mux.HandleFunc("GET /api/public/bestellung/{token}/etiketten/{groesse}", s.bestellung().OeffentlicheEtiketten())
	mux.HandleFunc("POST /api/public/bestellung/{token}/bestaetigen", s.bestellung().OeffentlichBestaetigen())
}
