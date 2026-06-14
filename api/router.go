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
	mahnRepo := repository.NewMahnwesenRepository(s.DB.Pool)

	orderSvc := NewOrderService(s.DB, bookRepo)
	pdfSvc := NewPDFService()

	// Initialize Inventur sub-module handlers
	_ = os.MkdirAll("uploads", 0750)
	_ = os.MkdirAll("uploads/fotos", 0750)
	invRepo := inventur.NewBookRepository(s.DB.Pool)
	invMeta := inventur.NeuerMetadatenClient()

	// Fail hard if JWT_SECRET is missing during route initialization
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("FATAL: JWT_SECRET environment variable is required and cannot be empty")
	}

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
	mux.HandleFunc("POST /login", auth.LoginHandler(s.DB.Pool, s.Auth, s.CookieSecure))

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	})

	// Delegate to domain-specific routers
	s.registerPublicRoutes(mux)
	s.registerCoreActionRoutes(mux, studentRepo, bookRepo, loanRepo)
	s.registerStudentRoutes(mux, studentRepo, mahnRepo, auditRepo)
	s.registerBookRoutes(mux, bookRepo, auditRepo)
	s.registerSystemRoutes(mux, auditRepo)
	s.registerOrderRoutes(mux, orderSvc, pdfSvc)

	actionHandler := s.ActionHandler(studentRepo, bookRepo, loanRepo)
	mux.Handle("POST /api/action", s.RequirePermission("view_students")(actionHandler))

	actionBatchHandler := s.ActionBatchHandler(studentRepo, bookRepo, loanRepo)
	mux.Handle("POST /api/action/batch", s.RequirePermission("view_students")(actionBatchHandler))

	// LITTERA CSV Import (Accessible by Admin)
	mux.Handle("POST /api/import/littera", s.RequirePermission("manage_inventory")(s.LitteraImportHandler()))

	// LUSD CSV Import (Accessible by Admin)
	mux.Handle("POST /api/import/lusd", s.RequirePermission("manage_inventory")(s.ImportLUSDHandler(studentRepo)))

	// Swagger interactive documentation
	mux.Handle("GET /swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))
	mux.HandleFunc("GET /swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
	})

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
	bodyLimiter := MaxBodySizeMiddleware(5 * 1024 * 1024) // 5MB limit
	rateLimiter := RateLimitMiddleware(50)

	// Chain: PanicRecovery -> SecurityHeaders -> CORS -> Logging -> HTTPSRedirect -> BodyLimiter -> RateLimiter -> CSRF -> RBACBlock -> ValidateUUIDParams -> Mux
	globalHandler := PanicRecoveryMiddleware(SecurityHeadersMiddleware(CORSMiddleware(s.HTTPSRedirectMiddleware(bodyLimiter(rateLimiter(s.CSRFMiddleware(s.RBACBlockMiddleware(ValidateUUIDParamsMiddleware(mux)))))))))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log incoming request without exposing IP addresses (.RemoteAddr stripped for DSGVO)
		// #nosec G706
		log.Printf("Incoming Request: %s %s", r.Method, r.URL.Path)
		globalHandler.ServeHTTP(w, r)
	})
}
