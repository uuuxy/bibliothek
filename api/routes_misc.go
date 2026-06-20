package api

import (
	"bibliothek/repository"
	"net/http"

	"bibliothek/internal/service"
)

func (s *Server) registerPublicRoutes(mux *http.ServeMux) {
	// ── PUBLIC ENDPOINTS ──
	mux.HandleFunc("GET /api/public/opac/suche", s.PublicCatalogSearchHandler())
	mux.HandleFunc("GET /api/antolin", s.AntolinHandler())
	mux.HandleFunc("GET /api/monitor/slides", s.GetMonitorSlidesHandler())
}

func (s *Server) registerCoreActionRoutes(mux *http.ServeMux, studentRepo repository.StudentRepository, bookRepo repository.BookRepository, loanRepo repository.LoanRepository, loanSvc service.LoanService) {
	// Central Omnibox Action Dispatcher
	actionHandler := s.ActionHandler(studentRepo, bookRepo, loanRepo, loanSvc)
	mux.Handle("POST /api/action", s.RequirePermission("view_students")(actionHandler))
	mux.Handle("POST /api/action/batch", s.RequirePermission("view_students")(s.ActionBatchHandler(studentRepo, bookRepo, loanRepo, loanSvc)))

	// Unified Fuzzy Search
	searchHandler := s.SearchHandler(studentRepo, bookRepo)
	mux.Handle("GET /api/search", s.RequirePermission("view_students")(searchHandler))

	// Inventory
	mux.Handle("POST /api/inventur/start", s.RequirePermission("manage_inventory")(s.InventurStartHandler()))
	mux.Handle("POST /api/inventur/scan", s.RequirePermission("inventory_scan")(s.InventurScanHandler()))
	mux.Handle("POST /api/inventur/finish", s.RequirePermission("manage_inventory")(s.InventurFinishHandler()))

	// Smart Scanner (Tresen-Weiche)
	mux.Handle("GET /api/scan", s.RequirePermission("view_students")(s.SmartScanHandler()))

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
