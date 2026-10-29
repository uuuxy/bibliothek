package katalog

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
)

// MarkierteTitelLoeschen ist die Tür DELETE /api/books: Sie löscht die in der Titelliste
// markierten Titel ({"ids": [...]}) samt allem, was an ihnen hängt, auch verliehene Exemplare
// (TitelAbfragen.LoescheTitelSamtAusleihen). Die Tür für einen einzelnen Titel lehnt einen
// verliehenen ab (TitelLoeschen). Im Protokoll steht, wer gelöscht hat, wie dort.
func (t *Tueren) MarkierteTitelLoeschen() http.HandlerFunc {
	return func(antwort http.ResponseWriter, anfrage *http.Request) {
		var eingabe struct {
			IDs []string `json:"ids"`
		}
		if fehler := json.NewDecoder(anfrage.Body).Decode(&eingabe); fehler != nil {
			apierrors.AntworteFehlerMitMeldung(antwort, http.StatusBadRequest, "Die Anfrage ließ sich nicht lesen", nil)
			return
		}

		if len(eingabe.IDs) == 0 {
			apierrors.AntworteFehlerMitMeldung(antwort, http.StatusBadRequest, "Keine Titel ausgewählt", nil)
			return
		}
		if !alleUUIDs(eingabe.IDs) {
			apierrors.AntworteFehlerMitMeldung(antwort, http.StatusBadRequest, "Eine der Kennungen ist ungültig", nil)
			return
		}
		claims, ok := auth.ClaimsAusKontext(anfrage.Context())
		if !ok {
			apierrors.AntworteFehlerMitMeldung(antwort, http.StatusUnauthorized, auth.MeldungNichtAngemeldet, nil)
			return
		}

		if fehler := t.titelAbfragen().LoescheTitelSamtAusleihen(anfrage.Context(), eingabe.IDs, claims.KontoID); fehler != nil {
			if errors.Is(fehler, repository.ErrTitelNichtGefunden) {
				apierrors.AntworteFehlerMitMeldung(antwort, http.StatusNotFound, "Keiner der ausgewählten Titel wurde gefunden", nil)
				return
			}
			slog.Error("Fehler beim Löschen von Büchern", "fehler", fehler)
			apierrors.AntworteFehlerMitMeldung(antwort, http.StatusInternalServerError, "Interner Serverfehler beim Löschen der Bücher", nil)
			return
		}

		tuer.AntworteJSON(antwort, http.StatusOK, map[string]string{"message": "Titel gelöscht"})
	}
}
