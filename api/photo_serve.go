package api

import (
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/internal/crypto"
)

// ServeStudentPhotoHandler lädt das verschlüsselte Foto eines Schülers aus der
// Datenbank, entschlüsselt es im Arbeitsspeicher und liefert es sicher an den Client aus.
func (s *Server) ServeStudentPhotoHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// {barcode_id}, nicht {id}: siehe Routen-Registrierung in routes_students.go —
		// unter dem Namen "id" hätte die UUID-Prüfung jeden Barcode abgewiesen.
		barcodeID := strings.TrimSpace(r.PathValue("barcode_id"))
		if barcodeID == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("fehlende schueler_id/barcode_id"))
			return
		}

		// Zuerst die UUID des Schülers anhand der Barcode-ID herausfinden und das Foto fetchen
		query := `
			SELECT sf.foto_encrypted 
			FROM schueler_fotos sf
			JOIN schueler s ON s.id = sf.schueler_id
			WHERE s.barcode_id = $1
		`

		var ciphertext []byte
		err := s.DB.Pool.QueryRow(ctx, query, barcodeID).Scan(&ciphertext)
		if errors.Is(err, pgx.ErrNoRows) {
			apierrors.SendHTTPError(w, http.StatusNotFound, fmt.Errorf("kein foto gefunden"))
			return
		}
		if err != nil {
			// Fehler-Kollaps (Sweep 29.08.2026): Ein DB-Fehler war vorher „kein foto".
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// Foto on the fly entschlüsseln und mit Sicherheits-Headern streamen
		if err := crypto.DecryptAndServe(w, ciphertext, "image/webp"); err != nil {
			// DecryptAndServe sendet bereits einen HTTP Error, wenn es hier fehlschlägt
			return
		}
	}
}
