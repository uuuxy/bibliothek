package tuer

import (
	"net/http"

	"bibliothek/pkg/httpresp"
)

// AntworteJSON encodes the payload as JSON and sends it with the given HTTP status code.
// It sets the Content-Type header to application/json.
func AntworteJSON(w http.ResponseWriter, status int, antwort any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	httpresp.SchreibeJSON(w, antwort)
}

// AntworteErfolg is a convenience function that sends a {"status": "success"} JSON response.
func AntworteErfolg(w http.ResponseWriter) {
	AntworteJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// Mehrfach genutzte String-Literale, zentral definiert (SonarQube go:S1192).

// HTTP-Header-Namen.
const (
	HeaderContentType        = "Content-Type"
	HeaderContentDisposition = "Content-Disposition"
	HeaderContentLength      = "Content-Length"
	HeaderCacheControl       = "Cache-Control"
)

// Content-Type-Werte.
const (
	ContentTypeJSON = "application/json"
	ContentTypePDF  = "application/pdf"
)

// Fehlermeldungen, die mehrere Handler wörtlich gleich ausgeben.
const (
	MeldungIDLeer                = "id darf nicht leer sein"
	MeldungEinstellungenUnlesbar = "Einstellungen konnten nicht gelesen werden"
)
