package schaden

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/pdf"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// ElternbriefPDF liefert den Elternbrief zu einem Schadensfall als PDF und vermerkt am Fall,
// dass er erzeugt ist.
func (t *Tueren) ElternbriefPDF() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.AntworteFehler(w, http.StatusBadRequest, errors.New("missing damage case ID parameter"))
			return
		}

		ctx := r.Context()

		info, aufBescheid, err := t.ladeSchadensfallBrief(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierrors.AntworteFehler(w, http.StatusNotFound, err)
				return
			}
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}
		// Steht die Forderung auf einem Schadensersatz-Bescheid, gilt der Bescheid: Ein
		// Elternbrief daneben nennte sie ein zweites Mal, mit eigener Frist und eigenem
		// Zahlungsweg. Dieselbe Regel wie bei der Ersatzforderung
		// (repository.LadeOffeneForderungenOhneBescheid).
		if aufBescheid {
			apierrors.AntworteFehler(w, http.StatusConflict, errors.New(
				"die Forderung steht auf einem Schadensersatz-Bescheid — dafür gilt der Bescheid, kein Elternbrief"))
			return
		}

		// Ein Lesefehler ist ein Fehler: Mit den Vorgaben nennte der Brief keine Schule und
		// die Bankverbindung der Vorgabe statt der eingetragenen.
		einstellungen, err := repository.NewEinstellungsAbfragen(t.DB.Pool).Lade(ctx)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, fmt.Errorf("einstellungen lesen: %w", err))
			return
		}
		schule := pdf.SchuleInfo{
			Name:    einstellungen.SchuleName,
			Strasse: einstellungen.SchuleStrasse,
			PLZ:     einstellungen.SchulePLZ,
			Ort:     einstellungen.SchuleOrt,
		}

		// Zahlstelle und Bankverbindung des Landes aus derselben Einstellung wie im
		// Bescheid — eine zweite Kontoangabe im selben Haus wäre eine zweite Wahrheit.
		angaben := repository.BescheidAngabenAus(einstellungen)
		zahlung := pdf.Zahlungsangaben{Zahlstelle: angaben.Zahlstelle, Bankverbindung: angaben.Bankverbindung}

		pdfDaten, err := pdf.GenerateSchadensfallPDF(info, schule, zahlung)
		if err != nil {
			log.Printf("PDF Generator: Generation error for case %s: %v", id, err)
			apierrors.AntworteFehler(w, http.StatusInternalServerError, errors.New("failed to generate PDF"))
			return
		}

		t.merkeElternbriefErzeugt(ctx, id)

		// Das PDF senden.
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=elternbrief_%s.pdf", info.SchuelerNachname))

		// #nosec G705 - der Körper ist das erzeugte PDF (application/pdf, als Anhang, nosniff); aus der Anfrage stammt nur die Kennung des Falls
		if _, err := w.Write(pdfDaten); err != nil {
			log.Printf("PDF Generator: Output error: %v", err)
			return
		}
	}
}

// ladeSchadensfallBrief lädt den Fall für den Elternbrief und sagt dazu, ob die Forderung
// schon auf einem Schadensersatz-Bescheid steht (dann gibt es keinen Brief).
func (t *Tueren) ladeSchadensfallBrief(ctx context.Context, id string) (pdf.SchadensfallInfo, bool, error) {
	fall, aufBescheid, err := repository.LadeSchadensfallBrief(ctx, t.DB.Pool, id)
	if err != nil {
		return pdf.SchadensfallInfo{}, false, err
	}

	return pdf.SchadensfallInfo{
		Beschreibung:     fall.Beschreibung,
		Betrag:           fall.Betrag,
		ErstelltAm:       fall.ErstelltAm,
		SchuelerVorname:  fall.SchuelerVorname,
		SchuelerNachname: fall.SchuelerNachname,
		SchuelerKlasse:   fall.SchuelerKlasse,
		Strasse:          fall.Strasse,
		Hausnummer:       fall.Hausnummer,
		PLZ:              fall.PLZ,
		Ort:              fall.Ort,
		BuchTitel:        fall.Titel,
		ExemplarBarcode:  fall.Barcode,
		Land:             fall.Land,
	}, aufBescheid, nil
}

func (t *Tueren) merkeElternbriefErzeugt(ctx context.Context, id string) {
	if err := repository.MerkeElternbriefErzeugt(ctx, t.DB.Pool, id); err != nil {
		log.Printf("PDF Generator: Database status update failed for case %s: %v", id, err)
	}
}
