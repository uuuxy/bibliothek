package inventur

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

func (handler *APIHandler) handleUpdateCover(writer http.ResponseWriter, request *http.Request) {
	id, ok := extractPathID(request.URL.Path, 4, "api", "books", "cover")
	if !ok {
		writeError(writer, http.StatusBadRequest, "ungültige route")
		return
	}

	if id == "" {
		writeError(writer, http.StatusBadRequest, "id darf nicht leer sein")
		return
	}

	var input struct {
		CoverURL string `json:"coverUrl"`
	}
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		writeError(writer, http.StatusBadRequest, "ungültiges JSON")
		return
	}

	coverURL := strings.TrimSpace(input.CoverURL)
	if coverURL == "" {
		writeError(writer, http.StatusBadRequest, "coverUrl darf nicht leer sein")
		return
	}

	// Sicherheits-Validierung: Nur HTTPS-URLs und lokale Uploads erlauben
	if !strings.HasPrefix(coverURL, "https://") && !strings.HasPrefix(coverURL, "/uploads/") {
		writeError(writer, http.StatusBadRequest, "coverUrl muss mit https:// oder /uploads/ beginnen")
		return
	}

	err := handler.repo.UpdateBookMetadata(request.Context(), id, "", "", coverURL)
	if err != nil {
		log.Printf("Fehler beim Cover-Update für Buch %s: %v", id, err)
		writeError(writer, http.StatusInternalServerError, "cover konnte nicht aktualisiert werden")
		return
	}

	book, err := handler.repo.GetBookByID(request.Context(), id)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "buch nach update nicht gefunden")
		return
	}

	writeJSON(writer, http.StatusOK, map[string]any{"message": "cover manuell aktualisiert", "data": book})
}
