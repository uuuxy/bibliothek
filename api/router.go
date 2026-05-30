package api

import (
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/db"
	_ "bibliothek/docs"
	"bibliothek/inventur"
	"bibliothek/repository"
	"bibliothek/sse"

	httpSwagger "github.com/swaggo/http-swagger"
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
			SameSite: http.SameSiteStrictMode,
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
	mux.Handle("DELETE /api/schueler/{id}", s.RequirePermission("delete_students")(s.DeleteStudentHandler(auditRepo)))

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

	// Mark copy as defective and create Schadensfaelle (Accessible by Admin and Mitarbeiter)
	mux.Handle("POST /api/buecher/exemplare/{id}/defekt", s.RequirePermission("edit_books")(s.MarkCopyDefektHandler()))

	// Undo a recent loan return within 1 hour (Accessible by Admin and Mitarbeiter)
	mux.Handle("DELETE /api/ausleihen/{id}/rueckgabe", s.RequirePermission("view_students")(s.UndoReturnHandler()))

	// Get system settings (Accessible by Admin)
	mux.Handle("GET /api/einstellungen", s.RequirePermission("manage_users")(s.GetSettingsHandler()))

	// Update system settings (Accessible by Admin)
	mux.Handle("PUT /api/einstellungen", s.RequirePermission("manage_users")(s.UpdateSettingsHandler()))

	// Delete physical copy (Accessible by Admin and Mitarbeiter)
	mux.Handle("DELETE /api/buecher/exemplare/{id}", s.RequirePermission("delete_books")(s.DeleteCopyHandler(auditRepo)))

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

	// Live ISBN metadata lookup + catalog upsert (Accessible by Admin and Mitarbeiter)
	mux.Handle("POST /api/buecher/aus-isbn", s.RequirePermission("create_orders")(s.ISBNZuTitelHandler()))

	// Get ordered copies in transit (Accessible by Admin and Mitarbeiter)
	mux.Handle("GET /api/bestellungen/zulauf", s.RequirePermission("view_orders")(s.GetIncomingShipmentsHandler()))

	// Klassen → Klassenlehrer-E-Mail Mapping (Admin only)
	mux.Handle("GET /api/klassen-mapping", s.RequirePermission("manage_users")(s.GetKlassenMappingHandler()))
	mux.Handle("POST /api/klassen-mapping", s.RequirePermission("manage_users")(s.UpsertKlassenMappingHandler()))
	mux.Handle("DELETE /api/klassen-mapping/{klasse}", s.RequirePermission("manage_users")(s.DeleteKlassenMappingHandler()))

	// Klassensatz Reservierungen (Lehrer submits; Admin/Mitarbeiter manages)
	mux.Handle("POST /api/reservierungen/klassensatz", s.RequirePermission("view_students")(s.CreateKlassensatzReservierungHandler()))
	mux.Handle("GET /api/reservierungen/klassensatz", s.RequirePermission("view_orders")(s.GetKlassensatzReservierungenHandler()))
	mux.Handle("GET /api/reservierungen/klassensatz/anzahl", s.RequirePermission("view_orders")(s.GetKlassensatzReservierungenAnzahlHandler()))
	mux.Handle("PUT /api/reservierungen/klassensatz/{id}/erledigen", s.RequirePermission("create_orders")(s.ErledigeKlassensatzReservierungHandler()))

	// Mahnwesen – overdue loans, PDF generation, SMTP dispatch
	mux.Handle("GET /api/mahnwesen", s.RequirePermission("view_students")(s.GetMahnwesenHandler()))
	mux.Handle("GET /api/mahnwesen/pdf", s.RequirePermission("view_students")(s.GetMahnwesenPDFHandler()))
	mux.Handle("POST /api/mahnwesen/senden", s.RequirePermission("create_orders")(s.SendMahnwesenHandler()))

	// Public OPAC catalog search (DSGVO-compliant: no loan data exposed)
	mux.HandleFunc("GET /api/opac/suche", s.PublicCatalogSearchHandler())

	// Antolin proxy – public, 24-hour in-memory cache
	mux.HandleFunc("GET /api/antolin", s.AntolinHandler())

	// Digital signage / info monitor data – public
	mux.HandleFunc("GET /api/monitor/slides", s.GetMonitorSlidesHandler())

	// Vormerkungen (individual book reservations / waitlist)
	mux.Handle("GET /api/vormerkungen", s.RequirePermission("view_books")(s.ListVormerkungHandler()))
	mux.Handle("POST /api/vormerkungen", s.RequirePermission("view_books")(s.CreateVormerkungHandler()))
	mux.Handle("DELETE /api/vormerkungen/{id}", s.RequirePermission("view_books")(s.DeleteVormerkungHandler()))

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

	// Swagger interactive documentation
	mux.Handle("GET /swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))
	mux.HandleFunc("GET /swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
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

	// Wrap mux in logging, rate limiting, HTTPS redirect, body size limit and RBAC blocking middlewares
	bodyLimiter := MaxBodySizeMiddleware(5 * 1024 * 1024) // 5MB limit
	rateLimiter := RateLimitMiddleware(50)

	// Chain: SecurityHeaders -> CORS -> Logging -> HTTPSRedirect -> BodyLimiter -> RateLimiter -> RBACBlock -> Mux
	globalHandler := SecurityHeadersMiddleware(CORSMiddleware(s.HTTPSRedirectMiddleware(bodyLimiter(rateLimiter(s.RBACBlockMiddleware(mux))))))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log incoming request without exposing IP addresses (.RemoteAddr stripped for DSGVO)
		log.Printf("Incoming Request: %s %s", r.Method, r.URL.Path)
		globalHandler.ServeHTTP(w, r)
	})
}

