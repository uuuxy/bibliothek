package inventur

import (
	"errors"
	"log"
	"net/http"
	"strings"
)

func (handler *APIHandler) handleLookup(writer http.ResponseWriter, request *http.Request) {
	isbn := strings.TrimSpace(request.PathValue("isbn"))
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
		if errors.Is(err, ErrKatalogdiensteNichtErreichbar) {
			// Netzausfall ist kein Nicht-Treffer: Bei 404 katalogisiert die Theke
			// während einer WLAN-Störung Bücher von Hand, die längst in der DNB stehen.
			log.Printf("isbn-lookup: Katalogdienste nicht erreichbar für %s: %v", isbn, err)
			writeError(writer, http.StatusBadGateway, "Katalogdienste (DNB, Google, OpenLibrary) nicht erreichbar — bitte später erneut versuchen")
			return
		}
		log.Printf("isbn-lookup fehlgeschlagen für %s: %v", isbn, err)
		writeError(writer, http.StatusNotFound, "metadaten nicht gefunden")
		return
	}

	// Send mapping back exactly as frontend expects it via REST JSON tags
	writeJSON(writer, http.StatusOK, map[string]any{
		"data": map[string]string{
			"title":      result.Titel,
			"subtitle":   result.Untertitel,
			"author":     result.Autor,
			"coverUrl":   result.CoverURL,
			"subject":    result.Fach,
			"grade":      result.KlassenStufe,
			"verlag":     result.Verlag,
			"jahr":       result.Jahr,
			"zielgruppe": result.Zielgruppe,
		},
	})
}
