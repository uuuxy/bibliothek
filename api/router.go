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
	mux.Handle("POST /api/action", s.RequirePermission("view_students")(actionHandler))

	// Unified Fuzzy Search (Accessible by Admin, Mitarbeiter, and Lehrer)
	searchHandler := s.SearchHandler(studentRepo, bookRepo)
	mux.Handle("GET /api/search", s.RequirePermission("view_students")(searchHandler))

	// LUSD CSV Student Import (Accessible by Admin and Mitarbeiter)
	importHandler := s.ImportStudentsHandler()
	mux.Handle("POST /api/import/students", s.RequirePermission("import_students")(importHandler))

	// New LUSD Import (Accessible by Admin only)
	mux.Handle("POST /api/students/import", s.RequirePermission("import_students")(s.ImportStudentsLUSDHandler()))

	// LUSD CSV Sync (Accessible by Admin and Mitarbeiter)
	mux.Handle("POST /api/import/lusd", s.RequirePermission("import_students")(s.ImportLUSDHandler()))

	// Upload student webcam passport photo (Accessible by Admin, Mitarbeiter, and Lehrer)
	mux.Handle("POST /api/schueler/{id}/photo", s.RequirePermission("upload_photos")(s.UploadStudentPhotoHandler()))

	// Get student profile (Accessible by Admin, Mitarbeiter, and Lehrer)
	mux.Handle("GET /api/schueler/{id}", s.RequirePermission("view_students")(s.GetStudentProfileHandler(studentRepo)))

	// Create student (Accessible by Admin and Mitarbeiter)
	mux.Handle("POST /api/schueler", s.RequirePermission("create_students")(s.CreateStudentHandler()))

	// Delete student (Accessible by Admin and Mitarbeiter)
	mux.Handle("DELETE /api/schueler/{id}", s.RequirePermission("delete_students")(s.DeleteStudentHandler()))

	// List distinct student classes (Accessible by Admin, Mitarbeiter, and Lehrer)
	mux.Handle("GET /api/klassen", s.RequirePermission("view_students")(s.GetClassesHandler()))

	// List or search students (Accessible by Admin, Mitarbeiter, and Lehrer)
	mux.Handle("GET /api/schueler", s.RequirePermission("view_students")(s.ListStudentsHandler()))

	// Parents damage letters PDF Generator (Accessible by Admin and Mitarbeiter)
	pdfHandler := s.GenerateDamagePDFHandler()
	mux.Handle("GET /api/schadensfaelle/{id}/pdf", s.RequirePermission("view_students")(pdfHandler))

	// Get copies of a book title (Accessible by Admin, Mitarbeiter, and Lehrer)
	mux.Handle("GET /api/buecher/titel/{id}/exemplare", s.RequirePermission("view_books")(s.GetTitleCopiesHandler()))

	// Update copy damage note (Accessible by Admin and Mitarbeiter)
	mux.Handle("POST /api/buecher/exemplare/{id}/schadensnotiz", s.RequirePermission("edit_books")(s.UpdateDamageNoteHandler()))

	// Delete physical copy (Accessible by Admin and Mitarbeiter)
	mux.Handle("DELETE /api/buecher/exemplare/{id}", s.RequirePermission("delete_books")(s.DeleteCopyHandler()))

	// Delete book title (Accessible by Admin and Mitarbeiter)
	mux.Handle("DELETE /api/buecher/titel/{id}", s.RequirePermission("delete_books")(s.DeleteTitleHandler(auditRepo)))

	// Delete user (Accessible by Admin)
	mux.Handle("DELETE /api/benutzer/{id}", s.RequirePermission("manage_users")(s.DeleteUserHandler(auditRepo)))

	// List users (Accessible by Admin)
	mux.Handle("GET /api/benutzer", s.RequirePermission("manage_users")(s.ListUsersHandler()))

	// Create user (Accessible by Admin)
	mux.Handle("POST /api/benutzer", s.RequirePermission("manage_users")(s.CreateUserHandler()))

	// Update user (Accessible by Admin)
	mux.Handle("PUT /api/benutzer/{id}", s.RequirePermission("manage_users")(s.UpdateUserHandler()))

	// View role permissions settings (Accessible by Admin)
	mux.Handle("GET /api/admin/permissions", s.RequirePermission("manage_users")(s.GetPermissionsHandler()))

	// Update role permission settings (Accessible by Admin)
	mux.Handle("PUT /api/admin/permissions", s.RequirePermission("manage_users")(s.UpdatePermissionsHandler()))

	// View audit logs (Accessible by Admin)
	mux.Handle("GET /api/audit", s.RequirePermission("audit_logs")(s.GetAuditLogsHandler()))

	// Get graduates live list (Accessible by Admin and Mitarbeiter)
	mux.Handle("GET /api/abgaenger", s.RequirePermission("view_graduates")(s.GetGraduatesHandler()))

	// Get reorder lists (Accessible by Admin and Mitarbeiter)
	mux.Handle("GET /api/bestellungen", s.RequirePermission("view_orders")(s.GetReordersHandler()))

	// Export reorder list as PDF (Accessible by Admin and Mitarbeiter)
	mux.Handle("GET /api/bestellungen/pdf", s.RequirePermission("view_orders")(s.ExportReordersPDFHandler()))

	// Scan items during active inventory (Accessible by Admin and Mitarbeiter)
	mux.Handle("POST /api/inventur/scan", s.RequirePermission("inventory_scan")(s.ScanInventoryHandler()))

	// Get system statistics (Accessible by Admin and Mitarbeiter)
	mux.Handle("GET /api/statistiken", s.RequirePermission("view_stats")(s.GetStatisticsHandler()))

	// Generate PNG/SVG Barcodes (Accessible by all staff roles)
	mux.Handle("GET /api/barcode", s.RequirePermission("view_books")(s.BarcodeHandler()))

	// Create supplier order and download vorab-barcode labels PDF (Accessible by Admin and Mitarbeiter)
	mux.Handle("POST /api/lieferanten/bestellen", s.RequirePermission("create_orders")(s.SupplierOrderHandler()))

	// One-click supplier order sending via email (Accessible by Admin and Mitarbeiter)
	mux.Handle("POST /api/bestellung/senden", s.RequirePermission("create_orders")(s.SendOrderMailHandler()))

	// Release all pending supplier orders (Accessible by Admin and Mitarbeiter)
	mux.Handle("POST /api/bestellungen/freigeben", s.RequirePermission("create_orders")(s.ReleaseOrdersHandler()))

	// List suppliers (Accessible by Admin and Mitarbeiter)
	mux.Handle("GET /api/lieferanten", s.RequirePermission("view_orders")(s.ListSuppliersHandler()))

	// Create supplier (Accessible by Admin and Mitarbeiter)
	mux.Handle("POST /api/lieferanten", s.RequirePermission("create_orders")(s.CreateSupplierHandler()))

	// Delete supplier (Accessible by Admin and Mitarbeiter)
	mux.Handle("DELETE /api/lieferanten/{id}", s.RequirePermission("create_orders")(s.DeleteSupplierHandler()))

	// Submit cart order with barcodes and PDF sending (Accessible by Admin and Mitarbeiter)
	mux.Handle("POST /api/orders", s.RequirePermission("create_orders")(s.SubmitOrderHandler()))

	// Get ordered copies in transit (Accessible by Admin and Mitarbeiter)
	mux.Handle("GET /api/bestellungen/zulauf", s.RequirePermission("view_orders")(s.GetIncomingShipmentsHandler()))

	// Real-time Event Stream (accessible by all authorized staff roles)
	sseHandler := s.Broker.Handler()
	mux.Handle("GET /events", s.RequirePermission("view_students")(sseHandler))

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
