package inventur

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

func (handler *APIHandler) handleListExternalCovers(writer http.ResponseWriter, request *http.Request) {
	books, err := handler.repo.ListExternalCoverBooks(request.Context(), 300)
	if err != nil {
		log.Printf("Fehler beim Laden externer Cover-Bücher: %v", err)
		writeError(writer, http.StatusInternalServerError, "externe cover konnten nicht geladen werden")
		return
	}

	writeJSON(writer, http.StatusOK, map[string]any{"data": books})
}

func (handler *APIHandler) handleRetryExternalCovers(writer http.ResponseWriter, request *http.Request) {
	var eingabe struct {
		IDs   []string `json:"ids"`
		Limit int      `json:"limit"`
	}
	if err := json.NewDecoder(request.Body).Decode(&eingabe); err != nil {
		writeError(writer, http.StatusBadRequest, "ungültiges JSON")
		return
	}
	if !alleUUIDs(eingabe.IDs) {
		writeError(writer, http.StatusBadRequest, "ids enthält eine ungültige Kennung")
		return
	}

	var (
		books []Book
		err   error
	)
	if len(eingabe.IDs) > 0 {
		books, err = handler.repo.ListBooksByIDs(request.Context(), eingabe.IDs)
	} else {
		books, err = handler.repo.ListExternalCoverBooks(request.Context(), eingabe.Limit)
	}
	if err != nil {
		log.Printf("Fehler beim Laden externer Cover für Retry: %v", err)
		writeError(writer, http.StatusInternalServerError, "cover-retry konnte nicht gestartet werden")
		return
	}

	zahlen := map[string]int{"retried": 0, coverAktualisiert: 0, coverUebersprungen: 0, coverGescheitert: 0}
	for _, book := range books {
		zahlen["retried"]++
		zahlen[handler.ladeCoverErneut(request.Context(), book)]++
	}

	writeJSON(writer, http.StatusOK, map[string]any{
		"message": "cover-retry abgeschlossen",
		"data":    zahlen,
	})
}

// Die Zahlen der Antwort, unter denen ein Titel nach dem erneuten Laden zählt.
const (
	coverAktualisiert  = "updated"
	coverUebersprungen = "skipped"
	coverGescheitert   = "failed"
)

// ladeCoverErneut fragt die Katalogdienste nach dem Cover eines Titels, trägt eine neue
// Adresse ein und nennt, unter welcher Zahl der Titel in der Antwort zählt.
func (handler *APIHandler) ladeCoverErneut(ctx context.Context, book Book) string {
	if !validiereISBN(book.ISBN) {
		return coverUebersprungen
	}

	lookup, lookupErr := handler.metadaten.SucheNachISBN(ctx, book.ISBN)
	if lookupErr != nil || lookup == nil || lookup.CoverURL == "" {
		return coverGescheitert
	}
	if !strings.HasPrefix(lookup.CoverURL, "http") || lookup.CoverURL == book.CoverURL {
		return coverUebersprungen
	}

	if updateErr := handler.repo.UpdateBookMetadata(ctx, book.ID, "", "", lookup.CoverURL); updateErr != nil {
		return coverGescheitert
	}
	return coverAktualisiert
}
