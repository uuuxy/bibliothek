package inventur

import (
	"net/http"
	"strings"
)

// withCORS erlaubt Frontend-Calls auf die API.
// Im Produktionsbetrieb (hinter Reverse Proxy auf derselben Domain) ist CORS
// eigentlich nicht nötig, aber für lokale Entwicklung trotzdem aktiviert.
func withCORS(next http.Handler, allowedOrigins map[string]struct{}) http.Handler {

	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		origin := request.Header.Get("Origin")

		if origin != "" {
			normalizedOrigin, err := normalizeOrigin(origin)
			if err == nil {
				if _, exists := allowedOrigins[strings.ToLower(normalizedOrigin)]; exists {
					writer.Header().Set("Access-Control-Allow-Origin", origin)
					writer.Header().Set("Access-Control-Allow-Credentials", "true")
					writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
					writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Token")
				}
			}
			writer.Header().Set("Vary", "Origin")
		}

		if request.Method == http.MethodOptions {
			writer.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(writer, request)
	})
}
