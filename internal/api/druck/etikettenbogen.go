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
	"bibliothek/internal/repository"
)

// etikettenbogenAnfrage ist ein Druckauftrag aus dem Druck-Center: Format, Startposition und
// die Etiketten.
type etikettenbogenAnfrage struct {
	FormatID      string                `json:"formatId"`
	Startposition int                   `json:"startPosition"`
	AlsQR         bool                  `json:"isQR"`
	Etiketten     []buchetikett.Etikett `json:"items"`
}

// MaxBuchEtiketten deckelt einen Druckauftrag. Jedes Etikett kostet ein Strichcode-Bild, und die
// Erzeugung läuft weiter, wenn die Anfrage abgelaufen ist; einige Aufträge mit 150.000 Einträgen
// zugleich legten den Server lahm. 5000 Etiketten sind über 200 Bögen.
const MaxBuchEtiketten = 5000

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
		if len(req.Etiketten) > MaxBuchEtiketten {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, fmt.Sprintf("Höchstens %d Etiketten je Druckauftrag, gewählt sind %d",
				MaxBuchEtiketten, len(req.Etiketten)), nil)
			return
		}

		ctx := r.Context()
		buchetikett.ErgaenzeServerfelder(ctx, t.DB.Pool, req.Etiketten)
		if req.FormatID == pdf.SignaturEtikettFormat {
			antworteSignaturEtiketten(w, req.Startposition, req.Etiketten)
			return
		}
		kopf := buchetikett.LadeKopf(ctx, t.DB.Pool)

		if req.FormatID == pdf.LernmittelEtikettFormat {
			antworteLernmittelEtiketten(w, req.Etiketten, kopf)
			return
		}

		// Ein unbekanntes Format druckte sonst still auf den Bogen der Vorgabe, und das fiele erst
		// am verdruckten Bogen auf; dieselbe Regel wie bei den Schüler-Etiketten.
		if fehler := pruefeBogenParameter(req.FormatID, &req.Startposition); fehler != nil {
			apierrors.AntworteFehler(w, http.StatusBadRequest, fehler)
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

// antworteLernmittelEtiketten liefert das große Lernmittel-Etikett mit demselben Erzeuger wie die
// Seite des Lieferanten, für ein Buch aus jedem Topf: Wer druckt, wählt die Größe. Den
// Eigentumsvermerk wählt der Kopf je Exemplar nach dem Topf.
func antworteLernmittelEtiketten(w http.ResponseWriter, etiketten []buchetikett.Etikett, kopf buchetikett.Kopf) {
	daten, err := pdf.ErzeugeLernmittelEtiketten(buchetikett.PDFEtiketten(etiketten, kopf))
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}
	tuer.AntworteDatei(w, tuer.ContentTypePDF, tuer.Zeigen, "etiketten_gross.pdf", daten)
}

// antworteSignaturEtiketten liefert die Signaturetiketten für den Buchrücken. Die Signatur kommt
// vom Titel (buchetikett.ErgaenzeServerfelder); ohne sie bliebe das Etikett leer. Die Tür lehnt
// dann den ganzen Auftrag ab und nennt die Barcodes, statt sie still auszulassen.
func antworteSignaturEtiketten(w http.ResponseWriter, startposition int, etiketten []buchetikett.Etikett) {
	var ohne []string
	aufschriften := make([][]string, 0, len(etiketten))
	for _, e := range etiketten {
		if strings.TrimSpace(e.Signatur) == "" {
			ohne = append(ohne, e.BarcodeID)
		}
		aufschriften = append(aufschriften, repository.SignaturZeilen(e.Signatur))
	}
	if len(ohne) > 0 {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest,
			"Ohne Signatur am Titel gibt es kein Signatur-Etikett: "+barcodeAufzaehlung(ohne), nil)
		return
	}

	bogen := pdf.ErzeugeSignaturEtiketten(startposition, aufschriften)
	tuer.SetzeDateiKopf(w, tuer.ContentTypePDF, tuer.Zeigen, "etiketten_signatur.pdf")
	if err := bogen.Output(w); err != nil {
		slog.Error("Fehler beim Senden des PDFs", "fehler", err)
	}
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
