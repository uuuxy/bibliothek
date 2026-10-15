package bestellung

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/internal/buchetikett"
	"bibliothek/pdf"
	"bibliothek/pkg/httpresp"
	"bibliothek/pkg/mitteltopf"
	"bibliothek/repository"
)

// Die Etikettenseite des Lieferanten-Links. Sie druckt genau die Etiketten, die auch im
// Mailanhang lagen — dieselben Barcodes, dieselbe Auswahl, dieselben Generatoren
// (klein = Bogen wie im Druck-Center, groß = Lernmittel-Etikett). Zwei Wege zum selben
// Buch dürfen nicht zwei verschiedene Aufkleber ergeben.

// OeffentlicheEtiketten liefert den Etikettenbogen als PDF (ohne Login, per Token).
func (t *Tueren) OeffentlicheEtiketten() http.HandlerFunc {
	return t.gibOeffentlicheEtiketten
}

// gibOeffentlicheEtiketten prüft Größe, Format und Token und liefert den Bogen der Bestellung.
func (t *Tueren) gibOeffentlicheEtiketten(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	groesse := r.PathValue("groesse")
	if groesse != "klein" && groesse != "gross" {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die Größe der Etiketten muss klein oder groß sein", nil)
		return
	}

	// Der Etikettenbogen muss zu den Bögen des Lieferanten passen, nicht zu unseren: Er
	// klebt die Etiketten auf sein eigenes Material, und davon gibt es verschiedene
	// Rastergrößen.
	//
	// Unbekannte Werte werden abgewiesen und nicht still auf die Vorgabe gedreht: Sonst
	// druckt der Lieferant nach einem Tippfehler im falschen Raster und merkt es erst am
	// verschnittenen Bogen. Leer heißt Vorgabe.
	format := r.URL.Query().Get("format")
	if format == "" {
		format = pdf.EtikettFormatVorgabe
	}
	if _, bekannt := pdf.FindeEtikettFormat(format); !bekannt {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, fmt.Sprintf("Unbekanntes Format der Etiketten: %q", format), nil)
		return
	}

	bestellungID, err := t.bestellungPerToken(ctx, r.PathValue("token"))
	if err != nil {
		sendeTokenFehler(w, err)
		return
	}

	// Das große Lernmittel-Etikett gibt es zu einer Bestellung für die Schülerbücherei
	// nicht — auch nicht für den, der die Adresse kennt. Die Seite blendet den Knopf
	// aus; die Tür ist es, die das Etikett verweigert.
	if groesse == "gross" {
		mittel, err := repository.LadeMittelDerBestellung(ctx, t.DB.Pool, bestellungID)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}
		if !mitteltopf.HatGrossesLernmittelEtikett(mittel) {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, "Zu dieser Bestellung gibt es kein großes Lernmittel-Etikett", nil)
			return
		}
	}

	etiketten, err := buchetikett.LadeDerBestellung(ctx, t.DB.Pool, bestellungID)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}
	if len(etiketten) == 0 {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, "Zu dieser Bestellung gibt es keine Etiketten", nil)
		return
	}

	daten, err := t.baueEtikettenPDF(ctx, groesse, format, etiketten)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set(tuer.HeaderContentType, tuer.ContentTypePDF)
	// inline: Der Lieferant soll den Bogen im Browser sehen und direkt drucken können.
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"etiketten_%s.pdf\"", groesse))
	httpresp.Schreibe(w, daten)
}

// baueEtikettenPDF erzeugt den Bogen in der gewünschten Größe.
//
// format gilt nur für "klein" — das große Lernmittel-Etikett hat ein festes Raster
// (2×2 auf A4) und wird ausgeschnitten, nicht auf vorgestanzte Bögen gedruckt.
func (t *Tueren) baueEtikettenPDF(ctx context.Context, groesse, format string, etiketten []buchetikett.Etikett) ([]byte, error) {
	eingabe := buchetikett.PDFEtiketten(etiketten, buchetikett.LadeKopf(ctx, t.DB.Pool))

	if groesse == "gross" {
		return pdf.ErzeugeLernmittelEtiketten(eingabe)
	}

	doc, err := pdf.ErzeugeBuchEtiketten(format, 1, false, eingabe)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := doc.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
