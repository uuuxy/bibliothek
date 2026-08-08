package api

import (
	"errors"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// GetBestelldetailHandler liefert EINE Bestellung mit allem, was zu ihr gehört.
//
// Warum ein eigener Endpunkt und nicht mehr Felder in /api/bestellhistorie: Diese Liste
// ist auf 200 Bestellungen gedeckelt, WEIL sie ohne Grenze 2,45 MB und knapp vier
// Sekunden gebraucht hat. Autor, Verlag, Cover und die Exemplarliste an jede der 200
// Zeilen zu hängen, holte genau dieses Problem zurück — für Daten, von denen jeweils
// eine einzige Bestellung angesehen wird.
//
// Kein SQL in dieser Datei: Ein neuer Handler nimmt repository/ (siehe schichtung_test.go).
func (s *Server) GetBestelldetailHandler(repo repository.BestelldetailRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("bestell-id fehlt"))
			return
		}

		detail, err := repo.GetBestellungDetail(r.Context(), id)
		if err != nil {
			// 404 statt 500: Eine unbekannte ID ist keine Störung, sondern ein veralteter
			// Verweis — und der Sanitizer macht aus jeder 500 ohnehin "interner
			// Datenbankfehler", womit die Ursache im Formular nie ankäme.
			if errors.Is(err, pgx.ErrNoRows) {
				apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("bestellung nicht gefunden"))
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, detail)
	}
}
