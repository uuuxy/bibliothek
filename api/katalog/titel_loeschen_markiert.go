package katalog

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"bibliothek/repository"
)

// MarkierteTitelLoeschen ist die Tür DELETE /api/books: Sie löscht die in der Titelliste
// markierten Titel ({"ids": [...]}) samt allem, was an ihnen hängt, auch verliehene Exemplare
// (TitelAbfragen.LoescheTitelSamtAusleihen). Die Tür für einen einzelnen Titel lehnt einen
// verliehenen ab (TitelLoeschen).
func (t *Tueren) MarkierteTitelLoeschen() http.HandlerFunc {
	return func(antwort http.ResponseWriter, anfrage *http.Request) {
		var eingabe struct {
			IDs []string `json:"ids"`
		}
		if fehler := json.NewDecoder(anfrage.Body).Decode(&eingabe); fehler != nil {
			schreibeFehler(antwort, http.StatusBadRequest, "Die Anfrage ließ sich nicht lesen")
			return
		}

		if len(eingabe.IDs) == 0 {
			schreibeFehler(antwort, http.StatusBadRequest, "Keine Titel ausgewählt")
			return
		}
		if !alleUUIDs(eingabe.IDs) {
			schreibeFehler(antwort, http.StatusBadRequest, "Eine der Kennungen ist ungültig")
			return
		}

		if fehler := t.titelAbfragen().LoescheTitelSamtAusleihen(anfrage.Context(), eingabe.IDs); fehler != nil {
			if errors.Is(fehler, repository.ErrTitelNichtGefunden) {
				schreibeFehler(antwort, http.StatusNotFound, "Keiner der ausgewählten Titel wurde gefunden")
				return
			}
			slog.Error("Fehler beim Löschen von Büchern", "fehler", fehler)
			schreibeFehler(antwort, http.StatusInternalServerError, "Interner Serverfehler beim Löschen der Bücher")
			return
		}

		schreibeJSON(antwort, http.StatusOK, map[string]string{"message": "Titel gelöscht"})
	}
}
