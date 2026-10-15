package katalog

import (
	"errors"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// titelSignaturAnfrage ist die Eingabe für TitelSignaturSetzen.
type titelSignaturAnfrage struct {
	Signatur string `json:"signatur"`
}

// TitelSignaturSetzen ändert nur die Signatur eines Titels — schlank wie
// ExemplarBarcodeSetzen/ExemplarStatusAendern, statt das große Buchformular
// (PUT /api/books/{id}) für eine einzelne Feldänderung im Bestellkorb zu bemühen.
//
// Genutzt beim Anlegen/Bestellen eines Buchs: Ein DNB-Treffer liefert nur einen
// Signatur-VORSCHLAG (siehe signaturVorschlagAusMetadaten), den das Sekretariat vor
// dem Bestellen noch korrigieren können muss. Ein bereits vorhandener Titel behält
// seine Signatur automatisch (dieser Endpunkt wird dafür nie automatisch aufgerufen,
// nur wenn im Bestellkorb tatsächlich editiert wird).
//
// @Summary      Update a title's signatur (shelf mark)
// @Tags         books
// @Accept       json
// @Produce      json
// @Param        id    path      string                      true  "Title ID"
// @Param        body  body      titelSignaturAnfrage  true  "New signatur"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Router       /buecher/titel/{id}/signatur [put]
func (t *Tueren) TitelSignaturSetzen() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest(tuer.MeldungIDLeer, errors.New("missing id"))
		}

		var req titelSignaturAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return nil
		}
		signatur := strings.TrimSpace(req.Signatur)

		neueSignatur, err := repository.SetzeTitelSignatur(r.Context(), t.DB.Pool, id, signatur)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apierrors.NotFound("Titel nicht gefunden", err)
			}
			return apierrors.Internal("Signatur konnte nicht gespeichert werden", err)
		}

		tuer.AntworteJSON(w, http.StatusOK, map[string]string{
			"id":       id,
			"signatur": neueSignatur,
		})
		return nil
	})
}
