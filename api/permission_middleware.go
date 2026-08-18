package api

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"bibliothek/apierrors"
	"bibliothek/auth"

	"github.com/jackc/pgx/v5"
)

var (
	permCache   = make(map[string]cacheEntry)
	permCacheMu sync.RWMutex
)

type cacheEntry struct {
	Allowed   bool
	ExpiresAt time.Time
}

// InvalidatePermissionCache clears all cached permission entries.
// Call this whenever roles or permissions are changed (e.g. UpdateUser, UpdatePermissions)
// to prevent stale cache entries from granting/denying access incorrectly.
func InvalidatePermissionCache() {
	permCacheMu.Lock()
	permCache = make(map[string]cacheEntry)
	permCacheMu.Unlock()
}

// claimsAusRequest liest das Session-Cookie und verifiziert das Token. Bei Fehler
// werden der passende HTTP-Status und der Fehler zurückgegeben.
func (s *Server) claimsAusRequest(r *http.Request) (*auth.Claims, int, error) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return nil, http.StatusUnauthorized, err
		}
		return nil, http.StatusBadRequest, err
	}
	// VerifyToken prüft neben Signatur und Blacklist auch den Echtzeit-Kontostatus
	// (aktiv / nicht gelöscht) direkt in der DB — siehe auth.Authenticator.VerifyToken.
	// Deshalb ist hier keine zusätzliche Zombie-Session-Prüfung nötig.
	claims, err := s.Auth.VerifyToken(cookie.Value)
	if err != nil {
		return nil, http.StatusUnauthorized, err
	}

	return claims, 0, nil
}

// erlaubeZugriff injiziert die Claims in den Request-Kontext und ruft den nächsten
// Handler auf.
func erlaubeZugriff(w http.ResponseWriter, r *http.Request, next http.Handler, claims *auth.Claims) {
	ctx := context.WithValue(r.Context(), auth.ClaimsContextKey, claims)
	next.ServeHTTP(w, r.WithContext(ctx))
}

// leseBerechtigungCache liefert eine noch gültige, gecachte Entscheidung.
func leseBerechtigungCache(cacheKey string) (allowed bool, found bool) {
	permCacheMu.RLock()
	entry, ok := permCache[cacheKey]
	permCacheMu.RUnlock()
	if ok && time.Now().Before(entry.ExpiresAt) {
		return entry.Allowed, true
	}
	return false, false
}

// ermittleUndCacheBerechtigung prüft das Recht in der DB und cacht ausschließlich
// stabile Entscheidungen (gewährt / explizit nicht gewährt). Transiente DB-Fehler
// (Timeout, Pool erschöpft, Verbindungsabriss) dürfen NICHT gecacht werden — sonst
// zementierte ein kurzer Aussetzer 60 s lang ein 403 für berechtigte Nutzer — und
// sind ein Server- (500), kein Berechtigungsproblem. pgx.ErrNoRows dagegen heißt
// "Recht nicht gewährt" und ist stabil cachebar.
func (s *Server) ermittleUndCacheBerechtigung(ctx context.Context, rolle, permission, cacheKey string) (bool, error) {
	var allowed bool
	query := `
		SELECT allowed
		FROM role_permissions
		WHERE UPPER(role) = UPPER($1) AND permission = $2
	`
	err := s.DB.Pool.QueryRow(ctx, query, rolle, permission).Scan(&allowed)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return false, err
	}
	finalAllowed := err == nil && allowed

	permCacheMu.Lock()
	// Abgelaufene Einträge beim Miss-Write opportunistisch entfernen, damit der Cache
	// nicht unbegrenzt wächst. Der Keyspace (Rolle × Permission) ist klein, daher ist
	// ein vollständiger Sweep hier günstig und kommt ohne Hintergrund-Goroutine aus.
	now := time.Now()
	for k, v := range permCache {
		if now.After(v.ExpiresAt) {
			delete(permCache, k)
		}
	}
	permCache[cacheKey] = cacheEntry{
		Allowed:   finalAllowed,
		ExpiresAt: now.Add(60 * time.Second),
	}
	permCacheMu.Unlock()

	return finalAllowed, nil
}

