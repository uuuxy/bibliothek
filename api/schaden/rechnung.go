package schaden

import (
	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/db"
	"bibliothek/pdf"
	"bibliothek/repository"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/google/uuid"
)

// ladeRechnungPosten lädt die offenen Forderungen eines Schülers ohne Bescheid als
// Rechnungspositionen (repository.LadeOffeneForderungenOhneBescheid).
func ladeRechnungPosten(ctx context.Context, datenbank db.Pool, schuelerID uuid.UUID) ([]pdf.RechnungsPosition, error) {
	posten, err := repository.LadeOffeneForderungenOhneBescheid(ctx, datenbank, schuelerID)
	if err != nil {
		return nil, err
	}
	var zeilen []pdf.RechnungsPosition
	for _, p := range posten {
		zeilen = append(zeilen, pdf.RechnungsPosition(p))
	}
	return zeilen, nil
}

// RechnungPDF liefert die Ersatzforderung an die Eltern eines Schülers als PDF: alle offenen
// Forderungen, die auf keinem Bescheid stehen.
func (t *Tueren) RechnungPDF() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		liefereRechnung(w, r, t.DB.Pool)
	}
}

// liefereRechnung lädt Anschrift und offene Forderungen ohne Bescheid und setzt den Brief.
func liefereRechnung(w http.ResponseWriter, r *http.Request, datenbank db.Pool) {
	ctx := r.Context()

	// Die Kennung des Schülers steht im Pfad: /api/print/rechnung/{schueler_id}
	idText := r.PathValue("schueler_id")
	schuelerID, err := uuid.Parse(idText)
	if err != nil {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, tuer.MeldungSchuelerKennungUngueltig, err)
		return
	}

	// Anschrift mitladen: Die Rechnung ist wie der Eltern-Mahnbrief ein Brief für das
	// DIN-Fensterkuvert (VVT-Zweck „gedruckte Rechnung / Elternbrief"). COALESCE ist
	// Pflicht: Die Spalten sind nullbar, die Go-Strings nicht.
	anschrift, err := repository.LadeBriefAnschrift(ctx, datenbank, schuelerID)
	schueler := pdf.Schueler(anschrift)
	if errors.Is(err, pgx.ErrNoRows) {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, "Schüler nicht gefunden", nil)
		return
	}
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}

	zeilen, err := ladeRechnungPosten(ctx, datenbank, schuelerID)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}

	if len(zeilen) == 0 {
		// Stehen die offenen Forderungen alle auf einem Bescheid, sagt die Meldung das.
		// Sonst hieße es „keine offenen Schadensfälle", während die Akte offene Beträge zeigt.
		aufBescheid, err := repository.HatOffeneForderungAufBescheid(ctx, datenbank, schuelerID)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}
		if aufBescheid {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, "Die offenen Forderungen stehen auf einem Schadensersatz-Bescheid. Dafür gilt der Bescheid, keine Ersatzforderung.", nil)
			return
		}
		apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, "Keine offenen Forderungen für diesen Schüler", nil)
		return
	}

	// Ein Lesefehler ist ein Fehler: Mit den Vorgaben nennte das Blatt keine Schule und die
	// Bankverbindung der Vorgabe statt der eingetragenen.
	einstellungen, err := repository.NewEinstellungsAbfragen(datenbank).Lade(ctx)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, fmt.Errorf("einstellungen lesen: %w", err))
		return
	}
	schule := tuer.SchuleAus(einstellungen)

	// Zahlstelle und Bankverbindung des Landes aus derselben Einstellung wie im
	// Bescheid — eine zweite Kontoangabe im selben Haus wäre eine zweite Wahrheit.
	angaben := repository.BescheidAngabenAus(einstellungen)
	zahlung := pdf.Zahlungsangaben{Zahlstelle: angaben.Zahlstelle, Bankverbindung: angaben.Bankverbindung}

	pdfDaten, err := pdf.ErzeugeRechnung(schueler, zeilen, schule, zahlung)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set(tuer.HeaderContentType, tuer.ContentTypePDF)
	w.Header().Set(tuer.HeaderContentDisposition, `inline; filename="Rechnung.pdf"`)
	w.Header().Set(tuer.HeaderContentLength, fmt.Sprint(len(pdfDaten)))

	http.ServeContent(w, r, "Rechnung.pdf", time.Now(), bytes.NewReader(pdfDaten))
}
