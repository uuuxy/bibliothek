package inventur

import (
	"net/http"
	"strings"
)

// handleAdminBooks ist der zentrale Router für admin-geschützte Buch-Operationen.
// Schreibende Operationen lösen automatisch ein Backup-Signal aus.
func (handler *APIHandler) handleAdminBooks(writer http.ResponseWriter, request *http.Request) {
	isMutation := request.Method == http.MethodPost ||
		request.Method == http.MethodPut ||
		request.Method == http.MethodDelete

	w := writer
	var recorder *statusRecorder
	if isMutation && handler.backup != nil {
		recorder = &statusRecorder{ResponseWriter: writer, status: http.StatusOK}
		w = recorder
	}

	path := request.URL.Path

	switch request.Method {
	case http.MethodGet:
		switch path {
		case "/api/admin/class-books":
			handler.handleClassBooks(w, request)
		case "/api/admin/books/external-covers":
			handler.handleListExternalCovers(w, request)
		case "/api/admin/books/export":
			handler.handleExportCSV(w, request)
		default:
			writeError(w, http.StatusNotFound, "route nicht gefunden")
		}
	case http.MethodPost:
		switch {
		case path == "/api/admin/class-books":
			handler.handleUpdateClassBooks(w, request)
		case path == "/api/admin/class-books/add":
			handler.handleAddClassBooks(w, request)
		case path == "/api/books/import":
			handler.handleImportExcel(w, request)
		case path == "/api/admin/books/retry-covers":
			handler.handleRetryExternalCovers(w, request)
		case path == "/api/admin/books/import":
			writeError(w, http.StatusNotImplemented, "Import noch nicht implementiert")
		case path == "/api/books":
			handler.BearbeiteBuchErstellen(w, request)
		case strings.HasSuffix(path, "/refresh-cover"):
			handler.handleRefreshCover(w, request)
		case strings.HasSuffix(path, "/cover-upload"):
			handler.handleUploadCover(w, request)
		default:
			writeError(w, http.StatusNotFound, "route nicht gefunden")
		}
	case http.MethodPut:
		switch {
		case path == "/api/admin/books/reorder":
			handler.handleReorderBooks(w, request)
		case strings.HasSuffix(path, "/cover"):
			handler.handleUpdateCover(w, request)
		case strings.HasPrefix(path, "/api/books/"):
			handler.BearbeiteBuchAktualisieren(w, request)
		default:
			writeError(w, http.StatusNotFound, "route nicht gefunden")
		}
	case http.MethodDelete:
		switch path {
		case "/api/admin/class-books":
			handler.handleDeleteClassGroup(w, request)
		case "/api/books":
			handler.BearbeiteBuecherLoeschen(w, request)
		default:
			writeError(w, http.StatusNotFound, "route nicht gefunden")
		}
	default:
		writeError(w, http.StatusNotFound, "route nicht gefunden")
	}

	// Bei erfolgreicher Mutation: Backup-Signal auslösen
	if recorder != nil && recorder.status >= 200 && recorder.status < 300 {
		handler.backup.NotifyChange()
	}
}

// statusRecorder ist ein http.ResponseWriter-Wrapper, der den HTTP-Statuscode
// aufzeichnet, um nach dem Handler-Aufruf zu prüfen, ob die Operation erfolgreich war.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
