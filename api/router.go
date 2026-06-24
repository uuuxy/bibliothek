package api

// router.go — HTTP route registration.
// All middleware functions live in middleware.go.
// Handler functions live in their respective domain files (copy_admin.go,
// user_admin.go, graduates.go, audit_handler.go, inventory.go, etc.).

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
	_ "bibliothek/docs"
	"bibliothek/internal/middleware"
	"bibliothek/internal/service"
	"bibliothek/inventur"
	"bibliothek/pkg/httpresp"
	"bibliothek/repository"
	"bibliothek/sse"

	sentryhttp "github.com/getsentry/sentry-go/http"
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
	mahnRepo := repository.NewMahnwesenRepository(s.DB.Pool)
	userRepo := repository.NewUserRepository(s.DB.Pool)

	loanSvc := service.NewLoanService(s.DB.Pool, studentRepo, bookRepo, loanRepo, auditRepo)
	deviceSvc := service.NewDeviceService(s.DB.Pool, studentRepo, loanRepo, auditRepo)
	omniboxSvc := service.NewOmniboxService(s.DB.Pool, studentRepo, bookRepo, loanRepo, loanSvc, deviceSvc)
	orderSvc := NewOrderService(s.DB, bookRepo)
	pdfSvc := NewPDFService()

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
		RequireAuth:      s.RequirePermission("view_books"),
		RequireAdminAuth: s.RequirePermission("edit_books"),
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

	// Public Endpoints
	mux.Handle("POST /login", AuthRateLimitMiddleware(http.HandlerFunc(auth.LoginHandler(s.DB.Pool, s.Auth, s.CookieSecure))))

	// Image Caching (Public)
	mux.HandleFunc("GET /api/images/cover", s.ServeCoverImageHandler())

	// CSRF token bootstrap (public): lets API clients obtain a token + cookie before any mutation
	mux.HandleFunc("GET /api/csrf-token", s.CSRFTokenHandler())

	// Token refresh (sliding window) — exempt from CSRF via middleware config
	mux.HandleFunc("POST /api/auth/refresh", auth.RefreshTokenHandler(s.Auth, s.CookieSecure))

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

	// Delegate to domain-specific routers
	s.registerPublicRoutes(mux)
	s.registerCoreActionRoutes(mux, studentRepo, bookRepo, omniboxSvc)
	s.registerStudentRoutes(mux, studentRepo, mahnRepo, auditRepo)
	s.registerBookRoutes(mux, bookRepo, auditRepo)
	s.registerSystemRoutes(mux, auditRepo, userRepo, s.DB.Pool)
	s.registerOrderRoutes(mux, orderSvc, pdfSvc)

	// LITTERA CSV Import (Accessible by Admin)
	mux.Handle("POST /api/import/littera", s.RequirePermission("manage_inventory")(s.LitteraImportHandler()))
	mux.Handle("POST /api/admin/import-bestand", s.RequirePermission("manage_inventory")(http.HandlerFunc(s.BestandImportHandler)))
	mux.Handle("POST /api/admin/sync-covers", s.RequirePermission("manage_inventory")(SyncCoversHandler(s.DB.Pool)))

	// Swagger interactive documentation (Only accessible in local/development mode)
	if os.Getenv("APP_ENV") == "local" || os.Getenv("APP_ENV") == "development" {
		mux.Handle("GET /swagger/", httpSwagger.Handler(
			httpSwagger.URL("/swagger/doc.json"),
		))
		mux.HandleFunc("GET /swagger", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
		})
	}

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

	// Wrap mux in logging, rate limiting, HTTPS redirect, body size limit and RBAC blocking middlewares
	bodyLimiter := MaxBodySizeMiddleware(100 * 1024 * 1024) // 100MB limit
	rateLimiter := RateLimitMiddleware(50)
	timeoutLimiter := TimeoutMiddleware(15 * time.Second)

	// Chain: PanicRecovery -> Sentry -> SecurityHeaders -> CORS -> Logging -> HTTPSRedirect -> BodyLimiter -> TimeoutLimiter -> RateLimiter -> CSRF -> ValidateUUIDParams -> Mux
	// Hinweis: Die frühere RBACBlockMiddleware (hartkodierte Pfad-Allowlist für LEHRER/HELFER)
	// wurde entfernt. Sie überstimmte das konfigurierbare role_permissions-System und sorgte dafür,
	// dass z. B. ein LEHRER seine im PermissionManager gewährten Rechte (view_students etc.) nicht
	// nutzen konnte. Autorisierung erfolgt nun einheitlich über RequirePermission/RequireRoles.
	sentryMiddleware := sentryhttp.New(sentryhttp.Options{Repanic: true}).Handle
	globalHandler := PanicRecoveryMiddleware(sentryMiddleware(middleware.SecurityHeadersMiddleware(CORSMiddleware(LoggingMiddleware(s.HTTPSRedirectMiddleware(bodyLimiter(timeoutLimiter(rateLimiter(s.CSRFMiddleware(ValidateUUIDParamsMiddleware(mux)))))))))))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log incoming request without exposing IP addresses (.RemoteAddr stripped for DSGVO)
		// #nosec G706
		log.Printf("Incoming Request: %s %s", r.Method, r.URL.Path)
		globalHandler.ServeHTTP(w, r)
	})
}
