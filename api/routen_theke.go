package api

import (
	"bibliothek/repository"
	"net/http"

	"bibliothek/internal/service"
)

func (s *Server) registriereThekenUndInventurRouten(mux *http.ServeMux, leserAbfragen repository.LeserAbfragen, katalogAbfragen repository.KatalogAbfragen, omniboxDienst service.OmniboxDienst, nachbuchDienst service.NachbuchDienst) {
	// Central Omnibox Action Dispatcher.
	// perform_actions (nicht view_students): das ist die Kiosk-/Terminal-Kernfunktion
	// (Ausleihe/Rückgabe/Scan/Suche). So kann die Helfer-Rolle am Terminal arbeiten,
	// OHNE die breiten view_students-Rechte (Schülerlisten, Mahnwesen, Bulk-Mahndruck).
	eingabeTuer := s.theke().EingabeVerarbeiten(omniboxDienst)
	mux.Handle("POST /api/action", s.verlangeRecht("perform_actions")(eingabeTuer))
	// Nachbuchen der Offline-Warteschlange (Stufe 2): dieselbe Theken-Rolle wie der Scan.
	mux.Handle("POST /api/action/nachbuchen", s.verlangeRecht("perform_actions")(s.theke().WarteschlangeNachbuchen(nachbuchDienst)))
	// Buch-Barcodes für die Offline-Einordnung (Stufe 2): nur Nummern, keine Personendaten.
	mux.Handle("GET /api/action/buchbarcodes", s.verlangeRecht("perform_actions")(s.theke().Buchbarcodes()))
	// Nachbuch-Meldungen (Migration 117): Liste und Quittieren nur mit view_students — die
	// Zeilen nennen Ausleiher und Vorbesitzer (entschieden am 13.09.2026); der Zähler
	// fürs Band ist eine Zahl und darf jeder Theken-Rolle gehören.
	mux.Handle("GET /api/action/nachbuch-meldungen", s.verlangeRecht("view_students")(s.theke().NachbuchMeldungen()))
	mux.Handle("GET /api/action/nachbuch-meldungen/anzahl", s.verlangeRecht("perform_actions")(s.theke().NachbuchMeldungenAnzahl()))
	mux.Handle("POST /api/action/nachbuch-meldungen/{id}/quittieren", s.verlangeRecht("view_students")(s.theke().NachbuchMeldungQuittieren()))

	// Unified Fuzzy Search
	sucheTuer := s.theke().Suche(leserAbfragen, katalogAbfragen)
	mux.Handle("GET /api/search", s.verlangeRecht("perform_actions")(sucheTuer))

	// Inventory
	mux.Handle("GET /api/inventur/sessions", s.verlangeRecht("inventory_scan")(s.inventurlauf().LaufendeInventuren()))
	mux.Handle("POST /api/inventur/start", s.verlangeRecht("manage_inventory")(s.inventurlauf().InventurStarten()))
	mux.Handle("POST /api/inventur/scan", s.verlangeRecht("inventory_scan")(s.inventurlauf().ScanVerbuchen()))
	mux.Handle("POST /api/inventur/finish", s.verlangeRecht("manage_inventory")(s.inventurlauf().InventurAbschliessen()))
	// Der Fehlbestand einer bereits abgeschlossenen Inventur — zum Nachsuchen und Ausdrucken.
	// Beide gehören zusammen: /abgeschlossen liefert die session_id, /fehlbestand die
	// Liste dazu. Ohne die erste war die zweite aus der Oberfläche nicht erreichbar.
	mux.Handle("GET /api/inventur/abgeschlossen", s.verlangeRecht("manage_inventory")(s.inventurlauf().AbgeschlosseneInventuren()))
	mux.Handle("GET /api/inventur/fehlbestand", s.verlangeRecht("manage_inventory")(s.inventurlauf().Fehlbestand()))
	mux.Handle("POST /api/inventur/abort", s.verlangeRecht("manage_inventory")(s.inventurlauf().InventurVerwerfen()))
	// Handlungsspielraum für den Fehlbestandsbericht: wiedergefunden oder endgültig weg.
	mux.Handle("POST /api/buecher/exemplare/{id}/gefunden", s.verlangeRecht("manage_inventory")(s.inventurlauf().FundMelden()))
	mux.Handle("POST /api/buecher/exemplare/verlust-endgueltig-loeschen", s.verlangeRecht("manage_inventory")(s.inventurlauf().VerlusteEndgueltigLoeschen()))

	// Smart Scanner (Tresen-Weiche) — Teil der Kiosk-Kernfunktion, siehe /api/action.

	// Hier standen bis zum 04.08.2026 zwei "Demo Dashboards": GET /admin/dashboard und
	// GET /teacher/dashboard, die nichts taten außer einen englischen Satz auszugeben
	// ("Access granted: Welcome to the Admin Dashboard."). Die Oberfläche ist eine SPA —
	// ihre Bildschirme leben unter / im Frontend, nicht hinter eigenen Server-Routen.
	// Die beiden waren korrekt rollengeschützt und damit harmlos, aber sie waren
	// Angriffsfläche ohne Gegenwert und der einzige Ort, an dem die Autorisierung noch
	// über Rollennamen statt über das konfigurierbare role_permissions-System lief.
	// Ersatzlos entfernt.
}
