package api

import (
	"bibliothek/repository"
	"net/http"

	"bibliothek/internal/service"
)

func (s *Server) registriereThekenUndInventurRouten(mux *http.ServeMux, leserAbfragen *repository.LeserAbfragen, katalogAbfragen *repository.KatalogAbfragen, omniboxDienst *service.OmniboxDienst, nachbuchDienst *service.NachbuchDienst, ausleihDienst *service.AusleihDienst) {
	// Die Eingabe der Omnibox: perform_actions, nicht view_students, weil das die Kernfunktion
	// der Theke ist (Ausleihe, Rückgabe, Scan, Suche). So arbeitet die Rolle Helfer an der Theke
	// ohne die breiten Rechte von view_students (Schülerlisten, Mahnwesen, Sammeldruck der
	// Mahnungen).
	eingabeTuer := s.theke().EingabeVerarbeiten(omniboxDienst)
	mux.Handle("POST /api/action", s.verlangeRecht("perform_actions")(eingabeTuer))
	// Nachbuchen der Warteschlange aus dem Offline-Betrieb: dieselbe Rolle wie der Scan.
	mux.Handle("POST /api/action/nachbuchen", s.verlangeRecht("perform_actions")(s.theke().WarteschlangeNachbuchen(nachbuchDienst)))
	// Die Barcodes der Bücher für die Einordnung im Offline-Betrieb: nur Nummern, keine
	// Personendaten.
	mux.Handle("GET /api/action/buchbarcodes", s.verlangeRecht("perform_actions")(s.theke().Buchbarcodes()))
	// Nachbuch-Meldungen: Liste und Quittieren nur mit view_students, weil die Zeilen Ausleiher
	// und Vorbesitzer nennen; der Zähler für das Band ist eine Zahl und darf jeder Rolle der Theke
	// gehören.
	mux.Handle("GET /api/action/nachbuch-meldungen", s.verlangeRecht("view_students")(s.theke().NachbuchMeldungen()))
	mux.Handle("GET /api/action/nachbuch-meldungen/anzahl", s.verlangeRecht("perform_actions")(s.theke().NachbuchMeldungenAnzahl()))
	mux.Handle("POST /api/action/nachbuch-meldungen/{id}/quittieren", s.verlangeRecht("view_students")(s.theke().NachbuchMeldungQuittieren()))

	// Die Suche der Theke über Leser und Titel
	sucheTuer := s.theke().Suche(leserAbfragen, katalogAbfragen)
	mux.Handle("GET /api/search", s.verlangeRecht("perform_actions")(sucheTuer))

	// Inventur
	mux.Handle("GET /api/inventur/sessions", s.verlangeRecht("inventory_scan")(s.inventurlauf().LaufendeInventuren()))
	mux.Handle("POST /api/inventur/start", s.verlangeRecht("manage_inventory")(s.inventurlauf().InventurStarten()))
	mux.Handle("POST /api/inventur/scan", s.verlangeRecht("inventory_scan")(s.inventurlauf().ScanVerbuchen(ausleihDienst)))
	mux.Handle("POST /api/inventur/finish", s.verlangeRecht("manage_inventory")(s.inventurlauf().InventurAbschliessen()))
	// Der Fehlbestand einer bereits abgeschlossenen Inventur — zum Nachsuchen und Ausdrucken.
	// Beide gehören zusammen: /abgeschlossen liefert die session_id, /fehlbestand die
	// Liste dazu. Ohne die erste wäre die zweite aus der Oberfläche nicht erreichbar.
	mux.Handle("GET /api/inventur/abgeschlossen", s.verlangeRecht("manage_inventory")(s.inventurlauf().AbgeschlosseneInventuren()))
	mux.Handle("GET /api/inventur/fehlbestand", s.verlangeRecht("manage_inventory")(s.inventurlauf().Fehlbestand()))
	mux.Handle("POST /api/inventur/abort", s.verlangeRecht("manage_inventory")(s.inventurlauf().InventurVerwerfen()))
	// Handlungsspielraum für den Fehlbestandsbericht: wiedergefunden oder endgültig weg.
	mux.Handle("POST /api/buecher/exemplare/{id}/gefunden", s.verlangeRecht("manage_inventory")(s.inventurlauf().FundMelden()))
	mux.Handle("POST /api/buecher/exemplare/verlust-endgueltig-loeschen", s.verlangeRecht("manage_inventory")(s.inventurlauf().VerlusteEndgueltigLoeschen()))

}
