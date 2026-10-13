package druck

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// MaxAusweisLayoutBytes begrenzt das Design (Base64-Logos können groß werden).
const MaxAusweisLayoutBytes = 5 << 20 // 5 MiB

// GetAusweisLayoutHandler liefert das gespeicherte Ausweis-Design als JSON.
// Ist noch keines gespeichert, wird "{}" zurückgegeben, damit das Frontend sauber
// auf seine Defaults zurückfällt.
//
// Ein Lesefehler ist kein „noch keines": Der Designer speichert nach einem leeren Objekt
// seine Vorgabewerte, für alle Arbeitsplätze. Der Fehler wird deshalb vor dem leeren Wert
// geprüft — nach einem gescheiterten Scan ist der Wert immer leer.
func (s *Tueren) GetAusweisLayoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wert, err := repository.LadeAusweisLayout(r.Context(), s.DB.Pool)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			wert = "{}"
		case err != nil:
			apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("ausweis-design laden: %w", err))
			return
		case strings.TrimSpace(wert) == "":
			wert = "{}"
		}
		w.Header().Set(tuer.HeaderContentType, "application/json; charset=utf-8")
		_, _ = w.Write([]byte(wert)) //nolint:errcheck // Antwort bereits committet
	}
}

// SaveAusweisLayoutHandler speichert das Ausweis-Design (validiertes JSON) zentral.
func (s *Tueren) SaveAusweisLayoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, MaxAusweisLayoutBytes+1))
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("konnte die Anfrage nicht lesen"))
			return
		}
		if len(body) > MaxAusweisLayoutBytes {
			apierrors.SendHTTPError(w, http.StatusRequestEntityTooLarge, errors.New("Ausweis-Design ist zu groß"))
			return
		}
		if !json.Valid(body) {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("ungültiges JSON"))
			return
		}

		if err := repository.SchreibeAusweisLayout(r.Context(), s.DB.Pool, string(body)); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("Ausweis-Design konnte nicht gespeichert werden"))
			return
		}

		tuer.RespondJSON(w, http.StatusOK, map[string]string{"status": "gespeichert"})
	}
}
