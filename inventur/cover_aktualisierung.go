package inventur

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"bibliothek/pkg/logger"
)

func (handler *APIHandler) handleRefreshCover(writer http.ResponseWriter, request *http.Request) {
	id, ok := buchIDAusPfad(writer, request)
	if !ok {
		return
	}

	book, err := handler.repo.GetBookByID(request.Context(), id)
	if errors.Is(err, ErrBookNotFound) {
		writeError(writer, http.StatusNotFound, "buch nicht gefunden")
		return
	}
	if err != nil {
		log.Printf("cover-refresh: buch %s laden: %v", logger.SanitizeLog(id), err)
		writeError(writer, http.StatusInternalServerError, "buch konnte nicht geladen werden")
		return
	}

	lookup, err := handler.metadaten.SucheNachISBN(request.Context(), book.ISBN)
	if err != nil || lookup == nil {
		if errors.Is(err, ErrKatalogdiensteNichtErreichbar) {
			// Netzausfall ist kein Nicht-Treffer — sonst sieht „Cover neu laden" bei
			// WLAN-Störung so aus, als gäbe es schlicht kein besseres Cover.
			writeError(writer, http.StatusBadGateway, "Katalogdienste (DNB, Google, OpenLibrary) nicht erreichbar — bitte später erneut versuchen")
			return
		}
		writeError(writer, http.StatusNotFound, "Kein Cover: Zu dieser ISBN kennen DNB, Google Books und OpenLibrary keinen Titel")
		return
	}

	// Nur das Cover. Bis zum 22.09.2026 schrieb die Tür auch Titel und Autor aus der
	// Suche zurück — ohne Aufrufer fiel das nicht auf. Mit dem Knopf „Cover neu holen" in
	// der Titel-Akte wäre es ein stilles Überschreiben von Handkorrekturen: Wer „Mathe 7,
	// Ausgabe Hessen" eingetippt hat, bekäme den Katalogtitel zurück, ohne es zu sehen.
	// Der Knopf verspricht ein Bild, also ändert die Tür nur das Bild.
	coverURL := strings.TrimSpace(lookup.CoverURL)
	if coverURL == "" {
		writeError(writer, http.StatusNotFound, "Bei DNB, Google Books und OpenLibrary gibt es kein Cover zu dieser ISBN")
		return
	}

	err = handler.repo.UpdateBookMetadata(request.Context(), id, "", "", coverURL)
	if err != nil {
		log.Printf("cover-refresh: update fehlgeschlagen für buch %s: %v", logger.SanitizeLog(id), err)
		writeError(writer, http.StatusInternalServerError, "cover konnte nicht gespeichert werden")
		return
	}

	book.CoverURL = coverURL
	writeJSON(writer, http.StatusOK, map[string]any{"message": "cover aktualisiert", "data": book})
}
