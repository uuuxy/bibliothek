package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

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

		var beschreibung string
		var betrag float64
		var erstelltAm time.Time
		var sVorname, sNachname, sKlasse string
		var tTitel, eBarcode string

		query := `
			SELECT 
				sf.beschreibung, sf.betrag, sf.erstellt_am,
				s.vorname, s.nachname, s.klasse,
				t.titel, e.barcode_id
			FROM schadensfaelle sf
			JOIN schueler s ON sf.schueler_id = s.id
			JOIN buecher_exemplare e ON sf.exemplar_id = e.id
			JOIN buecher_titel t ON e.titel_id = t.id
			WHERE sf.id = $1
		`

		err := s.DB.Pool.QueryRow(ctx, query, id).Scan(
			&beschreibung, &betrag, &erstelltAm,
			&sVorname, &sNachname, &sKlasse,
			&tTitel, &eBarcode,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierrors.SendHTTPError(w, http.StatusNotFound, err)
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
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

		info := pdf.SchadensfallInfo{
			Beschreibung:     beschreibung,
			Betrag:           betrag,
			ErstelltAm:       erstelltAm,
			SchuelerVorname:  sVorname,
			SchuelerNachname: sNachname,
			SchuelerKlasse:   sKlasse,
			BuchTitel:        tTitel,
			ExemplarBarcode:  eBarcode,
		}

		pdfBytes, err := pdf.GenerateSchadensfallPDF(info, schule)
		if err != nil {
			log.Printf("PDF Generator: Generation error for case %s: %v", id, err)
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("failed to generate PDF"))
			return
		}

		// Update database flag indicating that the letter was generated
		updateQuery := `
			UPDATE schadensfaelle
			SET elternbrief_generiert = true,
			    elternbrief_generiert_am = CURRENT_TIMESTAMP,
			    aktualisiert_am = CURRENT_TIMESTAMP
			WHERE id = $1
		`
		_, dbErr := s.DB.Pool.Exec(ctx, updateQuery, id)
		if dbErr != nil {
			log.Printf("PDF Generator: Database status update failed for case %s: %v", id, dbErr)
		}

		// Stream the generated PDF
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=elternbrief_%s.pdf", sNachname))

		if _, err := w.Write(pdfBytes); err != nil {
			log.Printf("PDF Generator: Output error: %v", err)
			return
		}
	}
}
