package api

import (
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/inventur"
	"bibliothek/repository"
	"bibliothek/sse"
)

// Server wraps the core application dependencies: database, auth, and real-time streaming.
type Server struct {
	DB           *db.Database
	Auth         *auth.Authenticator
	Broker       *sse.Broker
	CookieSecure bool
}

// NewServer constructs and returns a new Server instance.
func NewServer(database *db.Database, authenticator *auth.Authenticator, broker *sse.Broker, cookieSecure bool) *Server {
	return &Server{
		DB:           database,
		Auth:         authenticator,
		Broker:       broker,
		CookieSecure: cookieSecure,
	}
}

// Routes configures the HTTP multiplexer using modern Go (1.22+) enhanced routing patterns.
// Maps endpoints to their handlers and wraps protected endpoints in RBAC middleware.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// Instantiate repositories for database queries
	studentRepo := repository.NewStudentRepository(s.DB.Pool)
	bookRepo := repository.NewBookRepository(s.DB.Pool)
	loanRepo := repository.NewLoanRepository(s.DB.Pool)
	auditRepo := repository.NewAuditRepository(s.DB.Pool)

	// Initialize Inventur sub-module handlers
	_ = os.MkdirAll("uploads", 0755)
	invRepo := inventur.NewBookRepository(s.DB.Pool)
	invMeta := inventur.NeuerMetadatenClient()
	
	// Fail hard if JWT_SECRET is missing during route initialization
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("FATAL: JWT_SECRET environment variable is required and cannot be empty")
	}

	invHandler := inventur.NewAPIHandler(inventur.APIHandlerConfig{
		Repo:      invRepo,
		Metadaten: invMeta,
		JWTSecret: jwtSecret,
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
	mux.Handle("/api/auth/status", invHandler)

	// Public Endpoints
	mux.HandleFunc("POST /login/barcode", auth.LoginHandler(s.DB.Pool, s.Auth, s.CookieSecure))
	
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	})

	// Logout Endpoint: server-side JWT-Cookie invalidation with expiration in the past
	mux.HandleFunc("POST /api/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    "",
			Path:     "/",
			Expires:  time.Unix(0, 0),
			HttpOnly: true,
			Secure:   s.CookieSecure,
			SameSite: http.SameSiteLaxMode,
		})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"erfolgreich abgemeldet"}`))
	})

	// Protected Endpoints (RBAC Middleware checking roles: admin, lehrer, mitarbeiter)
	
	// Central Omnibox Action Dispatcher
	actionHandler := s.ActionHandler(studentRepo, bookRepo, loanRepo)
	mux.Handle("POST /api/action", s.Auth.RequireRoles(auth.RoleAdmin, auth.RoleLehrer, auth.RoleMitarbeiter)(actionHandler))

	// LUSD CSV Student Import (Accessible by Admin and Mitarbeiter)
	importHandler := s.ImportStudentsHandler()
	mux.Handle("POST /api/import/students", s.Auth.RequireRoles(auth.RoleAdmin, auth.RoleMitarbeiter)(importHandler))

	// LUSD CSV Sync (Accessible by Admin and Mitarbeiter)
	mux.Handle("POST /api/import/lusd", s.Auth.RequireRoles(auth.RoleAdmin, auth.RoleMitarbeiter)(s.ImportLUSDHandler()))

	// Upload student webcam passport photo (Accessible by Admin, Mitarbeiter, and Lehrer)
	mux.Handle("POST /api/schueler/{id}/photo", s.Auth.RequireRoles(auth.RoleAdmin, auth.RoleMitarbeiter, auth.RoleLehrer)(s.UploadStudentPhotoHandler()))

	// Get student profile (Accessible by Admin, Mitarbeiter, and Lehrer)
	mux.Handle("GET /api/schueler/{id}", s.Auth.RequireRoles(auth.RoleAdmin, auth.RoleMitarbeiter, auth.RoleLehrer)(s.GetStudentProfileHandler(studentRepo)))

	// List distinct student classes (Accessible by Admin, Mitarbeiter, and Lehrer)
	mux.Handle("GET /api/klassen", s.Auth.RequireRoles(auth.RoleAdmin, auth.RoleMitarbeiter, auth.RoleLehrer)(s.GetClassesHandler()))

	// List or search students (Accessible by Admin, Mitarbeiter, and Lehrer)
	mux.Handle("GET /api/schueler", s.Auth.RequireRoles(auth.RoleAdmin, auth.RoleMitarbeiter, auth.RoleLehrer)(s.ListStudentsHandler()))

	// Parents damage letters PDF Generator (Accessible by Admin and Mitarbeiter)
	pdfHandler := s.GenerateDamagePDFHandler()
	mux.Handle("GET /api/schadensfaelle/{id}/pdf", s.Auth.RequireRoles(auth.RoleAdmin, auth.RoleMitarbeiter)(pdfHandler))

	// Get copies of a book title (Accessible by Admin, Mitarbeiter, and Lehrer)
	mux.Handle("GET /api/buecher/titel/{id}/exemplare", s.Auth.RequireRoles(auth.RoleAdmin, auth.RoleMitarbeiter, auth.RoleLehrer)(s.GetTitleCopiesHandler()))

	// Update copy damage note (Accessible by Admin and Mitarbeiter)
	mux.Handle("POST /api/buecher/exemplare/{id}/schadensnotiz", s.Auth.RequireRoles(auth.RoleAdmin, auth.RoleMitarbeiter)(s.UpdateDamageNoteHandler()))

	// Delete physical copy (Accessible by Admin and Mitarbeiter)
	mux.Handle("DELETE /api/buecher/exemplare/{id}", s.Auth.RequireRoles(auth.RoleAdmin, auth.RoleMitarbeiter)(s.DeleteCopyHandler()))

	// Delete book title (Accessible by Admin and Mitarbeiter)
	mux.Handle("DELETE /api/buecher/titel/{id}", s.Auth.RequireRoles(auth.RoleAdmin, auth.RoleMitarbeiter)(s.DeleteTitleHandler(auditRepo)))

	// Delete user (Accessible by Admin)
	mux.Handle("DELETE /api/benutzer/{id}", s.Auth.RequireRoles(auth.RoleAdmin)(s.DeleteUserHandler(auditRepo)))

	// View audit logs (Accessible by Admin)
	mux.Handle("GET /api/audit", s.Auth.RequireRoles(auth.RoleAdmin)(s.GetAuditLogsHandler()))

	// Get graduates live list (Accessible by Admin and Mitarbeiter)
	mux.Handle("GET /api/abgaenger", s.Auth.RequireRoles(auth.RoleAdmin, auth.RoleMitarbeiter)(s.GetGraduatesHandler()))

	// Get reorder lists (Accessible by Admin and Mitarbeiter)
	mux.Handle("GET /api/bestellungen", s.Auth.RequireRoles(auth.RoleAdmin, auth.RoleMitarbeiter)(s.GetReordersHandler()))

	// Export reorder list as PDF (Accessible by Admin and Mitarbeiter)
	mux.Handle("GET /api/bestellungen/pdf", s.Auth.RequireRoles(auth.RoleAdmin, auth.RoleMitarbeiter)(s.ExportReordersPDFHandler()))

	// Scan items during active inventory (Accessible by Admin and Mitarbeiter)
	mux.Handle("POST /api/inventur/scan", s.Auth.RequireRoles(auth.RoleAdmin, auth.RoleMitarbeiter)(s.ScanInventoryHandler()))

	// Get system statistics (Accessible by Admin and Mitarbeiter)
	mux.Handle("GET /api/statistiken", s.Auth.RequireRoles(auth.RoleAdmin, auth.RoleMitarbeiter)(s.GetStatisticsHandler()))

	// Generate PNG/SVG Barcodes (Accessible by all staff roles)
	mux.Handle("GET /api/barcode", s.Auth.RequireRoles(auth.RoleAdmin, auth.RoleLehrer, auth.RoleMitarbeiter)(s.BarcodeHandler()))

	// Create supplier order and download vorab-barcode labels PDF (Accessible by Admin and Mitarbeiter)
	mux.Handle("POST /api/lieferanten/bestellen", s.Auth.RequireRoles(auth.RoleAdmin, auth.RoleMitarbeiter)(s.SupplierOrderHandler()))

	// One-click supplier order sending via email (Accessible by Admin and Mitarbeiter)
	mux.Handle("POST /api/bestellung/senden", s.Auth.RequireRoles(auth.RoleAdmin, auth.RoleMitarbeiter)(s.SendOrderMailHandler()))

	// Release all pending supplier orders (Accessible by Admin and Mitarbeiter)
	mux.Handle("POST /api/bestellungen/freigeben", s.Auth.RequireRoles(auth.RoleAdmin, auth.RoleMitarbeiter)(s.ReleaseOrdersHandler()))

	// Real-time Event Stream (accessible by all authorized staff roles)
	sseHandler := s.Broker.Handler()
	mux.Handle("GET /events", s.Auth.RequireRoles(auth.RoleAdmin, auth.RoleLehrer, auth.RoleMitarbeiter)(sseHandler))

	// Demo Admin-only Endpoint
	adminDashboard := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Access granted: Welcome to the Admin Dashboard."))
	})
	mux.Handle("GET /admin/dashboard", s.Auth.RequireRoles(auth.RoleAdmin)(adminDashboard))

	// Demo Teacher/Admin Endpoint
	teacherZone := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Access granted: Welcome to the Teacher Zone."))
	})
	mux.Handle("GET /teacher/dashboard", s.Auth.RequireRoles(auth.RoleAdmin, auth.RoleLehrer)(teacherZone))

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

	// Wrap mux in logging and rate limiting middleware
	rateLimiter := RateLimitMiddleware(50)
	globalHandler := rateLimiter(mux)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log incoming request without exposing IP addresses (.RemoteAddr stripped for DSGVO)
		log.Printf("Incoming Request: %s %s", r.Method, r.URL.Path)
		globalHandler.ServeHTTP(w, r)
	})
}
