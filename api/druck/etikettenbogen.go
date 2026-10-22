package druck

import (
	"fmt"
	"log"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/internal/buchetikett"
	"bibliothek/pdf"
)

// etikettenbogenAnfrage ist ein Druckauftrag aus dem Druck-Center: Format, Startposition und
// die Etiketten.
type etikettenbogenAnfrage struct {
	FormatID      string                `json:"formatId"`
	Startposition int                   `json:"startPosition"`
	AlsQR         bool                  `json:"isQR"`
	Etiketten     []buchetikett.Etikett `json:"items"`
}

// Etikettenbogen liefert die Etiketten eines Druckauftrags als Bogen im PDF. Signatur,
// Anschaffungsjahr und Topf trägt der Server nach (buchetikett.ErgaenzeServerfelder), weil der
// Browser nur Barcode, Titel und Autor schickt. Einen Auftrag mit einer LMF-Nummer lehnt er ab und
// nennt die Nummern (409, langeNummernAntwort): Für ein neues Etikett bekommt ein solches
// Exemplar erst eine kurze Nummer (VergebeNeueNummern).
func (t *Tueren) Etikettenbogen() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req etikettenbogenAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return
		}

		if len(req.Etiketten) == 0 {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Keine Exemplare angegeben", nil)
			return
		}
		nummern := make([]string, 0, len(req.Etiketten))
		for _, e := range req.Etiketten {
			nummern = append(nummern, e.BarcodeID)
		}
		if lang := buchetikett.NichtDruckbare(nummern); len(lang) > 0 {
			tuer.AntworteJSON(w, http.StatusConflict, langeNummernAntwort{Error: meldungLangeNummern, Nummern: lang})
			return
		}

		ctx := r.Context()
		buchetikett.ErgaenzeServerfelder(ctx, t.DB.Pool, req.Etiketten)

		bogen, err := pdf.ErzeugeBuchEtiketten(req.FormatID, req.Startposition, req.AlsQR, buchetikett.PDFEtiketten(req.Etiketten, buchetikett.LadeKopf(ctx, t.DB.Pool)))
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, fmt.Errorf("fehler bei der pdf generierung: %w", err))
			return
		}

		w.Header().Set(tuer.HeaderContentType, tuer.ContentTypePDF)
		w.Header().Set(tuer.HeaderContentDisposition, "inline; filename=\"etiketten_custom.pdf\"")

		if err := bogen.Output(w); err != nil {
			log.Printf("Fehler beim Senden des PDFs: %v", err)
		}
	}
}
