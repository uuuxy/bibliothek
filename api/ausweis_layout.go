package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"bibliothek/apierrors"

	"github.com/jackc/pgx/v5"
)

// ausweisLayoutKey ist der Schlüssel des Ausweis-Designs im Key-Value-Store
// system_einstellungen (wert TEXT). Das Design wird zentral gehalten, damit alle
// vernetzten Arbeitsplätze (Ausleihe vorn, Druck im Hintergrundbüro) exakt denselben
// Stand sehen — localStorage wäre bei mehreren PCs eine Sackgasse.
const ausweisLayoutKey = "ausweis_layout"

// maxAusweisLayoutBytes begrenzt das Design (Base64-Logos können groß werden).
const maxAusweisLayoutBytes = 5 << 20 // 5 MiB

// GetAusweisLayoutHandler liefert das gespeicherte Ausweis-Design als JSON.
// Ist noch keines gespeichert, wird "{}" zurückgegeben, damit das Frontend sauber
// auf seine Defaults zurückfällt.
func (s *Server) GetAusweisLayoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var wert string
		err := s.DB.Pool.QueryRow(r.Context(),
			`SELECT wert FROM system_einstellungen WHERE schluessel = $1`, ausweisLayoutKey).Scan(&wert)
		switch {
		case errors.Is(err, pgx.ErrNoRows) || strings.TrimSpace(wert) == "":
			wert = "{}"
		case err != nil:
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("Ausweis-Design konnte nicht geladen werden"))
			return
		}
		w.Header().Set(headerContentType, "application/json; charset=utf-8")
		_, _ = w.Write([]byte(wert)) //nolint:errcheck // Antwort bereits committet
	}
}

// SaveAusweisLayoutHandler speichert das Ausweis-Design (validiertes JSON) zentral.
func (s *Server) SaveAusweisLayoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, maxAusweisLayoutBytes+1))
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("konnte die Anfrage nicht lesen"))
			return
		}
		if len(body) > maxAusweisLayoutBytes {
			apierrors.SendHTTPError(w, http.StatusRequestEntityTooLarge, errors.New("Ausweis-Design ist zu groß"))
			return
		}
		if !json.Valid(body) {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("ungültiges JSON"))
			return
		}

		if _, err := s.DB.Pool.Exec(r.Context(),
			`INSERT INTO system_einstellungen (schluessel, wert, aktualisiert_am)
			 VALUES ($1, $2, CURRENT_TIMESTAMP)
			 ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert, aktualisiert_am = CURRENT_TIMESTAMP`,
			ausweisLayoutKey, string(body)); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("Ausweis-Design konnte nicht gespeichert werden"))
			return
		}

		RespondJSON(w, http.StatusOK, map[string]string{"status": "gespeichert"})
	}
}
