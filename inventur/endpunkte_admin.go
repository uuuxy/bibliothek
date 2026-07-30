package inventur

import (
	"net/http"
	"strings"
)

// handleAdminBooks ist der zentrale Router für admin-geschützte Buch-Operationen.
func (handler *APIHandler) handleAdminBooks(w http.ResponseWriter, request *http.Request) {
	path := request.URL.Path

	switch request.Method {
	case http.MethodGet:
		switch path {
		case routeClassBooks:
			handler.handleClassBooks(w, request)
		case "/api/admin/books/external-covers":
			handler.handleListExternalCovers(w, request)
		case "/api/admin/books/export":
			handler.handleExportCSV(w, request)
		default:
			writeError(w, http.StatusNotFound, routeNotFoundMsg)
		}
	case http.MethodPost:
		switch {
		case path == routeClassBooks:
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
			writeError(w, http.StatusNotFound, routeNotFoundMsg)
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
			writeError(w, http.StatusNotFound, routeNotFoundMsg)
		}
	case http.MethodDelete:
		switch path {
		case routeClassBooks:
			handler.handleDeleteClassGroup(w, request)
		case "/api/books":
			handler.BearbeiteBuecherLoeschen(w, request)
		default:
			writeError(w, http.StatusNotFound, routeNotFoundMsg)
		}
	default:
		writeError(w, http.StatusNotFound, routeNotFoundMsg)
	}
}
