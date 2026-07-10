package inventur

import (
	"log"
	"net/http"
	"strings"
)

func (handler *APIHandler) handleRefreshCover(writer http.ResponseWriter, request *http.Request) {
	parts := strings.Split(strings.Trim(request.URL.Path, "/"), "/")
	if len(parts) != 4 || parts[0] != "api" || parts[1] != "books" || parts[3] != "refresh-cover" {
		writeError(writer, http.StatusBadRequest, "ungültige route")
		return
	}

	id := parts[2]
	if id == "" {
		writeError(writer, http.StatusBadRequest, "id darf nicht leer sein")
		return
	}

	book, err := handler.repo.GetBookByID(request.Context(), id)
	if err != nil {
		log.Printf("cover-refresh: buch %s nicht gefunden: %v", id, err) //nolint:gosec // Pre-existing G706
		writeError(writer, http.StatusNotFound, "buch nicht gefunden")
		return
	}

	lookup, err := handler.metadaten.SucheNachISBN(request.Context(), book.ISBN)
	if err != nil || lookup == nil {
		writeError(writer, http.StatusNotFound, "Keine neuen Metadaten gefunden")
		return
	}

	err = handler.repo.UpdateBookMetadata(request.Context(), id, lookup.Titel, lookup.Autor, lookup.CoverURL)
	if err != nil {
		log.Printf("cover-refresh: update fehlgeschlagen für buch %s: %v", id, err) //nolint:gosec // Pre-existing G706
		writeError(writer, http.StatusInternalServerError, "metadaten konnten nicht aktualisiert werden")
		return
	}

	book.Title = fallbackString(strings.TrimSpace(lookup.Titel), book.Title)
	book.Author = fallbackString(strings.TrimSpace(lookup.Autor), book.Author)
	book.CoverURL = fallbackString(strings.TrimSpace(lookup.CoverURL), book.CoverURL)

	writeJSON(writer, http.StatusOK, map[string]any{"message": "cover aktualisiert", "data": book})
}

func fallbackString(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