// RequireAuthenticated verlangt nur eine gültige Sitzung, kein bestimmtes Recht.
//
// Für Endpunkte, die JEDER angemeldete Client öffnet, unabhängig von seiner Rolle — heute
// genau einer: der SSE-Stream /events, den der authStore direkt nach dem Login aufbaut.
// Ihn an ein Fachrecht zu hängen war eine Zeitbombe: Wer das Recht nicht hat, bekommt
// keine saubere Absage, sondern eine Reconnect-Schleife plus das Offline-Overlay aus
// App.svelte, das nach 25 s ohne Herzschlag zuschlägt. Für die Helfer-Rolle war das schon
// einmal der Grund, den Stream von view_students auf perform_actions umzuhängen (siehe
// routes_system.go) — mit der Portal-Rolle KOLLEGIUM wäre derselbe Fehler ein zweites Mal
// aufgetreten. Der Stream trägt nur operative Aktions-Events (Typ, IDs, Buchtitel,
// Zeitstempel) und keine Schüler-PII, deshalb ist die Sitzung hier die richtige Schwelle.
func (s *Server) RequireAuthenticated() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, status, err := s.claimsAusRequest(r)
			if err != nil {
				apierrors.SendHTTPError(w, status, err)
				return
			}
			erlaubeZugriff(w, r, next, claims)
		})
	}
}

// RequirePermission returns a middleware that validates if the authenticated user
// has the required permission dynamically defined in the database.
func (s *Server) RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Die UUID-Prüfung sitzt hier und NICHT mehr global um den Mux herum.
		//
		// Global gelegt las sie r.PathValue("id"), bevor der ServeMux die Route
		// aufgelöst hatte — der Wert war dort immer leer, die Prüfung lief also nie
		// (Audit-Befund vom 01.08.2026, belegt in middleware_frist_test.go). An dieser
		// Stelle ist die Route bereits getroffen und PathValue gefüllt.
		return ValidateUUIDParamsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.pruefeBerechtigung(w, r, next, permission)
		}))
	}
}

// pruefeBerechtigung setzt die RBAC-Prüfung für einen Request um: Admins dürfen immer,
// sonst wird die konfigurierte Rolle→Permission-Zuordnung (mit Cache) geprüft. Bei
// Erlaubnis wird an next delegiert, andernfalls 403 bzw. 500 gesendet.
func (s *Server) pruefeBerechtigung(w http.ResponseWriter, r *http.Request, next http.Handler, permission string) {
	claims, status, err := s.claimsAusRequest(r)
	if err != nil {
		apierrors.SendHTTPError(w, status, err)
		return
	}

	// Admin role always has all permissions allowed
	if strings.EqualFold(string(claims.Rolle), string(auth.RoleAdmin)) {
		erlaubeZugriff(w, r, next, claims)
		return
	}

	cacheKey := string(claims.Rolle) + ":" + permission
	if allowed, found := leseBerechtigungCache(cacheKey); found {
		if !allowed {
			log.Printf("Permission denied for role '%s' permission '%s' (FROM CACHE).", claims.Rolle, permission)
			apierrors.SendHTTPError(w, http.StatusForbidden, errors.New("keine Berechtigung für diese Aktion"))
			return
		}
		erlaubeZugriff(w, r, next, claims)
		return
	}

	finalAllowed, err := s.ermittleUndCacheBerechtigung(r.Context(), string(claims.Rolle), permission, cacheKey)
	if err != nil {
		log.Printf("Permission check DB error for role '%s' permission '%s': %v", claims.Rolle, permission, err)
		apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("berechtigung konnte nicht geprüft werden"))
		return
	}

	if !finalAllowed {
		log.Printf("Permission denied for role '%s' permission '%s'. allowed: %v", claims.Rolle, permission, finalAllowed)
		apierrors.SendHTTPError(w, http.StatusForbidden, errors.New("keine Berechtigung für diese Aktion"))
		return
	}

	erlaubeZugriff(w, r, next, claims)
}

// BesitztRecht prüft ein Zusatzrecht INNERHALB eines Handlers, dessen Route
// bereits hinter einem schwächeren Recht liegt — für Antworten, die je nach
// Aufrufer mehr oder weniger zeigen (z. B. Geräteliste: Verfügbarkeit für alle
// mit view_books, der Name des Ausleihers nur mit view_students; siehe
// bewertung/sicherheitsbefund-kiosk-suche.md). Fehler werten als "nein":
// Im Zweifel wird weniger gezeigt, nie mehr.
func (s *Server) BesitztRecht(r *http.Request, permission string) bool {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		return false
	}
	if strings.EqualFold(string(claims.Rolle), string(auth.RoleAdmin)) {
		return true
	}
	cacheKey := string(claims.Rolle) + ":" + permission
	if allowed, found := leseBerechtigungCache(cacheKey); found {
		return allowed
	}
	allowed, err := s.ermittleUndCacheBerechtigung(r.Context(), string(claims.Rolle), permission, cacheKey)
	if err != nil {
		return false
	}
	return allowed
}
