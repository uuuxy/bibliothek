package leser

import (
	"bytes"
	"fmt"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/internal/auskunft"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// DSGVOAuskunftPDF liefert die Betroffenenauskunft nach Art. 15 DSGVO als
// lesbares PDF. Inhaltlich identisch zur JSON-Variante (dieselbe sammleDSGVODaten-
// Quelle), nur menschenlesbar aufbereitet — gedacht zur Aushändigung an die Person bzw.
// die Erziehungsberechtigten (JSON ist für diesen Zweck ungeeignet). Die Erteilung wird
// wie bei der JSON-Auskunft im Protokoll festgehalten (Rechenschaftspflicht).
// @Summary      DSGVO-Betroffenenauskunft (Art. 15) als PDF
// @Tags         students
// @Produce      application/pdf
// @Param        id   path      string  true  "Kennung des Lesers (UUID)"
// @Success      200  {file}    binary
// @Router       /schueler/{id}/dsgvo-auskunft/pdf [get]
func (t *Tueren) DSGVOAuskunftPDF() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest("missing student ID parameter", nil)
		}
		ctx := r.Context()

		daten, err := t.sammleDSGVODaten(ctx, id, t.HatRecht(r, dsgvoKontoRecht))
		if err != nil {
			return err
		}

		einstellungen, err := repository.NewEinstellungsAbfragen(t.DB.Pool).Lade(ctx)
		if err != nil {
			return apierrors.Internal(tuer.MeldungEinstellungenUnlesbar, err)
		}
		schule := tuer.SchuleAus(einstellungen)

		pdfDaten, err := auskunft.ErzeugeBlatt(dsgvoAntwort(daten, schulzeit.Jetzt()), schule)
		if err != nil {
			return apierrors.Internal("PDF-Erzeugung fehlgeschlagen", err)
		}

		// Rechenschaftspflicht: Die erteilte Auskunft steht im Protokoll, wie bei der Auskunft
		// am Bildschirm. Erst hier, wenn das Blatt steht: Der Eintrag sagt, dass es erteilt ist.
		t.protokolliereDSGVOAuskunft(ctx, id)

		w.Header().Set(tuer.HeaderContentType, tuer.ContentTypePDF)
		w.Header().Set(tuer.HeaderContentDisposition, `attachment; filename="DSGVO-Auskunft.pdf"`)
		w.Header().Set(tuer.HeaderContentLength, fmt.Sprint(len(pdfDaten)))
		http.ServeContent(w, r, "DSGVO-Auskunft.pdf", schulzeit.Jetzt(), bytes.NewReader(pdfDaten))
		return nil
	})
}
