package druck

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/apierrors"
	"bibliothek/internal/buchetikett"
	"bibliothek/internal/pdf"
	"bibliothek/internal/pkg/mitteltopf"
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
// Browser nur Barcode, Titel und Autor schickt.
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

		ctx := r.Context()
		buchetikett.ErgaenzeServerfelder(ctx, t.DB.Pool, req.Etiketten)
		kopf := buchetikett.LadeKopf(ctx, t.DB.Pool)

		if req.FormatID == pdf.LernmittelEtikettFormat {
			antworteLernmittelEtiketten(w, req.Etiketten, kopf)
			return
		}

		bogen, err := pdf.ErzeugeBuchEtiketten(req.FormatID, req.Startposition, req.AlsQR, buchetikett.PDFEtiketten(req.Etiketten, kopf))
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, fmt.Errorf("fehler bei der pdf generierung: %w", err))
			return
		}

		tuer.SetzeDateiKopf(w, tuer.ContentTypePDF, tuer.Zeigen, "etiketten_custom.pdf")

		if err := bogen.Output(w); err != nil {
			slog.Error("Fehler beim Senden des PDFs", "fehler", err)
		}
	}
}

// antworteLernmittelEtiketten liefert das große Lernmittel-Etikett mit demselben Erzeuger und
// derselben Regel wie die Seite des Lieferanten: Für ein Buch der Schülerbücherei gibt es keins
// (mitteltopf.HatGrossesLernmittelEtikett). Die Tür lehnt dann den ganzen Auftrag ab und nennt die
// Barcodes, statt sie still auszulassen; die Vorschau zeigt die Meldung vor dem Druck.
func antworteLernmittelEtiketten(w http.ResponseWriter, etiketten []buchetikett.Etikett, kopf buchetikett.Kopf) {
	var ohne []string
	for _, e := range etiketten {
		if !mitteltopf.HatGrossesLernmittelEtikett(e.Topf) {
			ohne = append(ohne, e.BarcodeID)
		}
	}
	if len(ohne) > 0 {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest,
			"Bücher der Schülerbücherei bekommen kein großes Lernmittel-Etikett: "+barcodeAufzaehlung(ohne), nil)
		return
	}

	daten, err := pdf.ErzeugeLernmittelEtiketten(buchetikett.PDFEtiketten(etiketten, kopf))
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}
	tuer.AntworteDatei(w, tuer.ContentTypePDF, tuer.Zeigen, "etiketten_gross.pdf", daten)
}

// barcodeAufzaehlung nennt höchstens drei Barcodes und den Rest als Zahl, damit die Meldung in
// die Zeile unter der Vorschau passt.
func barcodeAufzaehlung(barcodes []string) string {
	const hoechstens = 3
	if len(barcodes) <= hoechstens {
		return strings.Join(barcodes, ", ")
	}
	return fmt.Sprintf("%s und %d weitere", strings.Join(barcodes[:hoechstens], ", "), len(barcodes)-hoechstens)
}
