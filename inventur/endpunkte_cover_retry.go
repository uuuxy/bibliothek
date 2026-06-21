package inventur

import (
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

	retried := 0
	updated := 0
	skipped := 0
	failed := 0

	for _, book := range books {
		retried++
		if !validiereISBN(book.ISBN) {
			skipped++
			continue
		}

		lookup, lookupErr := handler.metadaten.SucheNachISBN(request.Context(), book.ISBN)
		if lookupErr != nil || lookup == nil || lookup.CoverURL == "" {
			failed++
			continue
		}
		if !strings.HasPrefix(lookup.CoverURL, "http") || lookup.CoverURL == book.CoverURL {
			skipped++
			continue
		}

		if updateErr := handler.repo.UpdateBookMetadata(request.Context(), book.ID, "", "", lookup.CoverURL); updateErr != nil {
			failed++
			continue
		}
		updated++
	}

	writeJSON(writer, http.StatusOK, map[string]any{
		"message": "cover-retry abgeschlossen",
		"data": map[string]int{
			"retried": retried,
			"updated": updated,
			"skipped": skipped,
			"failed":  failed,
		},
	})
}
