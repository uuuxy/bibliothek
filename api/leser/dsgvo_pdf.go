package leser

import (
	"bytes"
	"fmt"
	"net/http"

	"bibliothek/api/tuer"
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
func (s *Tueren) DsgvoAuskunftPDFHandler() http.HandlerFunc {
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

		settings, err := repository.NewEinstellungsAbfragen(s.DB.Pool).Lade(ctx)
		if err != nil {
			return apierrors.Internal(tuer.MeldungEinstellungenUnlesbar, err)
		}
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

		// Rechenschaftspflicht: Die erteilte Auskunft steht im Protokoll, wie bei der Auskunft
		// am Bildschirm. Erst hier, wenn das Blatt steht: Der Eintrag sagt, dass es erteilt ist.
		s.protokolliereDsgvoAuskunft(ctx, id)

		w.Header().Set(tuer.HeaderContentType, tuer.ContentTypePDF)
		w.Header().Set(tuer.HeaderContentDisposition, `attachment; filename="DSGVO-Auskunft.pdf"`)
		w.Header().Set(tuer.HeaderContentLength, fmt.Sprint(len(pdfBytes)))
		http.ServeContent(w, r, "DSGVO-Auskunft.pdf", schulzeit.Jetzt(), bytes.NewReader(pdfBytes))
		return nil
	})
}
