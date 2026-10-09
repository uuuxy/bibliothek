package api

import (
	"bytes"
	"fmt"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/internal/auskunft"
	"bibliothek/pdf"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// DsgvoAuskunftPDFHandler liefert die Betroffenenauskunft nach Art. 15 DSGVO als
// lesbares PDF. Inhaltlich identisch zur JSON-Variante (dieselbe sammleDsgvoDaten-
// Quelle), nur menschenlesbar aufbereitet — gedacht zur Aushändigung an die Person bzw.
// die Erziehungsberechtigten (JSON ist für diesen Zweck ungeeignet). Die Erteilung wird
// wie bei der JSON-Auskunft im Audit-Log protokolliert (Rechenschaftspflicht).
// @Summary      DSGVO-Betroffenenauskunft (Art. 15) als PDF
// @Tags         students
// @Produce      application/pdf
// @Param        id   path      string  true  "Reader ID (UUID)"
// @Success      200  {file}    binary
// @Router       /schueler/{id}/dsgvo-auskunft/pdf [get]
func (s *Server) DsgvoAuskunftPDFHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest("missing student ID parameter", nil)
		}
		ctx := r.Context()

		daten, err := s.sammleDsgvoDaten(ctx, id, s.BesitztRecht(r, dsgvoKontoRecht))
		if err != nil {
			return err
		}

		// Rechenschaftspflicht: Auskunftserteilung protokollieren (wie JSON-Variante).
		s.protokolliereDsgvoAuskunft(ctx, id)

		settingsRepo := repository.NewSystemSettingsRepository(s.DB.Pool)
		settings, _ := settingsRepo.GetSettings(ctx) //nolint:errcheck // Header fällt sonst auf Defaults zurück
		schule := pdf.SchuleInfo{
			Name:    settings.SchuleName,
			Strasse: settings.SchuleStrasse,
			PLZ:     settings.SchulePLZ,
			Ort:     settings.SchuleOrt,
		}

		pdfBytes, err := auskunft.GenerateDsgvoAuskunftPDF(dsgvoAntwort(daten, schulzeit.Jetzt()), schule)
		if err != nil {
			return apierrors.Internal("PDF-Erzeugung fehlgeschlagen", err)
		}

		w.Header().Set(headerContentType, contentTypePDF)
		w.Header().Set(headerContentDisposition, `attachment; filename="DSGVO-Auskunft.pdf"`)
		w.Header().Set(headerContentLength, fmt.Sprint(len(pdfBytes)))
		http.ServeContent(w, r, "DSGVO-Auskunft.pdf", schulzeit.Jetzt(), bytes.NewReader(pdfBytes))
		return nil
	})
}
