package api

import (
	"bibliothek/repository"
	"net/http"

	"bibliothek/internal/service"
)

func (s *Server) registerPublicRoutes(mux *http.ServeMux) {
	// ── PUBLIC ENDPOINTS ──
	mux.HandleFunc("GET /api/public/opac/suche", s.portal().PublicCatalogSearchHandler())
	mux.HandleFunc("GET /api/monitor/slides", s.portal().GetMonitorSlidesHandler())

	// Bestätigungs-Link an den Lieferanten (Migration 063). Kein Login: Der Token aus dem
	// Link ist der Ausweis, und er öffnet ausschließlich diese eine Bestellung. Das
	// Bestätigen ist bewusst die einzige schreibende Route ohne Anmeldung im System —
	// Zuschnitt und Schadensobergrenze stehen in api/bestellung/bestellbestaetigung_public.go.
	mux.HandleFunc("GET /api/public/bestellung/{token}", s.bestellung().OeffentlicheBestellungHandler())
	mux.HandleFunc("GET /api/public/bestellung/{token}/etiketten/{groesse}", s.bestellung().OeffentlicheEtikettenHandler())
	mux.HandleFunc("POST /api/public/bestellung/{token}/bestaetigen", s.bestellung().OeffentlichBestaetigenHandler())
}

func (s *Server) registerCoreActionRoutes(mux *http.ServeMux, studentRepo repository.StudentRepository, bookRepo repository.KatalogAbfragen, omniboxSvc service.OmniboxService, nachbuchSvc service.NachbuchService) {
	// Central Omnibox Action Dispatcher.
	// perform_actions (nicht view_students): das ist die Kiosk-/Terminal-Kernfunktion
	// (Ausleihe/Rückgabe/Scan/Suche). So kann die Helfer-Rolle am Terminal arbeiten,
	// OHNE die breiten view_students-Rechte (Schülerlisten, Mahnwesen, Bulk-Mahndruck).
	actionHandler := s.theke().ActionHandler(omniboxSvc)
	mux.Handle("POST /api/action", s.RequirePermission("perform_actions")(actionHandler))
	// Nachbuchen der Offline-Warteschlange (Stufe 2): dieselbe Theken-Rolle wie der Scan.
	mux.Handle("POST /api/action/nachbuchen", s.RequirePermission("perform_actions")(s.theke().NachbuchenHandler(nachbuchSvc)))
	// Buch-Barcodes für die Offline-Einordnung (Stufe 2): nur Nummern, keine Personendaten.
	mux.Handle("GET /api/action/buchbarcodes", s.RequirePermission("perform_actions")(s.theke().BuchbarcodesHandler()))
	// Nachbuch-Meldungen (Migration 117): Liste und Quittieren nur mit view_students — die
	// Zeilen nennen Ausleiher und Vorbesitzer (entschieden am 13.09.2026); der Zähler
	// fürs Band ist eine Zahl und darf jeder Theken-Rolle gehören.
	mux.Handle("GET /api/action/nachbuch-meldungen", s.RequirePermission("view_students")(s.theke().NachbuchMeldungenListeHandler()))
	mux.Handle("GET /api/action/nachbuch-meldungen/anzahl", s.RequirePermission("perform_actions")(s.theke().NachbuchMeldungenAnzahlHandler()))
	mux.Handle("POST /api/action/nachbuch-meldungen/{id}/quittieren", s.RequirePermission("view_students")(s.theke().NachbuchMeldungQuittierenHandler()))

	// Unified Fuzzy Search
	searchHandler := s.theke().SearchHandler(studentRepo, bookRepo)
	mux.Handle("GET /api/search", s.RequirePermission("perform_actions")(searchHandler))

	// Inventory
	mux.Handle("GET /api/inventur/sessions", s.RequirePermission("inventory_scan")(s.inventurlauf().ListInventurSessionsHandler()))
	mux.Handle("POST /api/inventur/start", s.RequirePermission("manage_inventory")(s.inventurlauf().InventurStartHandler()))
	mux.Handle("POST /api/inventur/scan", s.RequirePermission("inventory_scan")(s.inventurlauf().InventurScanHandler()))
	mux.Handle("POST /api/inventur/finish", s.RequirePermission("manage_inventory")(s.inventurlauf().InventurFinishHandler()))
	// Der Fehlbestand einer bereits abgeschlossenen Inventur — zum Nachsuchen und Ausdrucken.
	// Beide gehören zusammen: /abgeschlossen liefert die session_id, /fehlbestand die
	// Liste dazu. Ohne die erste war die zweite aus der Oberfläche nicht erreichbar.
	mux.Handle("GET /api/inventur/abgeschlossen", s.RequirePermission("manage_inventory")(s.inventurlauf().ListAbgeschlosseneInventurenHandler()))
	mux.Handle("GET /api/inventur/fehlbestand", s.RequirePermission("manage_inventory")(s.inventurlauf().InventurFehlbestandHandler()))
	mux.Handle("POST /api/inventur/abort", s.RequirePermission("manage_inventory")(s.inventurlauf().InventurAbortHandler()))
	// Handlungsspielraum für den Fehlbestandsbericht: wiedergefunden oder endgültig weg.
	mux.Handle("POST /api/buecher/exemplare/{id}/gefunden", s.RequirePermission("manage_inventory")(s.inventurlauf().InventurVerlustGefundenHandler()))
	mux.Handle("POST /api/buecher/exemplare/verlust-endgueltig-loeschen", s.RequirePermission("manage_inventory")(s.inventurlauf().InventurVerlusteLoeschenHandler()))

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
