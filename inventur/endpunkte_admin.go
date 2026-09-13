package inventur

import (
	"net/http"
)

// handleAdminBooks ist der zentrale Router für admin-geschützte Buch-Operationen.
// Routen mit Buch-Kennung im Pfad stehen nicht hier, sondern mit Platzhalter in
// api_routen.go.
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
		switch path {
		case routeClassBooks:
			handler.handleUpdateClassBooks(w, request)
		case "/api/admin/class-books/add":
			handler.handleAddClassBooks(w, request)
		case "/api/books/import":
			handler.handleImportExcel(w, request)
		case "/api/admin/books/retry-covers":
			handler.handleRetryExternalCovers(w, request)
		case "/api/books":
			handler.BearbeiteBuchErstellen(w, request)
		default:
			writeError(w, http.StatusNotFound, routeNotFoundMsg)
		}
	case http.MethodPut:
		switch path {
		case "/api/admin/books/reorder":
			handler.handleReorderBooks(w, request)
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
