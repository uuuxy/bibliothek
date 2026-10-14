package leser

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/db"
	"bibliothek/pdf"
	"bibliothek/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// queryKontoauszugBuecher lädt die aktuell ausgeliehenen Bücher eines Schülers
// (Frist aufsteigend).
func queryKontoauszugBuecher(ctx context.Context, dbPool db.PgxPoolIface, schuelerID uuid.UUID) ([]pdf.KontoauszugBuch, error) {
	zeilen, err := repository.LadeKontoauszugBuecher(ctx, dbPool, schuelerID)
	if err != nil {
		return nil, err
	}
	var buecher []pdf.KontoauszugBuch
	for _, z := range zeilen {
		buecher = append(buecher, pdf.KontoauszugBuch(z))
	}
	return buecher, nil
}

// PrintKontoauszugHandler generates a simple receipt of all currently borrowed books for a student.
func (s *Tueren) PrintKontoauszugHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		idStr := r.PathValue("schueler_id")
		schuelerID, err := uuid.Parse(idStr)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		kopf, err := repository.LadeKontoauszugKopf(ctx, s.DB.Pool, schuelerID)
		schueler := pdf.KontoauszugSchueler(kopf)
		if errors.Is(err, pgx.ErrNoRows) {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("schüler nicht gefunden"))
			return
		}
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		buecher, err := queryKontoauszugBuecher(ctx, s.DB.Pool, schuelerID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if len(buecher) == 0 {
			apierrors.SendHTTPError(w, http.StatusNotFound, fmt.Errorf("keine aktiven Ausleihen gefunden"))
			return
		}

		pdfBytes, err := pdf.GenerateKontoauszug(schueler, buecher)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		filename := fmt.Sprintf("Kontoauszug_%s_%s.pdf", schueler.Vorname, schueler.Nachname)

		w.Header().Set(tuer.HeaderContentType, tuer.ContentTypePDF)
		w.Header().Set(tuer.HeaderContentDisposition, fmt.Sprintf(`inline; filename="%s"`, filename))
		w.Header().Set(tuer.HeaderContentLength, fmt.Sprint(len(pdfBytes)))

		http.ServeContent(w, r, filename, time.Now(), bytes.NewReader(pdfBytes))
	}
}
