package inventur

import (
	"log"
	"net/http"
	"strings"
)

func (handler *APIHandler) handleLookup(writer http.ResponseWriter, request *http.Request) {
	isbnRoh, ok := extractPathID(request.URL.Path, 3, "api", "lookup", "")
	if !ok {
		writeError(writer, http.StatusBadRequest, "ungültige route")
		return
	}

	isbn := strings.TrimSpace(isbnRoh)
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
