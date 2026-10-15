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
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		tuer.AntworteJSON(w, http.StatusOK, map[string]string{
			"next_barcode": barcode,
		})
	}
}

// exemplarBarcodeAnfrage ist der neue Barcode eines Exemplars.
type exemplarBarcodeAnfrage struct {
	Barcode string `json:"barcode"`
}

// ExemplarBarcodeSetzen setzt den Barcode eines Exemplars, etwa statt eines Platzhalters.
//
// @Summary      Barcode eines Exemplars setzen
// @Description  Setzt den Barcode eines Exemplars, etwa statt eines Platzhalters wie AUTO-. Eine Nummer, die schon ein anderes Exemplar oder ein Ausweis trägt, ist ein Konflikt (409).
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id    path      string                true  "Kennung des Exemplars (UUID)"
// @Param        body  body      exemplarBarcodeAnfrage  true  "Der neue Barcode"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Failure      409   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /buecher/exemplare/{id}/barcode [put]
func (t *Tueren) ExemplarBarcodeSetzen(katalogAbfragen repository.KatalogAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t.setzeExemplarBarcode(w, r, katalogAbfragen)
	}
}

// setzeExemplarBarcode setzt den Barcode eines Exemplars und ordnet die Ablehnungen ein.
func (t *Tueren) setzeExemplarBarcode(w http.ResponseWriter, r *http.Request, katalogAbfragen repository.KatalogAbfragen) {
	id := r.PathValue("id")
	if id == "" {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing copy ID parameter"))
		return
	}

	var req exemplarBarcodeAnfrage
	if !tuer.LeseAnfrage(w, r, &req) {
		return
	}

	if req.Barcode == "" {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("barcode cannot be empty"))
		return
	}

	ctx := r.Context()

	if err := katalogAbfragen.SetzeExemplarBarcode(ctx, id, req.Barcode); err != nil {
		if errors.Is(err, repository.ErrExemplarNichtGefunden) {
			apierrors.SendHTTPError(w, http.StatusNotFound, err)
			return
		}
		// Eine Nummer ist entweder Buch oder Ausweis: Die Theke löst einen Scan zuerst als Buch
		// auf; trüge ein Buch die Nummer eines Ausweises, lüde dieser Ausweis das Buch.
		if repository.IstNummerBuchOderAusweisKollision(err) {
			apierrors.SendHTTPError(w, http.StatusConflict, errors.New("diese Nummer ist der Ausweis eines Lesers und kann kein Buch-Barcode sein"))
			return
		}
		if repository.IstExemplarBarcodeVergeben(err) {
			apierrors.SendHTTPError(w, http.StatusConflict, errors.New("dieser Barcode wird bereits von einem anderen Exemplar verwendet"))
			return
		}
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	tuer.AntworteErfolg(w)
}
