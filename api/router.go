package api

// router.go — HTTP route registration.
// All middleware functions live in middleware.go.
// Handler functions live in their respective domain files (copy_admin.go,
// user_admin.go, graduates.go, audit_handler.go, inventory.go, etc.).

import (
	"log"
	"net/http"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	_ "bibliothek/docs"
	"bibliothek/internal/middleware"
	"bibliothek/internal/service"
	"bibliothek/repository"
	"bibliothek/sse"

	sentryhttp "github.com/getsentry/sentry-go/http"
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

	s.registerInventurRoutes(mux)

	s.registerAuthRoutes(mux)

	// Delegate to domain-specific routers
	s.registerPublicRoutes(mux)
	s.registerCoreActionRoutes(mux, studentRepo, bookRepo, omniboxSvc)
	s.registerStudentRoutes(mux, studentRepo, mahnRepo, auditRepo)
	s.registerBookRoutes(mux, bookRepo, auditRepo)
	s.registerSystemRoutes(mux, auditRepo, userRepo, s.DB.Pool)
	s.registerOrderRoutes(mux, orderSvc, pdfSvc)

	s.registerImportRoutes(mux)

	s.registerSwaggerRoutes(mux)

	s.registerFrontendRoutes(mux)

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
