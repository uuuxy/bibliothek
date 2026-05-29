package inventur

import (
	"net/http"
	"os"
	"strings"
	"time"
)

// neuteredFileSystem prevents directory listing by wrapping an http.FileSystem
// and returning an error if a requested path is a directory without an index.html.
type neuteredFileSystem struct {
	fs http.FileSystem
}

func (nfs neuteredFileSystem) Open(path string) (http.File, error) {
	f, err := nfs.fs.Open(path)
	if err != nil {
		return nil, err
	}

	s, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	if s.IsDir() {
		index := strings.TrimSuffix(path, "/") + "/index.html"
		if idx, err := nfs.fs.Open(index); err != nil {
			f.Close()
			return nil, err
		} else {
			idx.Close()
		}
	}

	return f, nil
}

type APIHandlerConfig struct {
	Repo          *BookRepository
	Metadaten     *MetadatenClient
	AdminPassword string
	GuestPassword string
	JWTSecret     string
	Backup        *BackupManager
}

type APIHandler struct {
	repo          *BookRepository
	metadaten     *MetadatenClient
	adminPassword string
	guestPassword string
	jwtKey        []byte
	jwtIssuer     string
	jwtAudience   string
	tokenVersion  int
	adminTokenTTL time.Duration
	guestTokenTTL time.Duration
	adminHandler  http.Handler
	loginLimiter  *IPBasedRateLimiter
	backup        *BackupManager
	adminCookie   string
	guestCookie   string
	csrfCookie    string
	csrfHeader    string
	cookieSecure  bool
	cookieDomain  string
}

func NewAPIHandler(config APIHandlerConfig) *APIHandler {
	jwtIssuer := strings.TrimSpace(os.Getenv("JWT_ISSUER"))
	if jwtIssuer == "" {
		jwtIssuer = "inventur-api"
	}

	jwtAudience := strings.TrimSpace(os.Getenv("JWT_AUDIENCE"))
	if jwtAudience == "" {
		jwtAudience = "inventur-frontend"
	}

	handler := &APIHandler{
		repo:          config.Repo,
		metadaten:     config.Metadaten,
		adminPassword: config.AdminPassword,
		guestPassword: config.GuestPassword,
		jwtKey:        []byte(config.JWTSecret),
		jwtIssuer:     jwtIssuer,
		jwtAudience:   jwtAudience,
		tokenVersion:  parseIntEnv("JWT_TOKEN_VERSION", 1),
		adminTokenTTL: parseDurationEnv("JWT_ADMIN_TTL", 12*time.Hour),
		guestTokenTTL: parseDurationEnv("JWT_GUEST_TTL", 24*time.Hour),
		// Strenger Login-Rate-Limit: 3 Requests/Minute, Burst 3
		loginLimiter: NewIPBasedRateLimiter(3.0/60.0, 3),
		backup:       config.Backup,
		adminCookie:  "inventur_admin_token",
		guestCookie:  "inventur_guest_token",
		csrfCookie:   "inventur_csrf",
		csrfHeader:   "X-CSRF-Token",
		cookieSecure: parseBoolEnv("COOKIE_SECURE", true),
		cookieDomain: strings.TrimSpace(os.Getenv("COOKIE_DOMAIN")),
	}
	handler.adminHandler = handler.requireAdmin(http.HandlerFunc(handler.handleAdminBooks))
	return handler
}

func (handler *APIHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Method == http.MethodPost && request.URL.Path == "/api/login" {
		handler.loginLimiter.Middleware(http.HandlerFunc(handler.handleLogin)).ServeHTTP(writer, request)
		return
	}

	if request.Method == http.MethodPost && request.URL.Path == "/api/login/guest" {
		handler.loginLimiter.Middleware(http.HandlerFunc(handler.handleLoginGuest)).ServeHTTP(writer, request)
		return
	}

	if request.Method == http.MethodPost && request.URL.Path == "/api/logout" {
		handler.handleLogout(writer, request)
		return
	}

	if request.Method == http.MethodGet && request.URL.Path == "/api/auth/status" {
		handler.handleAuthStatus(writer, request)
		return
	}

	if request.Method == http.MethodGet && request.URL.Path == "/api/books" {
		handler.requireAuth(http.HandlerFunc(handler.BearbeiteBuecherListe)).ServeHTTP(writer, request)
		return
	}

	if request.Method == http.MethodGet && request.URL.Path == "/api/class-books" {
		handler.requireAuth(http.HandlerFunc(handler.handleClassBooks)).ServeHTTP(writer, request)
		return
	}

	if request.Method == http.MethodGet && strings.HasPrefix(request.URL.Path, "/api/lookup/") {
		// ISBN-Lookup benötigt Authentifizierung, da es externe APIs anfragt
		// und sonst als offener Proxy missbraucht werden könnte.
		// Added handler.requireAuth to prevent unauthenticated access to the lookup API.
		handler.requireAuth(http.HandlerFunc(handler.handleLookup)).ServeHTTP(writer, request)
		return
	}

	if request.Method == http.MethodGet && request.URL.Path == "/api/subjects" {
		handler.requireAuth(http.HandlerFunc(handler.handleGetSubjects)).ServeHTTP(writer, request)
		return
	}

	if request.Method == http.MethodGet && strings.HasPrefix(request.URL.Path, "/uploads/") {
		http.StripPrefix("/uploads/", http.FileServer(neuteredFileSystem{http.Dir("uploads")})).ServeHTTP(writer, request)
		return
	}

	if strings.HasPrefix(request.URL.Path, "/api/books") || strings.HasPrefix(request.URL.Path, "/api/admin") {
		handler.adminHandler.ServeHTTP(writer, request)
		return
	}

	writeError(writer, http.StatusNotFound, "route nicht gefunden")
}

func (handler *APIHandler) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, err := handler.extractValidClaimsFromRequest(request)
		if err != nil {
			writeError(writer, http.StatusUnauthorized, err.Error())
			return
		}
		if err := handler.validateCSRF(request); err != nil {
			writeError(writer, http.StatusForbidden, err.Error())
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func (handler *APIHandler) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		claims, err := handler.extractValidClaimsFromRequest(request)
		if err != nil {
			writeError(writer, http.StatusUnauthorized, err.Error())
			return
		}
		if !claims.Admin {
			writeError(writer, http.StatusUnauthorized, "admin privileges required")
			return
		}
		if err := handler.validateCSRF(request); err != nil {
			writeError(writer, http.StatusForbidden, err.Error())
			return
		}
		next.ServeHTTP(writer, request)
	})
}
