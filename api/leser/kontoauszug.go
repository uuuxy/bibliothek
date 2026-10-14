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

// ladeKontoauszugBuecher lädt die aktuell ausgeliehenen Bücher eines Schülers
// (Frist aufsteigend).
func ladeKontoauszugBuecher(ctx context.Context, pool db.PgxPoolIface, schuelerID uuid.UUID) ([]pdf.KontoauszugBuch, error) {
	zeilen, err := repository.LadeKontoauszugBuecher(ctx, pool, schuelerID)
	if err != nil {
		return nil, err
	}
	var buecher []pdf.KontoauszugBuch
	for _, z := range zeilen {
		buecher = append(buecher, pdf.KontoauszugBuch(z))
	}
	return buecher, nil
}

// KontoauszugPDF liefert den Kontoauszug eines Lesers als PDF: alle Bücher, die er gerade
// ausgeliehen hat.
func (t *Tueren) KontoauszugPDF() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		idText := r.PathValue("schueler_id")
		schuelerID, err := uuid.Parse(idText)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		kopf, err := repository.LadeKontoauszugKopf(ctx, t.DB.Pool, schuelerID)
		schueler := pdf.KontoauszugSchueler(kopf)
		if errors.Is(err, pgx.ErrNoRows) {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("schüler nicht gefunden"))
			return
		}
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		buecher, err := ladeKontoauszugBuecher(ctx, t.DB.Pool, schuelerID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if len(buecher) == 0 {
			apierrors.SendHTTPError(w, http.StatusNotFound, fmt.Errorf("keine aktiven Ausleihen gefunden"))
			return
		}

		pdfDaten, err := pdf.GenerateKontoauszug(schueler, buecher)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		dateiname := fmt.Sprintf("Kontoauszug_%s_%s.pdf", schueler.Vorname, schueler.Nachname)

		w.Header().Set(tuer.HeaderContentType, tuer.ContentTypePDF)
		w.Header().Set(tuer.HeaderContentDisposition, fmt.Sprintf(`inline; filename="%s"`, dateiname))
		w.Header().Set(tuer.HeaderContentLength, fmt.Sprint(len(pdfDaten)))

		http.ServeContent(w, r, dateiname, time.Now(), bytes.NewReader(pdfDaten))
	}
}
