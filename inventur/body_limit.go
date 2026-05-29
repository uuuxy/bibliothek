package inventur

import (
	"net/http"
)

// maxJSONBodyBytes begrenzt die maximale Größe von JSON-Request-Bodys auf 1 MB.
// Dies verhindert DoS-Angriffe durch übergroße Payloads.
const maxJSONBodyBytes int64 = 1 << 20 // 1 MB

// begrenzeAnfrageGroesse ist eine Middleware, die den Request Body auf eine
// maximale Größe beschränkt. Wird auf alle Nicht-Multipart-Endpunkte angewendet.
// Multipart-Uploads (z.B. Cover, Excel) haben eigene, höhere Limits.
func begrenzeAnfrageGroesse(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// Multipart-Uploads haben eigene Limits in ihren Handlern
		// (z.B. upload_handler.go: 10 MB, excel_import.go: 10 MB)
		contentType := request.Header.Get("Content-Type")
		if request.Method != http.MethodGet &&
			request.Method != http.MethodOptions &&
			request.Method != http.MethodHead &&
			!isMultipartRequest(contentType) {
			request.Body = http.MaxBytesReader(writer, request.Body, maxJSONBodyBytes)
		}
		next.ServeHTTP(writer, request)
	})
}

// isMultipartRequest prüft, ob der Content-Type multipart/form-data ist.
func isMultipartRequest(contentType string) bool {
	return len(contentType) >= 19 && contentType[:19] == "multipart/form-data"
}
