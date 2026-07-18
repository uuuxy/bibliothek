package api

import (
	"bibliothek/pkg/httpresp"
	"bibliothek/repository"
	"net/http"

	"bibliothek/internal/service"
)

func (s *Server) registerPublicRoutes(mux *http.ServeMux) {
	// ── PUBLIC ENDPOINTS ──
	mux.HandleFunc("GET /api/public/opac/suche", s.PublicCatalogSearchHandler())
	mux.HandleFunc("GET /api/monitor/slides", s.GetMonitorSlidesHandler())
}

func (s *Server) registerCoreActionRoutes(mux *http.ServeMux, studentRepo repository.StudentRepository, bookRepo repository.BookRepository, omniboxSvc service.OmniboxService) {
	// Central Omnibox Action Dispatcher.
	// perform_actions (nicht view_students): das ist die Kiosk-/Terminal-Kernfunktion
	// (Ausleihe/Rückgabe/Scan/Suche). So kann die Helfer-Rolle am Terminal arbeiten,
	// OHNE die breiten view_students-Rechte (Schülerlisten, Mahnwesen, Bulk-Mahndruck).
	actionHandler := s.ActionHandler(omniboxSvc)
	mux.Handle("POST /api/action", s.RequirePermission("perform_actions")(actionHandler))
	mux.Handle("POST /api/action/batch", s.RequirePermission("perform_actions")(s.ActionBatchHandler(omniboxSvc)))

	// Unified Fuzzy Search
	searchHandler := s.SearchHandler(studentRepo, bookRepo)
	mux.Handle("GET /api/search", s.RequirePermission("perform_actions")(searchHandler))

	// Inventory
	mux.Handle("GET /api/inventur/sessions", s.RequirePermission("inventory_scan")(s.ListInventurSessionsHandler()))
	mux.Handle("POST /api/inventur/start", s.RequirePermission("manage_inventory")(s.InventurStartHandler()))
	mux.Handle("POST /api/inventur/scan", s.RequirePermission("inventory_scan")(s.InventurScanHandler()))
	mux.Handle("POST /api/inventur/finish", s.RequirePermission("manage_inventory")(s.InventurFinishHandler()))
	mux.Handle("POST /api/inventur/abort", s.RequirePermission("manage_inventory")(s.InventurAbortHandler()))

	// Smart Scanner (Tresen-Weiche) — Teil der Kiosk-Kernfunktion, siehe /api/action.
	mux.Handle("GET /api/scan", s.RequirePermission("perform_actions")(s.SmartScanHandler()))

	// Demo Dashboards
	adminDashboard := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpresp.Write(w, []byte("Access granted: Welcome to the Admin Dashboard."))
	})
	mux.Handle("GET /admin/dashboard", s.Auth.RequireRoles("admin")(adminDashboard))

	teacherZone := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpresp.Write(w, []byte("Access granted: Welcome to the Teacher Zone."))
	})
	mux.Handle("GET /teacher/dashboard", s.Auth.RequireRoles("admin", "lehrer")(teacherZone))
}
