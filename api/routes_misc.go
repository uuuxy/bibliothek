package api

import (
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/inventur"
	"bibliothek/pkg/httpresp"
	"bibliothek/repository"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"

	httpSwagger "github.com/swaggo/http-swagger"

	"bibliothek/internal/service"
)

func (s *Server) registerPublicRoutes(mux *http.ServeMux) {
	// ── PUBLIC ENDPOINTS ──
	mux.HandleFunc("GET /api/public/opac/suche", s.PublicCatalogSearchHandler())
	mux.HandleFunc("GET /api/monitor/slides", s.GetMonitorSlidesHandler())
}

func (s *Server) registerCoreActionRoutes(mux *http.ServeMux, studentRepo repository.StudentRepository, bookRepo repository.BookRepository, omniboxSvc service.OmniboxService) {
	// Central Omnibox Action Dispatcher
	actionHandler := s.ActionHandler(omniboxSvc)
	mux.Handle("POST /api/action", s.RequirePermission("view_students")(actionHandler))
	mux.Handle("POST /api/action/batch", s.RequirePermission("view_students")(s.ActionBatchHandler(omniboxSvc)))

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
		httpresp.Write(w, []byte("Access granted: Welcome to the Admin Dashboard."))
	})
	mux.Handle("GET /admin/dashboard", s.Auth.RequireRoles("admin")(adminDashboard))

	teacherZone := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpresp.Write(w, []byte("Access granted: Welcome to the Teacher Zone."))
	})
	mux.Handle("GET /teacher/dashboard", s.Auth.RequireRoles("admin", "lehrer")(teacherZone))
}

func (s *Server) registerInventurRoutes(mux *http.ServeMux) {

	// Initialize Inventur sub-module handlers
	if err := os.MkdirAll("uploads", 0750); err != nil {
		log.Printf("router: Upload-Verzeichnis konnte nicht angelegt werden: %v", err)
	}
	if err := os.MkdirAll("uploads/fotos", 0750); err != nil {
		log.Printf("router: Foto-Verzeichnis konnte nicht angelegt werden: %v", err)
	}
	invRepo := inventur.NewBookRepository(s.DB.Pool)
	invMeta := inventur.NeuerMetadatenClient()

	invHandler := inventur.NewAPIHandler(inventur.APIHandlerConfig{
		Repo:             invRepo,
		Metadaten:        invMeta,
		RequireViewBooks: s.RequirePermission("view_books"),
		RequireEditBooks: s.RequirePermission("edit_books"),
	})

	// Mount Inventur routes
	mux.Handle("/api/books", invHandler)
	mux.Handle("/api/books/", invHandler)
	mux.Handle("/api/class-books", invHandler)
	mux.Handle("/api/lookup/", invHandler)
	mux.Handle("/api/subjects", invHandler)
	mux.Handle("/api/admin", invHandler)
	mux.Handle("/api/admin/", invHandler)
	mux.Handle("/uploads/", invHandler)

}

func (s *Server) registerAuthRoutes(mux *http.ServeMux) {

	// Public Endpoints
	mux.Handle("POST /login", AuthRateLimitMiddleware(http.HandlerFunc(auth.LoginHandler(s.DB.Pool, s.Auth, s.CookieSecure))))

	// Image Caching (Public)
	mux.HandleFunc("GET /api/images/cover", s.ServeCoverImageHandler())

	// CSRF token bootstrap (public): lets API clients obtain a token + cookie before any mutation
	mux.HandleFunc("GET /api/csrf-token", s.CSRFTokenHandler())

	// Token refresh (sliding window) — exempt from CSRF via middleware config
	mux.HandleFunc("POST /api/auth/refresh", auth.RefreshTokenHandler(s.Auth, s.CookieSecure))

	// Session-Restore für den SPA-Boot: prüft Cookie + DB-Zustand, liefert Login-Shape
	mux.HandleFunc("GET /api/auth/me", auth.MeHandler(s.DB.Pool, s.Auth))

	// Logout — blacklists the current token and clears the session cookie
	mux.HandleFunc("POST /api/auth/logout", s.logoutHandler())

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := s.DB.Pool.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			httpresp.Write(w, []byte(`{"status":"unhealthy","error":"database unreachable"}`))
			return
		}
		httpresp.Write(w, []byte(`{"status":"healthy"}`))
	})

}

func (s *Server) registerImportRoutes(mux *http.ServeMux) {

	// LITTERA CSV Import (Accessible by Admin)
	mux.Handle("POST /api/import/littera", s.RequirePermission("manage_inventory")(s.LitteraImportHandler()))
	mux.Handle("POST /api/admin/import-bestand", s.RequirePermission("manage_inventory")(http.HandlerFunc(s.BestandImportHandler)))
	mux.Handle("POST /api/admin/sync-covers", s.RequirePermission("manage_inventory")(SyncCoversHandler(s.DB.Pool)))

}

func (s *Server) registerSwaggerRoutes(mux *http.ServeMux) {

	// Swagger interactive documentation (Only accessible in local/development mode)
	if os.Getenv("APP_ENV") == "local" || os.Getenv("APP_ENV") == "development" {
		mux.Handle("GET /swagger/", httpSwagger.Handler(
			httpSwagger.URL("/swagger/doc.json"),
		))
		mux.HandleFunc("GET /swagger", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
		})
	}

}

func (s *Server) registerFrontendRoutes(mux *http.ServeMux) {

	// Intercept missing favicon.ico to prevent fallback to index.html and 404 errors
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	// Serve Svelte frontend static assets from build directory
	fs := http.FileServer(http.Dir("./frontend/dist"))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			apierrors.SendHTTPError(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
			return
		}
		path := filepath.Join("./frontend/dist", r.URL.Path)
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			http.ServeFile(w, r, "./frontend/dist/index.html")
			return
		}
		fs.ServeHTTP(w, r)
	})

}