// HTTPSRedirectMiddleware automatically redirects unencrypted HTTP requests to HTTPS.
func (s *Server) HTTPSRedirectMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Bypass HTTPS redirection in local/development mode when CookieSecure is disabled
		if !s.CookieSecure {
			next.ServeHTTP(w, r)
			return
		}
		isHTTPS := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
		if !isHTTPS {
			target := "https://" + r.Host + r.URL.RequestURI()
			http.Redirect(w, r, target, http.StatusMovedPermanently)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// MaxBodySizeMiddleware limits the request body size to prevent DoS.
func MaxBodySizeMiddleware(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}

// RBACBlockMiddleware checks roles and enforces path access rules for LEHRER and HELFER roles.
func (s *Server) RBACBlockMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/health" || path == "/login/barcode" || path == "/api/auth/status" {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie("session_token")
		if err == nil && cookie.Value != "" {
			claims, err := s.Auth.VerifyToken(cookie.Value)
			if err == nil {
				role := strings.ToUpper(string(claims.Rolle))
				switch role {
				case "LEHRER":
					isAllowed := (r.Method == http.MethodGet && (path == "/api/search" || strings.HasPrefix(path, "/api/buecher/titel/") && strings.Contains(path, "/exemplare"))) ||
						(r.Method == http.MethodPost && path == "/api/auth/logout")

					if !isAllowed {
						apierrors.SendHTTPError(w, http.StatusForbidden, errors.New("zugriff verweigert für Lehrer"))
						return
					}
				case "HELFER":
					isAllowed := (r.Method == http.MethodPost && (path == "/api/action" || path == "/api/auth/logout")) ||
						(r.Method == http.MethodGet && path == "/events")

					if !isAllowed {
						apierrors.SendHTTPError(w, http.StatusForbidden, errors.New("zugriff verweigert für Helfer"))
						return
					}
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

// SecurityHeadersMiddleware sets HSTS, X-Frame-Options, X-Content-Type-Options and
// Referrer-Policy on every response. HSTS uses a 1-year max-age with includeSubDomains
// and preload to harden the school domain against protocol-downgrade attacks.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// HSTS: 1 year, include subdomains, eligible for preload list
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")
		// Prevent MIME-type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// Restrict referrer information
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// Basic CSP: allow same-origin resources only (adjust if CDN is added)
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; connect-src 'self'")
		next.ServeHTTP(w, r)
	})
}

// CORSMiddleware restricts cross-origin requests to the configured school domain.
// Set ALLOWED_ORIGIN env var to the school's frontend URL (e.g. https://bibliothek.schule.de).
// Falls back to same-origin only if not configured.
func CORSMiddleware(next http.Handler) http.Handler {
	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			// Same-origin request (no Origin header) – always allowed
			next.ServeHTTP(w, r)
			return
		}
		// Only allow explicitly configured origin; reject everything else
		if allowedOrigin != "" && origin == allowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
