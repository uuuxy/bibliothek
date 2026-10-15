package katalog

import (
	"errors"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"
)

// ExemplarBarcodeZiehen returns the next available internal B-XXXXX barcode as JSON.
//
// Speist den Knopf „Interne ID generieren" in der Exemplarkarte — der Weg, auf dem ein
// Altbestands-Exemplar mit Platzhalter (SYS-/AUTO-) seine echte Nummer bekommt.
//
// Die Nummer kommt aus derselben Sequenz wie die Vorab-Barcodes des Bestellwesens
// (repository/barcode_vergabe.go). Früher rechnete dieser Handler stattdessen
// MAX(Nummer)+1 aus dem Bestand. Das hatte zwei Folgen: Die so vergebene Nummer zog die
// nächste Bestellung erneut (UNIQUE-Verletzung, die die komplette Bestellung kippte),
// und nach einem harten Löschen — Titel löschen, Massenlöschung, „endgültig löschen" im
// Fehlbestand — kam die freigewordene Nummer zurück, obwohl ihr Etikett noch irgendwo
// klebte.
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

// exemplarBarcodeAnfrage holds the payload for updating a copy's barcode.
type exemplarBarcodeAnfrage struct {
	Barcode string `json:"barcode"`
}

// ExemplarBarcodeSetzen updates the barcode of a physical book copy.
// @Summary      Update copy barcode
// @Description  Updates the barcode of a physical book copy, replacing placeholders like AUTO-.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id    path      string                true  "Book copy ID (UUID)"
// @Param        body  body      exemplarBarcodeAnfrage  true  "New barcode payload"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string
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
		// Eine Nummer ist entweder Buch oder Ausweis (Migration 131): Die Theke löst
		// einen Scan zuerst als Buch auf — trüge ein Buch die Nummer eines Ausweises,
		// lüde dieser Ausweis das Buch.
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
