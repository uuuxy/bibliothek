package leser

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/apierrors"
	"bibliothek/internal/db"
	"bibliothek/internal/pdf"
	"bibliothek/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ladeKontoauszugBuecher lädt die aktuell ausgeliehenen Bücher und Geräte eines Lesers
// (Frist aufsteigend).
func ladeKontoauszugBuecher(ctx context.Context, pool db.Pool, schuelerID uuid.UUID) ([]pdf.KontoauszugBuch, error) {
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
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, tuer.MeldungSchuelerKennungUngueltig, err)
			return
		}

		kopf, err := repository.LadeKontoauszugKopf(ctx, t.DB.Pool, schuelerID)
		schueler := pdf.KontoauszugSchueler(kopf)
		if errors.Is(err, pgx.ErrNoRows) {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, meldungLeserNichtGefunden, nil)
			return
		}
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		buecher, err := ladeKontoauszugBuecher(ctx, t.DB.Pool, schuelerID)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		if len(buecher) == 0 {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, "Keine laufenden Ausleihen", nil)
			return
		}

		einstellungen, err := repository.NewEinstellungsAbfragen(t.DB.Pool).Lade(ctx)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, fmt.Errorf("einstellungen lesen: %w", err))
			return
		}

		pdfDaten, err := pdf.ErzeugeKontoauszug(schueler, buecher, tuer.SchuleAus(einstellungen))
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		dateiname := fmt.Sprintf("Kontoauszug_%s_%s.pdf", schueler.Vorname, schueler.Nachname)

		tuer.AntworteDatei(w, tuer.ContentTypePDF, tuer.Zeigen, dateiname, pdfDaten)
	}
}
