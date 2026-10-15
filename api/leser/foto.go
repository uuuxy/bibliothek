package leser

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/internal/crypto"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// Foto lädt das verschlüsselte Foto eines Schülers aus der
// Datenbank, entschlüsselt es im Arbeitsspeicher und liefert es an die Oberfläche aus.
func (t *Tueren) Foto() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// {barcode_id}, nicht {id}: siehe Routen-Registrierung in api/routen_leser.go —
		// unter dem Namen "id" hätte die UUID-Prüfung jeden Barcode abgewiesen.
		barcodeID := strings.TrimSpace(r.PathValue("barcode_id"))
		if barcodeID == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("fehlende schueler_id/barcode_id"))
			return
		}

		verschluesselt, err := repository.LadeFotoZurAusweisnummer(ctx, t.DB.Pool, barcodeID)
		if errors.Is(err, pgx.ErrNoRows) {
			apierrors.SendHTTPError(w, http.StatusNotFound, fmt.Errorf("kein foto gefunden"))
			return
		}
		if err != nil {
			// Ein Fehler der Datenbank ist kein „kein Foto": 500 statt 404.
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// Im Arbeitsspeicher entschlüsseln und mit den Sicherheits-Köpfen ausliefern.
		if err := crypto.SchreibeEntschluesselt(w, verschluesselt, "image/webp"); err != nil {
			// SchreibeEntschluesselt hat dann schon mit einem Fehler geantwortet.
			return
		}
	}
}
