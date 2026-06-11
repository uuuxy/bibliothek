package inventur

import (
	"log"
	"net/http"
	"strings"
)

func (handler *APIHandler) handleLookup(writer http.ResponseWriter, request *http.Request) {
	parts := strings.Split(strings.Trim(request.URL.Path, "/"), "/")
	if len(parts) != 3 || parts[0] != "api" || parts[1] != "lookup" {
		writeError(writer, http.StatusBadRequest, "ungültige route")
		return
	}

	isbn := strings.TrimSpace(parts[2])
	if isbn == "" {
		writeError(writer, http.StatusBadRequest, "isbn fehlt")
		return
	}

	if !validiereISBN(isbn) {
		writeError(writer, http.StatusBadRequest, "ungültiges ISBN-Format")
		return
	}

	result, err := handler.metadaten.SucheNachISBN(request.Context(), isbn)
	if err != nil {
		log.Printf("isbn-lookup fehlgeschlagen für %s: %v", isbn, err)
		writeError(writer, http.StatusNotFound, "metadaten nicht gefunden")
		return
	}

	// Send mapping back exactly as frontend expects it via REST JSON tags
	writeJSON(writer, http.StatusOK, map[string]any{
		"data": map[string]string{
			"title":    result.Titel,
			"subtitle": result.Untertitel,
			"author":   result.Autor,
			"coverUrl": result.CoverURL,
			"subject":  result.Fach,
			"grade":    result.KlassenStufe,
			"verlag":   result.Verlag,
			"jahr":     result.Jahr,
		},
	})
}
