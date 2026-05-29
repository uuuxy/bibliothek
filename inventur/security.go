package inventur

import "net/http"

// withSecurityHeaders setzt sicherheitsrelevante HTTP-Header für jede Antwort.
func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// Verhindert MIME-Type Sniffing
		writer.Header().Set("X-Content-Type-Options", "nosniff")

		// Verhindert Einbetten in iframes (Clickjacking-Schutz)
		writer.Header().Set("X-Frame-Options", "DENY")

		// Aktiviert HTTPS-Erzwingung im Browser (1 Jahr)
		writer.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Verhindert, dass der Browser die Seite als Referrer weitergibt
		writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Eingeschränkte Permissions Policy
		writer.Header().Set("Permissions-Policy", "camera=(self), microphone=()")

		// Content-Security-Policy: XSS-Schutz
		writer.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' https://covers.openlibrary.org https://portal.dnb.de https://books.google.com data:; "+
				"font-src 'self'; "+
				"connect-src 'self'; "+
				"frame-ancestors 'none'")

		next.ServeHTTP(writer, request)
	})
}
