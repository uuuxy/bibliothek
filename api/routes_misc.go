package api

import (
	"net/http"
	"bibliothek/repository"
)

func (s *Server) registerPublicRoutes(mux *http.ServeMux) {
	// ── PUBLIC ENDPOINTS ──
	mux.HandleFunc("GET /api/public/opac/suche", s.PublicCatalogSearchHandler())
	mux.HandleFunc("GET /api/antolin", s.AntolinHandler())
	mux.HandleFunc("GET /api/monitor/slides", s.GetMonitorSlidesHandler())
}

func (s *Server) registerCoreActionRoutes(mux *http.ServeMux, studentRepo repository.StudentRepository, bookRepo repository.BookRepository, loanRepo repository.LoanRepository) {
	// Central Omnibox Action Dispatcher
	actionHandler := s.ActionHandler(studentRepo, bookRepo, loanRepo)
	mux.Handle("POST /api/action", s.RequirePermission("view_students")(actionHandler))

	// Unified Fuzzy Search
	searchHandler := s.SearchHandler(studentRepo, bookRepo)
	mux.Handle("GET /api/search", s.RequirePermission("view_students")(searchHandler))

	// Inventory
	mux.Handle("POST /api/inventur/scan", s.RequirePermission("inventory_scan")(s.ScanInventoryHandler()))

	// Demo Dashboards
	adminDashboard := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Access granted: Welcome to the Admin Dashboard."))
	})
	mux.Handle("GET /admin/dashboard", s.Auth.RequireRoles("admin")(adminDashboard))

	teacherZone := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Access granted: Welcome to the Teacher Zone."))
	})
	mux.Handle("GET /teacher/dashboard", s.Auth.RequireRoles("admin", "lehrer")(teacherZone))
}
