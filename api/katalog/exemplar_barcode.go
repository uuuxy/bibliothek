package katalog

import (
	"errors"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"
)

// ExemplarBarcodeZiehen zieht die nächste freie interne Nummer (B-XXXXX) und nennt sie als
// JSON. Sie speist den Knopf „Interne ID generieren" in der Exemplarkarte, über den ein
// Exemplar aus dem Altbestand mit Platzhalter (SYS-, AUTO-) seine echte Nummer bekommt.
//
// Die Nummer kommt aus derselben Sequenz wie die Barcodes, die das Bestellwesen vorab vergibt
// (repository/barcode_vergabe.go). Eine Rechnung aus dem Bestand, die höchste Nummer plus
// eins, vergäbe eine Nummer, die die nächste Bestellung noch einmal zieht, und nach einem
// endgültigen Löschen die Nummer eines Etiketts, das noch an einem Buch klebt.
func (t *Tueren) ExemplarBarcodeZiehen() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		barcode, err := repository.ZieheFreienExemplarBarcode(r.Context(), t.DB.Pool)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		tuer.AntworteJSON(w, http.StatusOK, map[string]string{
			"next_barcode": barcode,
		})
	}
}

// meldungLMFNummerAlsBarcode lehnt eine LMF-Nummer als Barcode ab: Ein Buch trägt eine kurze
// Nummer, die LMF-Nummer seines alten Etiketts scannt über lmf_barcode weiter.
const meldungLMFNummerAlsBarcode = "Eine LMF-Nummer kann kein Barcode eines Buchs mehr sein; „Interne ID generieren“ vergibt eine kurze Nummer"

// exemplarBarcodeAnfrage ist der neue Barcode eines Exemplars.
type exemplarBarcodeAnfrage struct {
	Barcode string `json:"barcode"`
}

// ExemplarBarcodeSetzen setzt den Barcode eines Exemplars, etwa statt eines Platzhalters.
func (t *Tueren) ExemplarBarcodeSetzen(katalogAbfragen *repository.KatalogAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t.setzeExemplarBarcode(w, r, katalogAbfragen)
	}
}

// setzeExemplarBarcode setzt den Barcode eines Exemplars und ordnet die Ablehnungen ein.
func (t *Tueren) setzeExemplarBarcode(w http.ResponseWriter, r *http.Request, katalogAbfragen *repository.KatalogAbfragen) {
	id := r.PathValue("id")
	if id == "" {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die Kennung des Exemplars fehlt", nil)
		return
	}

	var req exemplarBarcodeAnfrage
	if !tuer.LeseAnfrage(w, r, &req) {
		return
	}

	if req.Barcode == "" {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Der Barcode darf nicht leer sein", nil)
		return
	}

	ctx := r.Context()

	if err := katalogAbfragen.SetzeExemplarBarcode(ctx, id, req.Barcode); err != nil {
		if errors.Is(err, repository.ErrExemplarNichtGefunden) {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, meldungExemplarNichtGefunden, err)
			return
		}
		// Eine Nummer ist entweder Buch oder Ausweis: Die Theke löst einen Scan zuerst als Buch
		// auf; trüge ein Buch die Nummer eines Ausweises, lüde dieser Ausweis das Buch.
		if repository.IstNummerBuchOderAusweisKollision(err) {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusConflict, "Diese Nummer ist der Ausweis eines Lesers und kann kein Barcode eines Buchs sein", nil)
			return
		}
		if repository.IstExemplarBarcodeVergeben(err) {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusConflict, "Diesen Barcode trägt bereits ein anderes Exemplar", nil)
			return
		}
		if repository.IstLMFNummerAlsBarcode(err) {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, meldungLMFNummerAlsBarcode, nil)
			return
		}
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}

	tuer.AntworteErfolg(w)
}
