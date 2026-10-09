package api

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

// GenerateDamagePDFHandler generates a formal PDF notification letter ("Elternbrief")
// for a student responsible for library book damage, marking the record in the DB.
func (s *Server) GenerateDamagePDFHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing damage case ID parameter"))
			return
		}

		ctx := r.Context()

		info, aufBescheid, err := s.fetchDamageCaseInfo(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierrors.SendHTTPError(w, http.StatusNotFound, err)
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		// Steht die Forderung auf einem Schadensersatz-Bescheid, gilt der Bescheid: Ein
		// Elternbrief daneben nennte sie ein zweites Mal, mit eigener Frist und eigenem
		// Zahlungsweg. Dieselbe Regel wie bei der Ersatzforderung
		// (repository.ListeOffeneForderungenOhneBescheid); docs/OFFEN.md 5.2, 23.09.2026.
		if aufBescheid {
			apierrors.SendHTTPError(w, http.StatusConflict, errors.New(
				"die Forderung steht auf einem Schadensersatz-Bescheid — dafür gilt der Bescheid, kein Elternbrief"))
			return
		}

		settingsRepo := repository.NewSystemSettingsRepository(s.DB.Pool)
		settings, _ := settingsRepo.GetSettings(ctx) //nolint:errcheck
		schule := pdf.SchuleInfo{
			Name:    settings.SchuleName,
			Strasse: settings.SchuleStrasse,
			PLZ:     settings.SchulePLZ,
			Ort:     settings.SchuleOrt,
		}

		// Zahlstelle und Bankverbindung des Landes aus derselben Einstellung wie im
		// Bescheid — eine zweite Kontoangabe im selben Haus wäre eine zweite Wahrheit.
		angaben := repository.BescheidAngabenAus(settings)
		zahlung := pdf.Zahlungsangaben{Zahlstelle: angaben.Zahlstelle, Bankverbindung: angaben.Bankverbindung}

		pdfBytes, err := pdf.GenerateSchadensfallPDF(info, schule, zahlung)
		if err != nil {
			log.Printf("PDF Generator: Generation error for case %s: %v", id, err)
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("failed to generate PDF"))
			return
		}

		s.markElternbriefGenerated(ctx, id)

		// Stream the generated PDF
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=elternbrief_%s.pdf", info.SchuelerNachname))

		// #nosec G705 - der Körper ist das erzeugte PDF (application/pdf, als Anhang, nosniff); aus der Anfrage stammt nur die Kennung des Falls
		if _, err := w.Write(pdfBytes); err != nil {
			log.Printf("PDF Generator: Output error: %v", err)
			return
		}
	}
}

// fetchDamageCaseInfo lädt den Fall für den Elternbrief und sagt dazu, ob die Forderung
// schon auf einem Schadensersatz-Bescheid steht (dann gibt es keinen Brief).
func (s *Server) fetchDamageCaseInfo(ctx context.Context, id string) (pdf.SchadensfallInfo, bool, error) {
	fall, aufBescheid, err := repository.LadeSchadensfallBrief(ctx, s.DB.Pool, id)
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
		BuchTitel:        fall.BuchTitel,
		ExemplarBarcode:  fall.ExemplarBarcode,
		Land:             fall.Land,
	}, aufBescheid, nil
}

func (s *Server) markElternbriefGenerated(ctx context.Context, id string) {
	if dbErr := repository.MerkeElternbriefErzeugt(ctx, s.DB.Pool, id); dbErr != nil {
		log.Printf("PDF Generator: Database status update failed for case %s: %v", id, dbErr)
	}
}
